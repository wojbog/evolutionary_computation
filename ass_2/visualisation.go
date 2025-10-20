package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func readCSV(filePath string, separator rune) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = separator

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	return records, nil
}

func getMaxCoordinates(points []Point) rl.Vector2 {
	var maxX, maxY float32
	for _, point := range points {
		if point.pos.X > maxX {
			maxX = point.pos.X
		}
		if point.pos.Y > maxY {
			maxY = point.pos.Y
		}
	}
	return rl.Vector2{X: maxX, Y: maxY}
}

func drawPath(path []uint32, points []Point, outputFile string, title string, display bool) {
	var maxWindowSize int32 = 1512

	var margin float32 = 64.0

	maxCoords := getMaxCoordinates(points)

	fmt.Println("Max Coordinates:", maxCoords)

	var windowWidth, windowHeight int32

	if maxCoords.X > maxCoords.Y {
		windowWidth = maxWindowSize
		ratio := maxCoords.Y / maxCoords.X
		windowHeight = int32(float32(windowWidth) * ratio)
	} else {
		windowHeight = maxWindowSize
		ratio := maxCoords.X / maxCoords.Y
		windowWidth = int32(float32(windowHeight) * ratio)
	}

	fmt.Println("Window Size:", windowWidth, "x", windowHeight)

	scaledPositions := make([]rl.Vector2, len(points))
	for i, point := range points {
		fmt.Println("Original Position:", point.pos)
		scaledPositions[i] = rl.Vector2{
			X: (point.pos.X/float32(maxCoords.X))*(float32(windowWidth)-margin) + margin/2,
			Y: (point.pos.Y/float32(maxCoords.Y))*(float32(windowHeight)-margin) + margin/2,
		}
		fmt.Println("0-1", point.pos.X/float32(windowWidth), point.pos.Y/float32(windowHeight))
		fmt.Println("Scaled Position:", scaledPositions[i])
		fmt.Println("---")
	}

	costs := make([]uint64, len(points))
	for i, point := range points {
		costs[i] = point.cost
	}

	var minCircleSize float32 = 3.0
	var maxCircleSize float32 = 9.0

	maxCost := uint64(0)
	minCost := uint64(math.MaxUint64)
	for _, cost := range costs {
		if cost > maxCost {
			maxCost = cost
		}
		if cost < minCost {
			minCost = cost
		}
	}

	rl.InitWindow(windowWidth, windowHeight, "Path Visualization")
	defer rl.CloseWindow()

	renderTexture := rl.LoadRenderTexture(windowWidth, windowHeight)
	defer rl.UnloadRenderTexture(renderTexture)

	rl.BeginTextureMode(renderTexture)

	rl.ClearBackground(rl.RayWhite)

	rl.DrawText(title, 10, 10, 20, rl.DarkGray)

	for i, position := range scaledPositions {
		cirlceSize := minCircleSize + (float32(costs[i])/float32(maxCost))*(maxCircleSize-minCircleSize)
		rl.DrawCircleV(position, cirlceSize, rl.Blue)
		rl.DrawCircleLinesV(position, cirlceSize, rl.Black)
		rl.DrawText(fmt.Sprintf("%d", i), int32(position.X)+5, int32(position.Y)+5, 10, rl.Black)
	}

	for i := 0; i < len(path)-1; i++ {
		start := scaledPositions[path[i]]
		end := scaledPositions[path[i+1]]
		distance := rl.Vector2Distance(start, end)

		rl.DrawLineV(start, end, rl.Red)
		rl.DrawText(fmt.Sprintf("%.0f", distance), int32((start.X+end.X)/2), int32((start.Y+end.Y)/2), 6, rl.Red)
	}

	rl.DrawLineV(scaledPositions[path[len(path)-1]], scaledPositions[path[0]], rl.Red)
	distance := rl.Vector2Distance(scaledPositions[path[len(path)-1]], scaledPositions[path[0]])
	rl.DrawText(fmt.Sprintf("%.0f", distance), int32((scaledPositions[path[len(path)-1]].X+scaledPositions[path[0]].X)/2), int32((scaledPositions[path[len(path)-1]].Y+scaledPositions[path[0]].Y)/2), 6, rl.Red)

	rl.EndTextureMode()

	image := rl.LoadImageFromTexture(renderTexture.Texture)
	defer rl.UnloadImage(image)

	rl.ImageFlipVertical(image)
	rl.ExportImage(*image, outputFile)
	fmt.Println("Image saved to", outputFile)

	if display {
		for !rl.WindowShouldClose() {
			rl.BeginDrawing()
			rl.DrawTextureRec(renderTexture.Texture,
				rl.NewRectangle(0, 0, float32(windowWidth), -float32(windowHeight)),
				rl.NewVector2(0, 0),
				rl.White)
			rl.EndDrawing()
		}
	}

}

func parsePathString(pathString string) ([]uint32, error) {
	pathString = strings.Trim(pathString, "[]")
	parts := strings.Split(pathString, " ")

	path := make([]uint32, 0, len(parts))

	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid number '%s' in path string: %v", part, err)
		}
		path = append(path, uint32(n))
	}

	return path, nil
}

type Result struct {
	name       string
	objective  uint64
	pathLength uint64
	totalCost  uint64
	path       []uint32
}

func getPathFromCSV(filename string) ([]Result, error) {
	records, err := readCSV(filename, ',')
	if err != nil {
		return nil, fmt.Errorf("Error reading CSV file: %v", err)
	}

	results := make([]Result, 0, len(records)-1)

	for _, rec := range records[1:] {
		name := rec[0]
		objective, err1 := strconv.ParseUint(rec[2], 10, 64)
		pathLength, err2 := strconv.ParseUint(rec[3], 10, 64)
		totalCost, err3 := strconv.ParseUint(rec[4], 10, 64)

		pathString := rec[6]
		path, err4 := parsePathString(pathString)
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			return nil, fmt.Errorf("Error parsing record fields: %v %v %v %v", err1, err2, err3, err4)
		}

		results = append(results, Result{
			name:       name,
			objective:  objective,
			pathLength: pathLength,
			totalCost:  totalCost,
			path:       path,
		})
	}

	return results, nil
}

type Point struct {
	pos  rl.Vector2
	cost uint64
}

func getPointsFromCSV(filename string) ([]Point, error) {
	records, err := readCSV(filename, ';')
	if err != nil {
		return nil, fmt.Errorf("Error reading CSV file: %v", err)
	}
	records = records[1:]

	points := make([]Point, 0, len(records)-1)
	for _, rec := range records {
		x, err1 := strconv.ParseFloat(rec[0], 32)
		y, err2 := strconv.ParseFloat(rec[1], 32)
		cost, err3 := strconv.ParseUint(rec[2], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil {
			return nil, fmt.Errorf("Error parsing point coordinates or cost: %v %v %v", err1, err2, err3)
		}

		points = append(points, Point{pos: rl.Vector2{X: float32(x), Y: float32(y)}, cost: cost})
	}

	return points, nil
}

func main() {
	filename_pointsA := "./TSPA.csv"
	points, err := getPointsFromCSV(filename_pointsA)
	if err != nil {
		fmt.Printf("Error parsing points: %v", err)
		return
	}

	filename_pathA := "ass_2/best_A.csv"
	results, err := getPathFromCSV(filename_pathA)
	if err != nil {
		fmt.Printf("Error parsing path string: %v", err)
		return
	}
	for _, result := range results {
		title := fmt.Sprintf("A: %v - Objective: %v (Distance: %v Cost: %v)", result.name, result.objective, result.pathLength, result.totalCost)
		outpath := fmt.Sprintf("ass_2/TSPA_%v.png", result.name)
		drawPath(result.path, points, outpath, title, true)
	}

	filename_pointsB := "./TSPB.csv"
	points, err = getPointsFromCSV(filename_pointsB)
	if err != nil {
		fmt.Printf("Error parsing points: %v", err)
		return
	}

	filename_pathB := "ass_2/best_B.csv"
	results, err = getPathFromCSV(filename_pathB)
	if err != nil {
		fmt.Printf("Error parsing path string: %v", err)
		return
	}
	for _, result := range results {
		title := fmt.Sprintf("B: %v - Objective: %v (Distance: %v Cost: %v)", result.name, result.objective, result.pathLength, result.totalCost)
		outpath := fmt.Sprintf("ass_2/TSPB_%v.png", result.name)
		drawPath(result.path, points, outpath, title, true)
	}
}
