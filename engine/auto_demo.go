package main

var autoDemoWait float64

// Run from fixed steps, including the ordinary idle-screen ticks. Completion
// and game-over screens therefore continue even after debris stops animating.
// Explicit pauses, replay and the editor retain control of the game.
func updateAutoDemo(dt float64) bool {
	if !autoPaddleEnabled || paused || replayModeActive || physicsEditorVisible || dt <= 0 {
		autoDemoWait = 0
		return false
	}
	if currentLevelIndex == 13 && !gameOver && !levelAdvancePending {
		startLevel(14)
		return true
	}
	if !gameOver && !levelAdvancePending {
		autoDemoWait = 0
		return false
	}
	autoDemoWait += dt
	if autoDemoWait+1e-9 < autoDemoTransitionSeconds {
		return false
	}
	autoDemoWait = 0
	if gameOver {
		// Both losing and finishing the whole game begin a fresh demo run.
		jumpToLevel(0)
	} else {
		advanceFromLevelComplete()
	}
	return true
}
