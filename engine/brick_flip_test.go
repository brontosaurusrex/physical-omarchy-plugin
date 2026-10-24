package main

import (
	"math"
	"reflect"
	"slices"
	"testing"
)

func TestBrickFlipKeepsGameplayAndReplayGeometry(t *testing.T) {
	oldBricks, oldGrid, oldCells := bricks, brickGrid, brickGridCells
	oldRows, oldCols, oldCellW, oldCellH := gridRows, gridCols, gridCellWidth, gridCellHeight
	oldTop, oldLeft, oldRemaining := gridOffsetTop, gridOffsetLeft, remainingBreakableBricks
	oldFlip, oldDirty, oldEligible := bricksVerticalFlipped, bricksDirty, levelRunRecordEligible
	oldOver, oldComplete, oldEditor, oldMode := gameOver, levelAdvancePending, physicsEditorVisible, replayModeActive
	oldFrames, oldDefs, oldMirror, oldRecording := replayFrames, replayBrickDefs, replayBrickMirrorSum, replayRecording
	oldStatuses, oldRun := statusMessages, fullGameRun
	t.Cleanup(func() {
		bricks, brickGrid, brickGridCells = oldBricks, oldGrid, oldCells
		gridRows, gridCols, gridCellWidth, gridCellHeight = oldRows, oldCols, oldCellW, oldCellH
		gridOffsetTop, gridOffsetLeft, remainingBreakableBricks = oldTop, oldLeft, oldRemaining
		bricksVerticalFlipped, bricksDirty, levelRunRecordEligible = oldFlip, oldDirty, oldEligible
		gameOver, levelAdvancePending, physicsEditorVisible, replayModeActive = oldOver, oldComplete, oldEditor, oldMode
		replayFrames, replayBrickDefs, replayBrickMirrorSum, replayRecording = oldFrames, oldDefs, oldMirror, oldRecording
		statusMessages, fullGameRun = oldStatuses, oldRun
	})
	gridRows, gridCols, gridCellWidth, gridCellHeight = 6, 3, 25, 20
	gridOffsetTop, gridOffsetLeft = 20, 10
	bricks = []brick{
		{x: 10, y: 40, w: 15, h: 10, row: 1, col: 0, tiltRadians: .1, alive: true},
		{x: 35, y: 80, w: 15, h: 10, row: 3, col: 1, tiltRadians: -.2, alive: true, magic: true},
		{x: 60, y: 100, w: 15, h: 10, row: 4, col: 2, tiltRadians: .3, unbreakable: true},
	}
	original := slices.Clone(bricks)
	remainingBreakableBricks = 2
	bricksVerticalFlipped, levelRunRecordEligible = false, true
	gameOver, levelAdvancePending, physicsEditorVisible, replayModeActive = false, false, false, false
	replayFrames, replayRecording = nil, true
	rememberReplayBrickDefs()
	appendReplayFrame(true)
	beforeBall, beforePaddle, beforePhysics := ball, paddle, physicsConfig
	beforeLives, beforeScore, beforePaused, beforeReady := lives, score, paused, waitingForStart

	flipBricksVertically()
	if !bricksVerticalFlipped || !bricksDirty || levelRunRecordEligible || remainingBreakableBricks != 2 {
		t.Fatal("flip lost its cache, assistance or remaining-brick state")
	}
	if ball != beforeBall || paddle != beforePaddle || physicsConfig != beforePhysics ||
		lives != beforeLives || score != beforeScore || paused != beforePaused || waitingForStart != beforeReady {
		t.Fatal("flipping bricks changed gameplay outside the brick field")
	}
	for i, br := range bricks {
		want := original[i]
		want.row = 5 - want.row
		want.y = gridOffsetTop + float64(want.row)*gridCellHeight
		want.tiltRadians = -want.tiltRadians
		if br != want {
			t.Fatalf("brick %d: got %+v, want %+v", i, br, want)
		}
		key := gridKey(br.row, br.col)
		if !slices.Contains(brickGrid[key], i) || !slices.Contains(brickGridCells[key], i) {
			t.Fatalf("collision/debris grid lost moved brick %d", i)
		}
		query := &Ball{x: br.x + br.w/2, y: br.y + br.h/2, r: 1}
		if !slices.Contains(candidateBrickIndices(query, nil), i) {
			t.Fatalf("collision lookup missed moved brick %d", i)
		}
		for frameIndex, expected := range []brick{original[i], br} {
			def := replayBrickAt(&replayFrames[frameIndex], i)
			if math.Abs(float64(def.y)-expected.y) > 1e-4 || math.Abs(float64(def.tilt)-expected.tiltRadians) > 1e-6 {
				t.Fatalf("replay frame %d lost brick %d orientation", frameIndex, i)
			}
		}
	}
	flipBricksVertically()
	if bricksVerticalFlipped || !reflect.DeepEqual(bricks, original) {
		t.Fatal("two flips did not restore the exact original layout")
	}
}
