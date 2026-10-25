package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	js "physical/internal/qtbridge"
	"strconv"
	"strings"
)

//go:embed levels/level*.txt
var nativeLevels embed.FS

type nativeCommand struct {
	DiagnosticSequence int64             `json:"_diag_seq"`
	DiagnosticSentUs   int64             `json:"_diag_sent_us"`
	Action             string            `json:"action"`
	Key                string            `json:"key"`
	Code               string            `json:"code"`
	Repeat             bool              `json:"repeat"`
	Shift              bool              `json:"shift"`
	Down               bool              `json:"down"`
	X                  float64           `json:"x"`
	Y                  float64           `json:"y"`
	DT                 float64           `json:"dt"`
	Value              float64           `json:"value"`
	Group              string            `json:"group"`
	Theme              map[string]string `json:"theme"`
}

var nativeVolume = 0.25

const nativeVolumeKey = "physical.audioVolume"

var nativeClipboard, nativeExport, nativeError string
var nativeSounds []map[string]any
var nativeVisible = true
var nativeTime float64
var nativeTunerDirty = true
var nativeLastRoom string

func nativeRoomSettings() map[string]any {
	preset, found := audioRoomPresetForName(currentAudioRoom)
	if !found {
		preset = defaultAudioRoomPresets["none"]
	}
	dry := preset.defaultDry
	if currentAudioRoomDry >= 0 {
		dry = clampFloat(currentAudioRoomDry, 0, 1)
	}
	if preset.name == "none" {
		dry = 1
	}
	return map[string]any{
		"name": preset.name, "dry": dry,
		"inputType": preset.inputFilterType, "inputHz": preset.inputFrequency, "inputQ": preset.inputQ,
		"outputType": preset.outputFilterType, "outputHz": preset.outputFrequency, "outputQ": preset.outputQ,
		"feedbackHz": math.Min(preset.outputFrequency, 5000), "feedbackQ": .45,
		"delay1": preset.delay1, "delay2": preset.delay2, "delay3": preset.delay3,
		"gain1": preset.tapGain1, "gain2": preset.tapGain2, "gain3": preset.tapGain3,
		"feedback": clampFloat(preset.feedback, 0, .75),
	}
}

// Carry the original oscillator and sample parameters without quantizing them.
func nativeTone(kind, wave string, start, end, duration, volume, delay, hitX float64) {
	if !enableSounds || !nativeVisible || len(nativeSounds) >= 64 {
		return
	}
	nativeSounds = append(nativeSounds, map[string]any{"kind": kind, "wave": wave, "args": []float64{start, end, duration, volume * audioMixerSynths, delay}, "pan": audioPanFromX(hitX)})
}
func nativeSampleSound(kind, feature string, impact, pitch, hitX float64) {
	if !enableSounds || !nativeVisible || len(nativeSounds) >= 64 {
		return
	}
	if pitch <= 0 {
		pitch = 1
	}
	strength := math.Sqrt(clampFloat((impact-minimumCollisionSoundSpeed)/(math.Max(physicsConfig.maxSpeed, minimumCollisionSoundSpeed+1)-minimumCollisionSoundSpeed), 0, 1))
	rate := audioRand(brickHitPlaybackRateMin, brickHitPlaybackRateMax) * pitch
	cutoff := (brickHitFilterMinHz + (brickHitFilterMaxHz-brickHitFilterMinHz)*strength) * audioRand(.86, 1.14)
	cutoff = clampFloat(cutoff, 800, 48000*.45)
	q := audioRand(.25, .85)
	gain := (brickHitGainMin + (brickHitGainMax-brickHitGainMin)*strength) * audioRand(.90, 1.08) * audioMixerBricks
	if feature != "" {
		rate = audioRand(magicFeaturePlaybackRateMin, magicFeaturePlaybackRateMax) * pitch
		cutoff = (magicFeatureFilterMinHz + (magicFeatureFilterMaxHz-magicFeatureFilterMinHz)*strength) * audioRand(.90, 1.10)
		// Preserve the original hard cap (500 Hz), including its clamp order.
		cutoff = clampFloat(cutoff, 800, magicFeatureHardMaxHz)
		q = audioRand(.20, .70)
		gain = (magicFeatureGainMin + (magicFeatureGainMax-magicFeatureGainMin)*strength) * audioRand(.94, 1.06) * audioMixerMagic
	}
	nativeSounds = append(nativeSounds, map[string]any{"kind": kind, "feature": strings.ToLower(strings.TrimSpace(feature)), "args": []float64{impact, pitch}, "rate": rate, "gain": gain, "cutoff": cutoff, "q": q, "pan": audioPanFromX(hitX)})
}

func nativeExportSVG(filename, svg string) bool {
	if svg == "" {
		return false
	}

	home, err := os.UserHomeDir()
	if err != nil {
		nativeError = err.Error()
		return false
	}
	dir := filepath.Join(home, "Pictures", "Physical")
	if custom := os.Getenv("PHYSICAL_EXPORT_DIR"); custom != "" {
		dir = custom
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		nativeError = err.Error()
		return false
	}
	path := filepath.Join(dir, filepath.Base(filename))
	if err = os.WriteFile(path, []byte(svg), 0644); err != nil {
		nativeError = err.Error()
		return false
	}
	nativeExport = path
	return true
}
func nativeKey(c nativeCommand) {
	event := js.ValueOf(map[string]any{"key": c.Key, "code": c.Code, "repeat": c.Repeat, "shiftKey": c.Shift, "target": map[string]any{"tagName": "CANVAS"}})
	if c.Down {
		keyDown.Invoke(event)
	} else {
		keyUp.Invoke(event)
	}
	if c.Key == "e" || c.Key == "E" {
		nativeTunerDirty = true
	}
}
func nativeTuner() []map[string]any {
	sliders := []map[string]any{}
	for _, s := range physicsEditorSliderSpecs {
		sliders = append(sliders, map[string]any{"kind": "physics", "key": s.key, "group": s.group, "label": s.label, "min": s.min, "max": s.max, "step": s.step, "precision": s.precision, "value": physicsEditorSpecValue(s)})
	}
	for _, s := range debrisEditorSliderSpecs {
		sliders = append(sliders, map[string]any{"kind": "debris", "key": s.key, "group": s.group, "label": s.label, "min": s.min, "max": s.max, "step": s.step, "precision": s.precision, "value": debrisEditorValue(s.key)})
	}
	for _, s := range autoPaddleEditorSliderSpecs {
		sliders = append(sliders, map[string]any{"kind": "auto", "key": s.key, "group": s.group, "label": s.label, "min": s.min, "max": s.max, "step": s.step, "precision": s.precision, "value": autoPaddleEditorValue(s.key)})
	}
	return sliders
}
func handleNativeCommand(c nativeCommand) {
	switch c.Action {
	case "hello":
		nativeTunerDirty = true
		nativeLastRoom = ""
	case "tick":
		if !nativeVisible {
			return
		}
		now := js.Now()
		dt := (now - nativeTime) / 1000
		if c.DT > 0 {
			dt = c.DT
			now = nativeTime + dt*1000
		}
		nativeTime = now
		if dt > physicsMaxFrameDelta {
			dt = physicsMaxFrameDelta
		}
		if dt < 0 {
			dt = 0
		}
		js.SetClock(now)
		// The original loop owns fixed steps, interpolation, diagnostics, and replay.
		gameLoop(js.Undefined(), nil)
	case "key":
		nativeKey(c)
	case "pointer-lock":
		mousePointerLocked = c.Down
		mouseControlActive = c.Down
		mousePaddleTargetX = paddle.x
		leftPressed, rightPressed = false, false
		paddle.vx = 0
		resetPaddleSpinHistory()
	case "relative-pointer":
		if !mousePointerLocked || paused || physicsEditorVisible || replayModeActive {
			return
		}
		mousePaddleTargetX = clampFloat(mousePaddleTargetX+c.X*defaultMousePointerLockSensitivity, 0, math.Max(0, canvasWidth-paddle.w))
		mouseControlActive = true
		leftPressed, rightPressed = false, false
		if !gameOver && !waitingForStart && !levelAdvancePending {
			paddle.x = mousePaddleTargetX
		}
	case "pointer":
		if mousePointerLocked {
			if c.Down {
				nativeKey(nativeCommand{Key: " ", Code: "Space", Down: true})
			}
			return
		}
		if physicsEditorVisible || replayModeActive {
			return
		}
		mouseControlActive = true
		mousePaddleTargetX = clampFloat(c.X-paddle.w/2, 0, canvasWidth-paddle.w)
		if c.Down {
			nativeKey(nativeCommand{Key: " ", Code: "Space", Down: true})
		}
	case "visibility":
		nativeVisible = c.Down
		nativeTime = js.Now()
		lastTime = nativeTime
		physicsAccumulator = 0
		leftPressed = false
		rightPressed = false
		mouseControlActive = false
		paddle.vx = 0
		if !nativeVisible {
			paused = true
			replayPlaybackDirection = 0
			nativeSounds = nil
		}
	case "pause":
		paused = true
		leftPressed = false
		rightPressed = false
		paddle.vx = 0
		replayPlaybackDirection = 0
	case "toggle-sound":
		toggleSound()
	case "replay-exit":
		exitReplay()
	case "restart":
		if physicsEditorVisible {
			closePhysicsEditor()
		}
		retryCurrentLevel()
		nativeTunerDirty = true
	case "volume":
		nativeVolume = clampFloat(c.Value, 0, 1)
		js.Global().Get("localStorage").Call("setItem", nativeVolumeKey, strconv.FormatFloat(nativeVolume, 'f', -1, 64))
	case "setting":
		markLevelRunAssisted()
		switch c.Group {
		case "physics":
			for _, s := range physicsEditorSliderSpecs {
				if s.key == c.Key {
					applyPhysicsEditorValue(s, c.Value)
				}
			}
		case "debris":
			for _, s := range debrisEditorSliderSpecs {
				if s.key == c.Key {
					applyDebrisEditorValue(s, c.Value)
				}
			}
		case "auto":
			for _, s := range autoPaddleEditorSliderSpecs {
				if s.key == c.Key {
					applyAutoPaddleEditorValue(s, c.Value)
				}
			}
		case "toggle":
			switch c.Key {
			case "live":
				physicsEditorLiveSimulation = c.Down
				paused = !c.Down
			case "debris":
				debrisEnabled = c.Down
			case "corner":
				physicsConfig.cornerPhysicsEnabled = c.Down
			case "allCorners":
				physicsConfig.cornerPhysicsAllBricks = c.Down
			}
		}
	case "tuner-reset":
		resetPhysicsEditorTo(defaultPhysicsSettings(), defaultDebrisEditorSnapshot(), defaultAutoPaddleHitVariation)
		nativeTunerDirty = true
	case "tuner-export":
		nativeClipboard = physicsEditorClipboardText()
	}
}
func nativeStatus() string {
	if len(statusMessages) == 0 {
		return ""
	}
	return statusMessages[len(statusMessages)-1].text
}
func emitNativeFrame(encoder *json.Encoder, action string, timing ...*nativeFrameTiming) error {
	if action != "tick" {
		if replayModeActive {
			drawReplayCurrentFrame()
		} else {
			draw(renderInterpolationAlpha)
		}
	}
	ops, caches := js.Frame()
	remaining := 0
	for _, b := range bricks {
		if b.alive && !b.unbreakable {
			remaining++
		}
	}
	packet := map[string]any{
		"type": "frame", "action": action, "ops": ops, "caches": caches, "width": canvasWidth, "height": canvasHeight,
		"state": map[string]any{"level": currentLevelIndex + 1, "levels": len(levels), "lives": lives, "score": score, "paused": paused, "ready": waitingForStart, "gameOver": gameOver, "win": win, "complete": levelAdvancePending, "balls": activeBallCount(), "remaining": remaining, "auto": autoPaddleEnabled, "gameSpeed": gameSpeedPercent, "runSpeed": levelRunStats.speedPercent, "mixedSpeed": levelRunStats.mixedSpeed, "levelTime": levelRunStats.elapsedSeconds, "recordEligible": levelRunRecordEligible, "flipped": bricksVerticalFlipped, "flipAnimating": brickFlipAnimation.active, "flipPhase": currentBrickWipePhase(), "flipReverse": brickFlipAnimation.reverse, "flipPowerActive": magicBrickFlipActive, "flipTimer": magicBrickFlipTimer, "sound": enableSounds, "volume": nativeVolume, "palette": palette, "fps": fpsCurrent, "replay": replayModeActive, "replayFrames": len(replayFrames), "replayFrame": replayFrameIndex, "replayDirection": replayPlaybackDirection, "replaySpeed": replayPlaybackSpeed, "editor": physicsEditorVisible, "live": physicsEditorLiveSimulation, "debris": debrisEnabled, "debrisMaster": debrisMasterEnabled, "debrisPieces": len(brickDebris), "corner": physicsConfig.cornerPhysicsEnabled, "allCorners": physicsConfig.cornerPhysicsAllBricks, "x": ball.x, "y": ball.y, "omega": ball.omega, "physicsSteps": physicsLastFrameSteps, "paddleX": paddle.x, "status": nativeStatus(), "magnet": magnetIsActive(), "zapper": zapperIsActive(), "fpsVisible": fpsMiniOverlayVisible, "fpsLowest": fpsLowest, "diagnostics": []bool{physicsOverlayVisible, debugOverlayVisible, statsOverlayVisible}, "unlocked": devAllLevelsUnlocked},
	}
	if shapes := js.DebrisShapes(); len(shapes) > 0 {
		packet["debrisShapes"] = shapes
	}
	if nativeTunerDirty {
		packet["sliders"] = nativeTuner()
		nativeTunerDirty = false
	}
	roomKey := fmt.Sprintf("%s/%g", currentAudioRoom, currentAudioRoomDry)
	if roomKey != nativeLastRoom {
		packet["room"] = nativeRoomSettings()
		nativeLastRoom = roomKey
	}
	if len(nativeSounds) > 0 {
		packet["sounds"] = nativeSounds
		nativeSounds = nil
	}
	if nativeClipboard != "" {
		packet["clipboard"] = nativeClipboard
		nativeClipboard = ""
	}
	if nativeExport != "" {
		packet["export"] = nativeExport
		nativeExport = ""
	}
	if err := js.StorageError(); err != "" {
		packet["error"] = err
	}
	if nativeError != "" {
		packet["error"] = nativeError
		nativeError = ""
	}
	if len(timing) > 0 && timing[0] != nil {
		return emitTimedNativeFrame(os.Stdout, packet, timing[0])
	}
	return encoder.Encode(packet)
}
func nativeMain() {
	stateHome := os.Getenv("XDG_DATA_HOME")
	if stateHome == "" {
		home, _ := os.UserHomeDir()
		stateHome = filepath.Join(home, ".local", "share")
	}
	statePath := filepath.Join(stateHome, "physical-omarchy", "progress.json")
	if custom := os.Getenv("PHYSICAL_STATE_PATH"); custom != "" {
		statePath = custom
	}
	canvas = js.Initialize(nativeLevels, statePath)
	storedVolume := js.Global().Get("localStorage").Call("getItem", nativeVolumeKey)
	if value, err := strconv.ParseFloat(storedVolume.String(), 64); !storedVolume.IsNull() && !storedVolume.IsUndefined() && err == nil && !math.IsNaN(value) && !math.IsInf(value, 0) {
		nativeVolume = clampFloat(value, 0, 1)
	}
	ctx = canvas.Call("getContext", "2d")
	ensureBrickCanvas()
	loadDebrisMasterPreference()
	loadGameSpeedPreference()
	loadLevels()
	setupInput()
	resetGame()
	syncRenderInterpolation()
	nativeTime = js.Now()
	lastTime = nativeTime
	encoder := json.NewEncoder(os.Stdout)
	js.BeginFrame()
	if err := emitNativeFrame(encoder, "init"); err != nil {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), 65536)
	for scanner.Scan() {
		readUs := nativeDiagnosticNowUs()
		var c nativeCommand
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			fmt.Fprintln(os.Stderr, "Invalid input:", err)
			continue
		}
		if c.Action == "quit" {
			return
		}
		if math.IsNaN(c.Value) || math.IsInf(c.Value, 0) {
			continue
		}
		js.BeginFrame()
		handleNativeCommand(c)
		if (c.Action == "pointer" && !c.Down) || c.Action == "relative-pointer" {
			continue
		}
		var timing *nativeFrameTiming
		if c.DiagnosticSequence > 0 {
			timing = &nativeFrameTiming{Sequence: c.DiagnosticSequence, SentUs: c.DiagnosticSentUs, ReadUs: readUs}
		}
		if err := emitNativeFrame(encoder, c.Action, timing); err != nil {
			return
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
