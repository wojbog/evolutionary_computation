package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Node struct {
	X, Y  int
	Cost  int
	Index int
}

type Solution struct {
	Method      string
	StartNode   int
	Selected    []int
	Tour        []int
	TourLen     int
	CostSum     int
	Obj         int
}

func main() {
	inFile := flag.String("in", "", "input CSV file (x,y,cost)")
	outFile := flag.String("out", "best_results.csv", "output CSV file path")
	alpha := flag.Float64("alpha", 1.0, "alpha weight for regret")
	beta := flag.Float64("beta", 1.0, "beta weight for best insertion cost")
	maxRuns := flag.Int("maxruns", 200, "maximum runs per method")
	verbose := flag.Bool("verbose", false, "print verbose output")
	flag.Parse()

	if *inFile == "" {
		log.Fatal("please provide -in CSV file path")
	}

	nodes, err := readNodesCSV(*inFile)
	if err != nil {
		log.Fatalf("failed reading nodes: %v", err)
	}
	n := len(nodes)
	k := (n + 1) / 2
	fmt.Printf("Loaded %d nodes. Selecting k=%d per tour.\n", n, k)

	D := computeDistanceMatrix(nodes)
	methods := []string{"regret", "weighted"}

	var bestResults []Solution
	for _, m := range methods {
		fmt.Printf("Running method: %s ...\n", m)
		count := min(*maxRuns, n)
		bestSol := Solution{Obj: math.MaxInt}
		for start := 0; start < count; start++ {
			var sol Solution
			switch m {
			case "regret":
				sol = greedyRegret(D, nodes, k, start, *verbose)
				sol.Method = "2-Regret insertion"
			case "weighted":
				sol = greedyWeighted(D, nodes, k, start, *alpha, *beta, *verbose)
				sol.Method = fmt.Sprintf("Weighted (α=%.2f,β=%.2f)", *alpha, *beta)
			}
			sol.StartNode = start
			if sol.Obj < bestSol.Obj {
				bestSol = sol
			}
		}
		fmt.Printf(" → Best objective for %s: %d (start %d)\n", bestSol.Method, bestSol.Obj, bestSol.StartNode)
		bestResults = append(bestResults, bestSol)
	}

	if err := writeResultsCSV(*outFile, bestResults); err != nil {
		log.Fatalf("failed writing results: %v", err)
	}
}

func readNodesCSV(path string) ([]Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = ';'
	lines, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var nodes []Node
	for i, rec := range lines {
		if len(rec) < 3 {
			continue
		}
		if i == 0 {
			// skip header
			if _, err := strconv.Atoi(strings.TrimSpace(rec[0])); err != nil {
				continue
			}
		}
		x, _ := strconv.Atoi(strings.TrimSpace(rec[0]))
		y, _ := strconv.Atoi(strings.TrimSpace(rec[1]))
		c, _ := strconv.Atoi(strings.TrimSpace(rec[2]))
		nodes = append(nodes, Node{X: x, Y: y, Cost: c, Index: len(nodes)})
	}
	return nodes, nil
}

func writeResultsCSV(path string, sols []Solution) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{"Method", "StartNode", "Objective", "TourLength", "SumCosts", "SelectedNodes", "TourOrder"}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, s := range sols {
		row := []string{
			s.Method,
			strconv.Itoa(s.StartNode),
			strconv.Itoa(s.Obj),
			strconv.Itoa(s.TourLen),
			strconv.Itoa(s.CostSum),
			intSliceToString(s.Selected),
			intSliceToString(s.Tour),
		}
		w.Write(row)
	}
	return w.Error()
}

func intSliceToString(a []int) string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for i, v := range a {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(strconv.Itoa(v))
	}
	sb.WriteString("]")
	return sb.String()
}

func computeDistanceMatrix(nodes []Node) [][]int {
	n := len(nodes)
	D := make([][]int, n)
	for i := 0; i < n; i++ {
		D[i] = make([]int, n)
		for j := 0; j < n; j++ {
			if i == j {
				D[i][j] = 0
			} else {
				dx := float64(nodes[i].X - nodes[j].X)
				dy := float64(nodes[i].Y - nodes[j].Y)
				D[i][j] = int(math.Round(math.Hypot(dx, dy)))
			}
		}
	}
	return D
}

func tourLength(tour []int, D [][]int) int {
	if len(tour) == 0 {
		return 0
	}
	sum := 0
	for i := 0; i < len(tour); i++ {
		a := tour[i]
		b := tour[(i+1)%len(tour)]
		sum += D[a][b]
	}
	return sum
}

func insertAt(tour []int, pos int, node int) []int {
	newT := append([]int{}, tour[:pos]...)
	newT = append(newT, node)
	newT = append(newT, tour[pos:]...)
	return newT
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

func countSelected(sel []bool) int {
	c := 0
	for _, v := range sel {
		if v {
			c++
		}
	}
	return c
}

// --- Greedy methods ---

func greedyRegret(D [][]int, nodes []Node, k int, start int, verbose bool) Solution {
	n := len(nodes)
	selected := make([]bool, n)
	selected[start] = true
	// second node: closest to start
	bestJ := -1
	bestVal := math.MaxInt
	for j := 0; j < n; j++ {
		if j == start {
			continue
		}
		val := D[start][j] + nodes[j].Cost
		if val < bestVal {
			bestVal = val
			bestJ = j
		}
	}
	selected[bestJ] = true
	tour := []int{start, bestJ}

	for countSelected(selected) < k {
		type cand struct {
			node, bestTot, secondTot, bestPos int
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
			cands = append(cands, cand{v, bestTot, secondTot, bestPos})
		}
		sort.Slice(cands, func(a, b int) bool {
			regA := cands[a].secondTot - cands[a].bestTot
			regB := cands[b].secondTot - cands[b].bestTot
			if regA != regB {
				return regA > regB
			}
			return cands[a].bestTot < cands[b].bestTot
		})
		ch := cands[0]
		selected[ch.node] = true
		tour = insertAt(tour, ch.bestPos, ch.node)
	}
	return finalizeSolution("2-Regret", D, nodes, selected, tour)
}

func greedyWeighted(D [][]int, nodes []Node, k int, start int, alpha, beta float64, verbose bool) Solution {
	n := len(nodes)
	selected := make([]bool, n)
	selected[start] = true
	// pick second node: nearest neighbor
	bestJ := -1
	bestVal := math.MaxInt
	for j := 0; j < n; j++ {
		if j == start {
			continue
		}
		val := D[start][j] + nodes[j].Cost
		if val < bestVal {
			bestVal = val
			bestJ = j
		}
	}
	selected[bestJ] = true
	tour := []int{start, bestJ}

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
	return finalizeSolution("Weighted", D, nodes, selected, tour)
}

// --- helpers ---

func finalizeSolution(name string, D [][]int, nodes []Node, selected []bool, tour []int) Solution {
	var selList []int
	sumCosts := 0
	for i, v := range selected {
		if v {
			selList = append(selList, i)
			sumCosts += nodes[i].Cost
		}
	}
	tlen := tourLength(tour, D)
	return Solution{
		Method:   name,
		Selected: selList,
		Tour:     tour,
		TourLen:  tlen,
		CostSum:  sumCosts,
		Obj:      tlen + sumCosts,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
