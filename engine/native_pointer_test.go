package main

import "testing"

func TestNativeRelativeMouseCapture(t *testing.T) {
	oldPaddle, oldWidth := paddle, canvasWidth
	oldLocked, oldControl, oldTarget := mousePointerLocked, mouseControlActive, mousePaddleTargetX
	oldPaused, oldReady, oldOver, oldComplete := paused, waitingForStart, gameOver, levelAdvancePending
	oldEditor, oldReplay := physicsEditorVisible, replayModeActive
	oldLeft, oldRight, oldHistory := leftPressed, rightPressed, paddleSpinHistory
	t.Cleanup(func() {
		paddle, canvasWidth = oldPaddle, oldWidth
		mousePointerLocked, mouseControlActive, mousePaddleTargetX = oldLocked, oldControl, oldTarget
		paused, waitingForStart, gameOver, levelAdvancePending = oldPaused, oldReady, oldOver, oldComplete
		physicsEditorVisible, replayModeActive = oldEditor, oldReplay
		leftPressed, rightPressed, paddleSpinHistory = oldLeft, oldRight, oldHistory
	})
	canvasWidth, paddle.x, paddle.w = 800, 300, 160
	paused, waitingForStart, gameOver, levelAdvancePending = false, false, false, false
	physicsEditorVisible, replayModeActive = false, false
	handleNativeCommand(nativeCommand{Action: "pointer-lock", Down: true})
	handleNativeCommand(nativeCommand{Action: "relative-pointer", X: 40})
	if paddle.x != 340 || !mouseControlActive {
		t.Fatalf("relative movement: x=%v, active=%v", paddle.x, mouseControlActive)
	}
	handleNativeCommand(nativeCommand{Action: "pointer", X: 0})
	if mousePaddleTargetX != 340 {
		t.Fatal("absolute cursor jumped captured paddle")
	}
	handleNativeCommand(nativeCommand{Action: "relative-pointer", X: 10000})
	if paddle.x != 640 {
		t.Fatal("captured paddle escaped right wall")
	}
	handleNativeCommand(nativeCommand{Action: "relative-pointer", X: -10000})
	if paddle.x != 0 {
		t.Fatal("captured paddle escaped left wall")
	}
	paused = true
	handleNativeCommand(nativeCommand{Action: "relative-pointer", X: 40})
	if paddle.x != 0 {
		t.Fatal("captured paddle moved while paused")
	}
	handleNativeCommand(nativeCommand{Action: "pointer-lock", Down: false})
	if mousePointerLocked || mouseControlActive || paddle.vx != 0 {
		t.Fatal("mouse capture did not release")
	}
}
