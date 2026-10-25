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
	oldAnimation, oldManual, oldMagic, oldTimer, oldPending := brickFlipAnimation, manualBrickFlip, magicBrickFlipActive, magicBrickFlipTimer, pendingMagicBrickFlips
	t.Cleanup(func() {
		bricks, brickGrid, brickGridCells = oldBricks, oldGrid, oldCells
		gridRows, gridCols, gridCellWidth, gridCellHeight = oldRows, oldCols, oldCellW, oldCellH
		gridOffsetTop, gridOffsetLeft, remainingBreakableBricks = oldTop, oldLeft, oldRemaining
		bricksVerticalFlipped, bricksDirty, levelRunRecordEligible = oldFlip, oldDirty, oldEligible
		gameOver, levelAdvancePending, physicsEditorVisible, replayModeActive = oldOver, oldComplete, oldEditor, oldMode
		replayFrames, replayBrickDefs, replayBrickMirrorSum, replayRecording = oldFrames, oldDefs, oldMirror, oldRecording
		statusMessages, fullGameRun = oldStatuses, oldRun
		brickFlipAnimation, manualBrickFlip, magicBrickFlipActive, magicBrickFlipTimer, pendingMagicBrickFlips = oldAnimation, oldManual, oldMagic, oldTimer, oldPending
	})
	gridRows, gridCols, gridCellWidth, gridCellHeight = 6, 3, 25, 20
	gridOffsetTop, gridOffsetLeft = 20, 10
	bricks = []brick{
		{x: 10, y: 40, w: 15, h: 10, row: 1, col: 0, tiltRadians: .1, alive: true},
		{x: 35, y: 80, w: 15, h: 10, row: 3, col: 1, tiltRadians: -.2, alive: true, magic: true},
		{x: 60, y: 100, w: 15, h: 10, row: 4, col: 2, tiltRadians: .3, unbreakable: true},
	}
	original := slices.Clone(bricks)
	resetBrickFlipState()
	remainingBreakableBricks = 2
	bricksVerticalFlipped, levelRunRecordEligible = false, true
	gameOver, levelAdvancePending, physicsEditorVisible, replayModeActive = false, false, false, false
	replayFrames, replayRecording = nil, true
	rememberReplayBrickDefs()
	appendReplayFrame(true)
	beforeBall, beforePaddle, beforePhysics := ball, paddle, physicsConfig
	beforeLives, beforeScore, beforePaused, beforeReady := lives, score, paused, waitingForStart

	flipBricksVertically()
	if !brickFlipAnimation.active || bricksVerticalFlipped {
		t.Fatal("manual flip did not start with an outgoing wipe")
	}
	advanceBrickFlipAnimation(brickWipeHalfSeconds / 2)
	if bricks[0].y != original[2].y || bricks[1].y != original[1].y || bricks[2].y != original[2].y {
		t.Fatal("wipe did not switch the left brick before the right bricks")
	}
	for i, br := range bricks {
		if br.h != original[i].h || br.w != original[i].w {
			t.Fatal("wipe resized a brick")
		}
	}
	appendReplayFrame(true)
	for i, br := range bricks {
		def := replayBrickAt(&replayFrames[len(replayFrames)-1], i)
		if math.Abs(float64(def.y)-br.y) > 1e-4 || math.Abs(float64(def.h)-br.h) > 1e-4 {
			t.Fatal("replay lost the partial wipe")
		}
	}
	advanceBrickFlipAnimation(2 * brickWipeHalfSeconds)
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
			index := 0
			if frameIndex == 1 {
				index = len(replayFrames) - 1
			}
			def := replayBrickAt(&replayFrames[index], i)
			if math.Abs(float64(def.y)-expected.y) > 1e-4 || math.Abs(float64(def.tilt)-expected.tiltRadians) > 1e-6 {
				t.Fatalf("replay frame %d lost brick %d orientation", frameIndex, i)
			}
		}
	}
	flipBricksVertically()
	if !brickFlipAnimation.reverse {
		t.Fatal("return did not choose a right-to-left wipe")
	}
	advanceBrickFlipAnimation(brickWipeHalfSeconds / 2)
	if bricks[2].y != original[2].y || bricks[0].y != original[2].y {
		t.Fatal("return wipe did not restore the right brick first")
	}
	appendReplayFrame(true)
	for i, br := range bricks {
		def := replayBrickAt(&replayFrames[len(replayFrames)-1], i)
		if math.Abs(float64(def.y)-br.y) > 1e-4 || float64(def.h) != br.h {
			t.Fatal("replay lost the return wipe direction")
		}
	}
	advanceBrickFlipAnimation(2 * brickWipeHalfSeconds)
	if bricksVerticalFlipped || !reflect.DeepEqual(bricks, original) {
		t.Fatal("two flips did not restore the exact original layout")
	}
	// Earned flips hold for the normal duration and refresh on another hit.
	levelRunRecordEligible = true
	pendingMagicBrickFlips = 1
	applyPendingMagicBrickFlip()
	if brickFlipAnimation.reverse {
		t.Fatal("earned flip did not choose left-to-right")
	}
	advanceBrickFlipAnimation(2 * brickWipeHalfSeconds)
	if !bricksVerticalFlipped || !levelRunRecordEligible {
		t.Fatal("earned flip changed record eligibility")
	}
	updateMagicBrickFlip(powerUpDuration / 2)
	pendingMagicBrickFlips = 1
	applyPendingMagicBrickFlip()
	if magicBrickFlipTimer != powerUpDuration {
		t.Fatal("another magic hit did not refresh the duration")
	}
	updateMagicBrickFlip(powerUpDuration + 1)
	if magicBrickFlipActive || !brickFlipAnimation.active || !brickFlipAnimation.reverse {
		t.Fatal("expiry did not start the return wipe")
	}
	advanceBrickFlipAnimation(2 * brickWipeHalfSeconds)
	if bricksVerticalFlipped || !reflect.DeepEqual(bricks, original) {
		t.Fatal("expired power-up did not restore exact geometry")
	}
	// A second key queues a return after the current pass finishes.
	flipBricksVertically()
	advanceBrickFlipAnimation(brickWipeHalfSeconds / 2)
	flipBricksVertically()
	advanceBrickFlipAnimation(2 * brickWipeHalfSeconds)
	if !brickFlipAnimation.active || !bricksVerticalFlipped {
		t.Fatal("second key did not finish the first pass and queue the return")
	}
	advanceBrickFlipAnimation(2 * brickWipeHalfSeconds)
	if brickFlipAnimation.active || !reflect.DeepEqual(bricks, original) {
		t.Fatal("queued early return changed the original geometry")
	}
	// A late key also leaves the current pass intact before returning.
	flipBricksVertically()
	advanceBrickFlipAnimation(1.5 * brickWipeHalfSeconds)
	beforeQueuedReturn := slices.Clone(bricks)
	flipBricksVertically()
	if !reflect.DeepEqual(bricks, beforeQueuedReturn) {
		t.Fatal("incoming request snapped to another orientation")
	}
	advanceBrickFlipAnimation(brickWipeHalfSeconds)
	if !brickFlipAnimation.active {
		t.Fatal("incoming request did not queue a return wipe")
	}
	advanceBrickFlipAnimation(2 * brickWipeHalfSeconds)
	if brickFlipAnimation.active || !reflect.DeepEqual(bricks, original) {
		t.Fatal("queued return did not restore exact geometry")
	}
	// Expiry restores the manually selected base, including a flipped base.
	flipBricksVertically()
	advanceBrickFlipAnimation(2 * brickWipeHalfSeconds)
	pendingMagicBrickFlips = 1
	applyPendingMagicBrickFlip()
	advanceBrickFlipAnimation(2 * brickWipeHalfSeconds)
	if bricksVerticalFlipped || !manualBrickFlip {
		t.Fatal("magic flip did not reverse the manual base orientation")
	}
	updateMagicBrickFlip(powerUpDuration + 1)
	advanceBrickFlipAnimation(2 * brickWipeHalfSeconds)
	if !bricksVerticalFlipped || !manualBrickFlip {
		t.Fatal("magic expiry lost the manual base orientation")
	}
}
