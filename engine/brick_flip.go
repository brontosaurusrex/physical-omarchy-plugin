package main

var bricksVerticalFlipped bool

// Include destroyed bricks so the mirror axis stays fixed throughout a level.
// Empty padding outside the brick field remains where the level placed it.
func brickVerticalBounds() (firstRow, lastRow int, mirrorSum float64) {
	if len(bricks) == 0 {
		return
	}
	firstRow, lastRow = bricks[0].row, bricks[0].row
	top, bottom := bricks[0].y, bricks[0].y+bricks[0].h
	for _, br := range bricks[1:] {
		firstRow = min(firstRow, br.row)
		lastRow = max(lastRow, br.row)
		top = min(top, br.y)
		bottom = max(bottom, br.y+br.h)
	}
	return firstRow, lastRow, top + bottom
}

func flipBricksVertically() {
	if len(bricks) == 0 || gameOver || levelAdvancePending || replayModeActive || physicsEditorVisible {
		return
	}
	first, last, _ := brickVerticalBounds()
	remaining := remainingBreakableBricks
	brickGrid = make(map[int][]int)
	brickGridCells = make([][]int, gridRows*gridCols)
	for i := range bricks {
		br := &bricks[i]
		br.row = first + last - br.row
		br.y = gridOffsetTop + float64(br.row)*gridCellHeight
		br.tiltRadians = -br.tiltRadians
		registerBrick(i)
	}
	// registerBrick also counts breakable bricks, including destroyed entries.
	remainingBreakableBricks = remaining
	bricksVerticalFlipped = !bricksVerticalFlipped
	bricksDirty = true
	markLevelRunAssisted()
	appendReplayFrame(true)
	if bricksVerticalFlipped {
		showStatus("Brick rows flipped vertically", 2.0)
	} else {
		showStatus("Original brick layout", 2.0)
	}
}
