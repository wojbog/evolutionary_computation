// local_search_tsp_select.go
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
		if row[0] == "x" {
			continue
		}
		if len(row) < 3 {
			return nil, fmt.Errorf("row %d has fewer than 3 columns", i)
		}
		x, err := strconv.Atoi(strings.TrimSpace(row[0]))
		if err != nil {
			return nil, err
		}
		y, err := strconv.Atoi(strings.TrimSpace(row[1]))
		if err != nil {
			return nil, err
		}
		cost, err := strconv.Atoi(strings.TrimSpace(row[2]))
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, Node{X: x, Y: y, Cost: cost})
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
				// mathematically rounded integer
				inst.Dist[i][j] = int(math.Round(dist))
			}
		}
	}
	// set K = ceil(N/2)
	inst.K = (inst.N + 1) / 2
	return inst, nil
}

// Objective calculation helpers (only used for reporting / verifying final soln)
//
// Tour is an order of selected node indices (0..N-1) with length == K
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

// CREATE STARTING SOLUTIONS

// Random starting solution: choose K distinct nodes uniformly, and random order
func RandomStart(inst *Instance, rnd *rand.Rand) ([]int, []bool) {
	N, K := inst.N, inst.K
	all := make([]int, N)
	for i := 0; i < N; i++ {
		all[i] = i
	}
	// shuffle and pick first K
	rnd.Shuffle(N, func(i, j int) { all[i], all[j] = all[j], all[i] })
	selected := all[:K]
	// random tour order
	rnd.Shuffle(len(selected), func(i, j int) { selected[i], selected[j] = selected[j], selected[i] })
	inSelected := make([]bool, N)
	for _, s := range selected {
		inSelected[s] = true
	}
	return selected, inSelected
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
	newT := append([]int{}, tour[:pos]...)
	newT = append(newT, node)
	newT = append(newT, tour[pos:]...)
	return newT
}

// Greedy construction using regret-2 insertion. Start from a specified starting node index.
func GreedyRegretStart(inst *Instance, startNode int) ([]int, []bool) {
	k := inst.K
	D:= inst.Dist
	alpha := 1.0
	beta := 1.0
	nodes := inst.Nodes
	n := len(nodes)
	selected := make([]bool, n)
	selected[startNode] = true
	// pick second node: nearest neighbor
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
			score := alpha*float64(regret) - beta*float64(bestTot)
			cands = append(cands, cand{v, bestTot, secondTot, bestPos, score})
		}
		sort.Slice(cands, func(a, b int) bool { return cands[a].score > cands[b].score })
		ch := cands[0]
		selected[ch.node] = true
		tour = insertAt(tour, ch.bestPos, ch.node)
	}
	

	return tour, selected
}

// LOCAL SEARCH moves and deltas
//
// We maintain:
// - tour: []int of length K in cycle order
// - inSel: []bool of length N whether node is selected
// - current tour length and selected cost can be tracked if desired, but deltas are used for evaluating moves
//
// Move types implemented:
// - intra nodes swap: swap tour positions i and j (i<j). Delta affects neighbors of both positions.
// - intra 2-opt (edge swap): 2-opt between positions i and j (assuming i<j): reverse tour segment (i+1..j) and reconnect.
// - inter exchange: replace tour[pos] (selected s) with unselected u (keeping the position in tour). Delta uses neighbors prev and next.
//
// Deltas compute only affected edges' lengths and change in node costs.

func mod(a, b int) int {
	v := a % b
	if v < 0 {
		v += b
	}
	return v
}

// delta for replacing node at tour[pos] (s) with new node u
func deltaReplaceAtPos(dist [][]int, nodes []Node, tour []int, pos int, u int) int {
	K := len(tour)
	s := tour[pos]
	prev := tour[mod(pos-1, K)]
	next := tour[mod(pos+1, K)]
	// old edges: prev - s, s - next
	// new edges: prev - u, u - next
	deltaLen := dist[prev][u] + dist[u][next] - dist[prev][s] - dist[s][next]
	deltaCost := nodes[u].Cost - nodes[s].Cost
	return deltaLen + deltaCost
}

// delta for swapping nodes at positions i and j in tour
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
	// neighbors
	Aprev := tour[mod(i-1, K)]
	Anext := tour[mod(i+1, K)]
	Bprev := tour[mod(j-1, K)]
	Bnext := tour[mod(j+1, K)]

	deltaLen := 0
	// If positions adjacent, careful with overlapping edges
	if i==0 && j==K-1 {
		// A and B adjacent, order ... B - A ...
		// old edges: Bprev-B, B-A, A-Anext
		// new edges: Bprev-A, A-B, B-Anext
		deltaLen += dist[Bprev][A] + dist[A][B] + dist[B][Anext]
		deltaLen -= dist[Bprev][B] + dist[B][A] + dist[A][Anext]
	} else if mod(i+1, K) == j {
		// A and B adjacent, order ... A - B ...
		// old edges: Aprev-A, A-B, B-Bnext
		// new edges: Aprev-B, B-A, A-Bnext
		deltaLen += dist[Aprev][B] + dist[B][A] + dist[A][Bnext]
		deltaLen -= dist[Aprev][A] + dist[A][B] + dist[B][Bnext]
	} else {
		// non-adjacent
		// old edges: Aprev-A, A-Anext, Bprev-B, B-Bnext
		// new edges: Aprev-B, B-Anext, Bprev-A, A-Bnext
		deltaLen += dist[Aprev][B] + dist[B][Anext] + dist[Bprev][A] + dist[A][Bnext]
		deltaLen -= dist[Aprev][A] + dist[A][Anext] + dist[Bprev][B] + dist[B][Bnext]
	}
	// cost change is zero (selected set unchanged)
	return deltaLen
}

// delta for 2-opt between edges (i,i+1) and (j,j+1) for i<j
// This corresponds to reversing tour segment i+1..j
func delta2Opt(dist [][]int, nodes []Node, tour []int, i int, j int) int {
	K := len(tour)
	if i == j {
		return 0
	}
	ai := tour[i]
	ai1 := tour[mod(i+1, K)]
	aj := tour[j]
	aj1 := tour[mod(j+1, K)]
	// old edges ai-ai1 and aj-aj1
	// new edges ai-aj and ai1-aj1 (but since we reverse, correct reconnection is ai-aj and ai1-aj1)
	deltaLen := dist[ai][aj] + dist[ai1][aj1] - dist[ai][ai1] - dist[aj][aj1]
	return deltaLen
}

// GREEDY local search: browse neighbors in randomized order, stop at first improving move
// returns whether an improvement was applied (true) and updates tour/inSel in-place
func LocalSearchGreedy(inst *Instance, tour []int, inSel []bool, intraMode string, rnd *rand.Rand, evalLimit int) (bool, int, int) {
	// intraMode: "nodes" or "edges"
	N := inst.N
	K := inst.K
	dist := inst.Dist
	nodes := inst.Nodes

	evals := 0
	improvements := 0

	// Build lists of candidate indices and shuffle ordering
	// For random browsing, we will create permuted indices for:
	// - inter moves: for each tour position pos and each unselected node u -> that's potentially large (K * (N-K))
	// To avoid creating a huge list in memory, we will generate randomized order by:
	//   1) create slice of tour positions permuted
	//   2) for each position pick a small random ordering of unselected nodes by shuffling the slice of unselected nodes each time
	// This still explores neighborhood in random order and avoids precomputing every possible move.
	// But since the assignment wants to browse the whole neighborhood in random order ideally, we will attempt to randomize both types in an interleaved way:
	// Implementation: create two action sequences:
	// - intra actions as pairs (i,j) (for nodes swap or 2-opt), enumerated but shuffled
	// - inter actions as (pos, u) enumerated but we'll shuffle pos list and for each pos produce randomized candidate unselected nodes
	// We'll interleave by alternating trying an intra move then an inter move until we find an improving move.

	// Enumerate intra moves indices and shuffle
	intraPairs := make([][2]int, 0)
	if intraMode == "nodes" {
		for i := 0; i < K; i++ {
			for j := i + 1; j < K; j++ {
				intraPairs = append(intraPairs, [2]int{i, j})
			}
		}
	} else {
		// edges (2-opt): consider i < j and not adjacent as separate moves as typical 2-opt
		for i := 0; i < K; i++ {
			for j := i + 1; j < K; j++ {
				// in standard TSP 2-opt any i<j is valid (adjacent handled too)
				intraPairs = append(intraPairs, [2]int{i, j})
			}
		}
	}
	rnd.Shuffle(len(intraPairs), func(i, j int) { intraPairs[i], intraPairs[j] = intraPairs[j], intraPairs[i] })

	// Prepare inter: list of tour positions and list of unselected nodes
	tourPosPerm := make([]int, K)
	for i := 0; i < K; i++ {
		tourPosPerm[i] = i
	}
	rnd.Shuffle(K, func(i, j int) { tourPosPerm[i], tourPosPerm[j] = tourPosPerm[j], tourPosPerm[i] })

	unselected := make([]int, 0)
	for v := 0; v < N; v++ {
		if !inSel[v] {
			unselected = append(unselected, v)
		}
	}

	// Interleave scanning: we'll iterate up to max(len(intraPairs), K) loops,
	// each loop attempt one intra candidate (next in sequence) and one inter candidate (pos + shuffled unselected).
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
			// Try an intra move if available
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
					// apply move
					if intraMode == "nodes" {
						tour[i], tour[j] = tour[j], tour[i]
					} else {
						// reverse segment i+1..j
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

			// Try an inter move
			if interIdx < K {
				pos := tourPosPerm[interIdx]
				interIdx++
				// shuffle unselected order (to avoid evaluating all in deterministic order)
				rnd.Shuffle(len(unselected), func(i, j int) { unselected[i], unselected[j] = unselected[j], unselected[i] })
				for _, u := range unselected {
					delta := deltaReplaceAtPos(dist, nodes, tour, pos, u)
					evals++
					if delta < 0 {
						// apply: swap membership: replace tour[pos] with u
						old := tour[pos]
						inSel[old] = false
						inSel[u] = true
						tour[pos] = u
						improvements++
						found = true
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

// STEEPEST local search: examine whole neighborhood (both intra & inter) and select best improving move
func LocalSearchSteepest(inst *Instance, tour []int, inSel []bool, intraMode string, evalLimit int) (bool, int, int) {
	N := inst.N
	K := inst.K
	dist := inst.Dist
	nodes := inst.Nodes

	bestDelta := 0
	bestMoveType := ""               // "intra_nodes", "intra_edges", "inter"
	bestParams := [3]int{-1, -1, -1} // meaning depends on move type

	evals := 0
	improvements := 0

	// Intra moves
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
			for j := i + 1; j < K; j++ {
				delta := delta2Opt(dist, nodes, tour, i, j)
				evals++
				if delta < bestDelta {
					bestDelta = delta
					bestMoveType = "intra_edges"
					bestParams = [3]int{i, j, 0}
				}
				if evals >= evalLimit && evalLimit > 0 {
					goto endSteep
				}
			}
		}
	}

	// Inter moves: for each tour pos and each unselected node
	for pos := 0; pos < K; pos++ {
		for u := 0; u < N; u++ {
			if inSel[u] {
				continue
			}
			delta := deltaReplaceAtPos(dist, nodes, tour, pos, u)
			evals++
			if delta < bestDelta {
				bestDelta = delta
				bestMoveType = "inter"
				bestParams = [3]int{pos, u, 0}
			}
			if evals >= evalLimit && evalLimit > 0 {
				goto endSteep
			}
		}
	}
endSteep:
	if bestDelta < 0 {
		// apply best move
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
			// shouldn't happen
			return false, evals, improvements
		}
		improvements++
		return true, evals, improvements
	}
	return false, evals, improvements
}

// Run local search until no improving move is found
func RunLocalSearch(inst *Instance, tour []int, inSel []bool, mode string, intraMode string, rnd *rand.Rand) (finalTour []int, finalInSel []bool, evalsTotal int, improvements int) {
	// mode: "steepest" or "greedy"
	tourCopy := make([]int, len(tour))
	copy(tourCopy, tour)
	inCopy := make([]bool, len(inSel))
	copy(inCopy, inSel)

	evalsTotal = 0
	improvements = 0
	iter := 0
	const evalLimitPerCall = 0 // 0 means unlimited

	for {
		iter++
		var changed bool
		var evals, imps int
		if mode == "greedy" {
			changed, evals, imps = LocalSearchGreedy(inst, tourCopy, inCopy, intraMode, rnd, evalLimitPerCall)
		} else {
			changed, evals, imps = LocalSearchSteepest(inst, tourCopy, inCopy, intraMode, evalLimitPerCall)
		}
		evalsTotal += evals
		improvements += imps
		if !changed {
			break
		}
		// if iter%50 == 0 {
		// 	fmt.Printf("  LS iter %d: evals so far %d, improvements %d\n", iter, evalsTotal, improvements)
		// }
	}
	return tourCopy, inCopy, evalsTotal, improvements
}

func runMethods(inst *Instance, runs int, seed int64, outPath string) error {
	rnd := rand.New(rand.NewSource(seed))
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()
	w := csv.NewWriter(outFile)
	defer w.Flush()

	// write header
	if err := w.Write([]string{"method", "run", "objective", "tour_length", "selected_costs", "evaluations", "improvements", "final_selected", "seed", "duration_ms"}); err != nil {
		return err
	}

	methods := []struct {
		mode      string // "steepest" or "greedy"
		intraMode string // "nodes" or "edges"
		startType string // "random" or "greedy"
	}{
		{"steepest", "nodes", "random"},
		{"steepest", "nodes", "greedy"},
		{"steepest", "edges", "random"},
		{"steepest", "edges", "greedy"},
		{"greedy", "nodes", "random"},
		{"greedy", "nodes", "greedy"},
		{"greedy", "edges", "random"},
		{"greedy", "edges", "greedy"},
	}
	for _, m := range methods {
		methodName := fmt.Sprintf("%s_intra:%s_start:%s", m.mode, m.intraMode, m.startType)
		fmt.Printf("Running method %s with %d runs...\n", methodName, runs)
		for run := 0; run < runs; run++ {
			// create a per-run RNG so results are reproducible
			runSeed := int64(rnd.Int63())
			runRnd := rand.New(rand.NewSource(runSeed))

			var tour []int
			var inSel []bool
			if m.startType == "random" {
				tour0, in0 := RandomStart(inst, runRnd)
				tour = tour0
				inSel = in0
			} else {
				// greedy start: use starting node = run % N (to emulate using different starting nodes)
				startNode := run % inst.N
				tour0, in0 := GreedyRegretStart(inst, startNode)
				tour = tour0
				inSel = in0
			}

			// run local search
			start := time.Now()
			finalTour, _, evals, imps := RunLocalSearch(inst, tour, inSel, m.mode, m.intraMode, runRnd)
			elapsed := time.Since(start)
			elapsedS := strconv.FormatFloat(elapsed.Seconds(), 'f', 6, 64)
			// compute objective values for output
			tLen := TourLength(inst.Dist, finalTour)
			sCost := SelectedCosts(inst.Nodes, finalTour)
			obj := tLen + sCost

			// prepare finalSelected list as semicolon separated indices
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
			// periodically flush
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
