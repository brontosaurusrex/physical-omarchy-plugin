package main

import (
	"math/rand"
	"testing"
)

func TestDebrisDirectGridMatchesOriginalQueries(t *testing.T) {
	oldBricks, oldGrid, oldCells := bricks, brickGrid, brickGridCells
	oldRows, oldCols := gridRows, gridCols
	oldWidth, oldHeight, oldLeft, oldTop, oldRange := gridCellWidth, gridCellHeight, gridOffsetLeft, gridOffsetTop, magnetRange
	oldRemaining := remainingBreakableBricks
	t.Cleanup(func() {
		bricks, brickGrid, brickGridCells = oldBricks, oldGrid, oldCells
		gridRows, gridCols = oldRows, oldCols
		gridCellWidth, gridCellHeight, gridOffsetLeft, gridOffsetTop, magnetRange = oldWidth, oldHeight, oldLeft, oldTop, oldRange
		remainingBreakableBricks = oldRemaining
	})
	gridRows, gridCols = 15, 44
	gridCellWidth, gridCellHeight, gridOffsetLeft, gridOffsetTop, magnetRange = 20, 32, 120, 45, 400
	brickGrid = make(map[int][]int)
	brickGridCells = make([][]int, gridRows*gridCols)
	bricks = nil
	random := rand.New(rand.NewSource(1))
	// Sparse cells, duplicate cells, dead and unbreakable bricks all retain order.
	for _, cell := range random.Perm(gridRows * gridCols) {
		if cell%4 == 0 {
			continue
		}
		for copy := 0; copy < 1+cell%2; copy++ {
			row, col := cell/gridCols, cell%gridCols
			bricks = append(bricks, brick{row: row, col: col, x: gridOffsetLeft + float64(col)*gridCellWidth, y: gridOffsetTop + float64(row)*gridCellHeight, w: 15, h: 25, alive: cell%5 != 0, unbreakable: cell%7 == 0})
			registerBrick(len(bricks) - 1)
		}
	}
	cells := brickGridCells
	for change := 0; change < 3; change++ {
		for i := 0; i < 1000; i++ {
			x, y := random.Float64()*2000-300, random.Float64()*1600-300
			brickGridCells = nil
			wantX, wantY, wantDistance, wantFound := nearestMagneticBrickForDebris(x, y)
			brickGridCells = cells
			gotX, gotY, gotDistance, gotFound := nearestMagneticBrickForDebris(x, y)
			if gotX != wantX || gotY != wantY || gotDistance != wantDistance || gotFound != wantFound {
				t.Fatalf("direct grid changed query at %g,%g: got %g,%g,%g,%v want %g,%g,%g,%v", x, y, gotX, gotY, gotDistance, gotFound, wantX, wantY, wantDistance, wantFound)
			}
		}
		for i := range bricks {
			if i%3 == change {
				bricks[i].alive = false
			}
		}
	}
}
