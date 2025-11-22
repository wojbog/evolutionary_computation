// local_search_tsp_lm.go
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// -------------------- Data types --------------------

// Node holds coordinate and cost
type Node struct {
	X, Y int
	Cost int
}

// Instance
type Instance struct {
	Nodes []Node
	Dist  [][]int // distance matrix (rounded Euclidean)
	N     int
	K     int // number of nodes to select (ceil(N/2))
}

// Result row to write to CSV
type ResultRow struct {
	Method        string
	Run           int
	Objective     int
	TourLength    int
	SelectedCosts int
	Evals         int
	Improvements  int
	FinalSelected []int
	Seed          int64
}

// -------------------- IO / utilities --------------------

// Utility: read instance CSV of rows: x,y,cost (integers), no header
func ReadInstanceCSV(path string) (*Instance, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = ';'
	all, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	nodes := make([]Node, 0, len(all))
	for i, row := range all {
		// skip empty rows
		if len(row) == 0 {
			continue
		}
		if len(row) < 3 {
			return nil, fmt.Errorf("row %d has fewer than 3 columns", i)
		}
		xs := strings.TrimSpace(row[0])
		ys := strings.TrimSpace(row[1])
		cs := strings.TrimSpace(row[2])

		x, errx := strconv.Atoi(xs)
		y, erry := strconv.Atoi(ys)
		cost, errc := strconv.Atoi(cs)
		if errx != nil || erry != nil || errc != nil {
			// treat as header if it's the first row
			if i == 0 {
				continue
			}
			return nil, fmt.Errorf("row %d: failed to parse integers: %v %v %v", i, errx, erry, errc)
		}
		nodes = append(nodes, Node{X: x, Y: y, Cost: cost})
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no data rows found in %s", path)
	}
	inst := &Instance{Nodes: nodes, N: len(nodes)}
	// compute distance matrix immediately
	inst.Dist = make([][]int, inst.N)
	for i := 0; i < inst.N; i++ {
		inst.Dist[i] = make([]int, inst.N)
		for j := 0; j < inst.N; j++ {
			if i == j {
				inst.Dist[i][j] = 0
			} else {
				dx := float64(inst.Nodes[i].X - inst.Nodes[j].X)
				dy := float64(inst.Nodes[i].Y - inst.Nodes[j].Y)
				dist := math.Hypot(dx, dy)
				inst.Dist[i][j] = int(math.Round(dist))
			}
		}
	}
	// set K = ceil(N/2)
	inst.K = (inst.N + 1) / 2
	return inst, nil
}

func TourLength(dist [][]int, tour []int) int {
	L := 0
	K := len(tour)
	if K == 0 {
		return 0
	}
	for i := 0; i < K; i++ {
		a := tour[i]
		b := tour[(i+1)%K]
		L += dist[a][b]
	}
	return L
}

func SelectedCosts(nodes []Node, tour []int) int {
	sum := 0
	for _, v := range tour {
		sum += nodes[v].Cost
	}
	return sum
}

func countSelected(sel []bool) int {
	c := 0
	for _, v := range sel {
		if v {
			c++
		}
	}
	return c
}

// -------------------- starting solutions --------------------

func RandomStart(inst *Instance, rnd *rand.Rand) ([]int, []bool) {
	N, K := inst.N, inst.K
	all := make([]int, N)
	for i := 0; i < N; i++ {
		all[i] = i
	}
	if N > 1 {
		rnd.Shuffle(N, func(i, j int) { all[i], all[j] = all[j], all[i] })
	}
	selected := all[:K]
	if K > 1 {
		rnd.Shuffle(len(selected), func(i, j int) { selected[i], selected[j] = selected[j], selected[i] })
	}
	inSelected := make([]bool, N)
	for _, s := range selected {
		inSelected[s] = true
	}
	return selected, inSelected
}

// Regret greedy start (robustified)
func GreedyRegretStart(inst *Instance, startNode int) ([]int, []bool) {
	k := inst.K
	D := inst.Dist
	nodes := inst.Nodes
	n := len(nodes)

	if n == 0 {
		return []int{}, []bool{}
	}
	if k <= 1 {
		in := make([]bool, n)
		in[startNode%n] = true
		return []int{startNode % n}, in
	}

	selected := make([]bool, n)
	startNode = startNode % n
	selected[startNode] = true

	bestJ := -1
	bestVal := math.MaxInt
	for j := 0; j < n; j++ {
		if j == startNode {
			continue
		}
		val := D[startNode][j] + nodes[j].Cost
		if val < bestVal {
			bestVal = val
			bestJ = j
		}
	}
	if bestJ == -1 {
		for j := 0; j < n; j++ {
			if j != startNode {
				bestJ = j
				break
			}
		}
		if bestJ == -1 {
			in := make([]bool, n)
			in[startNode] = true
			return []int{startNode}, in
		}
	}
	selected[bestJ] = true
	tour := []int{startNode, bestJ}

	for countSelected(selected) < k {
		type cand struct {
			node, bestTot, secondTot, bestPos int
			score                             float64
		}
		var cands []cand
		for v := 0; v < n; v++ {
			if selected[v] {
				continue
			}
			bestInc, bestPos := bestInsertion(v, tour, D)
			secondInc := math.MaxInt
			for i := 0; i < len(tour); i++ {
				a := tour[i]
				b := tour[(i+1)%len(tour)]
				inc := D[a][v] + D[v][b] - D[a][b]
				if inc < secondInc && i+1 != bestPos {
					secondInc = inc
				}
			}
			if secondInc == math.MaxInt {
				secondInc = bestInc
			}
			bestTot := bestInc + nodes[v].Cost
			secondTot := secondInc + nodes[v].Cost
			regret := secondTot - bestTot
			alpha := 1.0
			beta := 1.0
			score := alpha*float64(regret) - beta*float64(bestTot)
			cands = append(cands, cand{v, bestTot, secondTot, bestPos, score})
		}
		if len(cands) == 0 {
			break
		}
		sort.Slice(cands, func(a, b int) bool { return cands[a].score > cands[b].score })
		ch := cands[0]
		selected[ch.node] = true
		tour = insertAt(tour, ch.bestPos, ch.node)
	}
	return tour, selected
}

// -------------------- move delta helpers --------------------

func mod(a, b int) int {
	v := a % b
	if v < 0 {
		v += b
	}
	return v
}

func bestInsertion(node int, tour []int, D [][]int) (int, int) {
	best := math.MaxInt
	bestPos := 0
	m := len(tour)
	for i := 0; i < m; i++ {
		a := tour[i]
		b := tour[(i+1)%m]
		inc := D[a][node] + D[node][b] - D[a][b]
		if inc < best {
			best = inc
			bestPos = i + 1
		}
	}
	return best, bestPos
}

func insertAt(tour []int, pos int, node int) []int {
	if pos < 0 {
		pos = 0
	}
	if pos > len(tour) {
		pos = len(tour)
	}
	newT := append([]int{}, tour[:pos]...)
	newT = append(newT, node)
	newT = append(newT, tour[pos:]...)
	return newT
}

func deltaReplaceAtPos(dist [][]int, nodes []Node, tour []int, pos int, u int) int {
	K := len(tour)
	s := tour[pos]
	prev := tour[mod(pos-1, K)]
	next := tour[mod(pos+1, K)]
	deltaLen := dist[prev][u] + dist[u][next] - dist[prev][s] - dist[s][next]
	deltaCost := nodes[u].Cost - nodes[s].Cost
	return deltaLen + deltaCost
}

func deltaSwapPositions(dist [][]int, nodes []Node, tour []int, i int, j int) int {
	if i == j {
		return 0
	}
	if i > j {
		i, j = j, i
	}
	K := len(tour)
	A := tour[i]
	B := tour[j]
	Aprev := tour[mod(i-1, K)]
	Anext := tour[mod(i+1, K)]
	Bprev := tour[mod(j-1, K)]
	Bnext := tour[mod(j+1, K)]

	deltaLen := 0
	if i == 0 && j == K-1 {
		deltaLen += dist[Bprev][A] + dist[A][B] + dist[B][Anext]
		deltaLen -= dist[Bprev][B] + dist[B][A] + dist[A][Anext]
	} else if mod(i+1, K) == j {
		deltaLen += dist[Aprev][B] + dist[B][A] + dist[A][Bnext]
		deltaLen -= dist[Aprev][A] + dist[A][B] + dist[B][Bnext]
	} else {
		deltaLen += dist[Aprev][B] + dist[B][Anext] + dist[Bprev][A] + dist[A][Bnext]
		deltaLen -= dist[Aprev][A] + dist[A][Anext] + dist[Bprev][B] + dist[B][Bnext]
	}
	return deltaLen
}

func delta2Opt(dist [][]int, nodes []Node, tour []int, i int, j int) int {
	K := len(tour)
	if i == j {
		return 0
	}
	// if i > j {
	// 	i, j = j, i
	// }
	ai1 := tour[i]
	ai := tour[mod(i-1, K)]
	aj := tour[j]
	aj1 := tour[mod(j+1, K)]
	if ai1 == aj {
		return 0
	}
	if aj1 == ai {
		return 0
	}
	if ai == aj {
		return 0
	}

	deltaLen := dist[ai][aj] + dist[ai1][aj1] - dist[ai][ai1] - dist[aj][aj1]
	return deltaLen
}

func FindIndexTour(tour []int, node int) int {
	for i, v := range tour {
		if v == node {
			return i
		}
	}
	return -1
}

// -------------------- candidates builder --------------------

func buildCandidates(nodes []Node, dist [][]int, K int) [][]int {
	n := len(nodes)
	cands := make([][]int, n)
	for i := 0; i < n; i++ {
		type pair struct {
			v   int
			key int
		}
		arr := []pair{}
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			arr = append(arr, pair{v: j, key: dist[i][j] + nodes[j].Cost})
		}
		sort.Slice(arr, func(a, b int) bool { return arr[a].key < arr[b].key })
		limit := K
		if len(arr) < limit {
			limit = len(arr)
		}
		cands[i] = make([]int, limit)
		for k := 0; k < limit; k++ {
			cands[i][k] = arr[k].v
		}
	}
	return cands
}

// -------------------- Move LM structures --------------------

type MoveType int

const (
	MoveIntraSwap MoveType = iota
	MoveIntra2Opt
	MoveInterReplace
)

type MoveState int

const (
	MoveInvalid MoveState = iota
	MoveReversedOrFuture
	MoveApplicable
)

// Edge pair (u->v)
type Edge struct {
	U int
	V int
}

// Move stores minimal info to re-validate and apply the move
type Move struct {
	Type       MoveType
	PosA       int    // for intra moves: positions in tour (i)
	PosB       int    // second position (j)
	NodeU      int    // used for inter: candidate node to bring in
	Removed    []Edge // edges removed by the move (ordered as they were when move created)
	Delta      int
	CreatedFor []int // optional snapshot of tour positions (not used in validation)
}

// Check if directed edge u->v exists in tour and whether it's reversed
// returns (exists, reversed) where reversed=true if found as v->u (opposite direction)
func edgeExistsInTour(tour []int, u, v int) (bool, bool) {
	K := len(tour)
	for i := 0; i < K; i++ {
		a := tour[i]
		b := tour[(i+1)%K]
		if a == u && b == v {
			return true, false
		}
		if a == v && b == u {
			return true, true
		}
	}
	return false, false
}

// Validate a move according to the three situations described.
// - If any removed edge is absent -> MoveInvalid
// - If all removed edges present and none reversed -> MoveApplicable
// - If all removed edges present but at least one reversed -> MoveReversedOrFuture
func (m *Move) Validate(tour []int) MoveState {
	if len(m.Removed) == 0 {
		// No removed edges recorded: treat conservatively as Applicable so it can be applied
		// but in practice every move should have removed edges.
		return MoveApplicable
	}
	allPresent := true
	atLeastOneReversed := false
	for _, e := range m.Removed {
		ok, _ := edgeExistsInTour(tour, e.U, e.V)
		if !ok {
			allPresent = false
			break
		}
		// if rev {
		// 	atLeastOneReversed = true
		// }
	}
	if !allPresent {
		return MoveInvalid
	}
	if atLeastOneReversed {
		return MoveReversedOrFuture
	}
	if m.Type == MoveInterReplace && FindIndexTour(tour, m.NodeU) != -1 {
		return MoveReversedOrFuture
	}
	return MoveApplicable
}

// Apply move to tour and inSel. It assumes the move is applicable.
func applyMoveToSolution(m Move, tour []int, inSel []bool) {
	K := len(tour)
	if K == 0 {
		return
	}

	// helper to swap positions of two nodes (by value)
	swapNodes := func(aVal, bVal int) {
		ai := FindIndexTour(tour, aVal)
		bi := FindIndexTour(tour, bVal)
		if ai == -1 || bi == -1 {
			log.Fatalf("applyMoveToSolution: could not find nodes %d or %d in tour", aVal, bVal)
		}
		tour[ai], tour[bi] = tour[bi], tour[ai]
	}

	switch m.Type {
	case MoveIntraSwap:
		// BuildRemovedEdgesForSwap produces either 3 (adjacent/wrap) or 4 (non-adjacent) edges.
		if len(m.Removed) == 3 {
			// middle edge connects the two swapped nodes (A->B or B->A)
			A := m.Removed[1].U
			B := m.Removed[1].V
			swapNodes(A, B)
			return
		} else if len(m.Removed) == 4 {
			// removed: Aprev->A, A->Anext, Bprev->B, B->Bnext
			A := m.Removed[0].V
			B := m.Removed[2].V
			swapNodes(A, B)
			return
		}
		log.Fatal("applyMoveToSolution: unexpected removed-edges pattern for intra-swap")

	case MoveIntra2Opt:
		// // BuildRemovedEdgesFor2Opt returns [ai->ai1, aj->aj1]
		// if len(m.Removed) != 2 {
		// 	log.Fatal("applyMoveToSolution: unexpected removed-edges pattern for 2-opt")
		// }
		// ai1 := m.Removed[0].V
		// aj := m.Removed[1].U
		// posA := FindIndexTour(tour, ai1)
		// posB := FindIndexTour(tour, aj)
		// if posA == -1 || posB == -1 {
		// 	log.Fatalf("applyMoveToSolution: could not find nodes for 2-opt: %d or %d", ai1, aj)
		// }
		// if posA > posB {
		// 	posA, posB = posB, posA
		// }
		// // reverse segment [posA .. posB]
		// for a, b := posA, posB; a < b; a, b = a+1, b-1 {
		// 	tour[a], tour[b] = tour[b], tour[a]
		// }
		// return
		// BuildRemovedEdgesFor2Opt returns [ai->ai1, aj->aj1]
        // We want to reverse the segment starting at ai1 and ending at aj.
        if len(m.Removed) != 2 {
            log.Fatal("applyMoveToSolution: unexpected removed-edges pattern for 2-opt")
        }
    
		ai := m.Removed[0].U
        ai1 := m.Removed[0].V

        aj := m.Removed[1].U
		aj1 := m.Removed[1].V

		posai := FindIndexTour(tour, ai)
		posai1 := FindIndexTour(tour, ai1)

		var posA int
		var posB int

		if posai > posai1 {
			posA = posai
		} else {
			posA = posai1
		}

		posaj := FindIndexTour(tour, aj)
		posaj1 := FindIndexTour(tour, aj1)

		if posaj1 < posaj && posaj1 != 0 {
			posB = posaj1
		} else {
			posB = posaj
		}

        
        if posA == -1 || posB == -1 {
            log.Fatalf("applyMoveToSolution: could not find nodes for 2-opt: %d or %d", ai1, aj)
        }

        // Standard 2-opt reverses the segment FROM ai1 TO aj.
        if posA <= posB {
            // Linear case: ai1 comes before aj in array. 
            // Reverse the segment [posA ... posB]
            for a, b := posA, posB; a < b; a, b = a+1, b-1 {
                tour[a], tour[b] = tour[b], tour[a]
            }
        } else {
            // Wrap-around case: ai1 is later in the array than aj.
            // The segment wraps over the end. 
            // Reversing the wrap-around segment is topologically equivalent 
            // to reversing the "linear" complement segment: [aj1 ... ai].
            // aj1 is at posB+1, ai is at posA-1.
            
            // Note: We must handle bounds if posB is last or posA is first, 
            // but posA > posB implies posA > 0 and posB < N-1 usually.
            // If posB is the last element, posB+1 overflows? 
            // But aj is m.Removed[1].U, so aj->aj1 exists. 
            // If aj is last, aj1 is 0. But posA > posB implies posA != 0.
            
            start := posB + 1
            end := posA - 1
            
            // Reverse the complement segment
            for a, b := start, end; a < b; a, b = a+1, b-1 {
                tour[a], tour[b] = tour[b], tour[a]
            }
        }
        return

	case MoveInterReplace:
		// BuildRemovedEdgesForReplace returns [prev->s, s->next], so removed node s is m.Removed[0].V (or m.Removed[1].U).
		if len(m.Removed) != 2 {
			log.Fatal("applyMoveToSolution: unexpected removed-edges pattern for replace")
		}
		s := m.Removed[0].V
		pos := FindIndexTour(tour, s)
		if pos == -1 {
			log.Fatalf("applyMoveToSolution: could not find replaced node %d in tour", s)
		}
		u := m.NodeU
		// safety: ensure u not already selected
		if inSel[u] {
			// try to find an alternative: if u already selected, still replace position (caller should ensure validity)
			// but log warning
			log.Printf("applyMoveToSolution: replacing with node %d that is already selected", u)
		}
		inSel[s] = false
		inSel[u] = true
		tour[pos] = u
		return

	default:
		log.Fatal("applyMoveToSolution: unknown move type")
	}
}

// Build removed edges list for a move given the current tour and move type.
// This ensures the edges are recorded in the direction as they appear when created.
func BuildRemovedEdgesForSwap(tour []int, i int, j int) []Edge {
	K := len(tour)
	if i > j {
		i, j = j, i
	}
	A := tour[i]
	B := tour[j]
	Aprev := tour[mod(i-1, K)]
	Anext := tour[mod(i+1, K)]
	Bprev := tour[mod(j-1, K)]
	Bnext := tour[mod(j+1, K)]

	edges := []Edge{}
	if mod(i+1, K) == j { // adjacent case
		// old edges: Aprev-A, A-B, B-Bnext
		edges = append(edges, Edge{U: Aprev, V: A})
		edges = append(edges, Edge{U: A, V: B})
		edges = append(edges, Edge{U: B, V: Bnext})
	} else if i == 0 && j == K-1 {
		// adjacent in circular way: ... B - A ...
		// old edges: Bprev-B, B-A, A-Anext
		edges = append(edges, Edge{U: Bprev, V: B})
		edges = append(edges, Edge{U: B, V: A})
		edges = append(edges, Edge{U: A, V: Anext})
	} else {
		// non-adjacent: old edges: Aprev-A, A-Anext, Bprev-B, B-Bnext
		edges = append(edges, Edge{U: Aprev, V: A})
		edges = append(edges, Edge{U: A, V: Anext})
		edges = append(edges, Edge{U: Bprev, V: B})
		edges = append(edges, Edge{U: B, V: Bnext})
	}
	return edges
}

func BuildRemovedEdgesFor2Opt(tour []int, i int, j int) []Edge {
	K := len(tour)
	if i > j {
		i, j = j, i
	}
	ai := tour[i]
	ai1 := tour[mod(i+1, K)]
	aj := tour[j]
	aj1 := tour[mod(j+1, K)]
	// removed edges: ai-ai1, aj-aj1 (in that order)
	return []Edge{{U: ai, V: ai1}, {U: aj, V: aj1}}
}

func BuildRemovedEdgesForReplace(tour []int, pos int) []Edge {
	K := len(tour)
	s := tour[pos]
	prev := tour[mod(pos-1, K)]
	next := tour[mod(pos+1, K)]
	// removed edges: prev - s, s - next
	return []Edge{{U: prev, V: s}, {U: s, V: next}}
}

// sorting LM moves by delta (ascending)
func sortMovesByDelta(moves []Move) {
	sort.Slice(moves, func(i, j int) bool {
		return moves[i].Delta < moves[j].Delta
	})
}

// delete move at index from slice
func deleteMoveAtIndex(moves []Move, index int) []Move {
	if index ==0 {
		return moves[1:]
	} else if index == len(moves)-1 {
		return moves[:len(moves)-1]
	}
	return append(moves[:index], moves[index+1:]...)
}

func FillLm(inst *Instance, tour []int, inSel []bool, intraMode string, LM []Move) []Move {
	K := inst.K
	dist := inst.Dist
	nodes := inst.Nodes

	if intraMode == "nodes" {
		for i := 0; i < K; i++ {
			for j := i + 1; j < K; j++ {
				delta := deltaSwapPositions(dist, nodes, tour, i, j)
				if delta < 0 {
					// record move
					m := Move{
						Type:    MoveIntraSwap,
						PosA:    i,
						PosB:    j,
						Delta:   delta,
						Removed: BuildRemovedEdgesForSwap(tour, i, j),
					}
					LM = append(LM, m)
				}
			}
		}
	} else {
		for i := 0; i < K; i++ {
			for j := i + 1; j < K; j++ {
				delta := delta2Opt(dist, nodes, tour, i, j)
				if delta < 0 {
					m := Move{
						Type:    MoveIntra2Opt,
						PosA:    i,
						PosB:    j,
						Delta:   delta,
						Removed: BuildRemovedEdgesFor2Opt(tour, i, j),
					}
					LM = append(LM, m)
				}
			}
		}
		// Inter moves: consider all non-selected nodes (no candidate lists)
		n := len(nodes)
		for pos := 0; pos < K; pos++ {
			for u := 0; u < n; u++ {
				if inSel[u] {
					continue
				}
				delta := deltaReplaceAtPos(dist, nodes, tour, pos, u)
				if delta < 0 {
					m := Move{
						Type:    MoveInterReplace,
						PosA:    pos,
						NodeU:   u,
						Delta:   delta,
						Removed: BuildRemovedEdgesForReplace(tour, pos),
					}
					LM = append(LM, m)
				}
			}
		}

	}
	sortMovesByDelta(LM)
return LM
}

// -------------------- Local Search: Steepest with LM --------------------

// LocalSearchSteepestWithLM: tries LM first (validate and apply/apply/remove as described)
// If LM yields nothing applicable, scans neighborhood to find best improving move,
// applies it and adds it to LM for next iterations.
// Returns (changed bool, evals, improvements)
func LocalSearchSteepestWithLM(inst *Instance, tour []int, inSel []bool, intraMode string, evalLimit int, cand [][]int, rnd *rand.Rand, LM []Move) (bool, int, int, []Move) {
	// K := inst.K
	dist := inst.Dist
	nodes := inst.Nodes

	// We'll create LM local to this call and allow it to be returned if the caller wants to persist.
	// However the RunLocalSearch will maintain LM across iterations by passing it back between calls.
	// LM := make([]Move, 0)

	// initial evaluation: evaluate the full neighborhood once to populate LM with applicable improving moves
	evals := 0
	improvements := 0
	// bestDelta := 0
	// var bestMove Move

	// Evaluate all moves for the initial tour and add improving moves to LM.
	// In practice, this step can be limited or randomized; here we follow the assignment: evaluate currently applicable moves and their inverted edges.
	// Intra moves
	if len(LM) == 0 {

		LM = FillLm(inst, tour, inSel, intraMode, LM)
	}

	lengthLM := len(LM)


	for idxM :=0; idxM < lengthLM; idxM++ {
		if idxM >= len(LM) {
			break
		}
		state := LM[idxM].Validate(tour)
		if state == MoveInvalid {
			// drop it from LM
			LM = deleteMoveAtIndex(LM, idxM)
			
			idxM--
			// fmt.Println("Dropping invalid move from LM with delta")
		} else if state == MoveReversedOrFuture {
			// keep it, but do not apply now
			// fmt.Println("Keeping move from LM with delta")
			continue
		} else if state == MoveApplicable {
			

			var delta int
			if LM[idxM].Type == MoveInterReplace {
				sIdx := FindIndexTour(tour, LM[idxM].Removed[0].V)
				if sIdx == -1 {
					fmt.Print(LM[idxM].Removed[0].V)
					log.Fatalf("Error: LM move inter-replace node %d NOT in tour at position %d", LM[idxM].Removed[0].V, sIdx)
				}
				delta = deltaReplaceAtPos(dist, nodes, tour, sIdx, LM[idxM].NodeU)
			} else if LM[idxM].Type == MoveIntraSwap {
				log.Fatal("Error: LM move intra-swap should not be re-evaluated here")
				delta = deltaSwapPositions(dist, nodes, tour, LM[idxM].PosA, LM[idxM].PosB)
			} else if LM[idxM].Type == MoveIntra2Opt {
        		ai := LM[idxM].Removed[0].U
				ai1 := LM[idxM].Removed[0].V

				aj := LM[idxM].Removed[1].U
				aj1 := LM[idxM].Removed[1].V

				posai := FindIndexTour(tour, ai)
				posai1 := FindIndexTour(tour, ai1)
				var posA int
				var posB int

				if posai > posai1  {
					posA = posai
				} else {
					posA = posai1
				}

				posaj := FindIndexTour(tour, aj)
				posaj1 := FindIndexTour(tour, aj1)

				if posaj1 < posaj && posaj1 != 0 {
					posB = posaj1
				} else {
					posB = posaj
				}

				delta = delta2Opt(dist, nodes, tour, posA, posB)
			} else {
				log.Fatalf("Unknown move type in LM: %d", LM[idxM].Type)
			}

			if delta >= 0 {
				LM = deleteMoveAtIndex(LM, idxM)
				idxM--
				continue
			}




			applyMoveToSolution(LM[idxM], tour, inSel)
			LM = deleteMoveAtIndex(LM, idxM)
			idxM--
				

			improvements++
			// applied = true
			// After applying one LM move we follow the assignment: "perform (apply) the move and remove from LM"
			// Stop after first applied LM move to re-evaluate neighborhood next iteration.
			// Note: Could apply multiple LM moves in one go, but safer to apply one and re-validate.
			// Return updated LM without this move (others are in newLM).
			// Also do not re-add this move.
			// We return now with updated LM (newLM).
			return true, evals, improvements, LM
		}
	}

	// fmt.Println("No applicable LM moves found, scanning neighborhood for best move")
	// bestDelta := 0
	// var bestMove Move

	// if intraMode == "nodes" {
	// 	for i := 0; i < K; i++ {
	// 		for j := i + 1; j < K; j++ {
	// 			delta := deltaSwapPositions(dist, nodes, tour, i, j)
	// 			evals++
	// 			if delta < 0 {
	// 				// record move
	// 				m := Move{
	// 					Type:    MoveIntraSwap,
	// 					PosA:    i,
	// 					PosB:    j,
	// 					Delta:   delta,
	// 					Removed: BuildRemovedEdgesForSwap(tour, i, j),
	// 				}
	// 				if delta < bestDelta {
	// 					bestDelta = delta
	// 					bestMove = m
	// 				}
	// 				LM = append(LM, m)
	// 			}
	// 			if evalLimit > 0 && evals >= evalLimit {
	// 				return false, evals, improvements, LM
	// 			}
	// 		}
	// 	}
	// } else {
	// 	for i := 0; i < K; i++ {
	// 		for j := i + 1; j < K; j++ {
	// 			delta := delta2Opt(dist, nodes, tour, i, j)
	// 			evals++
	// 			if delta < 0 {
	// 				m := Move{
	// 					Type:    MoveIntra2Opt,
	// 					PosA:    i,
	// 					PosB:    j,
	// 					Delta:   delta,
	// 					Removed: BuildRemovedEdgesFor2Opt(tour, i, j),
	// 				}
	// 				if delta < bestDelta {
	// 					bestDelta = delta
	// 					bestMove = m
	// 				}
	// 				LM = append(LM, m)
	// 			}
	// 			if evalLimit > 0 && evals >= evalLimit {
	// 				return false, evals, improvements, LM
	// 			}
	// 		}
	// 	}
	// 	// Inter moves: consider all non-selected nodes (no candidate lists)
	// 	n := len(nodes)
	// 	for pos := 0; pos < K; pos++ {
	// 		for u := 0; u < n; u++ {
	// 			if inSel[u] {
	// 				continue
	// 			}
	// 			delta := deltaReplaceAtPos(dist, nodes, tour, pos, u)
	// 			evals++
	// 			if delta < 0 {
	// 				m := Move{
	// 					Type:    MoveInterReplace,
	// 					PosA:    pos,
	// 					NodeU:   u,
	// 					Delta:   delta,
	// 					Removed: BuildRemovedEdgesForReplace(tour, pos),
	// 				}
	// 				if delta < bestDelta {
	// 					bestDelta = delta
	// 					bestMove = m
	// 				}
	// 				LM = append(LM, m)
	// 			}
	// 			if evalLimit > 0 && evals >= evalLimit {
	// 				return false, evals, improvements, LM
	// 			}
	// 		}
	// 	}

	// }
	
	// if bestDelta < 0 {
	// 	sortMovesByDelta(LM)
	// 	applyMoveToSolution(bestMove, tour, inSel)
	// 	// LM = deleteMoveAtIndex(LM, idxM)
	// 	// idxM--
	// 	improvements++
	// 	return true, evals, improvements, LM
	// } 
	LM = FillLm(inst, tour, inSel, intraMode, LM)


	lengthLM = len(LM)


	for idxM :=0; idxM < lengthLM; idxM++ {
		if idxM >= len(LM) {
			break
		}
		state := LM[idxM].Validate(tour)
		if state == MoveInvalid {
			// drop it from LM
			LM = deleteMoveAtIndex(LM, idxM)
			
			idxM--
			// fmt.Println("Dropping invalid move from LM with delta")
		} else if state == MoveReversedOrFuture {
			// keep it, but do not apply now
			// fmt.Println("Keeping move from LM with delta")
			continue
		} else if state == MoveApplicable {
			

			var delta int
			if LM[idxM].Type == MoveInterReplace {
				sIdx := FindIndexTour(tour, LM[idxM].Removed[0].V)
				if sIdx == -1 {
					fmt.Print(LM[idxM].Removed[0].V)
					log.Fatalf("Error: LM move inter-replace node %d NOT in tour at position %d", LM[idxM].Removed[0].V, sIdx)
				}
				delta = deltaReplaceAtPos(dist, nodes, tour, sIdx, LM[idxM].NodeU)
			} else if LM[idxM].Type == MoveIntraSwap {
				log.Fatal("Error: LM move intra-swap should not be re-evaluated here")
				delta = deltaSwapPositions(dist, nodes, tour, LM[idxM].PosA, LM[idxM].PosB)
			} else if LM[idxM].Type == MoveIntra2Opt {
        		ai := LM[idxM].Removed[0].U
				ai1 := LM[idxM].Removed[0].V

				aj := LM[idxM].Removed[1].U
				aj1 := LM[idxM].Removed[1].V

				posai := FindIndexTour(tour, ai)
				posai1 := FindIndexTour(tour, ai1)
				var posA int
				var posB int

				if posai > posai1  {
					posA = posai
				} else {
					posA = posai1
				}

				posaj := FindIndexTour(tour, aj)
				posaj1 := FindIndexTour(tour, aj1)

				if posaj1 < posaj && posaj1 != 0 {
					posB = posaj1
				} else {
					posB = posaj
				}

				delta = delta2Opt(dist, nodes, tour, posA, posB)
			} else {
				log.Fatalf("Unknown move type in LM: %d", LM[idxM].Type)
			}

			if delta >= 0 {
				LM = deleteMoveAtIndex(LM, idxM)
				idxM--
				continue
			}



			applyMoveToSolution(LM[idxM], tour, inSel)
			LM = deleteMoveAtIndex(LM, idxM)
			idxM--
				
			improvements++
			// applied = true
			// After applying one LM move we follow the assignment: "perform (apply) the move and remove from LM"
			// Stop after first applied LM move to re-evaluate neighborhood next iteration.
			// Note: Could apply multiple LM moves in one go, but safer to apply one and re-validate.
			// Return updated LM without this move (others are in newLM).
			// Also do not re-add this move.
			// We return now with updated LM (newLM).
			return true, evals, improvements, LM
		}
	}

	return false, evals, improvements, LM
}

// LocalSearchSteepest_LM_Iter performs one iteration using existing LM first, then
// if none applicable scans neighborhood, applies best move and updates LM accordingly.
// It expects LM to be passed in and returns updated LM (with removed moves removed,
// moved ones applied and removed from LM, and reversed ones left).
func LocalSearchSteepest_LM_Iter(inst *Instance, tour []int, inSel []bool, intraMode string, evalLimit int, cand [][]int, LM []Move) (bool, int, int, []Move) {
	K := inst.K
	dist := inst.Dist
	nodes := inst.Nodes

	evals := 0
	improvements := 0

	// 1) Try LM moves first (validate each)
	newLM := make([]Move, 0, len(LM))
	// applied := false

	for _, m := range LM {
		state := m.Validate(tour)
		if state == MoveInvalid {
			// drop it from LM
			continue
		} else if state == MoveReversedOrFuture {
			// keep it, but do not apply now
			newLM = append(newLM, m)
			continue
		} else if state == MoveApplicable {
			// apply and remove from LM
			applyMoveToSolution(m, tour, inSel)
			improvements++
			// applied = true
			// After applying one LM move we follow the assignment: "perform (apply) the move and remove from LM"
			// Stop after first applied LM move to re-evaluate neighborhood next iteration.
			// Note: Could apply multiple LM moves in one go, but safer to apply one and re-validate.
			// Return updated LM without this move (others are in newLM).
			// Also do not re-add this move.
			// We return now with updated LM (newLM).
			return true, evals, improvements, newLM
		}
	}
	// replace LM with filtered newLM (reversed/or-future ones kept)
	LM = newLM

	// 2) If none of LM was applicable, scan neighborhood (steepest) to find best move
	bestDelta := 0
	var bestMove Move

	// intra
	if intraMode == "nodes" {
		for i := 0; i < K; i++ {
			for j := i + 1; j < K; j++ {
				delta := deltaSwapPositions(dist, nodes, tour, i, j)
				evals++
				if delta < bestDelta {
					bestDelta = delta
					bestMove = Move{
						Type:    MoveIntraSwap,
						PosA:    i,
						PosB:    j,
						Delta:   delta,
						Removed: BuildRemovedEdgesForSwap(tour, i, j),
					}
				}
				if evalLimit > 0 && evals >= evalLimit {
					goto endScan
				}
			}
		}
	} else {
		for i := 0; i < K; i++ {
			for j := i + 1; j < K; j++ {
				delta := delta2Opt(dist, nodes, tour, i, j)
				evals++
				if delta < bestDelta {
					bestDelta = delta
					bestMove = Move{
						Type:    MoveIntra2Opt,
						PosA:    i,
						PosB:    j,
						Delta:   delta,
						Removed: BuildRemovedEdgesFor2Opt(tour, i, j),
					}
				}
				if evalLimit > 0 && evals >= evalLimit {
					goto endScan
				}
			}
		}
	}

	// inter: consider candidate lists around pos as in previous code
	for pos := 0; pos < K; pos++ {
		for _, u := range cand[tour[mod(pos+1, K)]] {
			if inSel[u] {
				continue
			}
			delta := deltaReplaceAtPos(dist, nodes, tour, pos, u)
			evals++
			if delta < bestDelta {
				bestDelta = delta
				bestMove = Move{
					Type:    MoveInterReplace,
					PosA:    pos,
					NodeU:   u,
					Delta:   delta,
					Removed: BuildRemovedEdgesForReplace(tour, pos),
				}
			}
			if evalLimit > 0 && evals >= evalLimit {
				goto endScan
			}
		}
		for _, u := range cand[tour[mod(pos-1, K)]] {
			if inSel[u] {
				continue
			}
			delta := deltaReplaceAtPos(dist, nodes, tour, pos, u)
			evals++
			if delta < bestDelta {
				bestDelta = delta
				bestMove = Move{
					Type:    MoveInterReplace,
					PosA:    pos,
					NodeU:   u,
					Delta:   delta,
					Removed: BuildRemovedEdgesForReplace(tour, pos),
				}
			}
			if evalLimit > 0 && evals >= evalLimit {
				goto endScan
			}
		}
	}

endScan:
	// If we found a best improving move, apply it and add to LM for reuse in future iterations
	if bestDelta < 0 {
		applyMoveToSolution(bestMove, tour, inSel)
		improvements++
		// add bestMove to LM for future reuse
		LM = append(LM, bestMove)
		return true, evals, improvements, LM
	}

	// Nothing found
	return false, evals, improvements, LM
}

// -------------------- Local Search runner --------------------

// RunLocalSearch orchestrates iterative local search.
// mode: "steepest", "greedy", "steepest_lm"
// intraMode: "nodes" or "edges"
func RunLocalSearch(inst *Instance, tour []int, inSel []bool, mode string, intraMode string, rnd *rand.Rand, cand [][]int) (finalTour []int, finalInSel []bool, evalsTotal int, improvements int) {
	// copy
	tourCopy := make([]int, len(tour))
	copy(tourCopy, tour)
	inCopy := make([]bool, len(inSel))
	copy(inCopy, inSel)

	evalsTotal = 0
	improvements = 0
	iter := 0
	const evalLimitPerCall = 0 // 0 unlimited

	// For LM mode, we maintain LM across iterations
	var LM []Move

	LM = make([]Move, 0)
	for {
		iter++
		var changed bool
		var evals, imps int
		if mode == "greedy" {
			changed, evals, imps = LocalSearchGreedy(inst, tourCopy, inCopy, intraMode, rnd, evalLimitPerCall)
		} else if mode == "steepest_lm" {
			// first call: if LM empty, populate it by evaluating initial neighborhood once
			beforeObj := TourLength(inst.Dist, tourCopy) + SelectedCosts(inst.Nodes, tourCopy)
			ch, ev, im, lm := LocalSearchSteepestWithLM(inst, tourCopy, inCopy, intraMode, evalLimitPerCall, cand, rnd, LM)
			LM = lm
			// calculate objective and LM size
			obj := TourLength(inst.Dist, tourCopy) + SelectedCosts(inst.Nodes, tourCopy)
			if obj > beforeObj {
				break
			}
			changed = ch
			evals = ev
			imps = im
		} else {
			changed, evals, imps = LocalSearchSteepest(inst, tourCopy, inCopy, intraMode, evalLimitPerCall, cand)
		}
		evalsTotal += evals
		improvements += imps
		if !changed {
			break
		}
		// continue until no change
	}
	return tourCopy, inCopy, evalsTotal, improvements
}

// -------------------- Original Steepest (kept for comparison) --------------------

// STEEPEST local search: examine whole neighborhood (both intra & inter) and select best improving move
func LocalSearchSteepest(inst *Instance, tour []int, inSel []bool, intraMode string, evalLimit int, cand [][]int) (bool, int, int) {
	K := inst.K
	dist := inst.Dist
	nodes := inst.Nodes

	bestDelta := 0
	bestMoveType := ""
	bestParams := [3]int{-1, -1, -1}

	evals := 0
	improvements := 0

	if intraMode == "nodes" {
		for i := 0; i < K; i++ {
			for j := i + 1; j < K; j++ {
				delta := deltaSwapPositions(dist, nodes, tour, i, j)
				evals++
				if delta < bestDelta {
					bestDelta = delta
					bestMoveType = "intra_nodes"
					bestParams = [3]int{i, j, 0}
				}
				if evals >= evalLimit && evalLimit > 0 {
					goto endSteep
				}
			}
		}
	} else {
		for i := 0; i < K; i++ {
			for j := 0; j < len(cand[tour[i]]); j++ {
				if !inSel[cand[tour[i]][j]] {
					continue
				}
				indx := FindIndexTour(tour, cand[tour[i]][j])
				if indx == -1 {
					continue
				}
				delta := delta2Opt(dist, nodes, tour, i, indx)
				evals++
				if delta < bestDelta {
					bestDelta = delta
					bestMoveType = "intra_edges"
					if i < indx {
						bestParams = [3]int{i, indx, 0}
					} else {
						bestParams = [3]int{indx, i, 0}
					}
				}
				if evals >= evalLimit && evalLimit > 0 {
					goto endSteep
				}
			}
		}
	}

	for pos := 0; pos < K; pos++ {
		for u := 0; u < len(cand[tour[mod(pos+1, K)]]); u++ {
			if inSel[cand[tour[mod(pos+1, K)]][u]] {
				continue
			}
			delta := deltaReplaceAtPos(dist, nodes, tour, pos, cand[tour[mod(pos+1, K)]][u])
			evals++
			if delta < bestDelta {
				bestDelta = delta
				bestMoveType = "inter"
				bestParams = [3]int{pos, cand[tour[mod(pos+1, K)]][u], 0}
			}
			if evals >= evalLimit && evalLimit > 0 {
				goto endSteep
			}
		}
		for u := 0; u < len(cand[tour[mod(pos-1, K)]]); u++ {
			if inSel[cand[tour[mod(pos-1, K)]][u]] {
				continue
			}
			delta := deltaReplaceAtPos(dist, nodes, tour, pos, cand[tour[mod(pos-1, K)]][u])
			evals++
			if delta < bestDelta {
				bestDelta = delta
				bestMoveType = "inter"
				bestParams = [3]int{pos, cand[tour[mod(pos-1, K)]][u], 0}
			}
			if evals >= evalLimit && evalLimit > 0 {
				goto endSteep
			}
		}
	}
endSteep:
	if bestDelta < 0 {
		switch bestMoveType {
		case "intra_nodes":
			i, j := bestParams[0], bestParams[1]
			tour[i], tour[j] = tour[j], tour[i]
		case "intra_edges":
			i, j := bestParams[0], bestParams[1]
			start := i + 1
			end := j
			for a, b := start, end; a < b; a, b = a+1, b-1 {
				tour[a], tour[b] = tour[b], tour[a]
			}
		case "inter":
			pos, u := bestParams[0], bestParams[1]
			old := tour[pos]
			inSel[old] = false
			inSel[u] = true
			tour[pos] = u
		default:
			return false, evals, improvements
		}
		improvements++
		return true, evals, improvements
	}
	return false, evals, improvements
}

// GREEDY local search (kept unchanged except for safety)
func LocalSearchGreedy(inst *Instance, tour []int, inSel []bool, intraMode string, rnd *rand.Rand, evalLimit int) (bool, int, int) {
	N := inst.N
	K := inst.K
	dist := inst.Dist
	nodes := inst.Nodes

	evals := 0
	improvements := 0

	intraPairs := make([][2]int, 0)
	if intraMode == "nodes" {
		for i := 0; i < K; i++ {
			for j := i + 1; j < K; j++ {
				intraPairs = append(intraPairs, [2]int{i, j})
			}
		}
	} else {
		for i := 0; i < K; i++ {
			for j := i + 1; j < K; j++ {
				intraPairs = append(intraPairs, [2]int{i, j})
			}
		}
	}
	if len(intraPairs) > 1 {
		rnd.Shuffle(len(intraPairs), func(i, j int) { intraPairs[i], intraPairs[j] = intraPairs[j], intraPairs[i] })
	}

	tourPosPerm := make([]int, K)
	for i := 0; i < K; i++ {
		tourPosPerm[i] = i
	}
	if K > 1 {
		rnd.Shuffle(K, func(i, j int) { tourPosPerm[i], tourPosPerm[j] = tourPosPerm[j], tourPosPerm[i] })
	}

	buildUnselected := func() []int {
		out := make([]int, 0, N-K)
		for v := 0; v < N; v++ {
			if !inSel[v] {
				out = append(out, v)
			}
		}
		return out
	}

	unselected := buildUnselected()

	intraIdx := 0
	interIdx := 0
	maxLoops := len(intraPairs)
	if K > maxLoops {
		maxLoops = K
	}
	found := false

	for loop := 0; loop < maxLoops && !found; loop++ {

		doIntra := rnd.Intn(2) == 0
		if doIntra {
			if intraIdx < len(intraPairs) {
				p := intraPairs[intraIdx]
				intraIdx++
				i := p[0]
				j := p[1]
				var delta int
				if intraMode == "nodes" {
					delta = deltaSwapPositions(dist, nodes, tour, i, j)
				} else {
					delta = delta2Opt(dist, nodes, tour, i, j)
				}
				evals++
				if delta < 0 {
					if intraMode == "nodes" {
						tour[i], tour[j] = tour[j], tour[i]
					} else {
						start := i + 1
						end := j
						for a, b := start, end; a < b; a, b = a+1, b-1 {
							tour[a], tour[b] = tour[b], tour[a]
						}
					}
					improvements++
					found = true
					break
				}
				if evals >= evalLimit && evalLimit > 0 {
					return false, evals, improvements
				}
			}
		} else {
			if interIdx < K {
				pos := tourPosPerm[interIdx]
				interIdx++

				if len(unselected) == 0 {
					unselected = buildUnselected()
				}
				if len(unselected) > 1 {
					rnd.Shuffle(len(unselected), func(i, j int) { unselected[i], unselected[j] = unselected[j], unselected[i] })
				}

				for _, u := range unselected {
					if inSel[u] {
						continue
					}
					delta := deltaReplaceAtPos(dist, nodes, tour, pos, u)
					evals++
					if delta < 0 {
						old := tour[pos]
						inSel[old] = false
						inSel[u] = true
						tour[pos] = u
						improvements++
						found = true
						unselected = buildUnselected()
						break
					}
					if evals >= evalLimit && evalLimit > 0 {
						return false, evals, improvements
					}
				}
				if found {
					break
				}
			}
		}
	}
	return found, evals, improvements
}

// -------------------- runMethods and main --------------------

func runMethods(inst *Instance, runs int, seed int64, outPath string) error {
	rnd := rand.New(rand.NewSource(seed))
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()
	w := csv.NewWriter(outFile)
	defer w.Flush()

	// build candidates
	cand := buildCandidates(inst.Nodes, inst.Dist, 10)

	// write header: use duration_s (seconds)
	if err := w.Write([]string{"method", "run", "objective", "tour_length", "selected_costs", "evaluations", "improvements", "final_selected", "seed", "duration_s"}); err != nil {
		return err
	}

	methods := []struct {
		mode      string // "steepest" or "greedy" or "steepest_lm"
		intraMode string // "nodes" or "edges"
		startType string // "random" or "greedy"
	}{
		{"steepest_lm", "edges", "random"},
		// {"steepest", "edges", "random"},
		// {"greedy", "nodes", "random"},
	}

	for _, m := range methods {
		methodName := fmt.Sprintf("%s_intra:%s_start:%s", m.mode, m.intraMode, m.startType)
		fmt.Printf("Running method %s with %d runs...\n", methodName, runs)
		for run := 0; run < runs; run++ {
			runSeed := int64(rnd.Int63())
			runRnd := rand.New(rand.NewSource(runSeed))

			var tour []int
			var inSel []bool
			if m.startType == "random" {
				tour0, in0 := RandomStart(inst, runRnd)
				tour = tour0
				inSel = in0
			} else {
				startNode := run % inst.N
				tour0, in0 := GreedyRegretStart(inst, startNode)
				tour = tour0
				inSel = in0
			}

			start := time.Now()
			finalTour, _, evals, imps := RunLocalSearch(inst, tour, inSel, m.mode, m.intraMode, runRnd, cand)
			elapsed := time.Since(start)
			elapsedS := strconv.FormatFloat(elapsed.Seconds(), 'f', 6, 64)
			tLen := TourLength(inst.Dist, finalTour)
			sCost := SelectedCosts(inst.Nodes, finalTour)
			obj := tLen + sCost

			strSel := make([]string, len(finalTour))
			for i := range finalTour {
				strSel[i] = strconv.Itoa(finalTour[i])
			}
			if err := w.Write([]string{
				methodName,
				strconv.Itoa(run),
				strconv.Itoa(obj),
				strconv.Itoa(tLen),
				strconv.Itoa(sCost),
				strconv.Itoa(evals),
				strconv.Itoa(imps),
				strings.Join(strSel, ";"),
				strconv.FormatInt(runSeed, 10),
				elapsedS,
			}); err != nil {
				return err
			}
			if run%50 == 0 {
				w.Flush()
			}
		}
	}
	w.Flush()
	return nil
}

func main() {
	inPath := flag.String("in", "", "input CSV file path (rows: x,y,cost)")
	outPath := flag.String("out", "result.csv", "output CSV results path")
	runs := flag.Int("runs", 200, "number of runs per method")
	seed := flag.Int64("seed", time.Now().UnixNano(), "random seed")
	flag.Parse()
	if *inPath == "" || *outPath == "" {
		log.Fatalf("Please provide -in and -out paths. Example: ./app -in instance.csv -out results.csv")
	}
	inst, err := ReadInstanceCSV(*inPath)
	if err != nil {
		log.Fatalf("Failed to read instance: %v", err)
	}
	fmt.Printf("Read instance with N=%d nodes, selecting K=%d nodes\n", inst.N, inst.K)

	err = runMethods(inst, *runs, *seed, *outPath)
	if err != nil {
		log.Fatalf("runMethods failed: %v", err)
	}
	fmt.Printf("Done. Results written to %s\n", *outPath)
}
