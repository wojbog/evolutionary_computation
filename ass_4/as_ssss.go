// tsp_select.go
package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Node struct {
	x, y float64
	cost int
}

type MoveResult struct {
	improve  int
	tour     []int
	selected []bool
	moveType string
	details  string
	delta    int
	valid    bool
}

// Round mathematically
func roundInt(f float64) int {
	return int(math.Round(f))
}

// ----------- FILE READING -------------

// Reads from either CSV (x,y,cost) or whitespace-separated file
func readNodes(filename string) ([]Node, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".csv" {
		return readNodesCSV(filename)
	}
	return readNodesTXT(filename)
}

// CSV: x,y,cost (header optional)
func readNodesCSV(filename string) ([]Node, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading csv: %v", err)
	}

	nodes := []Node{}
	for i, rec := range records {
		if len(rec) < 3 {
			// maybe header row — skip if not numeric
			if i == 0 && (strings.Contains(strings.ToLower(rec[0]), "x") || strings.Contains(strings.ToLower(rec[1]), "y")) {
				continue
			}
			return nil, fmt.Errorf("csv line %d: expected 3 values (x,y,cost), got %v", i+1, rec)
		}
		x, err := strconv.ParseFloat(strings.TrimSpace(rec[0]), 64)
		if err != nil {
			if i == 0 && (strings.Contains(strings.ToLower(rec[0]), "x")) {
				continue
			}
			return nil, fmt.Errorf("csv line %d: bad x value: %v", i+1, err)
		}
		y, err := strconv.ParseFloat(strings.TrimSpace(rec[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("csv line %d: bad y value: %v", i+1, err)
		}
		c, err := strconv.Atoi(strings.TrimSpace(rec[2]))
		if err != nil {
			return nil, fmt.Errorf("csv line %d: bad cost value: %v", i+1, err)
		}
		nodes = append(nodes, Node{x, y, c})
	}
	return nodes, nil
}

// TXT: whitespace separated x y cost
func readNodesTXT(filename string) ([]Node, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	nodes := []Node{}
	lineNo := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNo++
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			return nil, fmt.Errorf("line %d: expected 3 fields", lineNo)
		}
		x, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: bad x value: %v", lineNo, err)
		}
		y, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: bad y value: %v", lineNo, err)
		}
		c, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("line %d: bad cost value: %v", lineNo, err)
		}
		nodes = append(nodes, Node{x, y, c})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nodes, nil
}

// ----------- CORE FUNCTIONS -------------

func computeDistanceMatrix(nodes []Node) [][]int {
	n := len(nodes)
	d := make([][]int, n)
	for i := 0; i < n; i++ {
		d[i] = make([]int, n)
		for j := 0; j < n; j++ {
			if i == j {
				d[i][j] = 0
			} else {
				dist := math.Hypot(nodes[i].x-nodes[j].x, nodes[i].y-nodes[j].y)
				d[i][j] = roundInt(dist)
			}
		}
	}
	return d
}

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
			arr = append(arr, pair{v: j, key: dist[i][j] + nodes[j].cost})
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

func randPerm(n int, rng *rand.Rand) []int {
	res := make([]int, n)
	for i := range res {
		res[i] = i
	}
	rng.Shuffle(n, func(i, j int) { res[i], res[j] = res[j], res[i] })
	return res
}

func buildRandomStart(nodes []Node, dist [][]int, rng *rand.Rand) (tour []int, selected []bool) {
	n := len(nodes)
	selectCount := (n + 1) / 2
	selected = make([]bool, n)
	perm := randPerm(n, rng)
	for i := 0; i < selectCount; i++ {
		selected[perm[i]] = true
	}
	selectedNodes := []int{}
	for i := range selected {
		if selected[i] {
			selectedNodes = append(selectedNodes, i)
		}
	}
	start := selectedNodes[rng.Intn(len(selectedNodes))]
	tour = []int{start}
	used := map[int]bool{start: true}
	for len(tour) < len(selectedNodes) {
		last := tour[len(tour)-1]
		best, bestd := -1, math.MaxInt32
		for _, v := range selectedNodes {
			if used[v] {
				continue
			}
			if dist[last][v] < bestd {
				best, bestd = v, dist[last][v]
			}
		}
		tour = append(tour, best)
		used[best] = true
	}
	return tour, selected
}

func objective(tour []int, selected []bool, dist [][]int, nodes []Node) int {
	total := 0
	m := len(tour)
	for i := 0; i < m; i++ {
		a, b := tour[i], tour[(i+1)%m]
		total += dist[a][b]
	}
	for i, s := range selected {
		if s {
			total += nodes[i].cost
		}
	}
	return total
}

func tourPosMap(tour []int) map[int]int {
	pos := make(map[int]int, len(tour))
	for i, v := range tour {
		pos[v] = i
	}
	return pos
}

func edgeIsCandidate(a, b int, cand [][]int) bool {
	for _, v := range cand[a] {
		if v == b {
			return true
		}
	}
	for _, v := range cand[b] {
		if v == a {
			return true
		}
	}
	return false
}

// ---------- LOCAL SEARCH (CANDIDATE + BASELINE) -------------

func steepestCandidate(nodes []Node, dist [][]int, cand [][]int, tour0 []int, sel0 []bool) ([]int, []bool, int) {
	tour := append([]int(nil), tour0...)
	selected := append([]bool(nil), sel0...)
	bestObj := objective(tour, selected, dist, nodes)

	for {
		var bestMove *MoveResult
		pos := tourPosMap(tour)
		n := len(nodes)
		m := len(tour)

		for s := 0; s < n; s++ {
			if !selected[s] {
				continue
			}
			for _, v := range cand[s] {
				if selected[v] {
					i, j := pos[s], pos[v]
					if i > j {
						i, j = j, i
					}
					if (i+1)%m == j {
						continue
					}
					a, b := tour[i], tour[(i+1)%m]
					c, d := tour[j], tour[(j+1)%m]
					delta := -dist[a][b] - dist[c][d] + dist[a][c] + dist[b][d]
					if !(edgeIsCandidate(a, c, cand) || edgeIsCandidate(b, d, cand)) {
						continue
					}
					if delta < 0 {
						newTour := append([]int(nil), tour...)
						for li, lj := i+1, j; li < lj; li, lj = li+1, lj-1 {
							newTour[li], newTour[lj] = newTour[lj], newTour[li]
						}
						mr := MoveResult{improve: -delta, tour: newTour, selected: append([]bool(nil), selected...), delta: delta, valid: true}
						if bestMove == nil || mr.improve > bestMove.improve {
							cp := mr
							bestMove = &cp
						}
					}
				} else {
					ps := pos[s]
					prev, next := tour[(ps-1+m)%m], tour[(ps+1)%m]
					delta := -dist[prev][s] - dist[s][next] + dist[prev][v] + dist[v][next]
					delta += -nodes[s].cost + nodes[v].cost
					if !(edgeIsCandidate(prev, v, cand) || edgeIsCandidate(v, next, cand)) {
						continue
					}
					if delta < 0 {
						newTour := append([]int(nil), tour...)
						newTour[ps] = v
						newSel := append([]bool(nil), selected...)
						newSel[s], newSel[v] = false, true
						mr := MoveResult{improve: -delta, tour: newTour, selected: newSel, delta: delta, valid: true}
						if bestMove == nil || mr.improve > bestMove.improve {
							cp := mr
							bestMove = &cp
						}
					}
				}
			}
		}
		if bestMove == nil {
			break
		}
		tour, selected = bestMove.tour, bestMove.selected
		bestObj = objective(tour, selected, dist, nodes)
	}
	return tour, selected, bestObj
}

func steepestBaseline(nodes []Node, dist [][]int, tour0 []int, sel0 []bool) ([]int, []bool, int) {
	tour := append([]int(nil), tour0...)
	selected := append([]bool(nil), sel0...)
	bestObj := objective(tour, selected, dist, nodes)
	m := len(tour)
	n := len(nodes)
	for {
		var bestMove *MoveResult
		pos := tourPosMap(tour)
		for i := 0; i < m; i++ {
			for j := i + 2; j < m; j++ {
				if i == 0 && j == m-1 {
					continue
				}
				a, b := tour[i], tour[(i+1)%m]
				c, d := tour[j], tour[(j+1)%m]
				delta := -dist[a][b] - dist[c][d] + dist[a][c] + dist[b][d]
				if delta < 0 {
					newTour := append([]int(nil), tour...)
					for li, lj := i+1, j; li < lj; li, lj = li+1, lj-1 {
						newTour[li], newTour[lj] = newTour[lj], newTour[li]
					}
					mr := MoveResult{improve: -delta, tour: newTour, selected: append([]bool(nil), selected...), delta: delta, valid: true}
					if bestMove == nil || mr.improve > bestMove.improve {
						cp := mr
						bestMove = &cp
					}
				}
			}
		}
		for s := 0; s < n; s++ {
			if !selected[s] {
				continue
			}
			ps := pos[s]
			prev, next := tour[(ps-1+m)%m], tour[(ps+1)%m]
			for u := 0; u < n; u++ {
				if selected[u] {
					continue
				}
				delta := -dist[prev][s] - dist[s][next] + dist[prev][u] + dist[u][next]
				delta += -nodes[s].cost + nodes[u].cost
				if delta < 0 {
					newTour := append([]int(nil), tour...)
					newTour[ps] = u
					newSel := append([]bool(nil), selected...)
					newSel[s], newSel[u] = false, true
					mr := MoveResult{improve: -delta, tour: newTour, selected: newSel, delta: delta, valid: true}
					if bestMove == nil || mr.improve > bestMove.improve {
						cp := mr
						bestMove = &cp
					}
				}
			}
		}
		if bestMove == nil {
			break
		}
		tour, selected = bestMove.tour, bestMove.selected
		bestObj = objective(tour, selected, dist, nodes)
	}
	return tour, selected, bestObj
}

// ---------- EXPERIMENT DRIVER -------------

func runExperiments(nodes []Node, dist [][]int, cand [][]int, runs int, rng *rand.Rand) {
	n := len(nodes)
	K := len(cand[0])
	fmt.Printf("Problem: n=%d, select=%d, candidate K=%d, runs=%d\n", n, (n+1)/2, K, runs)
	type Stats struct{ best int; sum int64 }
	cStat, bStat := Stats{math.MaxInt32, 0}, Stats{math.MaxInt32, 0}

	start := time.Now()
	for r := 0; r < runs; r++ {
		t0, s0 := buildRandomStart(nodes, dist, rng)
		_, _, objCand := steepestCandidate(nodes, dist, cand, t0, s0)
		_, _, objBase := steepestBaseline(nodes, dist, t0, s0)
		if objCand < cStat.best {
			cStat.best = objCand
		}
		cStat.sum += int64(objCand)
		if objBase < bStat.best {
			bStat.best = objBase
		}
		bStat.sum += int64(objBase)
		if (r+1)%50 == 0 {
			fmt.Printf("  completed %d runs\n", r+1)
		}
	}
	elapsed := time.Since(start)
	fmt.Printf("\n=== Results (%d runs, %v) ===\n", runs, elapsed)
	fmt.Printf("Candidate-based: best=%d avg=%.2f\n", cStat.best, float64(cStat.sum)/float64(runs))
	fmt.Printf("Baseline:        best=%d avg=%.2f\n", bStat.best, float64(bStat.sum)/float64(runs))
}

// ---------- MAIN -------------

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./tsp_select instance.csv")
		return
	}
	filename := os.Args[1]
	nodes, err := readNodes(filename)
	if err != nil {
		log.Fatalf("error reading %s: %v", filename, err)
	}
	if len(nodes) < 4 {
		log.Fatalf("need at least 4 nodes, got %d", len(nodes))
	}
	dist := computeDistanceMatrix(nodes)
	K := 10
	candidates := buildCandidates(nodes, dist, K)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	runExperiments(nodes, dist, candidates, 200, rng)
}
