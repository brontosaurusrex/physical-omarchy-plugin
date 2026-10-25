package main

import "math"

const brickWipeHalfSeconds = .45

type brickFlipPose struct {
	row             int
	y, height, tilt float64
}
type brickWipeBounds struct{ left, right float64 }
type brickFlipAnimationState struct {
	active      bool
	elapsed     float64
	target      bool
	reverse     bool
	nextReverse bool
	poses       []brickFlipPose
	first, last int
	mirrorSum   float64
	bounds      brickWipeBounds
}

var bricksVerticalFlipped bool
var manualBrickFlip bool
var pendingMagicBrickFlips int
var magicBrickFlipActive bool
var magicBrickFlipTimer float64
var brickFlipAnimation brickFlipAnimationState

func resetBrickFlipState() {
	bricksVerticalFlipped, manualBrickFlip, magicBrickFlipActive = false, false, false
	pendingMagicBrickFlips, magicBrickFlipTimer = 0, 0
	brickFlipAnimation = brickFlipAnimationState{}
}

// Include destroyed bricks so the mirror axis stays fixed throughout a level.
func brickVerticalBounds() (firstRow, lastRow int, mirrorSum float64) {
	if len(bricks) == 0 {
		return
	}
	if len(brickFlipAnimation.poses) == len(bricks) {
		return brickFlipAnimation.first, brickFlipAnimation.last, brickFlipAnimation.mirrorSum
	}
	firstRow, lastRow = bricks[0].row, bricks[0].row
	top, bottom := bricks[0].y, bricks[0].y+bricks[0].h
	for _, br := range bricks[1:] {
		firstRow, lastRow = min(firstRow, br.row), max(lastRow, br.row)
		top, bottom = min(top, br.y), max(bottom, br.y+br.h)
	}
	return firstRow, lastRow, top + bottom
}

func desiredBrickFlip() bool { return manualBrickFlip != magicBrickFlipActive }

func flipBricksVertically() {
	if len(bricks) == 0 || gameOver || levelAdvancePending || replayModeActive || physicsEditorVisible {
		return
	}
	manualBrickFlip = !manualBrickFlip
	markLevelRunAssisted()
	requestBrickFlipAnimation(!desiredBrickFlip())
	if desiredBrickFlip() {
		showStatus("Brick rows flipped vertically", 2)
	} else {
		showStatus("Original brick layout", 2)
	}
}

// Resolve earned effects after contacts have finished using the old geometry.
// Another hit refreshes the usual power-up timer instead of toggling it off.
func applyPendingMagicBrickFlip() {
	if pendingMagicBrickFlips == 0 {
		return
	}
	pendingMagicBrickFlips = 0
	magicBrickFlipActive = true
	magicBrickFlipTimer = powerUpDuration
	requestBrickFlipAnimation(false)
}

func updateMagicBrickFlip(dt float64) {
	if !magicBrickFlipActive || brickFlipAnimation.active {
		return
	}
	magicBrickFlipTimer = math.Max(0, magicBrickFlipTimer-dt)
	if magicBrickFlipTimer == 0 {
		magicBrickFlipActive = false
		requestBrickFlipAnimation(true)
		showStatus("Brick Flip ended", 2)
	}
}

func clearMagicBrickFlip() {
	active := magicBrickFlipActive
	magicBrickFlipActive = false
	magicBrickFlipTimer = 0
	if active {
		requestBrickFlipAnimation(true)
	}
}

func requestBrickFlipAnimation(reverse bool) {
	if len(bricks) == 0 {
		return
	}
	a := &brickFlipAnimation
	if a.active {
		// Finish the current pass before applying another request, so bricks
		// already crossed by the wipe never snap back in the middle of it.
		a.nextReverse = reverse
		return
	}
	if bricksVerticalFlipped == desiredBrickFlip() {
		return
	}
	if len(a.poses) != len(bricks) {
		a.first, a.last, a.mirrorSum = brickVerticalBounds()
		a.poses = make([]brickFlipPose, len(bricks))
		for i, br := range bricks {
			pose := brickFlipPose{br.row, br.y, br.h, br.tiltRadians}
			if bricksVerticalFlipped {
				pose.row = a.first + a.last - pose.row
				pose.y = a.mirrorSum - pose.y - pose.height
				pose.tilt = -pose.tilt
			}
			a.poses[i] = pose
			if i == 0 {
				a.bounds = brickWipeBounds{br.x, br.x + br.w}
			}
			a.bounds.left, a.bounds.right = min(a.bounds.left, br.x), max(a.bounds.right, br.x+br.w)
		}
	}
	a.active, a.elapsed, a.target = true, 0, desiredBrickFlip()
	a.reverse, a.nextReverse = reverse, reverse
}

// The screen wipe crosses whole bricks along every row. Each brick keeps
// its full shape and switches directly between the two layouts at the front.
func brickFlipAtWipe(base, target bool, phase float32, reverse bool, x, width float64, bounds brickWipeBounds) bool {
	if phase <= 0 {
		return base
	}
	position := .5
	if bounds.right > bounds.left {
		position = clampFloat((x+width/2-bounds.left)/(bounds.right-bounds.left), 0, 1)
	}
	if reverse {
		position = 1 - position
	}
	if float64(phase)/2 >= position {
		return target
	}
	return base
}

func currentBrickWipePhase() float32 {
	if !brickFlipAnimation.active {
		return 0
	}
	return float32(brickFlipAnimation.elapsed / brickWipeHalfSeconds)
}

func advanceBrickFlipAnimation(dt float64) {
	a := &brickFlipAnimation
	if !a.active || dt <= 0 {
		return
	}
	a.elapsed = math.Min(2*brickWipeHalfSeconds, a.elapsed+dt)
	phase := currentBrickWipePhase()
	changed := false
	for i, pose := range a.poses {
		br := &bricks[i]
		row, y, tilt := pose.row, pose.y, pose.tilt
		if brickFlipAtWipe(bricksVerticalFlipped, a.target, phase, a.reverse, br.x, br.w, a.bounds) {
			row = a.first + a.last - row
			y = a.mirrorSum - y - pose.height
			tilt = -tilt
		}
		changed = changed || br.row != row || br.y != y || br.tiltRadians != tilt
		br.row, br.y, br.h, br.tiltRadians = row, y, pose.height, tilt
	}
	if changed {
		remaining := remainingBreakableBricks
		brickGrid = make(map[int][]int)
		brickGridCells = make([][]int, gridRows*gridCols)
		for i := range bricks {
			registerBrick(i)
		}
		remainingBreakableBricks = remaining
	}
	bricksDirty = bricksDirty || changed
	if phase >= 2 {
		bricksVerticalFlipped = a.target
		a.active = false
		appendReplayFrame(true)
		requestBrickFlipAnimation(a.nextReverse)
	} else if changed {
		appendReplayFrame(true)
	}
}
