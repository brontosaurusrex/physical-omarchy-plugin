package main

import "testing"

func TestNativeContactAudio(t *testing.T) {
	oldSounds, oldEnabled, oldVisible := nativeSounds, enableSounds, nativeVisible
	oldBricks, oldPaddle, oldWidth, oldHeight := bricks, paddle, canvasWidth, canvasHeight
	oldPhysics, oldGravity, oldHole := physicsConfig, currentGravity, blackHoleActive
	t.Cleanup(func() {
		nativeSounds, enableSounds, nativeVisible = oldSounds, oldEnabled, oldVisible
		bricks, paddle, canvasWidth, canvasHeight = oldBricks, oldPaddle, oldWidth, oldHeight
		physicsConfig, currentGravity, blackHoleActive = oldPhysics, oldGravity, oldHole
	})
	bricks = nil
	canvasWidth = 800
	canvasHeight = 600
	paddle.x = 300
	paddle.y = 550
	paddle.w = 160
	paddle.h = 20
	paddle.vx = 0
	physicsConfig = defaultPhysicsSettings()
	currentGravity = 0
	blackHoleActive = false
	enableSounds = true
	nativeVisible = true
	for _, scenario := range []struct {
		name, kind string
		ball       Ball
	}{
		{"left", "wall", Ball{x: 7, y: 400, r: 8, vx: -400}},
		{"right", "wall", Ball{x: 793, y: 400, r: 8, vx: 400}},
		{"top", "wall", Ball{x: 400, y: 7, r: 8, vy: -400}},
		{"paddle", "paddle", Ball{x: 380, y: 542, r: 8, vy: 400}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			nativeSounds = nil
			b := scenario.ball
			updateBallStep(&b, physicsStepSeconds, false)
			if len(nativeSounds) != 1 || nativeSounds[0]["kind"] != scenario.kind {
				t.Fatalf("contact produced %v, want %s", nativeSounds, scenario.kind)
			}
			if b.soundCooldown <= 0 {
				t.Fatal("contact failed to apply sound cooldown")
			}
		})
	}
	for _, muted := range []bool{true, false} {
		enableSounds = !muted
		nativeVisible = muted
		nativeSounds = nil
		playWallHit()
		playPaddleHit()
		if len(nativeSounds) != 0 {
			t.Fatal("muted or hidden game emitted contact sounds")
		}
	}
}
