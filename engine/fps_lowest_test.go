package main

import "testing"

func TestLowestFPSOnlySamplesGameplay(t *testing.T) {
	oldLow, oldElapsed, oldFrames, oldSeen, oldPlaying := fpsLowest, fpsLowestElapsed, fpsLowestFrames, fpsVisibleSamplesSeen, fpsLowestPlaying
	t.Cleanup(func() {
		fpsLowest, fpsLowestElapsed, fpsLowestFrames, fpsVisibleSamplesSeen, fpsLowestPlaying = oldLow, oldElapsed, oldFrames, oldSeen, oldPlaying
	})
	fpsLowest, fpsLowestElapsed, fpsLowestFrames, fpsVisibleSamplesSeen, fpsLowestPlaying = 0, 0, 0, 0, false
	for i := 0; i < 100; i++ {
		sampleLowestRenderFPS(.1, false)
	}
	if fpsLowest != 0 {
		t.Fatal("title screen established minimum")
	}
	sampleLowestRenderFPS(.9, true)
	for i := 0; i < 150; i++ {
		sampleLowestRenderFPS(1./60, true)
	}
	if fpsLowest < 59.9 || fpsLowest > 60.1 {
		t.Fatalf("gameplay minimum = %v", fpsLowest)
	}
	for i := 0; i < 100; i++ {
		sampleLowestRenderFPS(.1, false)
	}
	sampleLowestRenderFPS(.9, true)
	for i := 0; i < 100; i++ {
		sampleLowestRenderFPS(1./60, true)
	}
	if fpsLowest < 59.9 {
		t.Fatalf("pause/title transition contaminated minimum: %v", fpsLowest)
	}
	for i := 0; i < 120; i++ {
		sampleLowestRenderFPS(1./30, true)
	}
	if fpsLowest < 29.9 || fpsLowest > 30.1 {
		t.Fatalf("real gameplay slowdown ignored: %v", fpsLowest)
	}
}
