package poi

import (
	"math"
	"math/rand"
)

// Point represents a 2D point with integer coordinates.
type Point struct {
	X int
	Y int
}

// PoissonDisk generates a set of points that are at least minDistance from each other.
// It returns a slice of points and the number of points generated.
// This implementation is a variation of Bridson's algorithm.
func PoissonDisk(minDistance, width, height int64, r *rand.Rand, noiseMap map[Point]float64, seaLevel float64) ([]Point, int) {
	
	// Data structures for the algorithm
	var points []Point
	var activePoints []Point
	
	// We use a grid to speed up the distance checks.
	cellSize := float64(minDistance) / math.Sqrt2
	gridWidth := int(math.Ceil(float64(width) / cellSize))
	gridHeight := int(math.Ceil(float64(height) / cellSize))
	grid := make([][]Point, gridWidth)
	for i := range grid {
		grid[i] = make([]Point, gridHeight)
	}
	
	// Add an initial random point on land
	var startPoint Point
	for {
		startPoint = Point{X: r.Intn(int(width)), Y: r.Intn(int(height))}
		if noiseMap[startPoint] >= seaLevel+0.05 {
			break
		}
	}
	
	points = append(points, startPoint)
	activePoints = append(activePoints, startPoint)
	gridX := int(float64(startPoint.X) / cellSize)
	gridY := int(float64(startPoint.Y) / cellSize)
	grid[gridX][gridY] = startPoint

	for len(activePoints) > 0 {
		randomIndex := r.Intn(len(activePoints))
		p := activePoints[randomIndex]
		
		foundCandidate := false
		for i := 0; i < 30; i++ { // Try up to 30 times
			
			// Generate a new candidate point in an annulus around the active point
			angle := r.Float64() * 2 * math.Pi
			dist := r.Float64()*(float64(minDistance)*2) + float64(minDistance)
			
			newPoint := Point{
				X: int(math.Round(float64(p.X) + math.Cos(angle)*dist)),
				Y: int(math.Round(float64(p.Y) + math.Sin(angle)*dist)),
			}
			
			// Check if the new point is within the bounds and on land
			if newPoint.X >= 0 && newPoint.X < int(width) && newPoint.Y >= 0 && newPoint.Y < int(height) && noiseMap[newPoint] >= seaLevel+0.05 {
				
				// Check if the candidate is far enough from existing points
				gridX = int(float64(newPoint.X) / cellSize)
				gridY = int(float64(newPoint.Y) / cellSize)
				
				ok := true
				for x := gridX - 2; x <= gridX+2; x++ {
					for y := gridY - 2; y <= gridY+2; y++ {
						if x >= 0 && x < gridWidth && y >= 0 && y < gridHeight && grid[x][y] != (Point{}) {
							dist := math.Sqrt(math.Pow(float64(newPoint.X-grid[x][y].X), 2) + math.Pow(float64(newPoint.Y-grid[x][y].Y), 2))
							if dist < float64(minDistance) {
								ok = false
							}
						}
					}
				}
				
				if ok {
					points = append(points, newPoint)
					activePoints = append(activePoints, newPoint)
					grid[gridX][gridY] = newPoint
					foundCandidate = true
					break
				}
			}
		}
		
		if !foundCandidate {
			activePoints = append(activePoints[:randomIndex], activePoints[randomIndex+1:]...)
		}
	}
	
	return points, len(points)
}
