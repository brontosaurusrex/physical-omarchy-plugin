package main

import "testing"

func TestNativeRoomFallbacks(t *testing.T) {
	oldName, oldDry := currentAudioRoom, currentAudioRoomDry
	t.Cleanup(func() { currentAudioRoom, currentAudioRoomDry = oldName, oldDry })
	currentAudioRoom, currentAudioRoomDry = "room_small", -1
	settings := nativeRoomSettings()
	if settings["name"] != "smallroom" || settings["dry"] != .78 {
		t.Fatalf("alias/default mix lost: %v", settings)
	}
	currentAudioRoomDry = .25
	if nativeRoomSettings()["dry"] != .25 {
		t.Fatal("level dry override ignored")
	}
	for _, name := range []string{"none", "unknown-preset"} {
		currentAudioRoom = name
		settings = nativeRoomSettings()
		if settings["name"] != "none" || settings["dry"] != float64(1) {
			t.Fatalf("dry/unknown preset leaked wet audio: %v", settings)
		}
	}
}

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
			want := 1
			if scenario.kind == "paddle" {
				want = 2
			}
			if len(nativeSounds) != want || nativeSounds[0]["kind"] != scenario.kind {
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

func TestNativePowerupAudioIdentity(t *testing.T) {
	oldSounds, oldEnabled, oldVisible := nativeSounds, enableSounds, nativeVisible
	t.Cleanup(func() { nativeSounds, enableSounds, nativeVisible = oldSounds, oldEnabled, oldVisible })
	enableSounds, nativeVisible = true, true
	for _, feature := range []string{"bigpaddle", "blackhole", "breakunbreakable", "dualballs", "influencer", "lowgravity", "magnet", "nuke", "passthrough", "reversegravity", "speedboost", "zapper"} {
		nativeSounds = nil
		playMagicFeature(" "+feature+" ", 500, .7, 300)
		if len(nativeSounds) != 1 || nativeSounds[0]["feature"] != feature || nativeSounds[0]["kind"] != "magic" {
			t.Fatalf("lost powerup identity: %v", nativeSounds)
		}
		args := nativeSounds[0]["args"].([]float64)
		if args[0] != 500 || args[1] != .7 {
			t.Fatalf("lost impact/pitch: %v", args)
		}
	}
	nativeSounds = nil
	enableSounds = false
	playMagicFeature("blackhole", 500, 1, 300)
	if len(nativeSounds) != 0 {
		t.Fatal("muted powerup emitted sound")
	}
	enableSounds, nativeVisible = true, false
	playMagicFeature("blackhole", 500, 1, 300)
	if len(nativeSounds) != 0 {
		t.Fatal("hidden powerup emitted sound")
	}
}

func TestNativeSynthParametersAndBrickBudget(t *testing.T) {
	oldSounds, oldEnabled, oldVisible, oldWidth, oldStarts := nativeSounds, enableSounds, nativeVisible, canvasWidth, brickHitStartsThisFrame
	t.Cleanup(func() {
		nativeSounds, enableSounds, nativeVisible, canvasWidth, brickHitStartsThisFrame = oldSounds, oldEnabled, oldVisible, oldWidth, oldStarts
	})
	nativeSounds = nil
	enableSounds = true
	nativeVisible = true
	canvasWidth = 800
	scheduleTonePanned("sawtooth", 920, 280, .08, .08, .012, 0)
	event := nativeSounds[0]
	args := event["args"].([]float64)
	if event["wave"] != "sawtooth" || args[0] != 920 || args[1] != 280 || args[2] != .08 || args[3] != .08*audioMixerSynths || args[4] != .012 || event["pan"] != -.75 {
		t.Fatalf("lost oscillator parameters: %v", event)
	}
	nativeSounds = nil
	brickHitStartsThisFrame = 0
	playMagicPitched(500, 1.32, 800)
	if len(nativeSounds) != 2 || nativeSounds[0]["wave"] != "sine" || nativeSounds[1]["wave"] != "triangle" {
		t.Fatalf("magic hit must use original generated pair: %v", nativeSounds)
	}
	first := nativeSounds[0]["args"].([]float64)[0]
	if first < 660*1.32*.94 || first > 660*1.32*1.06 {
		t.Fatalf("zapper magic pitch lost: %f", first)
	}
	nativeSounds = nil
	brickHitStartsThisFrame = 0
	for i := 0; i < 10; i++ {
		playBrickBreakPitched(500, 1, 400)
	}
	if len(nativeSounds) != brickHitMaxStartsPerFrame {
		t.Fatalf("brick burst budget: %d", len(nativeSounds))
	}
	nativeSounds = nil
	playMagicFeature("blackhole", 500, 1, 0)
	if nativeSounds[0]["cutoff"] != magicFeatureHardMaxHz || nativeSounds[0]["pan"] != -.75 {
		t.Fatalf("feature filter/pan differs from source: %v", nativeSounds[0])
	}
}
