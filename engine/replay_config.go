package main

// Replay/export defaults live in their own file so they can be dropped into the
// existing project without replacing config.go.
const (
	defaultReplayFPS               = 30.0
	defaultReplayMaxSeconds        = 600.0
	defaultReplayRecordDebris      = true
	defaultReplayMaxDebrisPerFrame = 150
	defaultReplayPlaybackMaxSpeed  = 8.0
	defaultReplayShowHUD           = true // replay viewer only; V/B exports never include HUD
	defaultReplayTrailFrames       = 8    // previous replay samples, plus current frame at full opacity
)
