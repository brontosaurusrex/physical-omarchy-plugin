package main

import (
	"fmt"
	"math"
	js "physical/internal/qtbridge"
	"slices"
	"strconv"
	"strings"
)

// Replay is deliberately visual-state based rather than a second physics
// simulation. That makes scrubbing exact and cheap: the live game is never
// rewound or mutated while replaying.
type replayBallState struct {
	x, y, angle float32
	vx, vy      float32
	omega       float32
	zapper      int32
}

type replayDebrisState struct {
	id          uint32
	x, y, angle float32
	opacity     float32
	front       bool
}

type replayFrame struct {
	timeSeconds float32
	balls       [defaultMaxBalls]replayBallState
	ballCount   uint8

	paddleX, paddleY float32
	paddleW, paddleH float32

	blackHoleX, blackHoleY float32
	blackHoleActive        bool
	zapperActive           bool
	drawBrickTilt          bool
	bricksFlipped          bool

	lives int32
	score int32

	brickAlive []uint64
	debris     []replayDebrisState
}

type replayBrickDef struct {
	x, y, w, h  float32
	tilt        float32
	magic       bool
	unbreakable bool
}

type replayDebrisShape struct {
	id         uint32
	circle     bool
	radius     float32
	roundness  float32
	pointCount int
	points     []float32
	fillColor  string
}

var (
	replayFrames         []replayFrame
	replayBrickDefs      []replayBrickDef
	replayBrickMirrorSum float32
	replayDebrisShapes   map[uint32]replayDebrisShape
	replayLevelIndex     int

	replayRecording          bool
	replayRecordingFinished  bool
	replayRecordingFull      bool
	replayCaptureAccumulator float64

	replayModeActive        bool
	replayFrameIndex        int
	replayPlaybackDirection int // -1 reverse, 0 pause, +1 forward
	replayPlaybackSpeed     = 1.0
	replayPlaybackAccum     float64
	replayReturnPaused      bool

	// A separate cache keeps scrubbing from changing the live brick mesh.
	replayBrickCanvas     js.Value
	replayBrickCtx        js.Value
	replayBrickCacheAlive []uint64
	replayBrickCacheStyle replayBrickCacheSettings
	replayBrickCacheValid bool
)

type replayBrickCacheSettings struct {
	width, height, radius float64
	tilt                  bool
	flipped               bool
	colors                [6]string
}

func replayFrameInterval() float64 {
	if defaultReplayFPS <= 0 {
		return 1.0 / 30.0
	}
	return 1.0 / defaultReplayFPS
}

func replayMaximumFrames() int {
	if defaultReplayMaxSeconds <= 0 || defaultReplayFPS <= 0 {
		return 0
	}
	return int(math.Ceil(defaultReplayMaxSeconds*defaultReplayFPS)) + 2
}

func replayAvailable() bool {
	return len(replayFrames) > 0
}

func beginReplayRecording(levelIndex int) {
	replayBrickCacheValid = false
	if replayModeActive {
		replayModeActive = false
	}
	replayFrames = nil
	replayBrickDefs = nil
	replayDebrisShapes = make(map[uint32]replayDebrisShape)
	replayLevelIndex = levelIndex
	replayRecording = true
	replayRecordingFinished = false
	replayRecordingFull = false
	replayCaptureAccumulator = 0
	replayFrameIndex = 0
	replayPlaybackDirection = 0
	replayPlaybackSpeed = 1
	replayPlaybackAccum = 0

	rememberReplayBrickDefs()
}

func rememberReplayBrickDefs() {
	_, _, mirrorSum := brickVerticalBounds()
	replayBrickMirrorSum = float32(mirrorSum)
	replayBrickDefs = make([]replayBrickDef, len(bricks))
	for i := range bricks {
		br := &bricks[i]
		replayBrickDefs[i] = replayBrickDef{
			x: float32(br.x), y: float32(br.y), w: float32(br.w), h: float32(br.h),
			tilt: float32(br.tiltRadians), magic: br.magic, unbreakable: br.unbreakable,
		}
		if bricksVerticalFlipped {
			replayBrickDefs[i].y = replayBrickMirrorSum - replayBrickDefs[i].y - replayBrickDefs[i].h
			replayBrickDefs[i].tilt = -replayBrickDefs[i].tilt
		}
	}
}

func replayBrickAt(frame *replayFrame, index int) replayBrickDef {
	def := replayBrickDefs[index]
	if frame.bricksFlipped {
		def.y = replayBrickMirrorSum - def.y - def.h
		def.tilt = -def.tilt
	}
	return def
}

func ensureReplayDebrisShape(fragment *debrisFragment) {
	if fragment == nil || fragment.renderID == 0 {
		return
	}
	if _, ok := replayDebrisShapes[fragment.renderID]; ok {
		return
	}
	shape := replayDebrisShape{
		id:         fragment.renderID,
		circle:     fragment.circle,
		radius:     float32(fragment.radius),
		roundness:  float32(fragment.roundness),
		pointCount: fragment.pointCount,
		fillColor:  fragment.fillColor,
	}
	if !fragment.circle && fragment.pointCount > 0 {
		count := fragment.pointCount * 2
		if count > len(fragment.points) {
			count = len(fragment.points)
		}
		shape.points = make([]float32, count)
		for i := 0; i < count; i++ {
			shape.points[i] = float32(fragment.points[i])
		}
	}
	replayDebrisShapes[fragment.renderID] = shape
}

func captureReplayFrameState(interpolate bool) replayFrame {
	frame := replayFrame{
		timeSeconds:     float32(levelRunStats.elapsedSeconds),
		paddleX:         float32(paddle.x),
		paddleY:         float32(paddle.y),
		paddleW:         float32(paddle.w),
		paddleH:         float32(paddle.h),
		blackHoleX:      float32(blackHoleX),
		blackHoleY:      float32(blackHoleY),
		blackHoleActive: blackHoleActive,
		zapperActive:    zapperIsActive(),
		drawBrickTilt:   physicsConfig.drawBrickTilt,
		bricksFlipped:   bricksVerticalFlipped,
		lives:           int32(lives),
		score:           int32(score),
	}

	var renderState renderSnapshot
	if interpolate {
		renderState = interpolatedRenderSnapshot(renderInterpolationAlpha)
	}

	count := activeBallCount()
	if count > defaultMaxBalls {
		count = defaultMaxBalls
	}
	frame.ballCount = uint8(count)
	for i := 0; i < count; i++ {
		b := activeBallAt(i)
		if b == nil {
			continue
		}
		x, y, angle := b.x, b.y, b.angle
		if interpolate && i < renderState.ballCount {
			x = renderState.balls[i].x
			y = renderState.balls[i].y
			angle = renderState.balls[i].angle
		}
		frame.balls[i] = replayBallState{
			x: float32(x), y: float32(y), angle: float32(angle),
			vx: float32(b.vx), vy: float32(b.vy), omega: float32(b.omega),
			zapper: int32(b.zapperTargetIndex),
		}
	}
	if interpolate {
		frame.paddleX = float32(renderState.paddleX)
		frame.blackHoleX = float32(renderState.blackHoleX)
		frame.blackHoleY = float32(renderState.blackHoleY)
		frame.blackHoleActive = renderState.blackHoleActive
	}

	wordCount := (len(bricks) + 63) / 64
	frame.brickAlive = make([]uint64, wordCount)
	for i := range bricks {
		if bricks[i].alive {
			frame.brickAlive[i/64] |= uint64(1) << uint(i%64)
		}
	}

	if defaultReplayRecordDebris && len(brickDebris) > 0 && defaultReplayMaxDebrisPerFrame != 0 {
		limit := len(brickDebris)
		if defaultReplayMaxDebrisPerFrame > 0 && limit > defaultReplayMaxDebrisPerFrame {
			limit = defaultReplayMaxDebrisPerFrame
		}
		frame.debris = make([]replayDebrisState, 0, limit)
		thresholdSquared := debrisFrontLayerSpeed * debrisFrontLayerSpeed
		for i := 0; i < limit; i++ {
			fragment := &brickDebris[i]
			opacity := debrisOpacity(fragment)
			if opacity <= 0 || (!fragment.circle && fragment.pointCount < 3) {
				continue
			}
			ensureReplayDebrisShape(fragment)
			x, y, angle := fragment.x, fragment.y, fragment.angle
			if interpolate {
				x = lerpFloat(fragment.previousX, fragment.x, renderInterpolationAlpha)
				y = lerpFloat(fragment.previousY, fragment.y, renderInterpolationAlpha)
				angle = lerpFloat(fragment.previousAngle, fragment.angle, renderInterpolationAlpha)
			}
			speedSquared := fragment.vx*fragment.vx + fragment.vy*fragment.vy
			front := debrisFrontLayerSpeed <= 0 || speedSquared >= thresholdSquared
			frame.debris = append(frame.debris, replayDebrisState{
				id: fragment.renderID, x: float32(x), y: float32(y), angle: float32(angle),
				opacity: float32(opacity), front: front,
			})
		}
	}

	return frame
}

func appendReplayFrame(force bool) {
	if !replayRecording {
		return
	}
	maximum := replayMaximumFrames()
	if maximum > 0 && len(replayFrames) >= maximum {
		replayRecording = false
		replayRecordingFull = true
		showStatusUnique("Replay buffer full", 2.0)
		return
	}
	frame := captureReplayFrameState(false)
	if !force && len(replayFrames) > 0 {
		last := replayFrames[len(replayFrames)-1]
		if math.Abs(float64(last.timeSeconds-frame.timeSeconds)) < 1e-7 {
			return
		}
	}
	replayFrames = append(replayFrames, frame)
}

// Called once after every active 240 Hz physics step. It samples the visual state
// at a configurable replay rate, so recording cost is bounded and independent of
// monitor refresh rate.
func recordReplayPhysicsStep(dt float64) {
	if !replayRecording || replayModeActive || dt <= 0 {
		return
	}

	if len(replayFrames) == 0 {
		appendReplayFrame(true)
	}

	replayCaptureAccumulator += dt
	interval := replayFrameInterval()
	for replayRecording && replayCaptureAccumulator+1e-12 >= interval {
		replayCaptureAccumulator -= interval
		appendReplayFrame(false)
	}

	if levelAdvancePending || gameOver {
		if replayRecording {
			currentTime := float32(levelRunStats.elapsedSeconds)
			if len(replayFrames) == 0 || math.Abs(float64(replayFrames[len(replayFrames)-1].timeSeconds-currentTime)) > 1e-6 {
				appendReplayFrame(true)
			}
		}
		replayRecording = false
		replayRecordingFinished = true
	}
}

func clearReplayInputState() {
	leftPressed = false
	rightPressed = false
	mobileLeftHeld = false
	mobileRightHeld = false
	touchControlActive = false
	mouseControlActive = false
	gamepadAxisX = 0
	paddle.vx = 0
	resetPaddleSpinHistory()
}

func enterReplay(direction int) bool {
	if !replayAvailable() {
		showStatus("No replay recorded yet", 1.5)
		return false
	}

	replayReturnPaused = paused
	replayModeActive = true
	paused = true
	clearReplayInputState()
	stopAllMagicFeatureVoices()
	if pointerLockActive() {
		releaseMousePointerLock()
	}
	physicsAccumulator = 0
	replayPlaybackAccum = 0
	replayPlaybackSpeed = 1
	replayPlaybackDirection = direction

	switch {
	case direction < 0:
		replayFrameIndex = len(replayFrames) - 1
	case direction > 0:
		replayFrameIndex = 0
	default:
		replayFrameIndex = len(replayFrames) - 1
	}
	return true
}

func exitReplay() {
	if !replayModeActive {
		return
	}
	replayModeActive = false
	replayPlaybackDirection = 0
	replayPlaybackSpeed = 1
	replayPlaybackAccum = 0
	paused = replayReturnPaused
	physicsAccumulator = 0
	clearReplayInputState()
	syncRenderInterpolation()
}

func setReplayDirection(direction int) {
	if !replayModeActive || direction == 0 {
		return
	}
	if direction < 0 && replayFrameIndex <= 0 {
		replayFrameIndex = len(replayFrames) - 1
	}
	if direction > 0 && replayFrameIndex >= len(replayFrames)-1 {
		replayFrameIndex = 0
	}
	if replayPlaybackDirection == direction {
		replayPlaybackSpeed = math.Min(defaultReplayPlaybackMaxSpeed, replayPlaybackSpeed*2)
	} else {
		replayPlaybackDirection = direction
		replayPlaybackSpeed = 1
	}
	replayPlaybackAccum = 0
}

func pauseReplay() {
	replayPlaybackDirection = 0
	replayPlaybackSpeed = 1
	replayPlaybackAccum = 0
}

func stepReplay(delta int) {
	if !replayModeActive || len(replayFrames) == 0 {
		return
	}
	pauseReplay()
	replayFrameIndex += delta
	if replayFrameIndex < 0 {
		replayFrameIndex = 0
	}
	if replayFrameIndex >= len(replayFrames) {
		replayFrameIndex = len(replayFrames) - 1
	}
}

func updateReplayPlayback(rawDt float64) {
	if !replayModeActive || replayPlaybackDirection == 0 || len(replayFrames) == 0 || rawDt <= 0 || rawDt >= 1 {
		return
	}
	replayPlaybackAccum += rawDt * replayPlaybackSpeed
	interval := replayFrameInterval()
	for replayPlaybackAccum+1e-12 >= interval {
		replayPlaybackAccum -= interval
		replayFrameIndex += replayPlaybackDirection
		if replayFrameIndex <= 0 {
			replayFrameIndex = 0
			if replayPlaybackDirection < 0 {
				pauseReplay()
				break
			}
		}
		if replayFrameIndex >= len(replayFrames)-1 {
			replayFrameIndex = len(replayFrames) - 1
			if replayPlaybackDirection > 0 {
				pauseReplay()
				break
			}
		}
	}
}

// J/K/L behave like an editor transport. In replay mode the cursor keys are
// deliberately repurposed for exact previous/next-frame stepping; outside replay
// they remain the normal live paddle controls.
func handleReplayShortcut(key, code string, shift, repeat bool) bool {
	if replayModeActive {
		if repeat {
			return true
		}
		switch key {
		case "j", "J":
			setReplayDirection(-1)
		case "k", "K":
			pauseReplay()
		case "l", "L":
			setReplayDirection(1)
		case "ArrowLeft":
			stepReplay(-1)
		case "ArrowRight":
			stepReplay(1)
		case "Home":
			pauseReplay()
			replayFrameIndex = 0
		case "End":
			pauseReplay()
			replayFrameIndex = len(replayFrames) - 1
		case "v", "V":
			exportCurrentFrameSVG()
		case "b", "B":
			exportCurrentFrameTrailSVG()
		case "Escape":
			exitReplay()
		}
		return true
	}

	if repeat {
		return false
	}
	switch key {
	case "j", "J":
		enterReplay(-1)
		return true
	case "k", "K":
		enterReplay(0)
		return true
	case "l", "L":
		enterReplay(1)
		return true
	case "v", "V":
		exportCurrentFrameSVG()
		return true
	case "b", "B":
		exportCurrentFrameTrailSVG()
		return true
	}
	_ = code
	_ = shift
	return false
}

func replayBrickIsAlive(frame *replayFrame, index int) bool {
	if frame == nil || index < 0 || index >= len(replayBrickDefs) {
		return false
	}
	word := index / 64
	if word < 0 || word >= len(frame.brickAlive) {
		return false
	}
	return frame.brickAlive[word]&(uint64(1)<<uint(index%64)) != 0
}

func replayBrickStyle(def replayBrickDef) (fill, stroke string, width float64) {
	if def.unbreakable {
		return palette[6], unbreakableStrokeColor, 2
	}
	if def.magic {
		return magicColor, magicStrokeColor, 2
	}
	return palette[2], brickStrokeColor, 1
}

func drawReplayBricks(frame *replayFrame) {
	if frame == nil {
		return
	}
	if replayBrickCanvas.IsUndefined() || replayBrickCanvas.IsNull() {
		replayBrickCanvas = doc.Call("createElement", "canvas")
		replayBrickCtx = replayBrickCanvas.Call("getContext", "2d")
	}
	style := replayBrickCacheSettings{
		width: canvasWidth, height: canvasHeight, radius: brickRadius,
		tilt:    frame.drawBrickTilt,
		flipped: frame.bricksFlipped,
		colors:  [6]string{palette[2], brickStrokeColor, magicColor, magicStrokeColor, palette[6], unbreakableStrokeColor},
	}
	if !replayBrickCacheValid || style != replayBrickCacheStyle || !slices.Equal(frame.brickAlive, replayBrickCacheAlive) {
		replayBrickCanvas.Set("width", int(canvasWidth))
		replayBrickCanvas.Set("height", int(canvasHeight))
		replayBrickCtx.Call("clearRect", 0, 0, canvasWidth, canvasHeight)
		data := make([]float64, 0, len(replayBrickDefs)*6)
		for i := range replayBrickDefs {
			if !replayBrickIsAlive(frame, i) {
				continue
			}
			def := replayBrickAt(frame, i)
			flags := 0.0
			if def.magic {
				flags = 1
			}
			if def.unbreakable {
				flags = 2
			}
			data = append(data, float64(def.x), float64(def.y), float64(def.w), float64(def.h), float64(def.tilt), flags)
		}
		drawBrickBatch(replayBrickCtx, data, frame.drawBrickTilt)
		replayBrickCacheAlive = append(replayBrickCacheAlive[:0], frame.brickAlive...)
		replayBrickCacheStyle = style
		replayBrickCacheValid = true
	}
	ctx.Call("drawImage", replayBrickCanvas, 0, 0)
}

func traceReplayDebrisShape(target js.Value, shape replayDebrisShape) {
	if shape.circle {
		target.Call("arc", 0, 0, float64(shape.radius), 0, 2*math.Pi)
		return
	}
	if shape.pointCount < 3 || len(shape.points) < shape.pointCount*2 {
		return
	}
	if shape.roundness <= 0.001 {
		target.Call("moveTo", float64(shape.points[0]), float64(shape.points[1]))
		for point := 1; point < shape.pointCount; point++ {
			target.Call("lineTo", float64(shape.points[point*2]), float64(shape.points[point*2+1]))
		}
	} else {
		cornerFraction := 0.08 + clampFloat(float64(shape.roundness), 0, 1)*0.34
		last := shape.pointCount - 1
		startX := float64(shape.points[last*2]) + (float64(shape.points[0])-float64(shape.points[last*2]))*(1-cornerFraction)
		startY := float64(shape.points[last*2+1]) + (float64(shape.points[1])-float64(shape.points[last*2+1]))*(1-cornerFraction)
		target.Call("moveTo", startX, startY)
		for point := 0; point < shape.pointCount; point++ {
			next := (point + 1) % shape.pointCount
			currentX := float64(shape.points[point*2])
			currentY := float64(shape.points[point*2+1])
			outX := currentX + (float64(shape.points[next*2])-currentX)*cornerFraction
			outY := currentY + (float64(shape.points[next*2+1])-currentY)*cornerFraction
			target.Call("quadraticCurveTo", currentX, currentY, outX, outY)
		}
	}
	target.Call("closePath")
}

func drawReplayDebris(frame *replayFrame, front bool) {
	if frame == nil || len(frame.debris) == 0 {
		return
	}
	ctx.Call("save")
	for _, state := range frame.debris {
		if state.front != front || state.opacity <= 0 {
			continue
		}
		shape, ok := replayDebrisShapes[state.id]
		if !ok {
			continue
		}
		ctx.Call("save")
		ctx.Set("globalAlpha", float64(state.opacity))
		ctx.Set("fillStyle", shape.fillColor)
		ctx.Call("translate", float64(state.x), float64(state.y))
		if !shape.circle {
			ctx.Call("rotate", float64(state.angle))
		}
		ctx.Call("beginPath")
		traceReplayDebrisShape(ctx, shape)
		ctx.Call("fill")
		ctx.Call("restore")
	}
	ctx.Call("restore")
}

func drawReplayZapperBolt(frameNumber, ballNumber int, startX, startY, endX, endY float64, color string) {
	const segments = 12
	dx := endX - startX
	dy := endY - startY
	length := math.Hypot(dx, dy)
	nx, ny := 0.0, 0.0
	if length > 0 {
		nx, ny = -dy/length, dx/length
	}
	ctx.Call("save")
	ctx.Set("strokeStyle", color)
	ctx.Set("lineWidth", 3)
	ctx.Call("beginPath")
	ctx.Call("moveTo", startX, startY)
	for i := 1; i < segments; i++ {
		t := float64(i) / float64(segments)
		jitter := math.Sin(float64(frameNumber*17+ballNumber*31+i*13)) * 12
		ctx.Call("lineTo", startX+dx*t+nx*jitter, startY+dy*t+ny*jitter)
	}
	ctx.Call("lineTo", endX, endY)
	ctx.Call("stroke")
	ctx.Call("restore")
}

func drawReplayZapperBolts(frame *replayFrame) {
	if frame == nil || !frame.zapperActive {
		return
	}
	for i := 0; i < int(frame.ballCount); i++ {
		target := int(frame.balls[i].zapper)
		if target < 0 || target >= len(replayBrickDefs) || !replayBrickIsAlive(frame, target) || replayBrickDefs[target].unbreakable {
			continue
		}
		def := replayBrickAt(frame, target)
		color := "#e8fbff"
		if i%2 == 1 {
			color = "#f3e8ff"
		}
		drawReplayZapperBolt(replayFrameIndex, i,
			float64(frame.balls[i].x), float64(frame.balls[i].y),
			float64(def.x+def.w/2), float64(def.y+def.h/2), color)
	}
}

func drawReplayHUD(frame *replayFrame) {
	if frame == nil || !defaultReplayShowHUD {
		return
	}
	const (
		hudTextX      = 22.0
		hudFirstLineY = 37.0
		hudLineStep   = 30.0
	)
	ctx.Call("save")
	ctx.Set("fillStyle", palette[4])
	ctx.Set("font", "18px GameFont, monospace")
	ctx.Set("textAlign", "left")
	ctx.Call("fillText", "Lives: "+strconv.Itoa(int(frame.lives)), hudTextX, hudFirstLineY)
	ctx.Call("fillText", "Level: "+strconv.Itoa(replayLevelIndex+1), hudTextX, hudFirstLineY+hudLineStep)
	ctx.Call("fillText", "Score: "+strconv.Itoa(int(frame.score)), hudTextX, hudFirstLineY+2*hudLineStep)
	if frame.ballCount > 0 {
		b := frame.balls[0]
		speed := math.Hypot(float64(b.vx), float64(b.vy))
		ctx.Call("fillText", "Speed: "+fmt.Sprintf("%.0f", speed), hudTextX, hudFirstLineY+3*hudLineStep)
		if math.Abs(float64(b.omega)) > 100 {
			ctx.Set("fillStyle", "#ff0000")
		}
		ctx.Call("fillText", "Spin:  "+fmt.Sprintf("%+.0f", b.omega), hudTextX, hudFirstLineY+4*hudLineStep)
	}
	ctx.Call("restore")
}

func replayTransportText() string {
	if len(replayFrames) == 0 {
		return "REPLAY --"
	}
	frame := replayFrames[replayFrameIndex]
	total := replayFrames[len(replayFrames)-1].timeSeconds
	state := "PAUSE"
	if replayPlaybackDirection < 0 {
		state = fmt.Sprintf("REV %.0fx", replayPlaybackSpeed)
	} else if replayPlaybackDirection > 0 {
		state = fmt.Sprintf("PLAY %.0fx", replayPlaybackSpeed)
	}
	bufferState := ""
	if replayRecordingFull {
		bufferState = "  BUFFER FULL"
	}
	return fmt.Sprintf("REPLAY  %s  %06d/%06d  %6.2fs/%6.2fs%s   J REV  K PAUSE  L PLAY   <-/-> FRAME   V SVG   B TRAIL   ESC EXIT",
		state, replayFrameIndex+1, len(replayFrames), frame.timeSeconds, total, bufferState)
}

func drawReplayTransportOverlay() {
	text := replayTransportText()
	ctx.Call("save")
	ctx.Set("font", "16px GameFont, monospace")
	width := ctx.Call("measureText", text).Get("width").Float() + 28
	height := 34.0
	x := (canvasWidth - width) / 2
	y := canvasHeight - 86.0
	ctx.Set("fillStyle", "rgba(0,0,0,0.72)")
	ctx.Call("fillRect", x, y, width, height)
	ctx.Set("fillStyle", palette[4])
	ctx.Set("textAlign", "center")
	ctx.Call("fillText", text, canvasWidth/2, y+23)
	ctx.Call("restore")
}

func drawReplayCurrentFrame() {
	if !replayModeActive || len(replayFrames) == 0 {
		return
	}
	if replayFrameIndex < 0 {
		replayFrameIndex = 0
	}
	if replayFrameIndex >= len(replayFrames) {
		replayFrameIndex = len(replayFrames) - 1
	}
	frame := &replayFrames[replayFrameIndex]

	ctx.Set("fillStyle", palette[0])
	ctx.Call("fillRect", 0, 0, canvasWidth, canvasHeight)

	drawReplayDebris(frame, false)
	drawReplayBricks(frame)
	drawReplayDebris(frame, true)
	drawReplayZapperBolts(frame)

	if frame.blackHoleActive && showBlackHole {
		ctx.Call("save")
		ctx.Set("fillStyle", "#000000")
		ctx.Set("strokeStyle", "#9d4edd")
		ctx.Set("lineWidth", 5)
		ctx.Call("beginPath")
		ctx.Call("arc", float64(frame.blackHoleX), float64(frame.blackHoleY), 28, 0, 2*math.Pi)
		ctx.Call("fill")
		ctx.Call("stroke")
		ctx.Set("strokeStyle", "#c77dff")
		ctx.Set("lineWidth", 2)
		ctx.Call("beginPath")
		ctx.Call("arc", float64(frame.blackHoleX), float64(frame.blackHoleY), 42, 0, 2*math.Pi)
		ctx.Call("stroke")
		ctx.Call("restore")
	}

	for i := 0; i < int(frame.ballCount); i++ {
		fillColor, spinMarkerColor := palette[3], palette[5]
		if i%2 == 1 {
			fillColor, spinMarkerColor = palette[7], palette[8]
		}
		b := frame.balls[i]
		drawBall(float64(b.x), float64(b.y), ballRadius, float64(b.angle), fillColor, spinMarkerColor)
	}

	ctx.Call("save")
	ctx.Set("fillStyle", palette[1])
	ctx.Call("beginPath")
	ctx.Call("roundRect", float64(frame.paddleX), float64(frame.paddleY), float64(frame.paddleW), float64(frame.paddleH), paddleRadius)
	ctx.Call("fill")
	ctx.Call("restore")

	drawReplayHUD(frame)
	drawReplayTransportOverlay()
}

func svgEscape(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	value = strings.ReplaceAll(value, `"`, "&quot;")
	value = strings.ReplaceAll(value, "'", "&apos;")
	return value
}

func svgNum(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}

func svgDebrisPath(shape replayDebrisShape) string {
	if shape.circle || shape.pointCount < 3 || len(shape.points) < shape.pointCount*2 {
		return ""
	}
	var b strings.Builder
	if shape.roundness <= 0.001 {
		b.WriteString("M ")
		b.WriteString(svgNum(float64(shape.points[0])))
		b.WriteByte(' ')
		b.WriteString(svgNum(float64(shape.points[1])))
		for point := 1; point < shape.pointCount; point++ {
			b.WriteString(" L ")
			b.WriteString(svgNum(float64(shape.points[point*2])))
			b.WriteByte(' ')
			b.WriteString(svgNum(float64(shape.points[point*2+1])))
		}
		b.WriteString(" Z")
		return b.String()
	}
	cornerFraction := 0.08 + clampFloat(float64(shape.roundness), 0, 1)*0.34
	last := shape.pointCount - 1
	startX := float64(shape.points[last*2]) + (float64(shape.points[0])-float64(shape.points[last*2]))*(1-cornerFraction)
	startY := float64(shape.points[last*2+1]) + (float64(shape.points[1])-float64(shape.points[last*2+1]))*(1-cornerFraction)
	b.WriteString("M ")
	b.WriteString(svgNum(startX))
	b.WriteByte(' ')
	b.WriteString(svgNum(startY))
	for point := 0; point < shape.pointCount; point++ {
		next := (point + 1) % shape.pointCount
		currentX := float64(shape.points[point*2])
		currentY := float64(shape.points[point*2+1])
		outX := currentX + (float64(shape.points[next*2])-currentX)*cornerFraction
		outY := currentY + (float64(shape.points[next*2+1])-currentY)*cornerFraction
		b.WriteString(" Q ")
		b.WriteString(svgNum(currentX))
		b.WriteByte(' ')
		b.WriteString(svgNum(currentY))
		b.WriteByte(' ')
		b.WriteString(svgNum(outX))
		b.WriteByte(' ')
		b.WriteString(svgNum(outY))
	}
	b.WriteString(" Z")
	return b.String()
}

func appendSVGDebris(builder *strings.Builder, frame *replayFrame, front bool) {
	for _, state := range frame.debris {
		if state.front != front || state.opacity <= 0 {
			continue
		}
		shape, ok := replayDebrisShapes[state.id]
		if !ok {
			continue
		}
		builder.WriteString(`<g transform="translate(`)
		builder.WriteString(svgNum(float64(state.x)))
		builder.WriteByte(' ')
		builder.WriteString(svgNum(float64(state.y)))
		builder.WriteString(`) rotate(`)
		builder.WriteString(svgNum(float64(state.angle) * 180 / math.Pi))
		builder.WriteString(`)" fill="`)
		builder.WriteString(svgEscape(shape.fillColor))
		builder.WriteString(`" fill-opacity="`)
		builder.WriteString(svgNum(float64(state.opacity)))
		builder.WriteString(`">`)
		if shape.circle {
			builder.WriteString(`<circle cx="0" cy="0" r="`)
			builder.WriteString(svgNum(float64(shape.radius)))
			builder.WriteString(`"/>`)
		} else if path := svgDebrisPath(shape); path != "" {
			builder.WriteString(`<path d="`)
			builder.WriteString(path)
			builder.WriteString(`"/>`)
		}
		builder.WriteString(`</g>`)
	}
}

func appendSVGCanvasStart(builder *strings.Builder) {
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	builder.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 `)
	builder.WriteString(svgNum(canvasWidth))
	builder.WriteByte(' ')
	builder.WriteString(svgNum(canvasHeight))
	builder.WriteString(`" width="`)
	builder.WriteString(svgNum(canvasWidth))
	builder.WriteString(`" height="`)
	builder.WriteString(svgNum(canvasHeight))
	builder.WriteString(`">`)
	builder.WriteString(`<rect x="0" y="0" width="100%" height="100%" fill="`)
	builder.WriteString(svgEscape(palette[0]))
	builder.WriteString(`"/>`)
}

func appendSVGBricks(builder *strings.Builder, frame *replayFrame) {
	if frame == nil {
		return
	}
	for i := range replayBrickDefs {
		if !replayBrickIsAlive(frame, i) {
			continue
		}
		def := replayBrickAt(frame, i)
		fill, stroke, width := replayBrickStyle(def)
		cx := float64(def.x + def.w/2)
		cy := float64(def.y + def.h/2)
		if frame.drawBrickTilt {
			builder.WriteString(`<g transform="rotate(`)
			builder.WriteString(svgNum(float64(def.tilt) * 180 / math.Pi))
			builder.WriteByte(' ')
			builder.WriteString(svgNum(cx))
			builder.WriteByte(' ')
			builder.WriteString(svgNum(cy))
			builder.WriteString(`)">`)
		}
		builder.WriteString(`<rect x="`)
		builder.WriteString(svgNum(float64(def.x)))
		builder.WriteString(`" y="`)
		builder.WriteString(svgNum(float64(def.y)))
		builder.WriteString(`" width="`)
		builder.WriteString(svgNum(float64(def.w)))
		builder.WriteString(`" height="`)
		builder.WriteString(svgNum(float64(def.h)))
		builder.WriteString(`" rx="`)
		builder.WriteString(svgNum(brickRadius))
		builder.WriteString(`" fill="`)
		builder.WriteString(svgEscape(fill))
		builder.WriteString(`" stroke="`)
		builder.WriteString(svgEscape(stroke))
		builder.WriteString(`" stroke-width="`)
		builder.WriteString(svgNum(width))
		builder.WriteString(`"/>`)
		if frame.drawBrickTilt {
			builder.WriteString(`</g>`)
		}
	}
}

type replaySVGPoint struct {
	x, y float64
}

func replayZapperRandom(state *uint32) float64 {
	x := *state
	if x == 0 {
		x = 0x6d2b79f5
	}
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	*state = x
	return float64(x&0x00ffffff) / float64(0x00ffffff)
}

func replayZapperSeed(frame *replayFrame, ballIndex, target int) uint32 {
	seed := uint32(ballIndex+1)*0x9e3779b9 ^ uint32(target+1)*0x85ebca6b
	seed ^= math.Float32bits(frame.balls[ballIndex].x) * 0xc2b2ae35
	seed ^= math.Float32bits(frame.balls[ballIndex].y) * 0x27d4eb2f
	seed ^= uint32(frame.score+1) * 0x165667b1
	if seed == 0 {
		seed = 1
	}
	return seed
}

func replayZapperPoints(startX, startY, endX, endY float64, seed *uint32, jitterScale, wavePhase float64) []replaySVGPoint {
	const segments = 12
	points := make([]replaySVGPoint, 0, segments+1)
	points = append(points, replaySVGPoint{x: startX, y: startY})
	dx := endX - startX
	dy := endY - startY
	length := math.Hypot(dx, dy)
	nx, ny := 0.0, 0.0
	if length > 0 {
		nx = -dy / length
		ny = dx / length
	}
	for segment := 1; segment < segments; segment++ {
		t := float64(segment) / float64(segments)
		x := startX + dx*t
		y := startY + dy*t
		// Keep both ends attached while allowing the middle of the bolt to crackle.
		jitterEnvelope := math.Sin(math.Pi * t)
		randomJitter := (replayZapperRandom(seed)*2 - 1) * 14 * jitterEnvelope * jitterScale
		waveJitter := math.Sin(wavePhase+t*math.Pi*3.0) * 4.5 * jitterEnvelope * jitterScale
		jitter := randomJitter + waveJitter
		x += nx * jitter
		y += ny * jitter
		points = append(points, replaySVGPoint{x: x, y: y})
	}
	points = append(points, replaySVGPoint{x: endX, y: endY})
	return points
}

func appendSVGPolyline(builder *strings.Builder, points []replaySVGPoint, stroke string, width, opacity float64) {
	if len(points) < 2 || opacity <= 0 {
		return
	}
	builder.WriteString(`<polyline points="`)
	for i, point := range points {
		if i > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(svgNum(point.x))
		builder.WriteByte(',')
		builder.WriteString(svgNum(point.y))
	}
	builder.WriteString(`" fill="none" stroke="`)
	builder.WriteString(stroke)
	builder.WriteString(`" stroke-width="`)
	builder.WriteString(svgNum(width))
	builder.WriteString(`" stroke-opacity="`)
	builder.WriteString(svgNum(clampFloat(opacity, 0, 1)))
	builder.WriteString(`" stroke-linecap="round" stroke-linejoin="round"/>`)
}

func appendSVGZapperForks(builder *strings.Builder, points []replaySVGPoint, color, shadowColor string, seed *uint32) {
	if len(points) < 8 {
		return
	}
	// Two short side branches make the exported bolt read as electricity instead
	// of a single polyline, while keeping the main strike visually dominant.
	for fork := 0; fork < 2; fork++ {
		baseIndex := 3 + fork*4
		if baseIndex >= len(points)-1 {
			break
		}
		base := points[baseIndex]
		prev := points[baseIndex-1]
		next := points[baseIndex+1]
		tx := next.x - prev.x
		ty := next.y - prev.y
		length := math.Hypot(tx, ty)
		if length <= 0 {
			continue
		}
		tx /= length
		ty /= length
		nx := -ty
		ny := tx
		side := 1.0
		if replayZapperRandom(seed) < 0.5 {
			side = -1
		}
		branchLength := 24 + replayZapperRandom(seed)*24
		forward := 8 + replayZapperRandom(seed)*16
		mid := replaySVGPoint{
			x: base.x + tx*forward*0.45 + nx*side*branchLength*0.45,
			y: base.y + ty*forward*0.45 + ny*side*branchLength*0.45,
		}
		end := replaySVGPoint{
			x: base.x + tx*forward + nx*side*branchLength,
			y: base.y + ty*forward + ny*side*branchLength,
		}
		forkPoints := []replaySVGPoint{base, mid, end}
		appendSVGPolyline(builder, forkPoints, shadowColor, 6, 0.20)
		appendSVGPolyline(builder, forkPoints, color, 1.6, 0.72)
	}
}

func appendSVGZapper(builder *strings.Builder, frame *replayFrame) {
	if frame == nil || !frame.zapperActive {
		return
	}
	for i := 0; i < int(frame.ballCount); i++ {
		target := int(frame.balls[i].zapper)
		if target < 0 || target >= len(replayBrickDefs) || !replayBrickIsAlive(frame, target) || replayBrickDefs[target].unbreakable {
			continue
		}
		def := replayBrickAt(frame, target)
		color, shadowColor := "#e8fbff", "#58d9ff"
		if i%2 == 1 {
			color, shadowColor = "#f3e8ff", "#b56cff"
		}
		startX := float64(frame.balls[i].x)
		startY := float64(frame.balls[i].y)
		endX := float64(def.x + def.w/2)
		endY := float64(def.y + def.h/2)
		seed := replayZapperSeed(frame, i, target)
		phaseBase := replayZapperRandom(&seed) * math.Pi * 2
		paths := [][]replaySVGPoint{
			replayZapperPoints(startX, startY, endX, endY, &seed, 0.85, phaseBase+0.00),
			replayZapperPoints(startX, startY, endX, endY, &seed, 1.05, phaseBase+1.75),
			replayZapperPoints(startX, startY, endX, endY, &seed, 1.25, phaseBase+3.10),
		}

		// Use several independently-jittered paths to better match the live zapper's
		// visual impression of multiple crackling zigzags overlaid on top of each other.
		for pathIndex, points := range paths {
			appendSVGPolyline(builder, points, shadowColor, 16, 0.07)
			appendSVGPolyline(builder, points, shadowColor, 8.5, 0.18+float64(pathIndex)*0.03)
		}
		for pathIndex, points := range paths {
			appendSVGPolyline(builder, points, color, 3.2, 0.55+float64(pathIndex)*0.10)
		}
		appendSVGPolyline(builder, paths[0], "#ffffff", 1.25, 0.82)
		appendSVGPolyline(builder, paths[1], color, 1.55, 0.92)
		appendSVGZapperForks(builder, paths[0], color, shadowColor, &seed)
		appendSVGZapperForks(builder, paths[1], color, shadowColor, &seed)
	}
}

func appendSVGBlackHole(builder *strings.Builder, frame *replayFrame, opacity float64) {
	if frame == nil || !frame.blackHoleActive || !showBlackHole || opacity <= 0 {
		return
	}
	builder.WriteString(`<g opacity="`)
	builder.WriteString(svgNum(clampFloat(opacity, 0, 1)))
	builder.WriteString(`">`)
	builder.WriteString(`<circle cx="`)
	builder.WriteString(svgNum(float64(frame.blackHoleX)))
	builder.WriteString(`" cy="`)
	builder.WriteString(svgNum(float64(frame.blackHoleY)))
	builder.WriteString(`" r="28" fill="#000" stroke="#9d4edd" stroke-width="5"/>`)
	builder.WriteString(`<circle cx="`)
	builder.WriteString(svgNum(float64(frame.blackHoleX)))
	builder.WriteString(`" cy="`)
	builder.WriteString(svgNum(float64(frame.blackHoleY)))
	builder.WriteString(`" r="42" fill="none" stroke="#c77dff" stroke-width="2"/>`)
	builder.WriteString(`</g>`)
}

func appendSVGBalls(builder *strings.Builder, frame *replayFrame, opacity float64) {
	if frame == nil || opacity <= 0 {
		return
	}
	opacity = clampFloat(opacity, 0, 1)
	for i := 0; i < int(frame.ballCount); i++ {
		fillColor, spinColor := palette[3], palette[5]
		if i%2 == 1 {
			fillColor, spinColor = palette[7], palette[8]
		}
		ballState := frame.balls[i]
		builder.WriteString(`<g opacity="`)
		builder.WriteString(svgNum(opacity))
		builder.WriteString(`" transform="translate(`)
		builder.WriteString(svgNum(float64(ballState.x)))
		builder.WriteByte(' ')
		builder.WriteString(svgNum(float64(ballState.y)))
		builder.WriteString(`) rotate(`)
		builder.WriteString(svgNum(float64(ballState.angle) * 180 / math.Pi))
		builder.WriteString(`)">`)
		builder.WriteString(`<circle cx="0" cy="0" r="`)
		builder.WriteString(svgNum(ballRadius))
		builder.WriteString(`" fill="`)
		builder.WriteString(svgEscape(fillColor))
		builder.WriteString(`"/>`)
		builder.WriteString(`<line x1="0" y1="-`)
		builder.WriteString(svgNum(ballRadius))
		builder.WriteString(`" x2="0" y2="`)
		builder.WriteString(svgNum(ballRadius))
		builder.WriteString(`" stroke="`)
		builder.WriteString(svgEscape(spinColor))
		builder.WriteString(`" stroke-width="2"/>`)
		builder.WriteString(`<circle cx="0" cy="0" r="2" fill="`)
		builder.WriteString(svgEscape(spinColor))
		builder.WriteString(`"/></g>`)
	}
}

func appendSVGPaddle(builder *strings.Builder, frame *replayFrame, opacity float64) {
	if frame == nil || opacity <= 0 {
		return
	}
	builder.WriteString(`<rect x="`)
	builder.WriteString(svgNum(float64(frame.paddleX)))
	builder.WriteString(`" y="`)
	builder.WriteString(svgNum(float64(frame.paddleY)))
	builder.WriteString(`" width="`)
	builder.WriteString(svgNum(float64(frame.paddleW)))
	builder.WriteString(`" height="`)
	builder.WriteString(svgNum(float64(frame.paddleH)))
	builder.WriteString(`" rx="`)
	builder.WriteString(svgNum(paddleRadius))
	builder.WriteString(`" fill="`)
	builder.WriteString(svgEscape(palette[1]))
	builder.WriteString(`" opacity="`)
	builder.WriteString(svgNum(clampFloat(opacity, 0, 1)))
	builder.WriteString(`"/>`)
}

func appendSVGDebrisOpacity(builder *strings.Builder, frame *replayFrame, front bool, opacity float64) {
	if frame == nil || opacity <= 0 {
		return
	}
	builder.WriteString(`<g opacity="`)
	builder.WriteString(svgNum(clampFloat(opacity, 0, 1)))
	builder.WriteString(`">`)
	appendSVGDebris(builder, frame, front)
	builder.WriteString(`</g>`)
}

// Clean vector frame export. HUD, status text, replay transport and diagnostics
// are intentionally excluded so V produces artwork rather than a UI screenshot.
func frameSVG(frame *replayFrame, frameNumber int) string {
	if frame == nil {
		return ""
	}
	var b strings.Builder
	b.Grow(32768)
	appendSVGCanvasStart(&b)
	appendSVGDebris(&b, frame, false)
	appendSVGBricks(&b, frame)
	appendSVGDebris(&b, frame, true)
	appendSVGZapper(&b, frame)
	appendSVGBlackHole(&b, frame, 1)
	appendSVGBalls(&b, frame, 1)
	appendSVGPaddle(&b, frame, 1)
	b.WriteString(`<!-- Physical clean vector frame export. HUD/transport/debug overlays intentionally omitted. -->`)
	b.WriteString(`</svg>`)
	_ = frameNumber
	return b.String()
}

func replayTrailOpacity(position, count int) float64 {
	if count <= 0 || position < 0 {
		return 0
	}
	// Exponential-ish fade: old samples are faint, recent samples remain visible.
	t := clampFloat(float64(position+1)/float64(count), 0, 1)
	return 0.04 + 0.36*math.Pow(t, 1.7)
}

func replayTrailFrameIndices(historyEnd int) []int {
	if historyEnd < 0 || len(replayFrames) == 0 || defaultReplayTrailFrames <= 0 {
		return nil
	}
	if historyEnd >= len(replayFrames) {
		historyEnd = len(replayFrames) - 1
	}
	start := historyEnd - defaultReplayTrailFrames + 1
	if start < 0 {
		start = 0
	}
	indices := make([]int, 0, historyEnd-start+1)
	for i := start; i <= historyEnd; i++ {
		indices = append(indices, i)
	}
	return indices
}

// B exports a temporal vector composite. Static geometry comes only from the
// current frame; prior replay samples are used only for moving objects. This
// keeps the image readable while creating a motion-blur/trail impression.
func trailFrameSVG(frame *replayFrame, frameNumber, historyEnd int) string {
	if frame == nil {
		return ""
	}
	var b strings.Builder
	b.Grow(65536)
	appendSVGCanvasStart(&b)

	history := replayTrailFrameIndices(historyEnd)
	for i, index := range history {
		appendSVGDebrisOpacity(&b, &replayFrames[index], false, replayTrailOpacity(i, len(history)))
	}
	appendSVGDebris(&b, frame, false)
	appendSVGBricks(&b, frame)
	for i, index := range history {
		appendSVGDebrisOpacity(&b, &replayFrames[index], true, replayTrailOpacity(i, len(history)))
	}
	appendSVGDebris(&b, frame, true)

	// Historical moving objects are drawn oldest-to-newest beneath the current
	// objects. Zapper is intentionally current-frame only so it stays crisp.
	for i, index := range history {
		opacity := replayTrailOpacity(i, len(history))
		appendSVGBlackHole(&b, &replayFrames[index], opacity)
		appendSVGBalls(&b, &replayFrames[index], opacity)
		appendSVGPaddle(&b, &replayFrames[index], opacity)
	}
	appendSVGZapper(&b, frame)
	appendSVGBlackHole(&b, frame, 1)
	appendSVGBalls(&b, frame, 1)
	appendSVGPaddle(&b, frame, 1)

	b.WriteString(`<!-- Physical temporal vector trail export. HUD/transport/debug overlays intentionally omitted. -->`)
	b.WriteString(`</svg>`)
	_ = frameNumber
	return b.String()
}

func downloadSVG(filename, svg string) bool {
	return nativeExportSVG(filename, svg)
}

func exportCurrentFrameSVG() {
	var frame replayFrame
	frameNumber := -1
	if replayModeActive && len(replayFrames) > 0 {
		frame = replayFrames[replayFrameIndex]
		frameNumber = replayFrameIndex
	} else {
		if len(replayBrickDefs) == 0 && len(bricks) > 0 {
			rememberReplayBrickDefs()
		}
		if replayDebrisShapes == nil {
			replayDebrisShapes = make(map[uint32]replayDebrisShape)
		}
		frame = captureReplayFrameState(true)
	}

	name := fmt.Sprintf("physical-level-%02d-live.svg", currentLevelIndex+1)
	if frameNumber >= 0 {
		name = fmt.Sprintf("physical-level-%02d-frame-%06d.svg", replayLevelIndex+1, frameNumber+1)
	}
	if downloadSVG(name, frameSVG(&frame, frameNumber)) {
		if !replayModeActive {
			showStatus("SVG exported", 1.5)
		}
	} else if !replayModeActive {
		showStatus("SVG export failed", 1.5)
	}
}

func exportCurrentFrameTrailSVG() {
	var frame replayFrame
	frameNumber := -1
	historyEnd := -1
	if replayModeActive && len(replayFrames) > 0 {
		frame = replayFrames[replayFrameIndex]
		frameNumber = replayFrameIndex
		historyEnd = replayFrameIndex - 1
	} else {
		if len(replayBrickDefs) == 0 && len(bricks) > 0 {
			rememberReplayBrickDefs()
		}
		if replayDebrisShapes == nil {
			replayDebrisShapes = make(map[uint32]replayDebrisShape)
		}
		frame = captureReplayFrameState(true)
		historyEnd = len(replayFrames) - 1
		// Do not ghost an almost-identical sample immediately beneath the live frame.
		if historyEnd >= 0 && math.Abs(float64(replayFrames[historyEnd].timeSeconds-frame.timeSeconds)) < replayFrameInterval()*0.5 {
			historyEnd--
		}
	}

	name := fmt.Sprintf("physical-level-%02d-live-trail.svg", currentLevelIndex+1)
	if frameNumber >= 0 {
		name = fmt.Sprintf("physical-level-%02d-frame-%06d-trail.svg", replayLevelIndex+1, frameNumber+1)
	}
	if downloadSVG(name, trailFrameSVG(&frame, frameNumber, historyEnd)) {
		if !replayModeActive {
			showStatus("Trail SVG exported", 1.5)
		}
	} else if !replayModeActive {
		showStatus("Trail SVG export failed", 1.5)
	}
}
