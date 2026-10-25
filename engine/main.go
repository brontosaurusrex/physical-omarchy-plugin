package main

import (
	"embed"
	"fmt"
	"io/fs"
	"math"
	"math/rand"
	js "physical/internal/qtbridge"
	"sort"
	"strconv"
	"strings"
)

// Compressed runtime samples are embedded from dedicated runtime directories.
// Editable WAV masters stay one directory above and are not embedded as long as
// they are kept out of runtime/. Supported compressed files are decoded
// sequentially at game startup so gameplay never waits for first-use decoding.
//
//go:embed sounds/brickHits/runtime sounds/magicBrickHits/runtime
var embeddedRuntimeAudio embed.FS

// Compile-time defaults and tuning values live in config.go.

// physicsSettings contains every physics value that is meaningful to override
// per level. Engine scheduling values such as the fixed-step rate remain global
// compile-time settings in config.go because changing them per level would alter
// simulation stability rather than level behavior.
type physicsSettings struct {
	gravity             float64
	restitution         float64
	frictionCoeff       float64
	paddleBoost         float64
	brickBoost          float64
	maxSpeed            float64
	maxSpin             float64
	stuckSpeedThreshold float64
	stuckDuration       float64
	tiltUpSpeed         float64
	tiltSideMin         float64
	tiltSideMax         float64

	magnusCoefficient       float64
	magnusAccelerationScale float64
	spinDrag                float64
	airDrag                 float64

	wallFrictionScale            float64
	brickFrictionScale           float64
	unbreakableFrictionScale     float64
	paddleFrictionScale          float64
	paddleSpinTransfer           float64
	collisionSpinCoupling        float64
	minimumCollisionGrip         float64
	minimumPaddleGrip            float64
	collisionSlop                float64
	paddleSpinGraceSeconds       float64
	mousePaddleSpinVelocityLimit float64
	overspeedHalfLife            float64

	wallNoiseCellSize      float64
	wallSideTiltDegrees    float64
	wallTopTiltDegrees     float64
	wallCornerFadeDistance float64

	// Brick-corner response. Amount 0 keeps the classic axis normal; 1 uses the
	// geometric circle-vs-corner normal on genuine corner hits. When
	// cornerPhysicsAllBricks is true, the normal passable-gap suppression is bypassed.
	cornerPhysicsEnabled   bool
	cornerPhysicsAmount    float64
	cornerPhysicsAllBricks bool

	brickTiltMinDegrees float64
	brickTiltMaxDegrees float64
	drawBrickTilt       bool

	orbitMinimumSpeed         float64
	orbitMinimumHitSpeed      float64
	orbitMinorSpeedRatio      float64
	orbitMinorSpeedFloor      float64
	orbitRequiredHits         int
	orbitDetectionWindow      float64
	orbitMaximumMinorProgress float64
	orbitHitCooldown          float64
	orbitEscapeSpeed          float64
	orbitEscapeDuration       float64
}

type physicsSliderSpec struct {
	group      string
	key        string
	label      string
	configName string
	min        float64
	max        float64
	step       float64
	precision  int
}

type debrisSliderSpec struct {
	group      string
	key        string
	label      string
	configName string
	min        float64
	max        float64
	step       float64
	precision  int
	integer    bool
}

type autoPaddleSliderSpec struct {
	group      string
	key        string
	label      string
	configName string
	min        float64
	max        float64
	step       float64
	precision  int
}

type debrisEditorSnapshot struct {
	enabled bool
	values  map[string]float64
}

func defaultPhysicsSettings() physicsSettings {
	return physicsSettings{
		gravity:             defaultPhysicsGravity,
		restitution:         defaultPhysicsRestitution,
		frictionCoeff:       defaultPhysicsFrictionCoeff,
		paddleBoost:         defaultPhysicsPaddleBoost,
		brickBoost:          defaultPhysicsBrickBoost,
		maxSpeed:            defaultPhysicsMaxSpeed,
		maxSpin:             defaultPhysicsMaxSpin,
		stuckSpeedThreshold: defaultPhysicsStuckSpeedThreshold,
		stuckDuration:       defaultPhysicsStuckDuration,
		tiltUpSpeed:         defaultPhysicsTiltUpSpeed,
		tiltSideMin:         defaultPhysicsTiltSideMin,
		tiltSideMax:         defaultPhysicsTiltSideMax,

		magnusCoefficient:       defaultPhysicsMagnusCoefficient,
		magnusAccelerationScale: defaultPhysicsMagnusAccelerationScale,
		spinDrag:                defaultPhysicsSpinDrag,
		airDrag:                 defaultPhysicsAirDrag,

		wallFrictionScale:            defaultPhysicsWallFrictionScale,
		brickFrictionScale:           defaultPhysicsBrickFrictionScale,
		unbreakableFrictionScale:     defaultPhysicsUnbreakableFrictionScale,
		paddleFrictionScale:          defaultPhysicsPaddleFrictionScale,
		paddleSpinTransfer:           defaultPhysicsPaddleSpinTransfer,
		collisionSpinCoupling:        defaultPhysicsCollisionSpinCoupling,
		minimumCollisionGrip:         defaultPhysicsMinimumCollisionGrip,
		minimumPaddleGrip:            defaultPhysicsMinimumPaddleGrip,
		collisionSlop:                defaultPhysicsCollisionSlop,
		paddleSpinGraceSeconds:       defaultPhysicsPaddleSpinGraceSeconds,
		mousePaddleSpinVelocityLimit: defaultPhysicsMousePaddleSpinVelocityLimit,
		overspeedHalfLife:            defaultPhysicsOverspeedHalfLife,

		wallNoiseCellSize:      defaultPhysicsWallNoiseCellSize,
		wallSideTiltDegrees:    defaultPhysicsWallSideTiltDegrees,
		wallTopTiltDegrees:     defaultPhysicsWallTopTiltDegrees,
		wallCornerFadeDistance: defaultPhysicsWallCornerFadeDistance,

		cornerPhysicsEnabled:   defaultPhysicsCornerPhysicsEnabled,
		cornerPhysicsAmount:    defaultPhysicsCornerPhysicsAmount,
		cornerPhysicsAllBricks: defaultPhysicsCornerPhysicsAllBricks,

		brickTiltMinDegrees: defaultPhysicsBrickTiltMinDegrees,
		brickTiltMaxDegrees: defaultPhysicsBrickTiltMaxDegrees,
		drawBrickTilt:       defaultPhysicsDrawBrickTilt,

		orbitMinimumSpeed:         defaultPhysicsOrbitMinimumSpeed,
		orbitMinimumHitSpeed:      defaultPhysicsOrbitMinimumHitSpeed,
		orbitMinorSpeedRatio:      defaultPhysicsOrbitMinorSpeedRatio,
		orbitMinorSpeedFloor:      defaultPhysicsOrbitMinorSpeedFloor,
		orbitRequiredHits:         defaultPhysicsOrbitRequiredHits,
		orbitDetectionWindow:      defaultPhysicsOrbitDetectionWindow,
		orbitMaximumMinorProgress: defaultPhysicsOrbitMaximumMinorProgress,
		orbitHitCooldown:          defaultPhysicsOrbitHitCooldown,
		orbitEscapeSpeed:          defaultPhysicsOrbitEscapeSpeed,
		orbitEscapeDuration:       defaultPhysicsOrbitEscapeDuration,
	}
}

var physicsConfig = defaultPhysicsSettings()

func setPhysicsFloatSetting(settings *physicsSettings, key string, value float64) {
	switch key {
	case "gravity":
		settings.gravity = value
	case "restitution":
		settings.restitution = value
	case "frictionCoeff":
		settings.frictionCoeff = value
	case "paddleBoost":
		settings.paddleBoost = value
	case "brickBoost":
		settings.brickBoost = value
	case "maxSpeed":
		settings.maxSpeed = value
	case "maxSpin":
		settings.maxSpin = value
	case "stuckSpeedThreshold":
		settings.stuckSpeedThreshold = value
	case "stuckDuration":
		settings.stuckDuration = value
	case "tiltUpSpeed":
		settings.tiltUpSpeed = value
	case "tiltSideMin":
		settings.tiltSideMin = value
	case "tiltSideMax":
		settings.tiltSideMax = value
	case "magnusCoefficient":
		settings.magnusCoefficient = value
	case "magnusAccelerationScale":
		settings.magnusAccelerationScale = value
	case "spinDrag":
		settings.spinDrag = value
	case "airDrag":
		settings.airDrag = value
	case "wallFrictionScale":
		settings.wallFrictionScale = value
	case "brickFrictionScale":
		settings.brickFrictionScale = value
	case "unbreakableFrictionScale":
		settings.unbreakableFrictionScale = value
	case "paddleFrictionScale":
		settings.paddleFrictionScale = value
	case "paddleSpinTransfer":
		settings.paddleSpinTransfer = value
	case "collisionSpinCoupling":
		settings.collisionSpinCoupling = value
	case "minimumCollisionGrip":
		settings.minimumCollisionGrip = value
	case "minimumPaddleGrip":
		settings.minimumPaddleGrip = value
	case "collisionSlop":
		settings.collisionSlop = value
	case "paddleSpinGraceSeconds":
		settings.paddleSpinGraceSeconds = value
	case "mousePaddleSpinVelocityLimit":
		settings.mousePaddleSpinVelocityLimit = value
	case "overspeedHalfLife":
		settings.overspeedHalfLife = value
	case "wallNoiseCellSize":
		settings.wallNoiseCellSize = value
	case "wallSideTiltDegrees":
		settings.wallSideTiltDegrees = value
	case "wallTopTiltDegrees":
		settings.wallTopTiltDegrees = value
	case "wallCornerFadeDistance":
		settings.wallCornerFadeDistance = value
	case "cornerPhysicsAmount":
		settings.cornerPhysicsAmount = value
	case "brickTiltMinDegrees":
		settings.brickTiltMinDegrees = value
	case "brickTiltMaxDegrees":
		settings.brickTiltMaxDegrees = value
	case "orbitMinimumSpeed":
		settings.orbitMinimumSpeed = value
	case "orbitMinimumHitSpeed":
		settings.orbitMinimumHitSpeed = value
	case "orbitMinorSpeedRatio":
		settings.orbitMinorSpeedRatio = value
	case "orbitMinorSpeedFloor":
		settings.orbitMinorSpeedFloor = value
	case "orbitDetectionWindow":
		settings.orbitDetectionWindow = value
	case "orbitMaximumMinorProgress":
		settings.orbitMaximumMinorProgress = value
	case "orbitHitCooldown":
		settings.orbitHitCooldown = value
	case "orbitEscapeSpeed":
		settings.orbitEscapeSpeed = value
	case "orbitEscapeDuration":
		settings.orbitEscapeDuration = value
	}
}

// ---- These are variables, reset each level ----
var (
	canvasWidth    = defaultCanvasWidth
	canvasHeight   = defaultCanvasHeight
	paddleWidth    = defaultPaddleWidth
	paddleHeight   = defaultPaddleHeight
	ballRadius     = defaultBallRadius
	brickRows      = defaultBrickRows
	brickCols      = defaultBrickCols
	brickWidth     = defaultBrickWidth
	brickHeight    = defaultBrickHeight
	brickPadding   = defaultBrickPadding
	brickOffsetTop = defaultBrickOffsetTop

	paddleRadius = defaultPaddleRadius
	brickRadius  = defaultBrickRadius

	unbreakableChance = defaultUnbreakableChance
	magicChance       = defaultMagicChance
	powerUpDuration   = defaultPowerUpDuration

	blackHoleStrength                = defaultBlackHoleStrength
	blackHoleRange                   = defaultBlackHoleRange
	blackHolePathVerticalRange       = defaultBlackHolePathVerticalRange
	blackHolePathCenterYOffset       = defaultBlackHolePathCenterYOffset
	blackHolePathHorizontalCyclesMin = defaultBlackHolePathHorizontalCyclesMin
	blackHolePathHorizontalCyclesMax = defaultBlackHolePathHorizontalCyclesMax
	blackHolePathVerticalCyclesMin   = defaultBlackHolePathVerticalCyclesMin
	blackHolePathVerticalCyclesMax   = defaultBlackHolePathVerticalCyclesMax
	blackHolePathWobble              = defaultBlackHolePathWobble
	magnetStrength                   = defaultMagnetStrength
	magnetRange                      = defaultMagnetRange
	influencerMultiplier             = defaultInfluencerMultiplier
	zapperRange                      = defaultZapperRange

	startBallX  = defaultStartBallX
	startBallY  = defaultStartBallY
	startBallVx = defaultStartBallVx
	startBallVy = defaultStartBallVy

	enableLowGravity       = defaultEnableLowGravity
	enablePassThrough      = defaultEnablePassThrough
	enableNuke             = defaultEnableNuke
	enableReverseGravity   = defaultEnableReverseGravity
	enableDualBalls        = defaultEnableDualBalls
	enableBlackHole        = defaultEnableBlackHole
	enableMagnet           = defaultEnableMagnet
	enableInfluencer       = defaultEnableInfluencer
	enableZapper           = defaultEnableZapper
	enableBreakUnbreakable = defaultEnableBreakUnbreakable
	enableBigPaddle        = defaultEnableBigPaddle
	enableBrickFlip        = defaultEnableBrickFlip

	enableSounds = defaultEnableSounds

	// Per-level room settings. A negative dry value means use the preset default.
	currentAudioRoom    = defaultAudioRoom
	currentAudioRoomDry = defaultAudioRoomDry

	// Persistent master gate for dynamic debris. The per-level debrisEnabled value
	// still controls each level, but the master switch may disable debris globally.
	debrisMasterEnabled = true

	// Per-level dynamic debris settings.
	debrisEnabled                  = defaultDebrisEnabled
	debrisShapeMode                = defaultDebrisShapeMode
	debrisPiecesMin                = defaultDebrisPiecesMin
	debrisPiecesMax                = defaultDebrisPiecesMax
	debrisLifetime                 = defaultDebrisLifetime
	debrisLifetimeVariationPercent = defaultDebrisLifetimeVariationPercent
	debrisFadeDuration             = defaultDebrisFadeDuration
	debrisStartOpacity             = defaultDebrisStartOpacity
	debrisStartOpacityVariation    = defaultDebrisStartOpacityVariation
	debrisFlashDuration            = defaultDebrisFlashDuration
	debrisFlashOpacity             = defaultDebrisFlashOpacity
	debrisImpactSpeedFactor        = defaultDebrisImpactSpeedFactor
	debrisBallPieceChance          = defaultDebrisBallPieceChance
	debrisTrianglePieceChance      = defaultDebrisTrianglePieceChance
	debrisStarPieceChance          = defaultDebrisStarPieceChance
	debrisStarPointsMin            = defaultDebrisStarPointsMin
	debrisStarPointsMax            = defaultDebrisStarPointsMax
	debrisGlassPieceChance         = defaultDebrisGlassPieceChance
	debrisGlassCornersMin          = defaultDebrisGlassCornersMin
	debrisGlassCornersMax          = defaultDebrisGlassCornersMax
	debrisSliverPieceChance        = defaultDebrisSliverPieceChance
	debrisMaxChunkAspectRatio      = defaultDebrisMaxChunkAspectRatio
	debrisSizeScale                = defaultDebrisSizeScale
	debrisBrickCollisionDelay      = defaultDebrisBrickCollisionDelay
	debrisGravityScale             = defaultDebrisGravityScale
	debrisAirDrag                  = defaultDebrisAirDrag
	debrisRestitution              = defaultDebrisRestitution
	debrisFriction                 = defaultDebrisFriction
	debrisExplosionSpeedMin        = defaultDebrisExplosionSpeedMin
	debrisExplosionSpeedMax        = defaultDebrisExplosionSpeedMax
	debrisAngularSpeedMin          = defaultDebrisAngularSpeedMin
	debrisAngularSpeedMax          = defaultDebrisAngularSpeedMax
	debrisAngularDrag              = defaultDebrisAngularDrag
	debrisAngularStopSpeed         = defaultDebrisAngularStopSpeed
	debrisBallInfluence            = defaultDebrisBallInfluence
	debrisFieldScale               = defaultDebrisFieldScale
	debrisMagnetScale              = defaultDebrisMagnetScale
	debrisMaxSpeed                 = defaultDebrisMaxSpeed
	debrisFrontLayerSpeed          = defaultDebrisFrontLayerSpeed
	debrisMaxActivePieces          = defaultDebrisMaxActivePieces
	debrisOffscreenMargin          = defaultDebrisOffscreenMargin

	ballRescueEnabled             = defaultBallRescueEnabled
	ballRescueFailureLimit        = defaultBallRescueFailureLimit
	ballRescueMinProgress         = defaultBallRescueMinProgress
	ballRescueCheckDuration       = defaultBallRescueCheckDuration
	ballRescueUpperScreenFraction = defaultBallRescueUpperScreenFraction
	ballRescueClearance           = defaultBallRescueClearance
	ballRescueLaunchSpeed         = defaultBallRescueLaunchSpeed
	ballRescueMinRealtimePercent  = defaultBallRescueMinRealtimePercent
	ballRescueMaxComputeLoad      = defaultBallRescueMaxComputeLoad
	ballBelowFloorGracePixels     = defaultBallBelowFloorGracePixels

	paused       bool
	leftPressed  bool
	rightPressed bool

	// Raw-mapped browser joystick state. The Vivanco controller reports its
	// horizontal stick on axis 0, primary fire as B0, and secondary fire as B1.
	gamepadAxisX                float64
	gamepadFireWasPressed       bool
	gamepadFullscreenWasPressed bool
	gamepadActiveIndex          = -1
	gamepadActiveID             string
	gamepadFullscreenCallbacks  []js.Func

	touchControlActive bool
	touchPointerID     int
	touchLastY         float64

	mouseControlActive bool
	mousePaddleTargetX float64
	mousePointerLocked bool

	mobileControlsEnabled  bool
	mobileControlMode      = defaultMobileControlMode
	mobileSelectorVisible  bool
	mobileLeftHeld         bool
	mobileRightHeld        bool
	mobileLeftPointerID    = -1
	mobileRightPointerID   = -1
	mobileControlCallbacks []js.Func

	phoneTiltAvailable       bool
	phoneTiltCalibrated      bool
	phoneTiltPermissionAsked bool
	phoneTiltListenerSet     bool
	phoneTiltNeutral         float64
	phoneTiltValue           float64
	phoneTiltTargetX         float64

	deviceOrientationCallback js.Func
	setPausedCallback         js.Func
	getPausedCallback         js.Func
)

// ---- Color palette runtime state ----
var palette = append([]string(nil), defaultPalette...)

var (
	magicColor             = defaultMagicColor
	magicStrokeColor       = defaultMagicStrokeColor
	unbreakableStrokeColor = defaultUnbreakableStrokeColor
	brickStrokeColor       = defaultBrickStrokeColor
)

// ---- Powerup types ----
const (
	POWER_NONE = iota
	POWER_LOW_GRAVITY
	POWER_PASS
	POWER_NUKE
	POWER_REVERSE_GRAVITY
	POWER_DUAL_BALLS
	POWER_BLACKHOLE
	POWER_MAGNET
	POWER_INFLUENCER
	POWER_ZAPPER
	POWER_BREAK_UNBREAKABLE
	POWER_BIG_PADDLE
	POWER_BRICK_FLIP
)

// ---- Ball struct ----
type Ball struct {
	x, y, r               float64
	vx, vy                float64
	omega, angle          float64
	stuckTimer            float64
	soundCooldown         float64
	ballBallSoundCooldown float64

	// Multi-ball transient state. New balls get a very short pair-collision grace
	// period so they can separate cleanly from the source ball they were cloned from.
	ballCollisionGrace float64
	zapperTargetIndex  int
	zapperHitTimer     float64

	orbitCandidateAxis   int
	orbitCandidateHits   int
	orbitCandidateTimer  float64
	orbitAnchorMinor     float64
	orbitHitCooldown     float64
	orbitEscapeAxis      int
	orbitEscapeDirection float64
	orbitEscapeTimer     float64

	rescueAttemptActive  bool
	rescueAttemptAxis    int
	rescueAttemptElapsed float64
	rescueStartX         float64
	rescueStartY         float64
	rescueFailureCount   int
}

// activeBallCount includes the primary ball. During normal gameplay ball 1
// always exists; extraBalls contains compacted balls 2..N.
func activeBallCount() int {
	return 1 + len(extraBalls)
}

func activeBallAt(index int) *Ball {
	if index == 0 {
		return &ball
	}
	extraIndex := index - 1
	if extraIndex < 0 || extraIndex >= len(extraBalls) {
		return nil
	}
	return &extraBalls[extraIndex]
}

func ballIndexOf(target *Ball) int {
	if target == nil {
		return -1
	}
	if target == &ball {
		return 0
	}
	for i := range extraBalls {
		if target == &extraBalls[i] {
			return i + 1
		}
	}
	return -1
}

func forEachActiveBall(fn func(*Ball, int)) {
	if fn == nil {
		return
	}
	fn(&ball, 0)
	for i := range extraBalls {
		fn(&extraBalls[i], i+1)
	}
}

func resetBallZapperState(b *Ball) {
	if b == nil {
		return
	}
	b.zapperTargetIndex = -1
	b.zapperHitTimer = 0
}

func resetAllBallZapperStates() {
	forEachActiveBall(func(b *Ball, _ int) {
		resetBallZapperState(b)
	})
}

func maxConfiguredBalls() int {
	if defaultMaxBalls < 1 {
		return 1
	}
	return defaultMaxBalls
}

// spawnAdditionalBall clones the primary ball but gives the new ball a small
// alternating angular spread. The brief pair-collision grace lets coincident
// spawn positions separate naturally before ball-ball collisions become active.
func spawnAdditionalBall() bool {
	if activeBallCount() >= maxConfiguredBalls() {
		return false
	}

	newIndex := activeBallCount() // zero-based index of the ball being appended
	spawn := ball
	speed := math.Hypot(ball.vx, ball.vy)
	if speed > 0 {
		angle := math.Atan2(ball.vy, ball.vx)
		rank := (newIndex + 1) / 2
		sign := 1.0
		if newIndex%2 == 1 {
			sign = -1
		}
		angle += sign * float64(rank) * defaultMultiBallSpawnSpreadDeg * math.Pi / 180.0
		spawn.vx = math.Cos(angle) * speed
		spawn.vy = math.Sin(angle) * speed
	}
	if newIndex%2 == 1 {
		spawn.omega = -ball.omega
	}
	spawn.stuckTimer = 0
	spawn.soundCooldown = 0
	spawn.ballBallSoundCooldown = 0
	spawn.ballCollisionGrace = defaultBallBallCollisionGraceSec
	resetFastOrbitState(&spawn)
	resetBallRescueState(&spawn, true)
	resetBallZapperState(&spawn)
	extraBalls = append(extraBalls, spawn)
	resetAutoPaddleHitPlan()
	return true
}

func boostAllActiveBalls(multiplier float64) {
	if multiplier <= 0 {
		return
	}
	forEachActiveBall(func(b *Ball, _ int) {
		b.vx *= multiplier
		b.vy *= multiplier
		recordMeasuredBallSpeed(b)
	})
}

// Big Paddle is allowed to shrink only while every surviving ball is moving up
// the screen. This prevents the paddle disappearing underneath a descending ball.
func allActiveBallsMovingUp() bool {
	movingUp := true
	forEachActiveBall(func(b *Ball, _ int) {
		if b.vy >= 0 {
			movingUp = false
		}
	})
	return movingUp
}

type statusMessage struct {
	text  string
	timer float64
}

type cornerPhysicsDebugEvent struct {
	count    int
	row, col int
	amount   float64
	nx, ny   float64
	impact   float64
}

// levelRunStats is reset when a level starts and frozen when it is cleared.
// Only clear, player-readable records live here: completion time, fastest ball
// speed reached on this level, fastest absolute spin, the largest brick chain
// caused by one physical ball-to-brick hit, and the fewest balls lost.
type levelRunStatsData struct {
	speedPercent   int
	mixedSpeed     bool
	elapsedSeconds float64
	fastestSpeed   float64
	fastestSpin    float64
	bestChain      int
	ballsLost      int
}

type levelBestStatsData struct {
	timeMs         int
	timeFound      bool
	fastestSpeed   int
	speedFound     bool
	fastestSpin    int
	spinFound      bool
	bestChain      int
	chainFound     bool
	ballsLost      int
	ballsLostFound bool
}

type levelRecordFlags struct {
	time      bool
	speed     bool
	spin      bool
	chain     bool
	ballsLost bool
}

// fullGameRunState tracks a continuous level-1-through-final-level attempt.
// It deliberately does not survive reloads. Pauses, READY screens, and manual
// level-complete screens do not add time; the run time is the sum of active
// per-level play time.
type fullGameRunStateData struct {
	speedPercent   int
	mixedSpeed     bool
	active         bool
	recordEligible bool
	elapsedSeconds float64
	nextLevel      int
	levelCount     int
}

type fullGameCompletionData struct {
	speedPercent       int
	mixedSpeed         bool
	valid              bool
	recordEligible     bool
	timeMs             int
	newRecord          bool
	previousBestTimeMs int
	previousBestFound  bool
	bestTimeMs         int
	bestFound          bool
	levelCount         int
}

// A feature sample may be long (for example blackhole.wav). Only one voice for
// a given feature is allowed at once; retriggering gently replaces it.
type magicFeatureVoice struct {
	source js.Value
	gain   js.Value
	ended  js.Func
	active bool
}

type paddleVelocitySample struct {
	vx  float64
	age float64
}

// renderSnapshot stores the last completed fixed-step state used for visual
// interpolation. Physics remains authoritative; only drawing is smoothed.
type ballRenderSnapshot struct {
	x, y, angle float64
}

type renderSnapshot struct {
	balls           [defaultMaxBalls]ballRenderSnapshot
	ballCount       int
	paddleX         float64
	blackHoleX      float64
	blackHoleY      float64
	blackHoleActive bool
}

// ---- Global state ----
var (
	doc    = js.Global().Get("document")
	canvas js.Value
	ctx    js.Value

	brickCanvas js.Value
	brickCtx    js.Value
	bricksDirty bool

	// Ball 1 is kept separately because it is the prize-taking primary ball.
	// extraBalls contains balls 2..N. When ball 1 is lost while another survives,
	// the first survivor is promoted to ball 1 and becomes the new prize taker.
	ball       Ball
	extraBalls []Ball

	paddle = struct {
		x, y, w, h float64
		vx         float64
	}{
		x:  0,
		y:  0,
		w:  defaultPaddleWidth,
		h:  defaultPaddleHeight,
		vx: 0,
	}

	bricks                 []brick
	brickDebris            []debrisFragment
	debrisRNG              = rand.New(rand.NewSource(0x52d3b715))
	debrisOpaqueColorCache = make(map[string][]string)
	debrisFieldTick        int

	debrisRenderIDCounter      uint32
	debrisAdaptiveBaselineFPS  float64
	debrisAdaptiveRenderStride = 1
	debrisRenderedLastFrame    int
	debrisSkippedLastFrame     int
	debrisPath2DChecked        bool
	debrisPath2DSupported      bool
	debrisPath2DConstructor    js.Value
	brickGrid                  map[int][]int
	brickGridCells             [][]int // Direct cell lookup for frequent debris field queries.
	remainingBreakableBricks   int
	initialBreakableBricks     int
	lastBrickSoundPlayed       bool
	score                      int
	lives                      int
	gameOver                   bool
	win                        bool
	waitingForStart            bool
	levelStartTitle            string
	levelCompleteTimer         float64
	levelAdvancePending        bool

	lastTime float64

	fpsCurrent            float64
	fpsLowest             float64
	fpsSampleElapsed      float64
	fpsSampleFrames       int
	fpsVisibleSamplesSeen int
	fpsLowestElapsed      float64
	fpsLowestFrames       int
	fpsLowestPlaying      bool
	fpsMiniOverlayVisible bool

	loopFunc           js.Func
	keyDown            js.Func
	keyUp              js.Func
	pointerMove        js.Func
	pointerDown        js.Func
	pointerUp          js.Func
	pointerCancel      js.Func
	mouseLeave         js.Func
	lockedMouseMove    js.Func
	pointerLockChange  js.Func
	pointerLockError   js.Func
	fullscreenToggle   js.Func
	browserUICallbacks []js.Func

	// Independent power-up states. Timed effects can coexist.
	lowGravityActive       bool
	lowGravityTimer        float64
	passActive             bool
	passTimer              float64
	reverseGravityActive   bool
	reverseGravityTimer    float64
	magnetPowerActive      bool
	magnetPowerTimer       float64
	zapperPowerActive      bool
	zapperPowerTimer       float64
	bigPaddleActive        bool
	bigPaddleTimer         float64
	bigPaddleExpiryPending bool
	currentGravity         float64

	blackHoleActive bool
	blackHoleTimer  float64
	blackHoleX      float64
	blackHoleY      float64

	blackHolePathElapsed          float64
	blackHolePathDuration         float64
	blackHolePathHorizontalCycles float64
	blackHolePathVerticalCycles   float64
	blackHolePathPhaseX           float64
	blackHolePathPhaseY           float64
	blackHolePathWobblePhaseX     float64
	blackHolePathWobblePhaseY     float64
	blackHoleRNG                  = rand.New(rand.NewSource(0x63b10c7))

	magnetCheat       bool
	zapperCheat       bool
	levelMagnetActive bool
	levelZapperActive bool

	influencerActive bool
	influencerTimer  float64

	// Zapper target/timer state now lives on each Ball so N balls are supported.

	statusMessages []statusMessage

	// Session-wide sequence number for genuine corner responses written to the
	// browser console. It intentionally does not reset between levels.
	cornerPhysicsDebugHitCount int
	cornerPhysicsDebugPending  []cornerPhysicsDebugEvent

	gridOffsetLeft float64
	gridOffsetTop  float64
	gridCellWidth  float64
	gridCellHeight float64
	gridRows       int
	gridCols       int

	paddlePreviousX float64

	// Recent fixed-step paddle velocities used only for spin transfer at impact.
	paddleSpinHistory []paddleVelocitySample

	hudLivesValue int
	hudLevelValue int
	hudScoreValue int
	hudLivesText  string
	hudLevelText  string
	hudScoreText  string

	// Per-level measured peaks. Speed is the fastest instantaneous ball speed
	// actually reached; spin is the greatest absolute spin reached. Both include
	// either active ball, survive life loss, and reset with the level.
	levelMeasuredMaxSpeed float64
	levelMeasuredMaxSpin  float64

	// Per-level run stats and persistent personal bests. The live HUD stays clean;
	// these appear only in diagnostics panel 3 and on the level-complete screen.
	levelRunStats             levelRunStatsData
	levelCompletedStats       levelRunStatsData
	levelCompletedStatsValid  bool
	levelPreviousBests        levelBestStatsData
	levelCurrentBests         levelBestStatsData
	levelCompletionNewRecords levelRecordFlags
	levelRunRecordEligible    bool

	// Continuous whole-game attempt and its final result. The persistent best is
	// scoped by the current number of levels so expanding the campaign starts a
	// fair new record table instead of comparing 31 levels against an old 30-level run.
	fullGameRun        fullGameRunStateData
	fullGameCompletion fullGameCompletionData

	// ---- Level system ----
	currentLevelIndex    int
	highestUnlockedLevel int
	unlockFrontier       int // Highest level index made available; may equal len(levels) after finishing all current levels.
	levels               []levelData

	devAllLevelsUnlocked  bool
	debugOverlayVisible   bool
	physicsOverlayVisible bool
	statsOverlayVisible   bool

	physicsEditorVisible                        bool
	physicsEditorPreviousPaused                 bool
	physicsEditorLiveSimulation                 bool
	physicsEditorOpeningConfig                  physicsSettings
	physicsEditorOpeningDebris                  debrisEditorSnapshot
	physicsEditorOpeningAutoHitVariation        float64
	physicsEditorPanel                          js.Value
	physicsEditorExportSelect                   js.Value
	physicsEditorLiveCheckbox                   js.Value
	physicsEditorDebrisCheckbox                 js.Value
	physicsEditorCornerPhysicsCheckbox          js.Value
	physicsEditorCornerPhysicsAllBricksCheckbox js.Value
	physicsEditorAutoPaddleCheck                js.Value
	physicsEditorInputs                         = make(map[string]js.Value)
	physicsEditorValueLabels                    = make(map[string]js.Value)
	physicsEditorCallbacks                      []js.Func

	// O toggles an automatic inspection paddle. It predicts the next crossing of
	// the paddle line and moves with bounded acceleration instead of teleporting.
	autoPaddleEnabled           bool
	autoPaddleTargetX           float64
	autoPaddleHitVariation      = defaultAutoPaddleHitVariation
	autoPaddleHitOffset         float64
	autoPaddleTargetBall        int
	autoPaddleNeedsNewHitOffset = true
	autoPaddleRNG               = rand.New(rand.NewSource(0x60a17f3d))

	physicsAccumulator       float64
	physicsStepRateCurrent   float64
	physicsRealtimePercent   float64
	physicsComputeLoad       float64
	physicsSampleElapsed     float64
	physicsSampleSteps       int
	physicsSampleComputeTime float64
	physicsSampleDroppedTime float64
	physicsDroppedTimeTotal  float64
	physicsWarningTimer      float64
	physicsLastFrameSteps    int
	physicsPeakFrameSteps    int

	previousRenderSnapshot   renderSnapshot
	renderSnapshotReady      bool
	renderInterpolationAlpha float64

	lastPaddleSpinValid  bool
	lastPaddleSpinBall   int
	lastPaddleSpinBefore float64
	lastPaddleSpinAdded  float64
	lastPaddleSpinAfter  float64

	lastCollisionValid                   bool
	lastCollisionSurface                 string
	lastCollisionIncomingAngle           float64
	lastCollisionOutgoingAngle           float64
	lastCollisionAngleChange             float64
	lastCollisionSpinAngleEffect         float64
	lastCollisionSpinBefore              float64
	lastCollisionSpinAfter               float64
	lastCollisionSpinChange              float64
	lastCollisionTangentialImpulse       float64
	lastCollisionNoSpinImpulse           float64
	lastCollisionSpinImpulseEffect       float64
	lastCollisionContactSpinSurfaceSpeed float64

	// ---- Audio ----
	audioCtx         js.Value
	audioMaster      js.Value
	audioBrickBus    js.Value
	audioMagicBus    js.Value
	audioSynthBus    js.Value
	audioInitialized bool

	// Reusable per-level room-effects graph. The three mixer buses feed
	// audioRoomInput; its dry and processed wet branches reunite at audioMaster.
	audioRoomInput          js.Value
	audioRoomDryGain        js.Value
	audioRoomWetGain        js.Value
	audioRoomInputFilter    js.Value
	audioRoomOutputFilter   js.Value
	audioRoomDelay1         js.Value
	audioRoomDelay2         js.Value
	audioRoomDelay3         js.Value
	audioRoomTapGain1       js.Value
	audioRoomTapGain2       js.Value
	audioRoomTapGain3       js.Value
	audioRoomWetSum         js.Value
	audioRoomFeedbackFilter js.Value
	audioRoomFeedbackGain   js.Value

	brickHitBuffers   []js.Value
	brickHitLastIndex = -1

	magicFeatureBuffers      = make(map[string]js.Value)
	magicFeatureVoices       = make(map[string]*magicFeatureVoice)
	magicFeatureActiveVoices int

	// Embedded compressed audio is read and decoded one file at a time in the
	// background. Empty/not-yet-ready sample banks use the existing synth
	// fallback immediately, so gameplay never waits for audio decoding.
	embeddedAudioCallbacks      []js.Func
	embeddedAudioPreloadStarted bool
	embeddedAudioQueue          []embeddedAudioSample
	embeddedAudioLoadActive     bool
	embeddedAudioMagicFeatures  = make(map[string]bool)
	embeddedAudioLoadedCount    int
	embeddedAudioFailedCount    int

	brickHitActiveVoices    int
	brickHitStartsThisFrame int
)

type embeddedAudioSample struct {
	path    string
	label   string
	feature string
	magic   bool
}

type brick struct {
	x, y, w, h  float64
	row, col    int
	tiltRadians float64
	alive       bool
	unbreakable bool
	magic       bool
}

// debrisFragment is a lightweight rigid shard. Rendering uses an irregular,
// single-fill-color local polygon, while collision uses its conservative bounding circle. This
// keeps hundreds of fragments practical inside the 240 Hz fixed-step loop.
type debrisFragment struct {
	x, y          float64
	previousX     float64
	previousY     float64
	vx, vy        float64
	angle         float64
	previousAngle float64
	omega         float64
	radius        float64
	mass          float64
	age           float64
	lifetime      float64
	pointCount    int
	points        [32]float64
	roundness     float64
	circle        bool
	startOpacity  float64
	fillColor     string

	// Cached rendering data. The Path2D outline is immutable until live size tuning
	// changes the shape; the opaque palette is shared by shards of the same color.
	renderID           uint32
	renderPath         js.Value
	renderPathReady    bool
	renderColorPalette []string
}

const debrisMasterStorageKey = "breakout.debrisMasterEnabled"

func debrisIsEnabled() bool {
	return debrisMasterEnabled && debrisEnabled
}

func loadDebrisMasterPreference() {
	// localStorage may be unavailable or blocked. Keep the safe default (ON) if
	// access throws or the saved value is missing/invalid.
	defer func() {
		if recover() != nil {
			debrisMasterEnabled = true
		}
	}()

	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return
	}
	value := storage.Call("getItem", debrisMasterStorageKey)
	if value.IsUndefined() || value.IsNull() {
		return
	}
	enabled, err := strconv.ParseBool(strings.TrimSpace(value.String()))
	if err == nil {
		debrisMasterEnabled = enabled
	}
}

func saveDebrisMasterPreference() {
	defer func() {
		_ = recover()
	}()

	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return
	}
	storage.Call("setItem", debrisMasterStorageKey, strconv.FormatBool(debrisMasterEnabled))
}

func toggleDebrisMaster() {
	debrisMasterEnabled = !debrisMasterEnabled
	if !debrisMasterEnabled {
		brickDebris = brickDebris[:0]
		debrisRenderedLastFrame = 0
		debrisSkippedLastFrame = 0
		debrisAdaptiveRenderStride = 1
	}
	saveDebrisMasterPreference()
	if debrisMasterEnabled {
		showStatus("Debris ON (saved)", 2.0)
	} else {
		showStatus("Debris OFF (saved)", 2.0)
	}
}

const (
	debrisShapeModeMixed     = "mixed"
	debrisShapeModeTriangles = "triangles"
	debrisShapeModeCircles   = "circles"
)

func parseDebrisShapeMode(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case debrisShapeModeMixed, "mix":
		return debrisShapeModeMixed, true
	case debrisShapeModeTriangles, "triangle":
		return debrisShapeModeTriangles, true
	case debrisShapeModeCircles, "circle":
		return debrisShapeModeCircles, true
	default:
		return "", false
	}
}

// ---- Level data ----
type levelData struct {
	layout []string
	config map[string]string
}

// ---- Helpers ----
func log(msg string) {
	js.Global().Get("console").Call("log", msg)
}

func sign(x float64) float64 {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}

// brickMicroTiltRadians returns a deterministic random-looking angle. It does
// not use math/rand, so brick construction cannot disturb power-up or gameplay
// randomness. Re-entering the same level produces the same brick angles.
func brickMicroTiltRadians(levelIndex, row, col int) float64 {
	hash := uint32(levelIndex+1)*0x9e3779b9 ^
		uint32(row+1)*0x85ebca6b ^
		uint32(col+1)*0xc2b2ae35
	hash ^= hash >> 16
	hash *= 0x7feb352d
	hash ^= hash >> 15
	hash *= 0x846ca68b
	hash ^= hash >> 16

	unit := float64(hash&0x00ffffff) / float64(0x00ffffff)
	magnitudeDegrees := physicsConfig.brickTiltMinDegrees +
		unit*(physicsConfig.brickTiltMaxDegrees-physicsConfig.brickTiltMinDegrees)
	if hash&0x80000000 != 0 {
		magnitudeDegrees = -magnitudeDegrees
	}
	return magnitudeDegrees * math.Pi / 180.0
}

func rotateVector(x, y, angle float64) (float64, float64) {
	cosAngle := math.Cos(angle)
	sinAngle := math.Sin(angle)
	return x*cosAngle - y*sinAngle, x*sinAngle + y*cosAngle
}

func wallNoiseHash(cell, wallID int) float64 {
	hash := uint32(cell+0x40000000)*0x9e3779b9 ^
		uint32(currentLevelIndex+1)*0x85ebca6b ^
		uint32(wallID)*0xc2b2ae35
	hash ^= hash >> 16
	hash *= 0x7feb352d
	hash ^= hash >> 15
	hash *= 0x846ca68b
	hash ^= hash >> 16
	return float64(hash&0x00ffffff)/float64(0x00ffffff)*2 - 1
}

// wallNoise returns smooth deterministic one-dimensional value noise.
func wallNoise(position float64, wallID int) float64 {
	if physicsConfig.wallNoiseCellSize <= 0 {
		return 0
	}

	x := position / physicsConfig.wallNoiseCellSize
	cell := int(math.Floor(x))
	t := x - float64(cell)
	t = t * t * (3 - 2*t)
	a := wallNoiseHash(cell, wallID)
	b := wallNoiseHash(cell+1, wallID)
	return a + (b-a)*t
}

func wallCornerFade(position, wallLength float64) float64 {
	if physicsConfig.wallCornerFadeDistance <= 0 || wallLength <= 0 {
		return 1
	}

	edgeDistance := math.Min(position, wallLength-position)
	t := clampFloat(edgeDistance/physicsConfig.wallCornerFadeDistance, 0, 1)
	return t * t * (3 - 2*t)
}

func roughWallNormal(
	nx, ny, position, wallLength float64,
	wallID int,
	maximumTiltDegrees float64,
) (float64, float64) {
	angleDegrees := wallNoise(position, wallID) * maximumTiltDegrees
	angleDegrees *= wallCornerFade(position, wallLength)
	return rotateVector(nx, ny, angleDegrees*math.Pi/180)
}

const (
	orbitAxisNone = iota
	orbitAxisVertical
	orbitAxisHorizontal
)

func resetFastOrbitCandidate(b *Ball) {
	b.orbitCandidateAxis = orbitAxisNone
	b.orbitCandidateHits = 0
	b.orbitCandidateTimer = 0
	b.orbitAnchorMinor = 0
}

func resetFastOrbitState(b *Ball) {
	resetFastOrbitCandidate(b)
	b.orbitHitCooldown = 0
	b.orbitEscapeAxis = orbitAxisNone
	b.orbitEscapeDirection = 0
	b.orbitEscapeTimer = 0
}

func fastOrbitAxis(b *Ball) int {
	speed := math.Hypot(b.vx, b.vy)
	if speed < physicsConfig.orbitMinimumSpeed {
		return orbitAxisNone
	}

	minorLimit := math.Max(physicsConfig.orbitMinorSpeedFloor, speed*physicsConfig.orbitMinorSpeedRatio)
	nearVertical := math.Abs(b.vx) <= minorLimit
	nearHorizontal := math.Abs(b.vy) <= minorLimit
	if nearVertical && !nearHorizontal {
		return orbitAxisVertical
	}
	if nearHorizontal && !nearVertical {
		return orbitAxisHorizontal
	}
	return orbitAxisNone
}

func orbitMinorPosition(b *Ball, axis int) float64 {
	if axis == orbitAxisVertical {
		return b.x
	}
	return b.y
}

func chooseOrbitEscapeDirection(b *Ball, axis int) float64 {
	minorVelocity := b.vy
	if axis == orbitAxisVertical {
		minorVelocity = b.vx
	}
	if math.Abs(minorVelocity) >= 1 {
		return sign(minorVelocity)
	}
	if math.Abs(b.omega) >= 0.01 {
		return sign(b.omega)
	}
	if axis == orbitAxisVertical {
		direction := sign(canvasWidth/2 - b.x)
		if direction != 0 {
			return direction
		}
	} else {
		direction := sign(canvasHeight/2 - b.y)
		if direction != 0 {
			return direction
		}
	}
	return 1
}

func enforceFastOrbitEscape(b *Ball) {
	if b.orbitEscapeTimer <= 0 || b.orbitEscapeAxis == orbitAxisNone {
		return
	}

	speed := math.Hypot(b.vx, b.vy)
	if speed <= 0 {
		return
	}
	minorSpeed := math.Min(physicsConfig.orbitEscapeSpeed, speed*0.35)
	majorSpeed := math.Sqrt(math.Max(0, speed*speed-minorSpeed*minorSpeed))

	if b.orbitEscapeAxis == orbitAxisVertical {
		majorDirection := sign(b.vy)
		if majorDirection == 0 {
			majorDirection = -1
		}
		b.vx = b.orbitEscapeDirection * minorSpeed
		b.vy = majorDirection * majorSpeed
		return
	}

	majorDirection := sign(b.vx)
	if majorDirection == 0 {
		majorDirection = 1
	}
	b.vx = majorDirection * majorSpeed
	b.vy = b.orbitEscapeDirection * minorSpeed
}

func activateFastOrbitEscape(b *Ball, axis int) {
	b.orbitEscapeAxis = axis
	b.orbitEscapeDirection = chooseOrbitEscapeDirection(b, axis)
	b.orbitEscapeTimer = physicsConfig.orbitEscapeDuration
	resetFastOrbitCandidate(b)
	enforceFastOrbitEscape(b)
	beginBallRescueAttempt(b, axis)
	showStatusUnique("Orbital tilt!", physicsOrbitMessageDuration)
}

func recordUnbreakableOrbitHit(b *Ball) {
	if b.orbitHitCooldown > 0 || b.orbitEscapeTimer > 0 {
		return
	}

	axis := fastOrbitAxis(b)
	if axis == orbitAxisNone {
		resetFastOrbitCandidate(b)
		return
	}

	minorPosition := orbitMinorPosition(b, axis)
	newCandidate := b.orbitCandidateAxis != axis ||
		b.orbitCandidateHits == 0 ||
		b.orbitCandidateTimer > physicsConfig.orbitDetectionWindow ||
		math.Abs(minorPosition-b.orbitAnchorMinor) > physicsConfig.orbitMaximumMinorProgress

	if newCandidate {
		b.orbitCandidateAxis = axis
		b.orbitCandidateHits = 1
		b.orbitCandidateTimer = 0
		b.orbitAnchorMinor = minorPosition
	} else {
		b.orbitCandidateHits++
	}
	b.orbitHitCooldown = physicsConfig.orbitHitCooldown

	if b.orbitCandidateHits >= physicsConfig.orbitRequiredHits {
		activateFastOrbitEscape(b, axis)
	}
}

func updateFastOrbitDetector(b *Ball, dt float64) {
	if b.orbitHitCooldown > 0 {
		b.orbitHitCooldown = math.Max(0, b.orbitHitCooldown-dt)
	}
	if b.orbitCandidateHits > 0 {
		b.orbitCandidateTimer += dt
		if b.orbitCandidateTimer > physicsConfig.orbitDetectionWindow {
			resetFastOrbitCandidate(b)
		}
	}
	if b.orbitEscapeTimer > 0 {
		b.orbitEscapeTimer = math.Max(0, b.orbitEscapeTimer-dt)
		enforceFastOrbitEscape(b)
		if b.orbitEscapeTimer == 0 {
			b.orbitEscapeAxis = orbitAxisNone
			b.orbitEscapeDirection = 0
		}
	}
}

func resetBallRescueState(b *Ball, resetFailures bool) {
	if b == nil {
		return
	}
	b.rescueAttemptActive = false
	b.rescueAttemptAxis = orbitAxisNone
	b.rescueAttemptElapsed = 0
	b.rescueStartX = b.x
	b.rescueStartY = b.y
	if resetFailures {
		b.rescueFailureCount = 0
	}
}

func beginBallRescueAttempt(b *Ball, progressAxis int) {
	if b == nil || !ballRescueEnabled || b.rescueAttemptActive {
		return
	}
	b.rescueAttemptActive = true
	b.rescueAttemptAxis = progressAxis
	b.rescueAttemptElapsed = 0
	b.rescueStartX = b.x
	b.rescueStartY = b.y
}

func ballRescueAttemptProgress(b *Ball) float64 {
	if b == nil || !b.rescueAttemptActive {
		return 0
	}
	switch b.rescueAttemptAxis {
	case orbitAxisVertical:
		return math.Abs(b.x - b.rescueStartX)
	case orbitAxisHorizontal:
		return math.Abs(b.y - b.rescueStartY)
	default:
		return math.Hypot(b.x-b.rescueStartX, b.y-b.rescueStartY)
	}
}

func ballRescuePerformanceHealthy() bool {
	if physicsStepRateCurrent <= 0 || physicsRealtimePercent <= 0 {
		return false
	}
	return physicsWarningTimer <= 0 &&
		physicsRealtimePercent >= ballRescueMinRealtimePercent &&
		physicsComputeLoad <= ballRescueMaxComputeLoad
}

func ballTeleportPositionClear(b *Ball, x, y, clearance float64) bool {
	if b == nil {
		return false
	}
	margin := b.r + math.Max(0, clearance)
	if x < margin || x > canvasWidth-margin || y < margin || y > paddle.y-margin {
		return false
	}

	for i := range bricks {
		br := &bricks[i]
		if !br.alive {
			continue
		}
		closestX := clampFloat(x, br.x, br.x+br.w)
		closestY := clampFloat(y, br.y, br.y+br.h)
		dx := x - closestX
		dy := y - closestY
		if dx*dx+dy*dy < margin*margin {
			return false
		}
	}

	positionClear := true
	forEachActiveBall(func(other *Ball, _ int) {
		if !positionClear || other == b {
			return
		}
		minimumDistance := b.r + other.r + math.Max(0, clearance)
		if math.Hypot(x-other.x, y-other.y) < minimumDistance {
			positionClear = false
		}
	})
	if !positionClear {
		return false
	}

	if blackHoleActive {
		minimumDistance := math.Max(48, b.r+math.Max(0, clearance))
		if math.Hypot(x-blackHoleX, y-blackHoleY) < minimumDistance {
			return false
		}
	}
	return true
}

func searchBallTeleportPosition(b *Ball, maximumY, clearance float64) (float64, float64, bool) {
	if b == nil {
		return 0, 0, false
	}
	minimumY := b.r + math.Max(12, clearance)
	maximumY = math.Min(maximumY, paddle.y-b.r-math.Max(12, clearance))
	if maximumY <= minimumY {
		return 0, 0, false
	}

	const columns = 25
	const rows = 14
	centerColumn := columns / 2
	for row := rows - 1; row >= 0; row-- {
		y := minimumY + (float64(row)+0.5)/float64(rows)*(maximumY-minimumY)
		for offset := 0; offset <= centerColumn; offset++ {
			indices := []int{centerColumn + offset}
			if offset > 0 {
				indices = append(indices, centerColumn-offset)
			}
			for _, column := range indices {
				if column < 0 || column >= columns {
					continue
				}
				x := (float64(column) + 0.5) / float64(columns) * canvasWidth
				if ballTeleportPositionClear(b, x, y, clearance) {
					return x, y, true
				}
			}
		}
	}
	return 0, 0, false
}

func safeBallTeleportPosition(b *Ball) (float64, float64, bool) {
	upperLimit := canvasHeight * clampFloat(ballRescueUpperScreenFraction, 0.10, 0.90)
	limits := []float64{upperLimit, canvasHeight * 0.55, paddle.y - b.r - 8}
	clearances := []float64{ballRescueClearance, ballRescueClearance * 0.5, 0}
	for _, maximumY := range limits {
		for _, clearance := range clearances {
			if x, y, ok := searchBallTeleportPosition(b, maximumY, clearance); ok {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func teleportBallToSafeArea(b *Ball, manual bool) bool {
	if b == nil {
		return false
	}
	x, y, found := safeBallTeleportPosition(b)
	if !found {
		showStatus("No safe teleport position", 1.5)
		return false
	}

	speed := ballRescueLaunchSpeed
	if speed <= 0 {
		speed = math.Max(300, math.Hypot(startBallVx, startBallVy))
	}
	if physicsConfig.maxSpeed > 0 {
		speed = math.Min(speed, physicsConfig.maxSpeed)
	}
	horizontalSpeed := speed * 0.35
	direction := 1.0
	ballIndex := ballIndexOf(b)
	if b.rescueFailureCount%2 != 0 || ballIndex%2 == 1 {
		direction = -1
	}
	verticalSpeed := math.Sqrt(math.Max(0, speed*speed-horizontalSpeed*horizontalSpeed))

	b.x, b.y = x, y
	b.vx = direction * horizontalSpeed
	b.vy = verticalSpeed
	b.omega *= 0.25
	b.stuckTimer = 0
	resetFastOrbitState(b)
	resetBallRescueState(b, true)
	syncRenderInterpolation()
	if manual {
		showStatus("Ball teleported (T)", 1.5)
	} else if ballIndex >= 0 {
		showStatus("Ball "+strconv.Itoa(ballIndex+1)+" rescue teleport!", 2.0)
	} else {
		showStatus("Ball rescue teleport!", 2.0)
	}
	return true
}

func updateBallRescueAttempt(b *Ball, dt float64) {
	if b == nil || !ballRescueEnabled {
		resetBallRescueState(b, true)
		return
	}
	if !b.rescueAttemptActive {
		return
	}
	b.rescueAttemptElapsed += dt
	if ballRescueAttemptProgress(b) >= ballRescueMinProgress {
		resetBallRescueState(b, true)
		return
	}
	if b.rescueAttemptElapsed < ballRescueCheckDuration {
		return
	}

	if !ballRescuePerformanceHealthy() {
		resetBallRescueState(b, false)
		return
	}

	b.rescueAttemptActive = false
	b.rescueAttemptAxis = orbitAxisNone
	b.rescueAttemptElapsed = 0
	b.rescueFailureCount++
	if b.rescueFailureCount >= ballRescueFailureLimit {
		teleportBallToSafeArea(b, false)
	}
}

// preventVerticalLock preserves total
// speed while giving nearly vertical trajectories a small deterministic
// horizontal component.
func preventVerticalLock(b *Ball, preferredDirection float64) {
	const minimumHorizontalSpeed = 60.0

	speed := math.Hypot(b.vx, b.vy)
	if speed < minimumHorizontalSpeed || math.Abs(b.vx) >= minimumHorizontalSpeed {
		return
	}

	direction := sign(preferredDirection)
	if direction == 0 {
		direction = sign(b.omega)
	}
	if direction == 0 {
		direction = sign(b.x - canvasWidth/2)
	}
	if direction == 0 {
		direction = 1
	}

	targetVx := direction * math.Min(minimumHorizontalSpeed, speed*0.35)
	vySign := sign(b.vy)
	if vySign == 0 {
		vySign = -1
	}

	remainingVySquared := math.Max(0, speed*speed-targetVx*targetVx)
	b.vx = targetVx
	b.vy = vySign * math.Sqrt(remainingVySquared)
}

func clearLastPaddleSpinDebug() {
	lastPaddleSpinValid = false
	lastPaddleSpinBall = 0
	lastPaddleSpinBefore = 0
	lastPaddleSpinAdded = 0
	lastPaddleSpinAfter = 0
}

func recordLastPaddleSpinDebug(isPrimary bool, before, after float64) {
	lastPaddleSpinValid = true
	if isPrimary {
		lastPaddleSpinBall = 1
	} else {
		lastPaddleSpinBall = 2
	}
	lastPaddleSpinBefore = before
	lastPaddleSpinAfter = after
	lastPaddleSpinAdded = after - before
}

func velocityAngleDegrees(vx, vy float64) float64 {
	return math.Atan2(vy, vx) * 180 / math.Pi
}

func normalizeAngleDegrees(angle float64) float64 {
	for angle > 180 {
		angle -= 360
	}
	for angle <= -180 {
		angle += 360
	}
	return angle
}

func clearLastCollisionDebug() {
	lastCollisionValid = false
	lastCollisionSurface = ""
	lastCollisionIncomingAngle = 0
	lastCollisionOutgoingAngle = 0
	lastCollisionAngleChange = 0
	lastCollisionSpinAngleEffect = 0
	lastCollisionSpinBefore = 0
	lastCollisionSpinAfter = 0
	lastCollisionSpinChange = 0
	lastCollisionTangentialImpulse = 0
	lastCollisionNoSpinImpulse = 0
	lastCollisionSpinImpulseEffect = 0
	lastCollisionContactSpinSurfaceSpeed = 0
}

func recordLastCollisionDebug(
	surface string,
	incomingAngle, outgoingAngle, noSpinOutgoingAngle float64,
	spinBefore, spinAfter, tangentialImpulse, noSpinImpulse, ballRadius float64,
) {
	lastCollisionValid = true
	lastCollisionSurface = surface
	lastCollisionIncomingAngle = incomingAngle
	lastCollisionOutgoingAngle = outgoingAngle
	lastCollisionAngleChange = normalizeAngleDegrees(outgoingAngle - incomingAngle)
	lastCollisionSpinAngleEffect = normalizeAngleDegrees(outgoingAngle - noSpinOutgoingAngle)
	lastCollisionSpinBefore = spinBefore
	lastCollisionSpinAfter = spinAfter
	lastCollisionSpinChange = spinAfter - spinBefore
	lastCollisionTangentialImpulse = tangentialImpulse
	lastCollisionNoSpinImpulse = noSpinImpulse
	lastCollisionSpinImpulseEffect = tangentialImpulse - noSpinImpulse
	lastCollisionContactSpinSurfaceSpeed = -spinBefore * ballRadius
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func moveToward(current, target, maxDelta float64) float64 {
	if current < target {
		return math.Min(current+maxDelta, target)
	}
	if current > target {
		return math.Max(current-maxDelta, target)
	}
	return target
}

func resetPaddleSpinHistory() {
	paddleSpinHistory = paddleSpinHistory[:0]
}

func updatePaddleSpinHistory(dt float64) {
	if dt <= 0 || physicsConfig.paddleSpinGraceSeconds <= 0 {
		resetPaddleSpinHistory()
		return
	}

	kept := paddleSpinHistory[:0]
	for _, sample := range paddleSpinHistory {
		sample.age += dt
		if sample.age <= physicsConfig.paddleSpinGraceSeconds {
			kept = append(kept, sample)
		}
	}
	paddleSpinHistory = append(kept, paddleVelocitySample{vx: paddle.vx})
}

func effectivePaddleSpinVelocity() float64 {
	best := paddle.vx
	for _, sample := range paddleSpinHistory {
		if sample.age < 0 || sample.age > physicsConfig.paddleSpinGraceSeconds {
			continue
		}
		weight := 1 - sample.age/physicsConfig.paddleSpinGraceSeconds
		candidate := sample.vx * weight
		if math.Abs(candidate) > math.Abs(best) {
			best = candidate
		}
	}
	return best
}

func showStatus(text string, duration float64) {
	statusMessages = append(statusMessages, statusMessage{text: text, timer: duration})
	if len(statusMessages) > statusMessageLimit {
		statusMessages = statusMessages[len(statusMessages)-statusMessageLimit:]
	}
}

func showStatusUnique(text string, duration float64) {
	for i := range statusMessages {
		if statusMessages[i].text == text {
			statusMessages[i].timer = duration
			return
		}
	}
	showStatus(text, duration)
}

// Keep a single High spin! entry in the existing top-left status stack.
// A new qualifying paddle hit refreshes its timer instead of adding duplicates.
func showHighSpinStatus() {
	if !enableHighSpinMessage {
		return
	}

	showStatusUnique("High spin!", highSpinMessageDuration)
}

func maybeShowHighSpin(before, after float64) {
	// Use the real post-clamp spin change caused by this paddle hit.
	if math.Abs(after-before) >= highSpinThreshold {
		showHighSpinStatus()
	}
}

func playImpactSound(b *Ball, impactSpeed float64, play func()) {
	if impactSpeed < minimumCollisionSoundSpeed || b.soundCooldown > 0 {
		return
	}
	play()
	b.soundCooldown = collisionSoundCooldown
}

func gridKey(row, col int) int {
	return row*gridCols + col
}

func registerBrick(index int) {
	br := &bricks[index]
	key := gridKey(br.row, br.col)
	brickGrid[key] = append(brickGrid[key], index)
	if key >= 0 && key < len(brickGridCells) {
		brickGridCells[key] = append(brickGridCells[key], index)
	}
	if !br.unbreakable {
		remainingBreakableBricks++
	}
}

func brickDebrisColor(br *brick) string {
	if br == nil {
		return palette[2]
	}
	if br.magic {
		return magicColor
	}
	if br.unbreakable {
		return palette[6]
	}
	return palette[2]
}

func normalizeDebrisSettings() {
	if mode, ok := parseDebrisShapeMode(debrisShapeMode); ok {
		debrisShapeMode = mode
	} else if mode, ok := parseDebrisShapeMode(defaultDebrisShapeMode); ok {
		debrisShapeMode = mode
	} else {
		debrisShapeMode = debrisShapeModeMixed
	}
	if debrisPiecesMin < 1 {
		debrisPiecesMin = 1
	}
	if debrisPiecesMax < 1 {
		debrisPiecesMax = 1
	}
	if debrisPiecesMin > debrisPiecesMax {
		debrisPiecesMin, debrisPiecesMax = debrisPiecesMax, debrisPiecesMin
	}
	if debrisPiecesMax > 32 {
		debrisPiecesMax = 32
	}
	if debrisPiecesMin > debrisPiecesMax {
		debrisPiecesMin = debrisPiecesMax
	}
	debrisLifetime = math.Max(0.05, debrisLifetime)
	debrisLifetimeVariationPercent = clampFloat(debrisLifetimeVariationPercent, 0, 95)
	debrisFadeDuration = clampFloat(debrisFadeDuration, 0, debrisLifetime)
	debrisStartOpacity = clampFloat(debrisStartOpacity, 0, 1)
	debrisStartOpacityVariation = clampFloat(debrisStartOpacityVariation, 0, 1)
	debrisFlashDuration = clampFloat(debrisFlashDuration, 0, debrisLifetime)
	debrisFlashOpacity = clampFloat(debrisFlashOpacity, 0, 1)
	debrisImpactSpeedFactor = math.Max(0, debrisImpactSpeedFactor)
	debrisBallPieceChance = clampFloat(debrisBallPieceChance, 0, 1)
	debrisTrianglePieceChance = clampFloat(debrisTrianglePieceChance, 0, 1)
	debrisStarPieceChance = clampFloat(debrisStarPieceChance, 0, 1)
	debrisGlassPieceChance = clampFloat(debrisGlassPieceChance, 0, 1)
	debrisSliverPieceChance = clampFloat(debrisSliverPieceChance, 0, 1)
	debrisStarPointsMin = max(3, min(8, debrisStarPointsMin))
	debrisStarPointsMax = max(3, min(8, debrisStarPointsMax))
	if debrisStarPointsMin > debrisStarPointsMax {
		debrisStarPointsMin, debrisStarPointsMax = debrisStarPointsMax, debrisStarPointsMin
	}
	debrisGlassCornersMin = max(7, min(16, debrisGlassCornersMin))
	debrisGlassCornersMax = max(7, min(16, debrisGlassCornersMax))
	if debrisGlassCornersMin > debrisGlassCornersMax {
		debrisGlassCornersMin, debrisGlassCornersMax = debrisGlassCornersMax, debrisGlassCornersMin
	}
	specialShapeChance := debrisBallPieceChance + debrisTrianglePieceChance +
		debrisStarPieceChance + debrisGlassPieceChance + debrisSliverPieceChance
	if specialShapeChance > 0.95 {
		scale := 0.95 / specialShapeChance
		debrisBallPieceChance *= scale
		debrisTrianglePieceChance *= scale
		debrisStarPieceChance *= scale
		debrisGlassPieceChance *= scale
		debrisSliverPieceChance *= scale
	}
	debrisMaxChunkAspectRatio = math.Max(1, debrisMaxChunkAspectRatio)
	debrisSizeScale = clampFloat(debrisSizeScale, 0.25, 3.0)
	debrisBrickCollisionDelay = math.Max(0, debrisBrickCollisionDelay)
	debrisGravityScale = math.Max(0, debrisGravityScale)
	debrisAirDrag = math.Max(0, debrisAirDrag)
	debrisRestitution = clampFloat(debrisRestitution, 0, 1.5)
	debrisFriction = clampFloat(debrisFriction, 0, 2)
	debrisExplosionSpeedMin = math.Max(0, debrisExplosionSpeedMin)
	debrisExplosionSpeedMax = math.Max(0, debrisExplosionSpeedMax)
	if debrisExplosionSpeedMin > debrisExplosionSpeedMax {
		debrisExplosionSpeedMin, debrisExplosionSpeedMax = debrisExplosionSpeedMax, debrisExplosionSpeedMin
	}
	debrisAngularSpeedMin = math.Max(0, debrisAngularSpeedMin)
	debrisAngularSpeedMax = math.Max(0, debrisAngularSpeedMax)
	if debrisAngularSpeedMin > debrisAngularSpeedMax {
		debrisAngularSpeedMin, debrisAngularSpeedMax = debrisAngularSpeedMax, debrisAngularSpeedMin
	}
	debrisAngularDrag = math.Max(0, debrisAngularDrag)
	debrisAngularStopSpeed = math.Max(0, debrisAngularStopSpeed)
	debrisBallInfluence = clampFloat(debrisBallInfluence, 0, 1)
	debrisFieldScale = math.Max(0, debrisFieldScale)
	debrisMagnetScale = math.Max(0, debrisMagnetScale)
	debrisMaxSpeed = math.Max(0, debrisMaxSpeed)
	debrisFrontLayerSpeed = math.Max(0, debrisFrontLayerSpeed)
	if debrisMaxActivePieces < 0 {
		debrisMaxActivePieces = 0
	}
	debrisOffscreenMargin = math.Max(0, debrisOffscreenMargin)
}

func normalizeBallRescueSettings() {
	if ballRescueFailureLimit < 1 {
		ballRescueFailureLimit = 1
	}
	ballRescueMinProgress = math.Max(0, ballRescueMinProgress)
	ballRescueCheckDuration = math.Max(physicsStepSeconds, ballRescueCheckDuration)
	ballRescueUpperScreenFraction = clampFloat(ballRescueUpperScreenFraction, 0.10, 0.90)
	ballRescueClearance = math.Max(0, ballRescueClearance)
	ballRescueLaunchSpeed = math.Max(0, ballRescueLaunchSpeed)
	ballRescueMinRealtimePercent = clampFloat(ballRescueMinRealtimePercent, 0, 100)
	ballRescueMaxComputeLoad = clampFloat(ballRescueMaxComputeLoad, 0, 1000)
	ballBelowFloorGracePixels = math.Max(0, ballBelowFloorGracePixels)
}

func trimOldestDebrisFor(additional int) {
	if additional <= 0 || debrisMaxActivePieces <= 0 {
		return
	}
	excess := len(brickDebris) + additional - debrisMaxActivePieces
	if excess <= 0 {
		return
	}
	if excess >= len(brickDebris) {
		brickDebris = brickDebris[:0]
		return
	}
	copy(brickDebris, brickDebris[excess:])
	brickDebris = brickDebris[:len(brickDebris)-excess]
}

func debrisRandomBetween(minimum, maximum float64) float64 {
	if maximum <= minimum {
		return minimum
	}
	return minimum + debrisRNG.Float64()*(maximum-minimum)
}

func debrisRandomInt(minimum, maximum int) int {
	if maximum <= minimum {
		return minimum
	}
	return minimum + debrisRNG.Intn(maximum-minimum+1)
}

func debrisPolygonArea(points [32]float64, pointCount int) float64 {
	if pointCount < 3 {
		return 0
	}
	area := 0.0
	for i := 0; i < pointCount; i++ {
		next := (i + 1) % pointCount
		area += points[i*2]*points[next*2+1] - points[next*2]*points[i*2+1]
	}
	return math.Abs(area) * 0.5
}

func rotateDebrisPoints(points *[32]float64, pointCount int, angle float64) {
	if points == nil || pointCount <= 0 || angle == 0 {
		return
	}
	for i := 0; i < pointCount; i++ {
		points[i*2], points[i*2+1] = rotateVector(points[i*2], points[i*2+1], angle)
	}
}

func debrisRadialPolygon(
	pointCount int,
	radiusX, radiusY,
	radiusMinimum, radiusMaximum,
	angleJitter, rotation float64,
) [32]float64 {
	var points [32]float64
	pointCount = max(3, min(16, pointCount))
	for i := 0; i < pointCount; i++ {
		angle := rotation + 2*math.Pi*float64(i)/float64(pointCount)
		angle += debrisRandomBetween(-angleJitter, angleJitter)
		radiusScale := debrisRandomBetween(radiusMinimum, radiusMaximum)
		points[i*2] = math.Cos(angle) * radiusX * radiusScale
		points[i*2+1] = math.Sin(angle) * radiusY * radiusScale
	}
	return points
}

func debrisStarPolygon(pointPairs int, radiusX, radiusY, rotation float64) ([32]float64, int) {
	var points [32]float64
	pointPairs = max(3, min(8, pointPairs))
	pointCount := pointPairs * 2
	innerScale := debrisRandomBetween(0.34, 0.62)
	step := math.Pi / float64(pointPairs)
	for i := 0; i < pointCount; i++ {
		angle := rotation + float64(i)*step + debrisRandomBetween(-step*0.16, step*0.16)
		radiusScale := debrisRandomBetween(0.86, 1.08)
		if i%2 == 1 {
			radiusScale *= innerScale * debrisRandomBetween(0.88, 1.12)
		}
		points[i*2] = math.Cos(angle) * radiusX * radiusScale
		points[i*2+1] = math.Sin(angle) * radiusY * radiusScale
	}
	return points, pointCount
}

// buildDebrisShape either forces one cheap family for browser comparisons or
// uses the existing mixed families. Collision remains one conservative circle.
func buildDebrisShape(cellWidth, cellHeight float64) (
	points [32]float64,
	pointCount int,
	radius, area, roundness float64,
	circle bool,
) {
	halfWidth := cellWidth * debrisRandomBetween(0.37, 0.50)
	halfHeight := cellHeight * debrisRandomBetween(0.36, 0.49)

	// Most debris should read as compact chunks rather than long sticks. Only the
	// deliberately rare sliver family may narrow itself further below.
	if halfWidth > halfHeight*debrisMaxChunkAspectRatio {
		halfWidth = halfHeight * debrisRandomBetween(1.0, debrisMaxChunkAspectRatio)
	}
	if halfHeight > halfWidth*debrisMaxChunkAspectRatio {
		halfHeight = halfWidth * debrisRandomBetween(1.0, debrisMaxChunkAspectRatio)
	}

	switch debrisShapeMode {
	case debrisShapeModeCircles:
		// A true circular chip. Drawing and collision use the same radius.
		radius = math.Min(halfWidth, halfHeight) * debrisRandomBetween(0.72, 0.96)
		radius = math.Max(2, radius)
		area = math.Pi * radius * radius
		return points, 0, radius, area, 1, true

	case debrisShapeModeTriangles:
		// Deliberately sharp triangles: exactly three straight edges and no rounded
		// corners, making this the cheapest non-circular Path2D test mode.
		pointCount = 3
		points = debrisRadialPolygon(
			pointCount,
			halfWidth,
			halfHeight,
			0.72,
			1.06,
			0.12,
			debrisRandomBetween(-math.Pi, math.Pi),
		)
		rotateDebrisPoints(&points, pointCount, debrisRandomBetween(-0.18, 0.18))
		for i := 0; i < pointCount; i++ {
			radius = math.Max(radius, math.Hypot(points[i*2], points[i*2+1]))
		}
		area = debrisPolygonArea(points, pointCount)
		return points, pointCount, math.Max(2, radius), math.Max(1, area), 0, false
	}

	shapeRoll := debrisRNG.Float64()
	ballLimit := debrisBallPieceChance
	triangleLimit := ballLimit + debrisTrianglePieceChance
	starLimit := triangleLimit + debrisStarPieceChance
	glassLimit := starLimit + debrisGlassPieceChance
	sliverLimit := glassLimit + debrisSliverPieceChance

	switch {
	case shapeRoll < ballLimit:
		// A true circular chip. Drawing and collision use the same radius.
		radius = math.Min(halfWidth, halfHeight) * debrisRandomBetween(0.72, 0.96)
		radius = math.Max(2, radius)
		area = math.Pi * radius * radius
		return points, 0, radius, area, 1, true

	case shapeRoll < triangleLimit:
		// Compact, visibly triangular fragments rather than narrow slivers.
		pointCount = 3
		points = debrisRadialPolygon(
			pointCount,
			halfWidth,
			halfHeight,
			0.72,
			1.06,
			0.12,
			debrisRandomBetween(-math.Pi, math.Pi),
		)
		roundness = debrisRandomBetween(0, 0.04)

	case shapeRoll < starLimit:
		// Concave stars vary from four to seven points (eight to fourteen corners).
		starPoints := debrisRandomInt(debrisStarPointsMin, debrisStarPointsMax)
		points, pointCount = debrisStarPolygon(
			starPoints,
			halfWidth,
			halfHeight,
			debrisRandomBetween(-math.Pi, math.Pi),
		)
		roundness = debrisRandomBetween(0, 0.025)

	case shapeRoll < glassLimit:
		// Sharp, many-cornered radial polygons resemble irregular glass chips.
		pointCount = debrisRandomInt(debrisGlassCornersMin, debrisGlassCornersMax)
		step := 2 * math.Pi / float64(pointCount)
		points = debrisRadialPolygon(
			pointCount,
			halfWidth,
			halfHeight,
			0.48,
			1.10,
			step*0.28,
			debrisRandomBetween(-math.Pi, math.Pi),
		)
		roundness = 0

	case shapeRoll < sliverLimit:
		// Rare narrow triangular or four-sided sliver.
		pointCount = 3 + debrisRNG.Intn(2)
		if debrisRNG.Intn(2) == 0 {
			halfWidth *= debrisRandomBetween(0.36, 0.58)
		} else {
			halfHeight *= debrisRandomBetween(0.36, 0.58)
		}
		points = debrisRadialPolygon(
			pointCount,
			halfWidth,
			halfHeight,
			0.68,
			1.05,
			0.16,
			debrisRandomBetween(-math.Pi, math.Pi),
		)
		roundness = debrisRandomBetween(0, 0.05)

	default:
		remainingChance := math.Max(0.0001, 1-sliverLimit)
		normalizedRoll := clampFloat((shapeRoll-sliverLimit)/remainingChance, 0, 1)
		switch {
		case normalizedRoll < 0.30:
			// Rough four-corner chunk, recognisably cut from a rectangular brick.
			pointCount = 4
			jitterX := halfWidth * 0.22
			jitterY := halfHeight * 0.24
			points = [32]float64{
				-halfWidth + debrisRandomBetween(-jitterX, jitterX), -halfHeight + debrisRandomBetween(-jitterY, jitterY),
				halfWidth + debrisRandomBetween(-jitterX, jitterX), -halfHeight + debrisRandomBetween(-jitterY, jitterY),
				halfWidth + debrisRandomBetween(-jitterX, jitterX), halfHeight + debrisRandomBetween(-jitterY, jitterY),
				-halfWidth + debrisRandomBetween(-jitterX, jitterX), halfHeight + debrisRandomBetween(-jitterY, jitterY),
			}
			roundness = debrisRandomBetween(0.02, 0.12)

		case normalizedRoll < 0.68:
			// Compact irregular chunks with a random four-to-seven-corner outline.
			pointCount = debrisRandomInt(4, 7)
			points = debrisRadialPolygon(
				pointCount,
				halfWidth,
				halfHeight,
				0.62,
				1.05,
				0.18,
				debrisRandomBetween(-math.Pi, math.Pi),
			)
			roundness = debrisRandomBetween(0.04, 0.18)

		default:
			// Eight-point rounded-rectangle silhouette retains some brick ancestry.
			pointCount = 8
			cutX := halfWidth * debrisRandomBetween(0.28, 0.45)
			cutY := halfHeight * debrisRandomBetween(0.28, 0.45)
			jitterX := halfWidth * 0.06
			jitterY := halfHeight * 0.07
			points = [32]float64{
				-halfWidth + cutX, -halfHeight + debrisRandomBetween(-jitterY, jitterY),
				halfWidth - cutX, -halfHeight + debrisRandomBetween(-jitterY, jitterY),
				halfWidth + debrisRandomBetween(-jitterX, jitterX), -halfHeight + cutY,
				halfWidth + debrisRandomBetween(-jitterX, jitterX), halfHeight - cutY,
				halfWidth - cutX, halfHeight + debrisRandomBetween(-jitterY, jitterY),
				-halfWidth + cutX, halfHeight + debrisRandomBetween(-jitterY, jitterY),
				-halfWidth + debrisRandomBetween(-jitterX, jitterX), halfHeight - cutY,
				-halfWidth + debrisRandomBetween(-jitterX, jitterX), -halfHeight + cutY,
			}
			roundness = debrisRandomBetween(0.58, 0.82)
		}
	}

	rotateDebrisPoints(&points, pointCount, debrisRandomBetween(-0.18, 0.18))
	for i := 0; i < pointCount; i++ {
		radius = math.Max(radius, math.Hypot(points[i*2], points[i*2+1]))
	}
	area = debrisPolygonArea(points, pointCount)
	return points, pointCount, math.Max(2, radius), math.Max(1, area), roundness, false
}

func spawnBrickDebris(br *brick, impactSpeed float64) {
	if !debrisIsEnabled() || br == nil || debrisMaxActivePieces <= 0 {
		return
	}
	normalizeDebrisSettings()

	pieceCount := debrisPiecesMin
	if debrisPiecesMax > debrisPiecesMin {
		pieceCount += debrisRNG.Intn(debrisPiecesMax - debrisPiecesMin + 1)
	}
	pieceCount = min(pieceCount, debrisMaxActivePieces)
	trimOldestDebrisFor(pieceCount)
	available := debrisMaxActivePieces - len(brickDebris)
	if available <= 0 {
		return
	}
	if pieceCount > available {
		pieceCount = available
	}

	fillColor := brickDebrisColor(br)
	impactSpeed = math.Max(0, impactSpeed)
	brickCenterX := br.x + br.w/2
	brickCenterY := br.y + br.h/2
	topCount := (pieceCount + 1) / 2
	bottomCount := pieceCount - topCount
	rowCount := 2
	if bottomCount == 0 {
		rowCount = 1
	}
	pieceIndex := 0

	spawnRow := func(row int, columns int) {
		if columns <= 0 {
			return
		}
		cellWidth := br.w / float64(columns)
		cellHeight := br.h / float64(rowCount)
		for column := 0; column < columns && pieceIndex < pieceCount; column++ {
			localCenterX := -br.w/2 + (float64(column)+0.5)*cellWidth
			localCenterY := -br.h/2 + (float64(row)+0.5)*cellHeight
			worldOffsetX, worldOffsetY := rotateVector(localCenterX, localCenterY, br.tiltRadians)

			points, pointCount, radius, shapeArea, roundness, circle := buildDebrisShape(cellWidth, cellHeight)
			if debrisSizeScale != 1 {
				for point := 0; point < pointCount; point++ {
					points[point*2] *= debrisSizeScale
					points[point*2+1] *= debrisSizeScale
				}
				radius *= debrisSizeScale
				shapeArea *= debrisSizeScale * debrisSizeScale
			}

			directionX, directionY := worldOffsetX, worldOffsetY
			// A small upward bias makes the break read as an explosion before gravity
			// takes over, while the radial component still follows the source brick.
			directionY -= br.h * 0.35
			length := math.Hypot(directionX, directionY)
			if length < 0.001 {
				angle := debrisRNG.Float64() * 2 * math.Pi
				directionX, directionY = math.Cos(angle), math.Sin(angle)
			} else {
				directionX /= length
				directionY /= length
			}
			areaFraction := shapeArea / math.Max(1, br.w*br.h)
			mass := clampFloat(areaFraction*1.8, 0.08, 0.60)
			referenceArea := cellWidth * cellHeight * 0.78
			referenceMass := clampFloat(referenceArea/math.Max(1, br.w*br.h)*1.8, 0.08, 0.60)
			sizeSpeedScale := clampFloat(math.Sqrt(referenceMass/mass), 0.80, 1.45)
			sizeSpinScale := clampFloat(math.Sqrt(referenceMass/mass), 0.85, 1.70)

			baseSpeed := debrisRandomBetween(debrisExplosionSpeedMin, debrisExplosionSpeedMax)
			speed := (baseSpeed + impactSpeed*debrisImpactSpeedFactor) * sizeSpeedScale
			tangentX, tangentY := -directionY, directionX
			tangentSpeed := debrisRandomBetween(-0.18*speed, 0.18*speed)
			angularSpeed := debrisRandomBetween(debrisAngularSpeedMin, debrisAngularSpeedMax) * sizeSpinScale
			if debrisRNG.Intn(2) == 0 {
				angularSpeed = -angularSpeed
			}

			initialAngle := br.tiltRadians + debrisRandomBetween(-0.10, 0.10)
			startOpacity := clampFloat(
				debrisStartOpacity+debrisRandomBetween(-debrisStartOpacityVariation, debrisStartOpacityVariation),
				0,
				1,
			)
			lifetimeVariation := debrisLifetimeVariationPercent / 100.0
			pieceLifetime := math.Max(
				0.05,
				debrisLifetime*debrisRandomBetween(1-lifetimeVariation, 1+lifetimeVariation),
			)
			fragment := debrisFragment{
				x:             brickCenterX + worldOffsetX,
				y:             brickCenterY + worldOffsetY,
				previousX:     brickCenterX + worldOffsetX,
				previousY:     brickCenterY + worldOffsetY,
				vx:            directionX*speed + tangentX*tangentSpeed,
				vy:            directionY*speed + tangentY*tangentSpeed,
				angle:         initialAngle,
				previousAngle: initialAngle,
				omega:         angularSpeed,
				radius:        radius,
				mass:          mass,
				age:           0,
				lifetime:      pieceLifetime,
				pointCount:    pointCount,
				points:        points,
				roundness:     roundness,
				circle:        circle,
				startOpacity:  startOpacity,
				fillColor:     fillColor,
			}
			debrisRenderIDCounter++
			if debrisRenderIDCounter == 0 {
				debrisRenderIDCounter++
			}
			fragment.renderID = debrisRenderIDCounter
			fragment.renderColorPalette = opaqueDebrisPalette(fillColor, palette[0])
			rebuildDebrisRenderPath(&fragment)
			clampDebrisSpeed(&fragment)
			brickDebris = append(brickDebris, fragment)
			pieceIndex++
		}
	}

	spawnRow(0, topCount)
	spawnRow(1, bottomCount)
}

func destroyBrick(br *brick, impactSpeed float64) bool {
	if br == nil || !br.alive || br.unbreakable {
		return false
	}
	spawnBrickDebris(br, impactSpeed)
	br.alive = false
	score++
	remainingBreakableBricks--
	if remainingBreakableBricks < 0 {
		remainingBreakableBricks = 0
	}
	bricksDirty = true
	maybePlayLastBrickSound()
	return true
}

func destroyAnyBrick(br *brick, impactSpeed float64) bool {
	if br == nil || !br.alive {
		return false
	}
	spawnBrickDebris(br, impactSpeed)
	br.alive = false
	score++
	if !br.unbreakable {
		remainingBreakableBricks--
		if remainingBreakableBricks < 0 {
			remainingBreakableBricks = 0
		}
	}
	bricksDirty = true
	maybePlayLastBrickSound()
	return true
}

func maybePlayLastBrickSound() {
	if remainingBreakableBricks == 0 && !lastBrickSoundPlayed {
		lastBrickSoundPlayed = true
		playLastBrick()
	}
}

func recordMeasuredBallSpin(b *Ball) {
	if b == nil {
		return
	}

	spin := math.Abs(b.omega)
	levelMeasuredMaxSpin = math.Max(levelMeasuredMaxSpin, spin)
	levelRunStats.fastestSpin = math.Max(levelRunStats.fastestSpin, spin)
}

func recordMeasuredBallSpeed(b *Ball) {
	if b == nil {
		return
	}

	speed := math.Hypot(b.vx, b.vy)
	levelMeasuredMaxSpeed = math.Max(levelMeasuredMaxSpeed, speed)
	levelRunStats.fastestSpeed = math.Max(levelRunStats.fastestSpeed, speed)
}

func recordIncomingCollisionSpeed(speed float64) {
	levelMeasuredMaxSpeed = math.Max(levelMeasuredMaxSpeed, speed)
	levelRunStats.fastestSpeed = math.Max(levelRunStats.fastestSpeed, speed)
}

func updateHUDCache() {
	level := currentLevelIndex + 1
	if hudLivesValue != lives {
		hudLivesValue = lives
		hudLivesText = "Lives: " + strconv.Itoa(lives)
	}
	if hudLevelValue != level {
		hudLevelValue = level
		hudLevelText = "Level: " + strconv.Itoa(level)
	}
	if hudScoreValue != score {
		hudScoreValue = score
		hudScoreText = "Score: " + strconv.Itoa(score)
	}
}

func toggleSound() {
	enableSounds = !enableSounds

	if enableSounds {
		unlockAudioFromGesture()
		showStatus("Sound on", 2.0)
	} else {
		stopAllMagicFeatureVoices()
		if audioInitialized && !audioMaster.IsUndefined() && !audioMaster.IsNull() {
			audioMaster.Get("gain").Call("cancelScheduledValues", audioCtx.Get("currentTime").Float())
			audioMaster.Get("gain").Set("value", 0)
		}
		showStatus("Sound off", 2.0)
	}
}

// ---- Audio (non-blocking, scheduled via Web Audio) ----

func supportedCompressedAudioFile(name string) bool {
	name = strings.TrimSpace(name)

	switch strings.ToLower(filepathExtension(name)) {
	case ".mp3", ".ogg", ".oga", ".opus", ".webm", ".m4a", ".aac", ".flac":
		return true
	default:
		// WAV is intentionally excluded. WAV masters belong outside runtime/ and
		// are never read or decoded by this loader.
		return false
	}
}

func filepathExtension(name string) string {
	index := strings.LastIndexByte(name, '.')
	if index < 0 {
		return ""
	}
	return name[index:]
}

func keepEmbeddedAudioCallback(callback js.Func) {
	embeddedAudioCallbacks = append(embeddedAudioCallbacks, callback)
}

func queueEmbeddedAudioDirectory(directory, label string, magic bool) {
	entries, err := fs.ReadDir(embeddedRuntimeAudio, directory)
	if err != nil {
		log("Could not read embedded " + label + " directory " + directory + ": " + err.Error())
		return
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !supportedCompressedAudioFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	for _, fileName := range names {
		sample := embeddedAudioSample{
			path:  directory + "/" + fileName,
			label: label,
			magic: magic,
		}

		if magic {
			sample.feature = magicFeatureNameFromFilename(fileName)
			if sample.feature == "" || embeddedAudioMagicFeatures[sample.feature] {
				continue
			}
			embeddedAudioMagicFeatures[sample.feature] = true
		}

		embeddedAudioQueue = append(embeddedAudioQueue, sample)
	}
}

func finishEmbeddedAudioSampleLoad(sample embeddedAudioSample, loaded bool) {
	if loaded {
		embeddedAudioLoadedCount++
	} else {
		embeddedAudioFailedCount++
	}
	embeddedAudioLoadActive = false
	scheduleNextEmbeddedAudioLoad()
}

func decodeEmbeddedAudio(sample embeddedAudioSample) {
	data, err := embeddedRuntimeAudio.ReadFile(sample.path)
	if err != nil {
		log("Could not read embedded " + sample.label + " sample " + sample.path + ": " + err.Error())
		finishEmbeddedAudioSampleLoad(sample, false)
		return
	}
	if len(data) == 0 {
		log("Embedded " + sample.label + " sample was empty: " + sample.path)
		finishEmbeddedAudioSampleLoad(sample, false)
		return
	}

	byteArray := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(byteArray, data)

	var decodeSuccessCallback js.Func
	var decodeFailureCallback js.Func
	finished := false
	finish := func(loaded bool) {
		if finished {
			return
		}
		finished = true
		finishEmbeddedAudioSampleLoad(sample, loaded)
	}

	decodeSuccessCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 || args[0].IsUndefined() || args[0].IsNull() {
			log("Decoded embedded " + sample.label + " sample was empty: " + sample.path)
			finish(false)
			return nil
		}

		if sample.magic {
			magicFeatureBuffers[sample.feature] = args[0]
		} else {
			brickHitBuffers = append(brickHitBuffers, args[0])
		}
		finish(true)
		return nil
	})

	decodeFailureCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		reason := "decode failed"
		if len(args) > 0 {
			reason = fmt.Sprint(args[0])
		}
		log("Could not decode embedded " + sample.label + " sample " + sample.path + ": " + reason)
		finish(false)
		return nil
	})

	keepEmbeddedAudioCallback(decodeSuccessCallback)
	keepEmbeddedAudioCallback(decodeFailureCallback)

	defer func() {
		if recovered := recover(); recovered != nil {
			log("Could not start decoding embedded " + sample.label + " sample " + sample.path + ": " + fmt.Sprint(recovered))
			finish(false)
		}
	}()
	audioCtx.Call("decodeAudioData", byteArray.Get("buffer"), decodeSuccessCallback, decodeFailureCallback)
}

func scheduleNextEmbeddedAudioLoad() {
	if embeddedAudioLoadActive {
		return
	}
	if len(embeddedAudioQueue) == 0 {
		if embeddedAudioPreloadStarted {
			log(fmt.Sprintf(
				"Embedded audio preload finished: %d loaded, %d failed; brick=%d magic=%d",
				embeddedAudioLoadedCount,
				embeddedAudioFailedCount,
				len(brickHitBuffers),
				len(magicFeatureBuffers),
			))
		}
		return
	}

	sample := embeddedAudioQueue[0]
	embeddedAudioQueue = embeddedAudioQueue[1:]
	embeddedAudioLoadActive = true

	timerCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if !audioInitialized || audioCtx.IsUndefined() || audioCtx.IsNull() {
			finishEmbeddedAudioSampleLoad(sample, false)
			return nil
		}
		decodeEmbeddedAudio(sample)
		return nil
	})
	keepEmbeddedAudioCallback(timerCallback)
	js.Global().Call("setTimeout", timerCallback, embeddedAudioDecodeYieldMS)
}

func startEmbeddedAudioPreload() {
	if embeddedAudioPreloadStarted || !audioInitialized {
		return
	}

	embeddedAudioPreloadStarted = true
	embeddedAudioLoadedCount = 0
	embeddedAudioFailedCount = 0
	embeddedAudioQueue = embeddedAudioQueue[:0]
	clear(embeddedAudioMagicFeatures)
	brickHitBuffers = brickHitBuffers[:0]
	brickHitLastIndex = -1
	clear(magicFeatureBuffers)

	queueEmbeddedAudioDirectory(brickHitSampleDirectory, "normal-brick hit", false)
	queueEmbeddedAudioDirectory(magicFeatureSampleDirectory, "magic-feature", true)
	sort.Slice(embeddedAudioQueue, func(i, j int) bool {
		return embeddedAudioQueue[i].path < embeddedAudioQueue[j].path
	})

	if len(embeddedAudioQueue) == 0 {
		log("No embedded compressed runtime audio found; using synthesized fallbacks")
		return
	}

	log(fmt.Sprintf("Embedded compressed audio queued: %d file(s)", len(embeddedAudioQueue)))
	scheduleNextEmbeddedAudioLoad()
}

func magicFeatureNameFromFilename(name string) string {
	name = strings.TrimSpace(name)
	if slash := strings.LastIndexByte(name, '/'); slash >= 0 {
		name = name[slash+1:]
	}
	extension := filepathExtension(name)
	return strings.ToLower(strings.TrimSpace(strings.TrimSuffix(name, extension)))
}

type audioRoomPreset struct {
	name       string
	defaultDry float64

	inputFilterType string
	inputFrequency  float64
	inputQ          float64

	outputFilterType string
	outputFrequency  float64
	outputQ          float64

	delay1, delay2, delay3       float64
	tapGain1, tapGain2, tapGain3 float64
	feedback                     float64
}

func normalizeAudioRoomName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer("_", "", "-", "", " ", "")
	name = replacer.Replace(name)
	switch name {
	case "", "none", "off", "dry", "disabled":
		return "none"
	case "smallroom", "roomsmall":
		return "smallroom"
	case "bigopenroom", "openroom", "bigroom", "roomopen":
		return "bigopenroom"
	case "smallhall", "hallsmall":
		return "smallhall"
	case "bighall", "largehall", "hallbig":
		return "bighall"
	case "underwater", "water":
		return "underwater"
	case "space":
		return "space"
	case "cave", "cavern":
		return "cave"
	case "matrix", "digital":
		return "matrix"
	default:
		return name
	}
}

func audioRoomPresetForName(name string) (audioRoomPreset, bool) {
	name = normalizeAudioRoomName(name)
	preset, ok := defaultAudioRoomPresets[name]
	return preset, ok
}

func smoothAudioParam(param js.Value, value, now float64) {
	if param.IsUndefined() || param.IsNull() {
		return
	}
	defer func() { _ = recover() }()
	current := param.Get("value").Float()
	param.Call("cancelScheduledValues", now)
	param.Call("setValueAtTime", current, now)
	param.Call("linearRampToValueAtTime", value, now+audioRoomTransitionSeconds)
}

func applyAudioRoomSettings() {
	if !audioInitialized || audioCtx.IsUndefined() || audioCtx.IsNull() {
		return
	}

	preset, found := audioRoomPresetForName(currentAudioRoom)
	if !found {
		log("Unknown audioRoom preset: " + currentAudioRoom + "; using none")
		preset, _ = audioRoomPresetForName("none")
		currentAudioRoom = defaultAudioRoom
	}

	dry := preset.defaultDry
	if currentAudioRoomDry >= 0 {
		dry = clampFloat(currentAudioRoomDry, 0, 1)
	}
	if preset.name == "none" {
		dry = 1
	}
	wet := 1 - dry

	now := audioCtx.Get("currentTime").Float()

	// Filter type changes are instantaneous; frequency, Q, delay, gain, and
	// feedback are ramped to avoid clicks while moving between levels.
	audioRoomInputFilter.Set("type", preset.inputFilterType)
	audioRoomOutputFilter.Set("type", preset.outputFilterType)
	audioRoomFeedbackFilter.Set("type", "lowpass")

	smoothAudioParam(audioRoomInputFilter.Get("frequency"), preset.inputFrequency, now)
	smoothAudioParam(audioRoomInputFilter.Get("Q"), preset.inputQ, now)
	smoothAudioParam(audioRoomOutputFilter.Get("frequency"), preset.outputFrequency, now)
	smoothAudioParam(audioRoomOutputFilter.Get("Q"), preset.outputQ, now)
	smoothAudioParam(audioRoomFeedbackFilter.Get("frequency"), math.Min(preset.outputFrequency, 5000), now)
	smoothAudioParam(audioRoomFeedbackFilter.Get("Q"), 0.45, now)

	smoothAudioParam(audioRoomDelay1.Get("delayTime"), preset.delay1, now)
	smoothAudioParam(audioRoomDelay2.Get("delayTime"), preset.delay2, now)
	smoothAudioParam(audioRoomDelay3.Get("delayTime"), preset.delay3, now)
	smoothAudioParam(audioRoomTapGain1.Get("gain"), preset.tapGain1, now)
	smoothAudioParam(audioRoomTapGain2.Get("gain"), preset.tapGain2, now)
	smoothAudioParam(audioRoomTapGain3.Get("gain"), preset.tapGain3, now)
	smoothAudioParam(audioRoomFeedbackGain.Get("gain"), clampFloat(preset.feedback, 0, 0.75), now)
	smoothAudioParam(audioRoomDryGain.Get("gain"), dry, now)
	smoothAudioParam(audioRoomWetGain.Get("gain"), wet, now)

	log(fmt.Sprintf("Audio room: %s (dry %.2f, wet %.2f)", preset.name, dry, wet))
}

func initAudio() {
	if audioInitialized || !enableSounds {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			audioInitialized = false
			audioCtx = js.Undefined()
			audioMaster = js.Undefined()
			audioBrickBus = js.Undefined()
			audioMagicBus = js.Undefined()
			audioSynthBus = js.Undefined()
			audioRoomInput = js.Undefined()
			audioRoomDryGain = js.Undefined()
			audioRoomWetGain = js.Undefined()
			audioRoomInputFilter = js.Undefined()
			audioRoomOutputFilter = js.Undefined()
			audioRoomDelay1 = js.Undefined()
			audioRoomDelay2 = js.Undefined()
			audioRoomDelay3 = js.Undefined()
			audioRoomTapGain1 = js.Undefined()
			audioRoomTapGain2 = js.Undefined()
			audioRoomTapGain3 = js.Undefined()
			audioRoomWetSum = js.Undefined()
			audioRoomFeedbackFilter = js.Undefined()
			audioRoomFeedbackGain = js.Undefined()
			js.Global().Get("console").Call("error", "Audio init panic:", fmt.Sprint(r))
		}
	}()

	window := js.Global().Get("window")
	ctor := window.Get("AudioContext")
	if ctor.IsUndefined() || ctor.IsNull() {
		ctor = window.Get("webkitAudioContext")
	}
	if ctor.IsUndefined() || ctor.IsNull() {
		log("Web Audio API is not supported by this browser")
		return
	}

	audioCtx = ctor.New()
	if audioCtx.IsUndefined() || audioCtx.IsNull() {
		log("Failed to create AudioContext")
		return
	}

	audioMaster = audioCtx.Call("createGain")
	audioBrickBus = audioCtx.Call("createGain")
	audioMagicBus = audioCtx.Call("createGain")
	audioSynthBus = audioCtx.Call("createGain")

	audioRoomInput = audioCtx.Call("createGain")
	audioRoomDryGain = audioCtx.Call("createGain")
	audioRoomWetGain = audioCtx.Call("createGain")
	audioRoomInputFilter = audioCtx.Call("createBiquadFilter")
	audioRoomOutputFilter = audioCtx.Call("createBiquadFilter")
	audioRoomDelay1 = audioCtx.Call("createDelay", 2.0)
	audioRoomDelay2 = audioCtx.Call("createDelay", 2.0)
	audioRoomDelay3 = audioCtx.Call("createDelay", 2.0)
	audioRoomTapGain1 = audioCtx.Call("createGain")
	audioRoomTapGain2 = audioCtx.Call("createGain")
	audioRoomTapGain3 = audioCtx.Call("createGain")
	audioRoomWetSum = audioCtx.Call("createGain")
	audioRoomFeedbackFilter = audioCtx.Call("createBiquadFilter")
	audioRoomFeedbackGain = audioCtx.Call("createGain")

	// Start fully dry and silent on the wet branch. The selected level preset
	// then fades in from this safe state, so an omitted audioRoom can never leak
	// the default Biquad/Delay settings during initial audio startup.
	audioRoomDryGain.Get("gain").Set("value", 1)
	audioRoomWetGain.Get("gain").Set("value", 0)
	audioRoomTapGain1.Get("gain").Set("value", 0)
	audioRoomTapGain2.Get("gain").Set("value", 0)
	audioRoomTapGain3.Get("gain").Set("value", 0)
	audioRoomDelay1.Get("delayTime").Set("value", 0.01)
	audioRoomDelay2.Get("delayTime").Set("value", 0.02)
	audioRoomDelay3.Get("delayTime").Set("value", 0.04)
	audioRoomFeedbackGain.Get("gain").Set("value", 0)

	audioBrickBus.Call("connect", audioRoomInput)
	audioMagicBus.Call("connect", audioRoomInput)
	audioSynthBus.Call("connect", audioRoomInput)

	// Completely dry branch.
	audioRoomInput.Call("connect", audioRoomDryGain)
	audioRoomDryGain.Call("connect", audioMaster)

	// Three reflection taps plus a damped feedback tail form the wet branch.
	audioRoomInput.Call("connect", audioRoomInputFilter)
	audioRoomInputFilter.Call("connect", audioRoomDelay1)
	audioRoomInputFilter.Call("connect", audioRoomDelay2)
	audioRoomInputFilter.Call("connect", audioRoomDelay3)
	audioRoomDelay1.Call("connect", audioRoomTapGain1)
	audioRoomDelay2.Call("connect", audioRoomTapGain2)
	audioRoomDelay3.Call("connect", audioRoomTapGain3)
	audioRoomTapGain1.Call("connect", audioRoomWetSum)
	audioRoomTapGain2.Call("connect", audioRoomWetSum)
	audioRoomTapGain3.Call("connect", audioRoomWetSum)
	audioRoomWetSum.Call("connect", audioRoomOutputFilter)
	audioRoomOutputFilter.Call("connect", audioRoomWetGain)
	audioRoomWetGain.Call("connect", audioMaster)

	// DelayNode in the cycle makes this legal Web Audio feedback routing.
	audioRoomDelay3.Call("connect", audioRoomFeedbackFilter)
	audioRoomFeedbackFilter.Call("connect", audioRoomFeedbackGain)
	audioRoomFeedbackGain.Call("connect", audioRoomDelay3)

	audioMaster.Call("connect", audioCtx.Get("destination"))

	audioInitialized = true
	setAudioMixerGains()
	applyAudioRoomSettings()
	startEmbeddedAudioPreload()
	log("Audio graph initialized; embedded compressed samples are preloading")
}

func setAudioMixerGains() {
	if !audioInitialized {
		return
	}

	if !audioMaster.IsUndefined() && !audioMaster.IsNull() {
		audioMaster.Get("gain").Set("value", audioMixerMaster)
	}
	if !audioBrickBus.IsUndefined() && !audioBrickBus.IsNull() {
		audioBrickBus.Get("gain").Set("value", audioMixerBricks)
	}
	if !audioMagicBus.IsUndefined() && !audioMagicBus.IsNull() {
		audioMagicBus.Get("gain").Set("value", audioMixerMagic)
	}
	if !audioSynthBus.IsUndefined() && !audioSynthBus.IsNull() {
		audioSynthBus.Get("gain").Set("value", audioMixerSynths)
	}
}

func ensureAudioRunning() {
	if !enableSounds ||
		!audioInitialized || audioCtx.IsUndefined() || audioCtx.IsNull() {
		return
	}
	setAudioMixerGains()
	state := audioCtx.Get("state")
	if state.Type() == js.TypeString && state.String() == "suspended" {
		audioCtx.Call("resume")
	}
}

func unlockAudioFromGesture() {

}

func scheduleAudioPreparation() {

}

func brickCenterX(br *brick) float64 {
	if br == nil {
		return canvasWidth / 2
	}
	return br.x + br.w/2
}

func audioPanFromX(hitX float64) float64 {
	if canvasWidth <= 0 {
		return 0
	}
	pan := hitX/canvasWidth*2 - 1
	return clampFloat(pan, -brickSoundPanLimit, brickSoundPanLimit)
}

// Connect an audio node through a StereoPannerNode and then into its mixer bus.
// Browsers without stereo-panner support fall back to a centered bus connection.
func connectAudioNodePanned(node, destination js.Value, hitX, when float64) {
	connected := false
	defer func() {
		if recover() != nil && !connected {
			defer func() { _ = recover() }()
			node.Call("connect", destination)
		}
	}()

	createPanner := audioCtx.Get("createStereoPanner")
	if createPanner.Type() != js.TypeFunction {
		node.Call("connect", destination)
		return
	}

	panner := audioCtx.Call("createStereoPanner")
	panner.Get("pan").Call("setValueAtTime", audioPanFromX(hitX), when)
	node.Call("connect", panner)
	panner.Call("connect", destination)
	connected = true
}

func scheduleTone(
	oscType string,
	startFreq float64,
	endFreq float64,
	duration float64,
	volume float64,
	delay float64,
) {
	scheduleTonePanned(oscType, startFreq, endFreq, duration, volume, delay, canvasWidth/2)
}

func scheduleTonePanned(
	oscType string,
	startFreq float64,
	endFreq float64,
	duration float64,
	volume float64,
	delay float64,
	hitX float64,
) {
	if enableSounds {
		nativeTone("tone", oscType, startFreq, endFreq, duration, volume, delay, hitX)
	}
}

func scheduleChord(freqs []float64, oscType string, duration, volume, delay float64) {
	scheduleChordPanned(freqs, oscType, duration, volume, delay, canvasWidth/2)
}

func scheduleChordPanned(freqs []float64, oscType string, duration, volume, delay, hitX float64) {
	if len(freqs) == 0 {
		return
	}
	perVoice := volume / float64(len(freqs))
	for i, freq := range freqs {
		detune := 1.0 + float64(i)*0.002
		scheduleTonePanned(oscType, freq*detune, freq*0.98, duration, perVoice, delay, hitX)
	}
}

func audioRand(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func varyFreq(freq float64, amount float64) float64 {
	return freq * audioRand(1-amount, 1+amount)
}

func playWallHit() {
	nativeTone("wall", "triangle", varyFreq(230, 0.10), varyFreq(175, 0.10), audioRand(0.038, 0.055), audioRand(0.11, 0.16), 0, canvasWidth/2)
}

func playPaddleHit() {
	nativeTone("paddle", "triangle", varyFreq(180, 0.08), varyFreq(320, 0.08), audioRand(0.065, 0.085), audioRand(0.16, 0.22), 0, canvasWidth/2)
	nativeTone("paddle", "sine", varyFreq(90, 0.06), varyFreq(70, 0.06), audioRand(0.08, 0.11), audioRand(0.05, 0.08), 0, canvasWidth/2)
}

// Ball-ball impacts use a short, light metallic/glass-like tick. Loudness and
// pitch follow only the closing speed along the collision normal, while the
// stereo position follows the midpoint between the two balls. This stays
// intentionally quieter than brick and paddle impacts in dense multi-ball play.
func playBallBallHit(hitX, impactSpeed float64) {
	if !defaultBallBallSoundEnabled || impactSpeed < defaultBallBallSoundMinSpeed {
		return
	}

	referenceSpeed := math.Max(physicsConfig.maxSpeed, defaultBallBallSoundMinSpeed+1)
	strength := clampFloat(
		(impactSpeed-defaultBallBallSoundMinSpeed)/(referenceSpeed-defaultBallBallSoundMinSpeed),
		0, 1,
	)
	strength = math.Sqrt(strength)
	volume := defaultBallBallSoundGainMin +
		(defaultBallBallSoundGainMax-defaultBallBallSoundGainMin)*strength
	pitch := 0.90 + 0.35*strength

	scheduleTonePanned(
		"sine",
		varyFreq(1080*pitch, 0.035),
		varyFreq(760*pitch, 0.035),
		audioRand(0.024, 0.036),
		volume,
		0,
		hitX,
	)
	scheduleTonePanned(
		"triangle",
		varyFreq(1880*pitch, 0.045),
		varyFreq(1320*pitch, 0.045),
		audioRand(0.016, 0.026),
		volume*0.30,
		0.001,
		hitX,
	)
}

func playSynthBrickBreakPitched(pitchScale, hitX float64) {
	if pitchScale <= 0 {
		pitchScale = 1
	}
	scheduleTonePanned("square", varyFreq(520*pitchScale, 0.12), varyFreq(360*pitchScale, 0.12), audioRand(0.045, 0.065), audioRand(0.07, 0.11), 0, hitX)
	scheduleTonePanned("triangle", varyFreq(760*pitchScale, 0.10), varyFreq(520*pitchScale, 0.10), audioRand(0.032, 0.05), audioRand(0.035, 0.06), audioRand(0.005, 0.012), hitX)
}

func playSynthBrickBreak(hitX float64) {
	playSynthBrickBreakPitched(1, hitX)
}

func chooseSampleIndex(buffers []js.Value, lastIndex int) int {
	count := len(buffers)
	if count <= 1 {
		return count - 1
	}
	if lastIndex < 0 || lastIndex >= count {
		return rand.Intn(count)
	}

	index := rand.Intn(count - 1)
	if index >= lastIndex {
		index++
	}
	return index
}

func playSampledBrickImpact(
	impactSpeed float64,
	hitX float64,
	buffers []js.Value,
	lastIndex *int,
	playbackRateMin float64,
	playbackRateMax float64,
	filterMinHz float64,
	filterMaxHz float64,
	gainMin float64,
	gainMax float64,
	label string,
	fallback func(),
) {
	if !enableSounds {
		return
	}

	samplePlaybackReady := audioInitialized &&
		!audioCtx.IsNull() && !audioCtx.IsUndefined() &&
		!audioMaster.IsNull() && !audioMaster.IsUndefined() &&
		!audioBrickBus.IsNull() && !audioBrickBus.IsUndefined() &&
		len(buffers) > 0

	// The budget is shared by normal and magic bricks. Extra bricks are still
	// destroyed; only excess sounds in a large burst are skipped.
	if brickHitStartsThisFrame >= brickHitMaxStartsPerFrame {
		return
	}
	if samplePlaybackReady && brickHitActiveVoices >= brickHitMaxActiveVoices {
		return
	}
	brickHitStartsThisFrame++

	if !samplePlaybackReady {
		fallback()
		return
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			log(label + " sample playback failed: " + fmt.Sprint(recovered))
			fallback()
		}
	}()

	ensureAudioRunning()

	index := chooseSampleIndex(buffers, *lastIndex)
	if index < 0 || index >= len(buffers) {
		fallback()
		return
	}
	*lastIndex = index

	referenceSpeed := math.Max(physicsConfig.maxSpeed, minimumCollisionSoundSpeed+1)
	strength := clampFloat(
		(impactSpeed-minimumCollisionSoundSpeed)/(referenceSpeed-minimumCollisionSoundSpeed),
		0, 1,
	)
	strength = math.Sqrt(strength)

	now := audioCtx.Get("currentTime").Float()
	source := audioCtx.Call("createBufferSource")
	filter := audioCtx.Call("createBiquadFilter")
	gain := audioCtx.Call("createGain")

	source.Set("buffer", buffers[index])
	source.Get("playbackRate").Call(
		"setValueAtTime",
		audioRand(playbackRateMin, playbackRateMax),
		now,
	)

	filter.Set("type", "lowpass")
	cutoff := filterMinHz + (filterMaxHz-filterMinHz)*strength
	cutoff *= audioRand(0.86, 1.14)
	nyquistMargin := audioCtx.Get("sampleRate").Float() * 0.45
	filter.Get("frequency").Call("setValueAtTime", clampFloat(cutoff, 800, nyquistMargin), now)
	filter.Get("Q").Call("setValueAtTime", audioRand(0.25, 0.85), now)

	volume := gainMin + (gainMax-gainMin)*strength
	volume *= audioRand(0.90, 1.08)
	gain.Get("gain").Call("setValueAtTime", volume, now)

	source.Call("connect", filter)
	filter.Call("connect", gain)
	connectAudioNodePanned(gain, audioBrickBus, hitX, now)

	// Install the completion callback before starting the source, then release
	// the Go callback as soon as this one-shot voice ends.
	var ended js.Func
	ended = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if brickHitActiveVoices > 0 {
			brickHitActiveVoices--
		}
		source.Set("onended", js.Null())
		ended.Release()
		return nil
	})
	source.Set("onended", ended)
	source.Call("start", now)
	brickHitActiveVoices++
}

func playBrickBreakPitched(impactSpeed, pitchScale, hitX float64) {
	if enableSounds {
		if brickHitStartsThisFrame >= brickHitMaxStartsPerFrame {
			return
		}
		brickHitStartsThisFrame++
		nativeSampleSound("brick", "", impactSpeed, pitchScale, hitX)
	}
}

func playBrickBreak(impactSpeed, hitX float64) {
	playBrickBreakPitched(impactSpeed, 1, hitX)
}

func retireMagicFeatureVoice(voice *magicFeatureVoice, now float64) {
	if voice == nil {
		return
	}
	if voice.active {
		voice.active = false
		if magicFeatureActiveVoices > 0 {
			magicFeatureActiveVoices--
		}
	}

	defer func() { _ = recover() }()
	gainParam := voice.gain.Get("gain")
	currentGain := math.Max(gainParam.Get("value").Float(), 0.0001)
	gainParam.Call("cancelScheduledValues", now)
	gainParam.Call("setValueAtTime", currentGain, now)
	gainParam.Call("linearRampToValueAtTime", 0.0001, now+magicFeatureRetriggerFade)
	voice.source.Call("stop", now+magicFeatureRetriggerFade+0.004)
}

func stopMagicFeatureVoice(feature string) {
	nativeSounds = append(nativeSounds, map[string]any{"kind": "stop-feature", "feature": strings.ToLower(strings.TrimSpace(feature))})
	feature = strings.ToLower(strings.TrimSpace(feature))
	voice := magicFeatureVoices[feature]
	if voice == nil {
		return
	}
	delete(magicFeatureVoices, feature)

	now := 0.0
	if audioInitialized && !audioCtx.IsNull() && !audioCtx.IsUndefined() {
		now = audioCtx.Get("currentTime").Float()
	}
	retireMagicFeatureVoice(voice, now)
}

func stopAllMagicFeatureVoices() {
	nativeSounds = append(nativeSounds, map[string]any{"kind": "stop-features"})
	for feature := range magicFeatureVoices {
		stopMagicFeatureVoice(feature)
	}
}

func playMagicFeature(feature string, impactSpeed, pitchScale, hitX float64) {
	if enableSounds {
		nativeSampleSound("magic", feature, impactSpeed, pitchScale, hitX)
	}
}

func playMagicPitched(impactSpeed, pitchScale, hitX float64) {
	if enableSounds {
		if brickHitStartsThisFrame >= brickHitMaxStartsPerFrame {
			return
		}
		brickHitStartsThisFrame++
		playSynthMagicPitched(pitchScale, hitX)
	}
}

func playMagic(impactSpeed, hitX float64) {
	playMagicPitched(impactSpeed, 1, hitX)
}

func playZapperDestroyedBrick(br *brick) {
	if br == nil {
		return
	}

	referenceSpeed := math.Max(physicsConfig.maxSpeed, minimumCollisionSoundSpeed+1)
	impactSpeed := minimumCollisionSoundSpeed +
		(referenceSpeed-minimumCollisionSoundSpeed)*zapperBrickStrength

	hitX := brickCenterX(br)
	if br.magic {
		playMagicPitched(impactSpeed, zapperBrickPitchScale, hitX)
	} else {
		playBrickBreakPitched(impactSpeed, zapperBrickPitchScale, hitX)
	}
}

func playUnbreakable(hitX float64) {
	// Low, rounded impact with small natural variation.
	base := varyFreq(82, 0.10)
	scheduleTonePanned("sine", base, varyFreq(52, 0.08), audioRand(0.09, 0.14), audioRand(0.12, 0.17), 0, hitX)
	scheduleTonePanned("triangle", varyFreq(46, 0.08), varyFreq(34, 0.08), audioRand(0.11, 0.16), audioRand(0.05, 0.08), 0.004, hitX)
}

func playSynthMagicPitched(pitchScale, hitX float64) {
	if pitchScale <= 0 {
		pitchScale = 1
	}
	scheduleTonePanned("sine", varyFreq(660*pitchScale, 0.06), varyFreq(990*pitchScale, 0.06), audioRand(0.09, 0.13), audioRand(0.08, 0.11), 0, hitX)
	scheduleTonePanned("triangle", varyFreq(990*pitchScale, 0.05), varyFreq(1480*pitchScale, 0.05), audioRand(0.11, 0.15), audioRand(0.055, 0.08), audioRand(0.05, 0.075), hitX)
}

func playSynthMagic(hitX float64) {
	playSynthMagicPitched(1, hitX)
}

func playPowerup(hitX float64) {
	root := varyFreq(330, 0.05)
	scheduleTonePanned("triangle", root, root*4/3, audioRand(0.09, 0.12), audioRand(0.08, 0.11), 0, hitX)
	scheduleTonePanned("triangle", root*4/3, root*2, audioRand(0.10, 0.14), audioRand(0.07, 0.10), audioRand(0.07, 0.10), hitX)
	scheduleChordPanned([]float64{root * 2, root * 2.5, root * 3}, "sine", audioRand(0.18, 0.24), audioRand(0.11, 0.15), audioRand(0.14, 0.19), hitX)
}

func playDie() {
	// Audible life-loss cue: short falling mid tone plus low body.
	scheduleTone(
		"triangle",
		varyFreq(420, 0.06),
		varyFreq(150, 0.06),
		audioRand(0.20, 0.26),
		audioRand(0.14, 0.18),
		0,
	)
	scheduleTone(
		"sine",
		varyFreq(150, 0.05),
		varyFreq(58, 0.05),
		audioRand(0.24, 0.32),
		audioRand(0.08, 0.11),
		0.025,
	)
}

func playGameOver() {
	scheduleTone("sawtooth", varyFreq(260, 0.03), varyFreq(120, 0.03), audioRand(0.24, 0.30), audioRand(0.08, 0.11), 0)
	scheduleTone("triangle", varyFreq(170, 0.03), varyFreq(62, 0.03), audioRand(0.36, 0.44), audioRand(0.08, 0.11), 0.20)
	scheduleChord([]float64{55, 65.4, 82.4}, "sine", audioRand(0.42, 0.52), audioRand(0.10, 0.14), 0.42)
}

func playLastBrick() {
	root := varyFreq(520, 0.04)
	scheduleTone("triangle", root, root*1.5, audioRand(0.07, 0.10), audioRand(0.08, 0.11), 0)
	scheduleTone("sine", root*1.5, root*2, audioRand(0.11, 0.15), audioRand(0.07, 0.10), 0.055)
	scheduleChord([]float64{root * 2, root * 2.5}, "sine", audioRand(0.16, 0.22), audioRand(0.10, 0.14), 0.14)
}

func playYouWin() {
	root := varyFreq(392, 0.02)
	scheduleTone("triangle", root, root*4/3, 0.14, 0.09, 0)
	scheduleTone("triangle", root*4/3, root*5/3, 0.15, 0.09, 0.13)
	scheduleTone("triangle", root*5/3, root*2, 0.17, 0.09, 0.27)
	scheduleChord([]float64{root, root * 5 / 4, root * 3 / 2, root * 2}, "sine", 0.48, 0.18, 0.43)
}

func playZapper(hitX float64) {
	scheduleTonePanned("sawtooth", varyFreq(920, 0.10), varyFreq(280, 0.10), 0.08, 0.08, 0, hitX)
	scheduleTonePanned("square", varyFreq(1450, 0.08), varyFreq(520, 0.08), 0.045, 0.045, 0.008, hitX)
}

func playLevelComplete() {
	root := varyFreq(440, 0.04)
	scheduleTone("triangle", root, root*1.5, audioRand(0.085, 0.115), audioRand(0.07, 0.09), 0.08)
	scheduleChord([]float64{root * 1.5, root * 1.875}, "sine", audioRand(0.14, 0.19), audioRand(0.09, 0.12), audioRand(0.16, 0.20))
}

// ---- Reset globals to defaults ----
func resetGlobals() {
	physicsConfig = defaultPhysicsSettings()
	canvasWidth = defaultCanvasWidth
	canvasHeight = defaultCanvasHeight
	paddleWidth = defaultPaddleWidth
	paddleHeight = defaultPaddleHeight
	ballRadius = defaultBallRadius
	brickRows = defaultBrickRows
	brickCols = defaultBrickCols
	brickWidth = defaultBrickWidth
	brickHeight = defaultBrickHeight
	brickPadding = defaultBrickPadding
	brickOffsetTop = defaultBrickOffsetTop
	paddleRadius = defaultPaddleRadius
	brickRadius = defaultBrickRadius
	unbreakableChance = defaultUnbreakableChance
	magicChance = defaultMagicChance
	powerUpDuration = defaultPowerUpDuration
	blackHoleStrength = defaultBlackHoleStrength
	blackHoleRange = defaultBlackHoleRange
	blackHolePathVerticalRange = defaultBlackHolePathVerticalRange
	blackHolePathCenterYOffset = defaultBlackHolePathCenterYOffset
	blackHolePathHorizontalCyclesMin = defaultBlackHolePathHorizontalCyclesMin
	blackHolePathHorizontalCyclesMax = defaultBlackHolePathHorizontalCyclesMax
	blackHolePathVerticalCyclesMin = defaultBlackHolePathVerticalCyclesMin
	blackHolePathVerticalCyclesMax = defaultBlackHolePathVerticalCyclesMax
	blackHolePathWobble = defaultBlackHolePathWobble
	magnetStrength = defaultMagnetStrength
	magnetRange = defaultMagnetRange
	influencerMultiplier = defaultInfluencerMultiplier
	zapperRange = defaultZapperRange
	startBallX = defaultStartBallX
	startBallY = defaultStartBallY
	startBallVx = defaultStartBallVx
	startBallVy = defaultStartBallVy
	enableLowGravity = defaultEnableLowGravity
	enablePassThrough = defaultEnablePassThrough
	enableNuke = defaultEnableNuke
	enableReverseGravity = defaultEnableReverseGravity
	enableDualBalls = defaultEnableDualBalls
	enableBlackHole = defaultEnableBlackHole
	enableMagnet = defaultEnableMagnet
	enableInfluencer = defaultEnableInfluencer
	enableZapper = defaultEnableZapper
	enableBreakUnbreakable = defaultEnableBreakUnbreakable
	enableBigPaddle = defaultEnableBigPaddle
	enableBrickFlip = defaultEnableBrickFlip
	currentAudioRoom = defaultAudioRoom
	currentAudioRoomDry = defaultAudioRoomDry
	debrisEnabled = defaultDebrisEnabled
	debrisShapeMode = defaultDebrisShapeMode
	debrisPiecesMin = defaultDebrisPiecesMin
	debrisPiecesMax = defaultDebrisPiecesMax
	debrisLifetime = defaultDebrisLifetime
	debrisLifetimeVariationPercent = defaultDebrisLifetimeVariationPercent
	debrisFadeDuration = defaultDebrisFadeDuration
	debrisStartOpacity = defaultDebrisStartOpacity
	debrisStartOpacityVariation = defaultDebrisStartOpacityVariation
	debrisFlashDuration = defaultDebrisFlashDuration
	debrisFlashOpacity = defaultDebrisFlashOpacity
	debrisImpactSpeedFactor = defaultDebrisImpactSpeedFactor
	debrisBallPieceChance = defaultDebrisBallPieceChance
	debrisTrianglePieceChance = defaultDebrisTrianglePieceChance
	debrisStarPieceChance = defaultDebrisStarPieceChance
	debrisStarPointsMin = defaultDebrisStarPointsMin
	debrisStarPointsMax = defaultDebrisStarPointsMax
	debrisGlassPieceChance = defaultDebrisGlassPieceChance
	debrisGlassCornersMin = defaultDebrisGlassCornersMin
	debrisGlassCornersMax = defaultDebrisGlassCornersMax
	debrisSliverPieceChance = defaultDebrisSliverPieceChance
	debrisMaxChunkAspectRatio = defaultDebrisMaxChunkAspectRatio
	debrisSizeScale = defaultDebrisSizeScale
	debrisBrickCollisionDelay = defaultDebrisBrickCollisionDelay
	debrisGravityScale = defaultDebrisGravityScale
	debrisAirDrag = defaultDebrisAirDrag
	debrisRestitution = defaultDebrisRestitution
	debrisFriction = defaultDebrisFriction
	debrisExplosionSpeedMin = defaultDebrisExplosionSpeedMin
	debrisExplosionSpeedMax = defaultDebrisExplosionSpeedMax
	debrisAngularSpeedMin = defaultDebrisAngularSpeedMin
	debrisAngularSpeedMax = defaultDebrisAngularSpeedMax
	debrisAngularDrag = defaultDebrisAngularDrag
	debrisAngularStopSpeed = defaultDebrisAngularStopSpeed
	debrisBallInfluence = defaultDebrisBallInfluence
	debrisFieldScale = defaultDebrisFieldScale
	debrisMagnetScale = defaultDebrisMagnetScale
	debrisMaxSpeed = defaultDebrisMaxSpeed
	debrisFrontLayerSpeed = defaultDebrisFrontLayerSpeed
	debrisMaxActivePieces = defaultDebrisMaxActivePieces
	debrisOffscreenMargin = defaultDebrisOffscreenMargin
	autoPaddleHitVariation = defaultAutoPaddleHitVariation
	autoPaddleNeedsNewHitOffset = true
	autoPaddleTargetBall = 0
	autoPaddleHitOffset = 0
	ballRescueEnabled = defaultBallRescueEnabled
	ballRescueFailureLimit = defaultBallRescueFailureLimit
	ballRescueMinProgress = defaultBallRescueMinProgress
	ballRescueCheckDuration = defaultBallRescueCheckDuration
	ballRescueUpperScreenFraction = defaultBallRescueUpperScreenFraction
	ballRescueClearance = defaultBallRescueClearance
	ballRescueLaunchSpeed = defaultBallRescueLaunchSpeed
	ballRescueMinRealtimePercent = defaultBallRescueMinRealtimePercent
	ballRescueMaxComputeLoad = defaultBallRescueMaxComputeLoad
	ballBelowFloorGracePixels = defaultBallBelowFloorGracePixels
	brickDebris = brickDebris[:0]
	debrisFieldTick = 0
	palette = append([]string(nil), defaultPalette...)
	magicColor = defaultMagicColor
	magicStrokeColor = defaultMagicStrokeColor
	unbreakableStrokeColor = defaultUnbreakableStrokeColor
	brickStrokeColor = defaultBrickStrokeColor
	levelStartTitle = "READY"
}

func parseLevelStartTitle(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "READY"
	}

	// Level files may use title="S L A Y". strconv.Unquote handles quoted
	// strings and escaped characters; unquoted values are accepted as-is.
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		}
	} else if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		value = value[1 : len(value)-1]
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "READY"
	}
	return value
}

// ---- Apply config overrides from map ----
func applyConfig(config map[string]string) {
	keys := make([]string, 0, len(config))
	for key := range config {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		val := config[key]
		if field, physicsKey := parsePhysicsFloatConfigKey(key); physicsKey {
			f, err := strconv.ParseFloat(val, 64)
			if err != nil || !validPhysicsFloatSetting(field, f) {
				log("Invalid physics level variable: " + key + "=" + val)
				continue
			}
			setPhysicsFloatSetting(&physicsConfig, field, f)
			continue
		}

		switch key {
		case "title":
			levelStartTitle = parseLevelStartTitle(val)
		case "audioRoom":
			currentAudioRoom = normalizeAudioRoomName(val)
		case "audioRoomDry":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1 {
				currentAudioRoomDry = f
			} else {
				log("audioRoomDry must be between 0 and 1")
			}
		case "backgroundColor":
			palette[0] = val
		case "paddleColor":
			palette[1] = val
		case "brickColor":
			palette[2] = val
		case "ballColor":
			palette[3] = val
		case "textColor":
			palette[4] = val
		case "spinColor":
			palette[5] = val
		case "unbreakableColor":
			palette[6] = val
		case "secondBallColor":
			palette[7] = val
		case "secondBallSpinColor":
			palette[8] = val
		case "magicColor":
			magicColor = val
		case "magicStrokeColor":
			magicStrokeColor = val
		case "unbreakableStrokeColor":
			unbreakableStrokeColor = val
		case "brickStrokeColor":
			brickStrokeColor = val
		case "orbitRequiredHits":
			if i, err := strconv.Atoi(val); err == nil && i > 0 {
				physicsConfig.orbitRequiredHits = i
			} else {
				log("orbitRequiredHits must be a positive integer")
			}
		case "drawBrickTilt":
			if b, err := strconv.ParseBool(val); err == nil {
				physicsConfig.drawBrickTilt = b
			} else {
				log("drawBrickTilt must be true or false")
			}
		case "cornerPhysics", "cornerPhysicsEnabled":
			if b, err := strconv.ParseBool(val); err == nil {
				physicsConfig.cornerPhysicsEnabled = b
			} else {
				log(key + " must be true or false")
			}
		case "cornerPhysicsAllBricks", "allBrickCorners":
			if b, err := strconv.ParseBool(val); err == nil {
				physicsConfig.cornerPhysicsAllBricks = b
			} else {
				log(key + " must be true or false")
			}
		case "powerUpDuration":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				powerUpDuration = f
			}
		case "magnet":
			if b, err := strconv.ParseBool(val); err == nil {
				levelMagnetActive = b
			}
		case "zapper":
			if b, err := strconv.ParseBool(val); err == nil {
				levelZapperActive = b
			}
		case "blackHoleStrength":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				blackHoleStrength = f
			} else {
				log("blackHoleStrength must be zero or greater")
			}
		case "blackHoleRange":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				blackHoleRange = f
			} else {
				log("blackHoleRange must be zero or greater")
			}
		case "blackHolePathVerticalRange":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				blackHolePathVerticalRange = f
			} else {
				log("blackHolePathVerticalRange must be zero or greater")
			}
		case "blackHolePathCenterYOffset":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				blackHolePathCenterYOffset = f
			}
		case "blackHolePathHorizontalCyclesMin":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				blackHolePathHorizontalCyclesMin = f
			} else {
				log("blackHolePathHorizontalCyclesMin must be greater than zero")
			}
		case "blackHolePathHorizontalCyclesMax":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				blackHolePathHorizontalCyclesMax = f
			} else {
				log("blackHolePathHorizontalCyclesMax must be greater than zero")
			}
		case "blackHolePathVerticalCyclesMin":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				blackHolePathVerticalCyclesMin = f
			} else {
				log("blackHolePathVerticalCyclesMin must be greater than zero")
			}
		case "blackHolePathVerticalCyclesMax":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				blackHolePathVerticalCyclesMax = f
			} else {
				log("blackHolePathVerticalCyclesMax must be greater than zero")
			}
		case "blackHolePathWobble":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 0.45 {
				blackHolePathWobble = f
			} else {
				log("blackHolePathWobble must be between 0 and 0.45")
			}
		case "magnetStrength":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				magnetStrength = f
			}
		case "magnetRange":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				magnetRange = f
			}
		case "influencerMultiplier":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				influencerMultiplier = f
			}
		case "zapperRange":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				zapperRange = f
			}
		case "brickOffsetTop":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				brickOffsetTop = f
			}
		case "brickWidth", "defaultBrickWidth":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				brickWidth = f
			}
		case "brickHeight", "defaultBrickHeight":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				brickHeight = f
			}
		case "brickPadding", "defaultBrickPadding":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				brickPadding = f
			}
		case "brickRows":
			if i, err := strconv.Atoi(val); err == nil {
				brickRows = i
			}
		case "brickCols":
			if i, err := strconv.Atoi(val); err == nil {
				brickCols = i
			}
		case "ballX":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				startBallX = f
			}
		case "ballY":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				startBallY = f
			}
		case "ballVx":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				startBallVx = f
			}
		case "ballVy":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				startBallVy = f
			}
		case "ballRadius":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				ballRadius = f
			}
		case "enableLowGravity":
			if b, err := strconv.ParseBool(val); err == nil {
				enableLowGravity = b
			}
		case "enablePassThrough":
			if b, err := strconv.ParseBool(val); err == nil {
				enablePassThrough = b
			}
		case "enableNuke":
			if b, err := strconv.ParseBool(val); err == nil {
				enableNuke = b
			}
		case "enableReverseGravity":
			if b, err := strconv.ParseBool(val); err == nil {
				enableReverseGravity = b
			}
		case "enableDualBalls":
			if b, err := strconv.ParseBool(val); err == nil {
				enableDualBalls = b
			}
		case "enableBlackHole":
			if b, err := strconv.ParseBool(val); err == nil {
				enableBlackHole = b
			}
		case "enableMagnet":
			if b, err := strconv.ParseBool(val); err == nil {
				enableMagnet = b
			}
		case "enableInfluencer":
			if b, err := strconv.ParseBool(val); err == nil {
				enableInfluencer = b
			}
		case "enableZapper":
			if b, err := strconv.ParseBool(val); err == nil {
				enableZapper = b
			}
		case "enableBreakUnbreakable":
			if b, err := strconv.ParseBool(val); err == nil {
				enableBreakUnbreakable = b
			}
		case "enableBigPaddle":
			if b, err := strconv.ParseBool(val); err == nil {
				enableBigPaddle = b
			}
		case "enableBrickFlip":
			if b, err := strconv.ParseBool(val); err == nil {
				enableBrickFlip = b
			}
		case "paddleWidth":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				paddleWidth = f
			}
		case "paddleHeight":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				paddleHeight = f
			}
		case "autoPaddleHitVariation":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 0.90 {
				autoPaddleHitVariation = f
				autoPaddleNeedsNewHitOffset = true
			} else {
				log("autoPaddleHitVariation must be from 0 to 0.90")
			}
		case "debris", "debrisEnabled":
			if b, err := strconv.ParseBool(val); err == nil {
				debrisEnabled = b
			} else {
				log(key + " must be true or false")
			}
		case "debrisShapeMode":
			if mode, ok := parseDebrisShapeMode(val); ok {
				debrisShapeMode = mode
				config[key] = mode
			} else {
				log("debrisShapeMode must be mixed, triangles, or circles")
			}
		case "debrisPiecesMin":
			if i, err := strconv.Atoi(val); err == nil && i >= 1 && i <= 32 {
				debrisPiecesMin = i
			} else {
				log("debrisPiecesMin must be from 1 to 32")
			}
		case "debrisPiecesMax":
			if i, err := strconv.Atoi(val); err == nil && i >= 1 && i <= 32 {
				debrisPiecesMax = i
			} else {
				log("debrisPiecesMax must be from 1 to 32")
			}
		case "debrisMaxActivePieces":
			if i, err := strconv.Atoi(val); err == nil && i >= 0 && i <= 2000 {
				debrisMaxActivePieces = i
			} else {
				log("debrisMaxActivePieces must be from 0 to 2000")
			}
		case "debrisLifetime":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				debrisLifetime = f
			} else {
				log("debrisLifetime must be greater than zero")
			}
		case "debrisLifetimeVariationPercent":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 95 {
				debrisLifetimeVariationPercent = f
			} else {
				log("debrisLifetimeVariationPercent must be from 0 to 95")
			}
		case "debrisFadeDuration":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisFadeDuration = f
			} else {
				log("debrisFadeDuration must be zero or greater")
			}
		case "debrisStartOpacity":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1 {
				debrisStartOpacity = f
			} else {
				log("debrisStartOpacity must be from 0 to 1")
			}
		case "debrisStartOpacityVariation":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1 {
				debrisStartOpacityVariation = f
			} else {
				log("debrisStartOpacityVariation must be from 0 to 1")
			}
		case "debrisFlashDuration":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisFlashDuration = f
			} else {
				log("debrisFlashDuration must be zero or greater")
			}
		case "debrisFlashOpacity":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1 {
				debrisFlashOpacity = f
			} else {
				log("debrisFlashOpacity must be from 0 to 1")
			}
		case "debrisImpactSpeedFactor":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisImpactSpeedFactor = f
			} else {
				log("debrisImpactSpeedFactor must be zero or greater")
			}
		case "debrisBallPieceChance":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1 {
				debrisBallPieceChance = f
			} else {
				log("debrisBallPieceChance must be from 0 to 1")
			}
		case "debrisTrianglePieceChance":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1 {
				debrisTrianglePieceChance = f
			} else {
				log("debrisTrianglePieceChance must be from 0 to 1")
			}
		case "debrisStarPieceChance":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1 {
				debrisStarPieceChance = f
			} else {
				log("debrisStarPieceChance must be from 0 to 1")
			}
		case "debrisStarPointsMin":
			if i, err := strconv.Atoi(val); err == nil && i >= 3 && i <= 8 {
				debrisStarPointsMin = i
			} else {
				log("debrisStarPointsMin must be from 3 to 8")
			}
		case "debrisStarPointsMax":
			if i, err := strconv.Atoi(val); err == nil && i >= 3 && i <= 8 {
				debrisStarPointsMax = i
			} else {
				log("debrisStarPointsMax must be from 3 to 8")
			}
		case "debrisGlassPieceChance":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1 {
				debrisGlassPieceChance = f
			} else {
				log("debrisGlassPieceChance must be from 0 to 1")
			}
		case "debrisGlassCornersMin":
			if i, err := strconv.Atoi(val); err == nil && i >= 7 && i <= 16 {
				debrisGlassCornersMin = i
			} else {
				log("debrisGlassCornersMin must be from 7 to 16")
			}
		case "debrisGlassCornersMax":
			if i, err := strconv.Atoi(val); err == nil && i >= 7 && i <= 16 {
				debrisGlassCornersMax = i
			} else {
				log("debrisGlassCornersMax must be from 7 to 16")
			}
		case "debrisSliverPieceChance":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1 {
				debrisSliverPieceChance = f
			} else {
				log("debrisSliverPieceChance must be from 0 to 1")
			}
		case "debrisMaxChunkAspectRatio":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 1 && f <= 4 {
				debrisMaxChunkAspectRatio = f
			} else {
				log("debrisMaxChunkAspectRatio must be from 1 to 4")
			}
		case "debrisSizeScale":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0.25 && f <= 3 {
				debrisSizeScale = f
			} else {
				log("debrisSizeScale must be from 0.25 to 3")
			}
		case "debrisBrickCollisionDelay":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisBrickCollisionDelay = f
			} else {
				log("debrisBrickCollisionDelay must be zero or greater")
			}
		case "debrisGravityScale":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisGravityScale = f
			}
		case "debrisAirDrag":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisAirDrag = f
			}
		case "debrisRestitution":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1.5 {
				debrisRestitution = f
			}
		case "debrisFriction":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 2 {
				debrisFriction = f
			}
		case "debrisExplosionSpeedMin":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisExplosionSpeedMin = f
			}
		case "debrisExplosionSpeedMax":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisExplosionSpeedMax = f
			}
		case "debrisAngularSpeedMin":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisAngularSpeedMin = f
			}
		case "debrisAngularSpeedMax":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisAngularSpeedMax = f
			}
		case "debrisAngularDrag":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisAngularDrag = f
			} else {
				log("debrisAngularDrag must be zero or greater")
			}
		case "debrisAngularStopSpeed":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisAngularStopSpeed = f
			} else {
				log("debrisAngularStopSpeed must be zero or greater")
			}
		case "debrisBallInfluence":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1 {
				debrisBallInfluence = f
			}
		case "debrisFieldScale":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisFieldScale = f
			}
		case "debrisMagnetScale":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisMagnetScale = f
			}
		case "debrisMaxSpeed":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisMaxSpeed = f
			}
		case "debrisFrontLayerSpeed":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisFrontLayerSpeed = f
			} else {
				log("debrisFrontLayerSpeed must be zero or greater")
			}
		case "debrisOffscreenMargin":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				debrisOffscreenMargin = f
			}
		case "ballRescue", "ballRescueEnabled":
			if b, err := strconv.ParseBool(val); err == nil {
				ballRescueEnabled = b
			} else {
				log(key + " must be true or false")
			}
		case "ballRescueFailureLimit":
			if i, err := strconv.Atoi(val); err == nil && i >= 1 && i <= 100 {
				ballRescueFailureLimit = i
			} else {
				log("ballRescueFailureLimit must be from 1 to 100")
			}
		case "ballRescueMinProgress":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				ballRescueMinProgress = f
			}
		case "ballRescueCheckDuration":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				ballRescueCheckDuration = f
			}
		case "ballRescueUpperScreenFraction":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0.10 && f <= 0.90 {
				ballRescueUpperScreenFraction = f
			} else {
				log("ballRescueUpperScreenFraction must be from 0.10 to 0.90")
			}
		case "ballRescueClearance":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				ballRescueClearance = f
			}
		case "ballRescueLaunchSpeed":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				ballRescueLaunchSpeed = f
			}
		case "ballRescueMinRealtimePercent":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 100 {
				ballRescueMinRealtimePercent = f
			}
		case "ballRescueMaxComputeLoad":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				ballRescueMaxComputeLoad = f
			}
		case "ballBelowFloorGracePixels", "ballFloorDeathMargin":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 5000 {
				ballBelowFloorGracePixels = f
			} else {
				log("ballBelowFloorGracePixels must be from 0 to 5000")
			}
		default:
			log("Unknown level variable: " + key)
		}
	}
	normalizeDebrisSettings()
	normalizeBallRescueSettings()
}

// ---- Physics ----

// resolveCollisionBallImproved uses the actual contact point and a solid-disk
// tangential impulse. It is used only while physics is selected.
func resolveCollisionBall(b *Ball, nx, ny, surfVx, surfVy, frictionScale float64) (float64, bool) {
	physics := &physicsConfig
	// nx,ny point from the surface toward the ball, so the contact point is
	// on the opposite side of the ball centre.
	cx := b.x - nx*b.r
	cy := b.y - ny*b.r

	contactVx := b.vx - b.omega*(cy-b.y)
	contactVy := b.vy + b.omega*(cx-b.x)
	relVx := contactVx - surfVx
	relVy := contactVy - surfVy

	vn := relVx*nx + relVy*ny
	tx := -ny
	ty := nx
	translationVt := (b.vx-surfVx)*tx + (b.vy-surfVy)*ty
	spinSurfaceSpeed := -b.omega * b.r
	vt := translationVt + spinSurfaceSpeed*physics.collisionSpinCoupling
	if vn >= 0 {
		return 0, false
	}

	vnNew := -physics.restitution * vn
	deltaVn := vnNew - vn

	// For a solid disk, no-slip correction is -vt/3 because tangential
	// impulse changes both translation and rotation. Different surfaces use
	// different friction scales: paddle strongest, bricks medium, walls weak.
	effectiveFriction := math.Max(
		physics.frictionCoeff*frictionScale,
		physics.minimumCollisionGrip,
	)
	maxFriction := effectiveFriction * math.Abs(deltaVn)
	desiredDeltaVt := -vt / 3.0
	deltaVt := clampFloat(desiredDeltaVt, -maxFriction, maxFriction)

	b.vx += deltaVn*nx + deltaVt*tx
	b.vy += deltaVn*ny + deltaVt*ty
	b.omega -= 2 * deltaVt / b.r
	b.omega = clampFloat(b.omega, -physics.maxSpin, physics.maxSpin)

	return deltaVt, true
}

// Resolve one collision while measuring how much the ball's spin changed the
// outgoing direction. Diagnostics are recorded only for Ball 1.
func resolveCollisionDebug(
	b *Ball,
	nx, ny, surfVx, surfVy float64,
	frictionScale float64,
	surface string,
	record bool,
) {
	incomingAngle := velocityAngleDegrees(b.vx, b.vy)
	incomingSpeed := math.Hypot(b.vx, b.vy)
	spinBefore := b.omega

	// A zero-spin clone gives a direct A/B measurement of the angle caused by
	// spin at this exact collision, using the same incoming velocity and normal.
	noSpin := *b
	noSpin.omega = 0
	noSpinImpulse := 0.0
	if impulse, collided := resolveCollisionBall(&noSpin, nx, ny, surfVx, surfVy, frictionScale); collided {
		noSpinImpulse = impulse
	}

	tangentialImpulse := 0.0
	if impulse, collided := resolveCollisionBall(b, nx, ny, surfVx, surfVy, frictionScale); collided {
		recordIncomingCollisionSpeed(incomingSpeed)
		tangentialImpulse = impulse
		recordMeasuredBallSpeed(b)
		recordMeasuredBallSpin(b)
	}

	if record {
		recordLastCollisionDebug(
			surface,
			incomingAngle,
			velocityAngleDegrees(b.vx, b.vy),
			velocityAngleDegrees(noSpin.vx, noSpin.vy),
			spinBefore,
			b.omega,
			tangentialImpulse,
			noSpinImpulse,
			b.r,
		)
	}
}

// ---- Build bricks from level layout (with auto-scaling) ----
func buildBricksFromLevel(lvl levelData, levelIndex int) {
	resetBrickFlipState()
	bricks = nil
	brickGrid = make(map[int][]int)
	brickGridCells = nil
	remainingBreakableBricks = 0
	lastBrickSoundPlayed = false
	if len(lvl.layout) == 0 {
		initBricksDefault(levelIndex)
		return
	}

	rows := len(lvl.layout)
	cols := 0
	for _, line := range lvl.layout {
		trimmed := strings.TrimRight(line, " ")
		if len(trimmed) > cols {
			cols = len(trimmed)
		}
	}
	if rows == 0 || cols == 0 {
		initBricksDefault(levelIndex)
		return
	}

	totalWidth := float64(cols)*(brickWidth+brickPadding) - brickPadding
	totalHeight := float64(rows)*(brickHeight+brickPadding) - brickPadding
	margin := 20.0
	scale := math.Min((canvasWidth-2*margin)/totalWidth, (canvasHeight-2*margin)/totalHeight)
	if scale > 1 {
		scale = 1
	}

	scaledWidth := brickWidth * scale
	scaledHeight := brickHeight * scale
	scaledPadding := brickPadding * scale
	scaledTotalWidth := float64(cols)*(scaledWidth+scaledPadding) - scaledPadding
	scaledTotalHeight := float64(rows)*(scaledHeight+scaledPadding) - scaledPadding
	gridOffsetLeft = (canvasWidth - scaledTotalWidth) / 2

	if brickOffsetTop >= 0 {
		gridOffsetTop = math.Max(brickOffsetTop, margin)
		if gridOffsetTop+scaledTotalHeight > canvasHeight-margin {
			gridOffsetTop = canvasHeight - margin - scaledTotalHeight
		}
	} else {
		gridOffsetTop = (canvasHeight - scaledTotalHeight) / 2
	}
	gridCellWidth = scaledWidth + scaledPadding
	gridCellHeight = scaledHeight + scaledPadding
	gridRows, gridCols = rows, cols
	brickGridCells = make([][]int, gridRows*gridCols)

	for row := 0; row < rows; row++ {
		line := strings.TrimRight(lvl.layout[row], " ")
		for col := 0; col < cols; col++ {
			ch := byte(' ')
			if col < len(line) {
				ch = line[col]
			}
			// X is a randomized block using the level's existing chances:
			// unbreakable first, then magic, otherwise a normal brick.
			if ch == 'X' {
				if rand.Float64() < unbreakableChance {
					ch = 'U'
				} else if rand.Float64() < magicChance {
					ch = 'M'
				} else {
					ch = '#'
				}
			}

			if ch == ' ' {
				continue
			}
			br := brick{
				x:           gridOffsetLeft + float64(col)*gridCellWidth,
				y:           gridOffsetTop + float64(row)*gridCellHeight,
				w:           scaledWidth,
				h:           scaledHeight,
				row:         row,
				col:         col,
				tiltRadians: brickMicroTiltRadians(levelIndex, row, col),
				alive:       true,
			}
			br.unbreakable = ch == 'U'
			br.magic = ch == 'M'
			bricks = append(bricks, br)
			registerBrick(len(bricks) - 1)
		}
	}
	initialBreakableBricks = remainingBreakableBricks
	bricksDirty = true
}

// ---- Fallback default bricks (also scaled) ----
func initBricksDefault(levelIndex int) {
	resetBrickFlipState()
	bricks = nil
	brickGrid = make(map[int][]int)
	brickGridCells = nil
	remainingBreakableBricks = 0
	lastBrickSoundPlayed = false

	// The built-in fallback behaves as if it has empty rows below its
	// visible bricks. The complete imaginary layout is still vertically
	// centered, which moves the actual brick grid upward.
	const extraEmptyRowsBelow = 5

	totalWidth := float64(brickCols)*(brickWidth+brickPadding) - brickPadding
	layoutRows := brickRows + extraEmptyRowsBelow
	totalHeight := float64(layoutRows)*(brickHeight+brickPadding) - brickPadding
	margin := 20.0
	scale := math.Min((canvasWidth-2*margin)/totalWidth, (canvasHeight-2*margin)/totalHeight)
	if scale > 1 {
		scale = 1
	}

	scaledWidth := brickWidth * scale
	scaledHeight := brickHeight * scale
	scaledPadding := brickPadding * scale
	scaledTotalWidth := float64(brickCols)*(scaledWidth+scaledPadding) - scaledPadding
	scaledTotalHeight := float64(layoutRows)*(scaledHeight+scaledPadding) - scaledPadding
	gridOffsetLeft = (canvasWidth - scaledTotalWidth) / 2
	if brickOffsetTop >= 0 {
		gridOffsetTop = math.Max(brickOffsetTop, margin)
		if gridOffsetTop+scaledTotalHeight > canvasHeight-margin {
			gridOffsetTop = canvasHeight - margin - scaledTotalHeight
		}
	} else {
		gridOffsetTop = (canvasHeight - scaledTotalHeight) / 2
	}
	gridCellWidth = scaledWidth + scaledPadding
	gridCellHeight = scaledHeight + scaledPadding
	gridRows, gridCols = brickRows, brickCols
	brickGridCells = make([][]int, gridRows*gridCols)

	rand.Seed(1)
	for row := 0; row < brickRows; row++ {
		for col := 0; col < brickCols; col++ {
			unbreakable := rand.Float64() < unbreakableChance
			magic := !unbreakable && rand.Float64() < magicChance
			bricks = append(bricks, brick{
				x: gridOffsetLeft + float64(col)*gridCellWidth,
				y: gridOffsetTop + float64(row)*gridCellHeight,
				w: scaledWidth, h: scaledHeight,
				row: row, col: col,
				tiltRadians: brickMicroTiltRadians(levelIndex, row, col),
				alive:       true, unbreakable: unbreakable, magic: magic,
			})
			registerBrick(len(bricks) - 1)
		}
	}
	initialBreakableBricks = remainingBreakableBricks
	bricksDirty = true
}

// ---- Nuke ----
func nukeBricks(hitBrick *brick, impactSpeed float64) {
	if hitBrick == nil {
		return
	}
	for i := range bricks {
		br := &bricks[i]
		if !br.alive {
			continue
		}
		if (br.row == hitBrick.row && abs(br.col-hitBrick.col) <= 2) ||
			(br.col == hitBrick.col && abs(br.row-hitBrick.row) <= 2) {
			destroyAnyBrick(br, impactSpeed)
		}
	}
}

// Destroy one random living unbreakable brick.
func breakRandomUnbreakable(impactSpeed float64) bool {
	var candidates []int
	for i := range bricks {
		if bricks[i].alive && bricks[i].unbreakable {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		return false
	}

	index := candidates[rand.Intn(len(candidates))]
	return destroyAnyBrick(&bricks[index], impactSpeed)
}

// Resize the paddle while preserving its center and keeping it on-screen.
func setPaddleSize(width, height float64) {
	center := paddle.x + paddle.w/2
	paddle.w = width
	paddle.h = height
	paddle.x = center - paddle.w/2
	paddle.x = math.Max(0, math.Min(paddle.x, canvasWidth-paddle.w))
	paddlePreviousX = paddle.x
}

// Gravity effects coexist. Black hole overrides both; otherwise reverse
// gravity has priority over low gravity.
func refreshCurrentGravity() {
	gravity := physicsConfig.gravity
	if blackHoleActive {
		currentGravity = 0
	} else if reverseGravityActive {
		currentGravity = -gravity
	} else if lowGravityActive {
		currentGravity = gravity / 3
	} else {
		currentGravity = gravity
	}
}

// randomBlackHolePathValue samples one path parameter without consuming the
// gameplay RNG used for power-up selection.
func randomBlackHolePathValue(minimum, maximum float64) float64 {
	if minimum > maximum {
		minimum, maximum = maximum, minimum
	}
	if maximum <= minimum {
		return minimum
	}
	return minimum + blackHoleRNG.Float64()*(maximum-minimum)
}

// updateBlackHolePathPosition evaluates a smooth randomised Lissajous-style
// curve. Phase-modulated harmonics keep the motion organic without sharp
// corners, while the configured ranges keep it near the middle of the screen.
func updateBlackHolePathPosition() {
	duration := math.Max(blackHolePathDuration, 0.001)
	progress := clampFloat(blackHolePathElapsed/duration, 0, 1)
	angle := 2 * math.Pi * progress

	wobble := clampFloat(blackHolePathWobble, 0, 0.45)
	xDetail := math.Sin(angle*blackHolePathHorizontalCycles*2.73 + blackHolePathWobblePhaseX)
	xCurve := math.Sin(angle*blackHolePathHorizontalCycles + blackHolePathPhaseX + wobble*xDetail)

	yDetail := math.Sin(angle*blackHolePathVerticalCycles*1.91 + blackHolePathWobblePhaseY)
	yCurve := math.Sin(angle*blackHolePathVerticalCycles + blackHolePathPhaseY + wobble*0.72*yDetail)

	const visualMargin = 48.0
	centerX := canvasWidth / 2
	centerY := clampFloat(canvasHeight/2+blackHolePathCenterYOffset, visualMargin, canvasHeight-visualMargin)
	horizontalRange := math.Min(math.Max(0, blackHoleRange), math.Max(0, centerX-visualMargin))
	verticalRange := math.Min(
		math.Max(0, blackHolePathVerticalRange),
		math.Max(0, math.Min(centerY-visualMargin, canvasHeight-visualMargin-centerY)),
	)

	blackHoleX = centerX + horizontalRange*xCurve
	blackHoleY = centerY + verticalRange*yCurve
}

func startBlackHolePath() {
	blackHolePathElapsed = 0
	blackHolePathDuration = math.Max(powerUpDuration, 0.001)
	blackHolePathHorizontalCycles = randomBlackHolePathValue(
		blackHolePathHorizontalCyclesMin,
		blackHolePathHorizontalCyclesMax,
	)
	blackHolePathVerticalCycles = randomBlackHolePathValue(
		blackHolePathVerticalCyclesMin,
		blackHolePathVerticalCyclesMax,
	)
	blackHolePathPhaseX = blackHoleRNG.Float64() * 2 * math.Pi
	blackHolePathPhaseY = blackHoleRNG.Float64() * 2 * math.Pi
	blackHolePathWobblePhaseX = blackHoleRNG.Float64() * 2 * math.Pi
	blackHolePathWobblePhaseY = blackHoleRNG.Float64() * 2 * math.Pi
	updateBlackHolePathPosition()
}

func resetBlackHolePath() {
	blackHolePathElapsed = 0
	blackHolePathDuration = 0
	blackHolePathHorizontalCycles = 0
	blackHolePathVerticalCycles = 0
	blackHolePathPhaseX = 0
	blackHolePathPhaseY = 0
	blackHolePathWobblePhaseX = 0
	blackHolePathWobblePhaseY = 0
}

func clearTimedPowerUps() {
	clearMagicBrickFlip()
	stopAllMagicFeatureVoices()
	lowGravityActive = false
	lowGravityTimer = 0
	passActive = false
	passTimer = 0
	reverseGravityActive = false
	reverseGravityTimer = 0
	magnetPowerActive = false
	magnetPowerTimer = 0
	zapperPowerActive = false
	zapperPowerTimer = 0
	bigPaddleActive = false
	bigPaddleTimer = 0
	bigPaddleExpiryPending = false
	blackHoleActive = false
	blackHoleTimer = 0
	resetBlackHolePath()
	influencerActive = false
	influencerTimer = 0
	extraBalls = extraBalls[:0]
	resetAllBallZapperStates()
	setPaddleSize(paddleWidth, paddleHeight)
	refreshCurrentGravity()
}

// Activate a magic-brick feature and play the compressed sample whose basename
// matches that feature. Missing/failed samples use the established generator.
func activatePowerUpWithBrick(hitBrick *brick, impactSpeed float64) bool {

	var available []int
	if enableLowGravity {
		available = append(available, POWER_LOW_GRAVITY)
	}
	if enablePassThrough {
		available = append(available, POWER_PASS)
	}
	if enableNuke {
		available = append(available, POWER_NUKE)
	}
	if enableReverseGravity {
		available = append(available, POWER_REVERSE_GRAVITY)
	}
	if enableDualBalls {
		available = append(available, POWER_DUAL_BALLS)
	}
	if enableBlackHole {
		available = append(available, POWER_BLACKHOLE)
	}
	if enableMagnet {
		available = append(available, POWER_MAGNET)
	}
	if enableInfluencer {
		available = append(available, POWER_INFLUENCER)
	}
	if enableZapper {
		available = append(available, POWER_ZAPPER)
	}
	if enableBreakUnbreakable {
		for i := range bricks {
			if bricks[i].alive && bricks[i].unbreakable {
				available = append(available, POWER_BREAK_UNBREAKABLE)
				break
			}
		}
	}
	if enableBigPaddle {
		available = append(available, POWER_BIG_PADDLE)
	}
	if enableBrickFlip {
		available = append(available, POWER_BRICK_FLIP)
	}

	if len(available) == 0 {
		return false
	}

	p := available[rand.Intn(len(available))]
	feature := ""
	activated := true

	switch p {
	case POWER_LOW_GRAVITY:
		feature = "lowgravity"
		lowGravityActive = true
		lowGravityTimer = powerUpDuration
		refreshCurrentGravity()
		showStatus("Low Gravity!", powerUpDuration)
	case POWER_PASS:
		feature = "passthrough"
		passActive = true
		passTimer = powerUpDuration
		showStatus("Pass Through!", powerUpDuration)
	case POWER_NUKE:
		feature = "nuke"
		nukeBricks(hitBrick, impactSpeed)
		showStatus("Nuke!", 2.0)
	case POWER_REVERSE_GRAVITY:
		feature = "reversegravity"
		reverseGravityActive = true
		reverseGravityTimer = powerUpDuration
		refreshCurrentGravity()
		showStatus("Reverse Gravity!", powerUpDuration)
	case POWER_DUAL_BALLS:
		if spawnAdditionalBall() {
			feature = "dualballs"
			showStatus("Multi Ball! "+strconv.Itoa(activeBallCount())+"/"+strconv.Itoa(maxConfiguredBalls()), 2.0)
		} else {
			feature = "speedboost"
			boostAllActiveBalls(defaultMultiBallSpeedBoost)
			showStatus("Speed Boost!", 2.0)
		}
	case POWER_BLACKHOLE:
		feature = "blackhole"
		blackHoleActive = true
		blackHoleTimer = powerUpDuration
		currentGravity = 0

		// Generate a fresh smooth curve around the slightly raised screen centre.
		// Path randomness is isolated from gameplay randomness.
		startBlackHolePath()

		showStatus("Black Hole!", powerUpDuration)
		refreshCurrentGravity()
	case POWER_MAGNET:
		feature = "magnet"
		magnetPowerActive = true
		magnetPowerTimer = powerUpDuration
		showStatus("Magnets!", powerUpDuration)
	case POWER_INFLUENCER:
		feature = "influencer"
		influencerActive = true
		influencerTimer = powerUpDuration
		showStatus("Influencer!", powerUpDuration)
	case POWER_ZAPPER:
		feature = "zapper"
		zapperPowerActive = true
		zapperPowerTimer = powerUpDuration
		resetAllBallZapperStates()
		showStatus("Zapper!", powerUpDuration)
	case POWER_BREAK_UNBREAKABLE:
		feature = "breakunbreakable"
		if breakRandomUnbreakable(impactSpeed) {
			showStatus("Unbreakable destroyed!", 2.0)
		} else {
			activated = false
		}
	case POWER_BIG_PADDLE:
		feature = "bigpaddle"
		bigPaddleActive = true
		bigPaddleTimer = powerUpDuration
		bigPaddleExpiryPending = false
		setPaddleSize(paddleWidth*2, paddleHeight)
		showStatus("Big Paddle!", powerUpDuration)
	case POWER_BRICK_FLIP:
		feature = "brickflip"
		pendingMagicBrickFlips++
		showStatus("Brick Flip!", 2.0)
	}

	if !activated || feature == "" {
		return false
	}
	playMagicFeature(feature, impactSpeed, 1, brickCenterX(hitBrick))
	return true
}

// ---- Load levels ----
func loadLevels() {
	levelIndex := 1
	for {
		// Cache-bust each request while developing so edited level files
		// are always fetched fresh instead of reused from browser cache.
		timestamp := int64(js.Global().Get("Date").Call("now").Float())
		url := "levels/level" + strconv.Itoa(levelIndex) + ".txt?t=" + strconv.FormatInt(timestamp, 10)

		xhr := js.Global().Get("XMLHttpRequest").New()
		xhr.Call("open", "GET", url, false)
		xhr.Call("setRequestHeader", "Cache-Control", "no-cache")
		xhr.Call("setRequestHeader", "Pragma", "no-cache")
		xhr.Call("send")
		status := xhr.Get("status").Int()
		if status == 404 {
			break
		}
		if status != 200 {
			log("Error loading level " + strconv.Itoa(levelIndex) + ": status " + strconv.Itoa(status))
			break
		}
		text := xhr.Get("responseText").String()
		lines := strings.Split(text, "\n")
		var configLines []string
		var layoutLines []string
		inLayout := false
		for _, rawLine := range lines {
			line := strings.TrimSpace(rawLine)
			if line == "---" {
				inLayout = true
				continue
			}
			if !inLayout {
				if line != "" && !strings.HasPrefix(line, "#") {
					configLines = append(configLines, line)
				}
			} else {
				layoutLine := strings.TrimRight(rawLine, "\r\n ")
				layoutLines = append(layoutLines, layoutLine)
			}
		}
		configMap := make(map[string]string)
		for _, cfg := range configLines {
			parts := strings.SplitN(cfg, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			configMap[key] = val
		}
		levels = append(levels, levelData{layout: layoutLines, config: configMap})
		levelIndex++
	}
	if len(levels) == 0 {
		log("No level files found; using default layout")
		defaultLayout := []string{
			"##################",
			"#  #  #  #  #  #",
			"#  #  #  #  #  #",
			"##################",
		}
		levels = append(levels, levelData{layout: defaultLayout, config: make(map[string]string)})
	}
}

// ---- Remember progression in browser storage ----
const (
	savedLevelKey           = "breakout.currentLevel"
	highestUnlockedLevelKey = "breakout.highestUnlockedLevel" // Legacy key.
	unlockFrontierKey       = "breakout.unlockFrontier"
)

func readStoredInt(key string) (int, bool) {
	value := 0
	found := false
	func() {
		defer func() { _ = recover() }()
		storage := js.Global().Get("localStorage")
		if storage.IsUndefined() || storage.IsNull() {
			return
		}
		stored := storage.Call("getItem", key)
		if stored.IsUndefined() || stored.IsNull() {
			return
		}
		parsed, err := strconv.Atoi(stored.String())
		if err != nil {
			return
		}
		value = parsed
		found = true
	}()
	return value, found
}

func writeStoredInt(key string, value int) {
	defer func() {
		if r := recover(); r != nil {
			js.Global().Get("console").Call("warn", "Could not save progression:", fmt.Sprint(r))
		}
	}()
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return
	}
	storage.Call("setItem", key, strconv.Itoa(value))
}

func levelBestStorageKey(levelIndex int, metric string) string {
	return speedRecordKey("breakout.best.v1.level."+strconv.Itoa(levelIndex+1)+"."+metric, levelRunStats.speedPercent)
}

func fullGameBestStorageKey(levelCount, speedPercent int) string {
	return speedRecordKey("breakout.best.v1.fullGame."+strconv.Itoa(levelCount)+".timeMs", speedPercent)
}

func loadFullGameBestTime(levelCount, speedPercent int) (int, bool) {
	if levelCount <= 0 {
		return 0, false
	}
	return readStoredInt(fullGameBestStorageKey(levelCount, speedPercent))
}

func beginFullGameRun() {
	fullGameRun = fullGameRunStateData{
		speedPercent:   gameSpeedPercent,
		active:         true,
		recordEligible: !autoPaddleEnabled,
		nextLevel:      0,
		levelCount:     len(levels),
	}
	fullGameCompletion = fullGameCompletionData{}
}

func cancelFullGameRun() {
	fullGameRun.active = false
	fullGameRun.recordEligible = false
}

// recordFullGameLevelCompletion accepts only the exact next level in a run that
// began at level 1. Manual navigation, game-over retries from later levels, or
// any other break in sequence therefore cannot produce a full-game record.
func recordFullGameLevelCompletion() {
	if !fullGameRun.active {
		return
	}
	if fullGameRun.levelCount != len(levels) ||
		currentLevelIndex != fullGameRun.nextLevel ||
		currentLevelIndex < 0 || currentLevelIndex >= len(levels) {
		cancelFullGameRun()
		return
	}

	fullGameRun.elapsedSeconds += levelCompletedStats.elapsedSeconds
	if !levelRunRecordEligible {
		fullGameRun.recordEligible = false
	}
	fullGameRun.nextLevel++

	if fullGameRun.nextLevel < fullGameRun.levelCount {
		return
	}

	result := fullGameCompletionData{
		speedPercent:   fullGameRun.speedPercent,
		mixedSpeed:     fullGameRun.mixedSpeed,
		valid:          true,
		recordEligible: fullGameRun.recordEligible,
		timeMs:         int(math.Round(fullGameRun.elapsedSeconds * 1000)),
		levelCount:     fullGameRun.levelCount,
	}
	if !result.mixedSpeed {
		result.previousBestTimeMs, result.previousBestFound = loadFullGameBestTime(result.levelCount, result.speedPercent)
	}
	result.bestTimeMs = result.previousBestTimeMs
	result.bestFound = result.previousBestFound
	if result.recordEligible && (!result.previousBestFound || result.timeMs < result.previousBestTimeMs) {
		writeStoredInt(fullGameBestStorageKey(result.levelCount, result.speedPercent), result.timeMs)
		result.bestTimeMs = result.timeMs
		result.bestFound = true
		result.newRecord = true
	}
	fullGameCompletion = result
	fullGameRun.active = false
}

func loadLevelBestStats(levelIndex int) levelBestStatsData {
	var best levelBestStatsData
	best.timeMs, best.timeFound = readStoredInt(levelBestStorageKey(levelIndex, "timeMs"))
	best.fastestSpeed, best.speedFound = readStoredInt(levelBestStorageKey(levelIndex, "fastestSpeed"))
	best.fastestSpin, best.spinFound = readStoredInt(levelBestStorageKey(levelIndex, "fastestSpin"))
	best.bestChain, best.chainFound = readStoredInt(levelBestStorageKey(levelIndex, "bestChain"))
	best.ballsLost, best.ballsLostFound = readStoredInt(levelBestStorageKey(levelIndex, "ballsLost"))
	return best
}

func saveBestIfLower(levelIndex int, metric string, value int, oldValue int, found bool) (int, bool, bool) {
	if !found || value < oldValue {
		writeStoredInt(levelBestStorageKey(levelIndex, metric), value)
		return value, true, true
	}
	return oldValue, found, false
}

func saveBestIfHigher(levelIndex int, metric string, value int, oldValue int, found bool) (int, bool, bool) {
	if !found || value > oldValue {
		writeStoredInt(levelBestStorageKey(levelIndex, metric), value)
		return value, true, true
	}
	return oldValue, found, false
}

func resetLevelRunStats(levelIndex int) {
	levelRunStats = levelRunStatsData{speedPercent: gameSpeedPercent}
	levelCompletedStats = levelRunStatsData{}
	levelCompletedStatsValid = false
	levelCompletionNewRecords = levelRecordFlags{}
	levelCurrentBests = loadLevelBestStats(levelIndex)
	levelPreviousBests = levelCurrentBests
	levelRunRecordEligible = !autoPaddleEnabled
}

func markLevelRunAssisted() {
	if levelAdvancePending || gameOver {
		return
	}
	levelRunRecordEligible = false
	if fullGameRun.active {
		fullGameRun.recordEligible = false
	}
}

func recordLevelCompletionStats() {
	// Freeze the measured peaks into the result snapshot before saving records.
	levelRunStats.fastestSpeed = math.Max(levelRunStats.fastestSpeed, levelMeasuredMaxSpeed)
	levelRunStats.fastestSpin = math.Max(levelRunStats.fastestSpin, levelMeasuredMaxSpin)
	levelCompletedStats = levelRunStats
	levelCompletedStatsValid = true
	levelCompletionNewRecords = levelRecordFlags{}
	// Preserve the personal bests exactly as they were before this clear. The
	// results table uses this snapshot for the WAS column even when a new record
	// immediately replaces the stored BEST value below.
	levelPreviousBests = levelCurrentBests
	if !levelRunRecordEligible {
		return
	}

	timeMs := int(math.Round(levelCompletedStats.elapsedSeconds * 1000))
	speed := int(math.Round(levelCompletedStats.fastestSpeed))
	spin := int(math.Round(levelCompletedStats.fastestSpin))
	levelCurrentBests.timeMs, levelCurrentBests.timeFound, levelCompletionNewRecords.time =
		saveBestIfLower(currentLevelIndex, "timeMs", timeMs, levelCurrentBests.timeMs, levelCurrentBests.timeFound)
	levelCurrentBests.fastestSpeed, levelCurrentBests.speedFound, levelCompletionNewRecords.speed =
		saveBestIfHigher(currentLevelIndex, "fastestSpeed", speed, levelCurrentBests.fastestSpeed, levelCurrentBests.speedFound)
	levelCurrentBests.fastestSpin, levelCurrentBests.spinFound, levelCompletionNewRecords.spin =
		saveBestIfHigher(currentLevelIndex, "fastestSpin", spin, levelCurrentBests.fastestSpin, levelCurrentBests.spinFound)
	levelCurrentBests.bestChain, levelCurrentBests.chainFound, levelCompletionNewRecords.chain =
		saveBestIfHigher(currentLevelIndex, "bestChain", levelCompletedStats.bestChain, levelCurrentBests.bestChain, levelCurrentBests.chainFound)
	levelCurrentBests.ballsLost, levelCurrentBests.ballsLostFound, levelCompletionNewRecords.ballsLost =
		saveBestIfLower(currentLevelIndex, "ballsLost", levelCompletedStats.ballsLost, levelCurrentBests.ballsLost, levelCurrentBests.ballsLostFound)
}

func formatLevelTimeSeconds(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	return formatLevelTimeMs(int(math.Round(seconds * 1000)))
}

func formatLevelTimeMs(milliseconds int) string {
	if milliseconds < 0 {
		milliseconds = 0
	}
	minutes := milliseconds / 60000
	seconds := (milliseconds / 1000) % 60
	hundredths := (milliseconds % 1000) / 10
	return fmt.Sprintf("%02d:%02d.%02d", minutes, seconds, hundredths)
}

func durationUnit(value int, singular string) string {
	if value == 1 {
		return singular
	}
	return singular + "s"
}

func formatFullGameDurationMs(milliseconds int) string {
	if milliseconds < 0 {
		milliseconds = 0
	}
	totalSeconds := int(math.Round(float64(milliseconds) / 1000.0))
	hours := totalSeconds / 3600
	minutes := (totalSeconds / 60) % 60
	seconds := totalSeconds % 60
	if hours > 0 {
		return fmt.Sprintf("%d %s %d %s and %d %s",
			hours, durationUnit(hours, "hour"),
			minutes, durationUnit(minutes, "minute"),
			seconds, durationUnit(seconds, "second"))
	}
	if minutes > 0 {
		return fmt.Sprintf("%d %s and %d %s",
			minutes, durationUnit(minutes, "minute"),
			seconds, durationUnit(seconds, "second"))
	}
	return fmt.Sprintf("%d %s", seconds, durationUnit(seconds, "second"))
}

func formatBestInt(value int, found bool) string {
	if !found {
		return "--"
	}
	return strconv.Itoa(value)
}

func formatBestTime(best levelBestStatsData) string {
	if !best.timeFound {
		return "--"
	}
	return formatLevelTimeMs(best.timeMs)
}

func saveCurrentLevel() {
	writeStoredInt(savedLevelKey, currentLevelIndex)
}

func permanentUnlockedLimit() int {
	if len(levels) == 0 {
		return 0
	}
	limit := unlockFrontier
	if limit < 0 {
		limit = 0
	}
	if limit >= len(levels) {
		limit = len(levels) - 1
	}
	return limit
}

func activeUnlockedLimit() int {
	if len(levels) == 0 {
		return 0
	}
	if devAllLevelsUnlocked {
		return len(levels) - 1
	}
	return permanentUnlockedLimit()
}

func refreshHighestUnlockedLevel() {
	highestUnlockedLevel = permanentUnlockedLimit()
}

func saveUnlockFrontier() {
	writeStoredInt(unlockFrontierKey, unlockFrontier)
	// Keep the old key updated so older builds still see sensible progress.
	writeStoredInt(highestUnlockedLevelKey, permanentUnlockedLimit())
}

func loadUnlockFrontier() int {
	if len(levels) == 0 {
		return 0
	}

	if frontier, ok := readStoredInt(unlockFrontierKey); ok {
		if frontier < 0 {
			return 0
		}
		return frontier
	}

	legacyHighest, legacyFound := readStoredInt(highestUnlockedLevelKey)
	if !legacyFound {
		return 0
	}
	if legacyHighest < 0 {
		legacyHighest = 0
	}

	frontier := legacyHighest

	// Legacy builds could not store one-past-the-final-level. When more levels
	// were later added, both stored values still pointed at the old final level,
	// forcing it to be replayed. Treat that exact legacy state as completed.
	if saved, ok := readStoredInt(savedLevelKey); ok &&
		saved == legacyHighest && legacyHighest < len(levels)-1 {
		frontier = legacyHighest + 1
		log("Migrated legacy final-level progress to level " + strconv.Itoa(frontier+1))
	}

	unlockFrontier = frontier
	saveUnlockFrontier()
	return frontier
}

func unlockNextLevel() {
	// Visiting a locked level through the temporary developer toggle must not
	// permanently skip normal progression.
	if devAllLevelsUnlocked && currentLevelIndex > permanentUnlockedLimit() {
		return
	}

	next := currentLevelIndex + 1
	if next > unlockFrontier {
		unlockFrontier = next
		refreshHighestUnlockedLevel()
		saveUnlockFrontier()
		if next < len(levels) {
			showStatus("Level "+strconv.Itoa(next+1)+" unlocked!", 3.0)
		}
	}
}

func loadSavedLevel() int {
	if len(levels) == 0 {
		return 0
	}

	savedIndex, ok := readStoredInt(savedLevelKey)
	if !ok {
		savedIndex = 0
	}
	if savedIndex < 0 {
		savedIndex = 0
	}
	if savedIndex >= len(levels) {
		savedIndex = len(levels) - 1
	}

	// Resume the last level actually played. Progression is stored separately
	// in unlockFrontier; clamp only if the saved level is no longer available
	// under normal progression (for example, it was visited with the U toggle).
	if savedIndex > permanentUnlockedLimit() {
		savedIndex = permanentUnlockedLimit()
	}
	return savedIndex
}

// ---- Start a level ----
func startLevel(index int) {
	if autoPaddleEnabled && index == 13 {
		index++ // Level 14 is omitted only from demo play.
	}
	autoDemoWait = 0
	stopAllMagicFeatureVoices()
	if index >= len(levels) {
		gameOver = true
		win = true
		return
	}
	paused = false
	waitingForStart = true
	levelCompleteTimer = 0
	levelAdvancePending = false
	leftPressed = false
	rightPressed = false
	touchControlActive = false
	mouseControlActive = false
	mousePaddleTargetX = 0
	mobileLeftHeld = false
	mobileRightHeld = false
	mobileLeftPointerID = -1
	mobileRightPointerID = -1
	paddle.vx = 0
	resetPaddleSpinHistory()

	// User toggles never carry into a new level.
	magnetCheat = false
	zapperCheat = false
	resetLevelRunStats(index)

	// Level 1 always begins a fresh whole-game attempt. Later levels preserve that
	// attempt only when they are the exact next sequential level reached through
	// normal completion.
	if index == 0 {
		beginFullGameRun()
	} else if !fullGameRun.active || fullGameRun.levelCount != len(levels) || index != fullGameRun.nextLevel {
		cancelFullGameRun()
	} else if autoPaddleEnabled {
		fullGameRun.recordEligible = false
	}

	// Reset all globals and level-only state before applying config.
	resetGlobals()
	levelMagnetActive = false
	levelZapperActive = false

	// Apply level-specific config. A level can use magnet=true, audioRoom=cave,
	// and audioRoomDry=0.65. If audioRoom is omitted, resetGlobals leaves it dry.
	applyConfig(levels[index].config)
	applyAudioRoomSettings()

	paddle.w = paddleWidth
	paddle.h = paddleHeight

	ball.x, ball.y = startBallX, startBallY
	ball.vx, ball.vy = startBallVx, startBallVy
	ball.omega, ball.angle = 0, 0
	ball.stuckTimer = 0
	ball.soundCooldown = 0
	ball.ballBallSoundCooldown = 0
	ball.r = ballRadius
	// Use the real launch state as the baseline. Peaks then track the fastest
	// instantaneous ball speed and absolute spin actually reached on this level.
	levelMeasuredMaxSpeed = math.Hypot(ball.vx, ball.vy)
	levelMeasuredMaxSpin = math.Abs(ball.omega)
	levelRunStats.fastestSpeed = levelMeasuredMaxSpeed
	levelRunStats.fastestSpin = levelMeasuredMaxSpin
	resetFastOrbitState(&ball)
	resetBallRescueState(&ball, true)
	resetBallZapperState(&ball)
	ball.ballCollisionGrace = 0
	extraBalls = extraBalls[:0]
	paddle.x = (canvasWidth - paddle.w) / 2
	paddle.y = canvasHeight - 40
	paddle.vx = 0
	paddlePreviousX = paddle.x
	mousePaddleTargetX = paddle.x
	clearLastPaddleSpinDebug()
	clearLastCollisionDebug()

	lowGravityActive = false
	lowGravityTimer = 0
	passActive = false
	passTimer = 0
	reverseGravityActive = false
	reverseGravityTimer = 0
	magnetPowerActive = false
	magnetPowerTimer = 0
	zapperPowerActive = false
	zapperPowerTimer = 0
	bigPaddleActive = false
	bigPaddleTimer = 0
	bigPaddleExpiryPending = false
	blackHoleActive = false
	blackHoleTimer = 0
	blackHoleX = canvasWidth / 2
	blackHoleY = canvasHeight/2 + blackHolePathCenterYOffset
	resetBlackHolePath()
	refreshCurrentGravity()
	influencerActive = false
	influencerTimer = 0
	resetAllBallZapperStates()
	statusMessages = nil
	if levelMagnetActive {
		showStatus("Magnets enabled by level", 2.0)
	}
	if levelZapperActive {
		showStatus("Zapper enabled by level", 2.0)
	}

	debrisRNG.Seed(int64(index+1)*0x52d3b715 + 1)
	autoPaddleRNG.Seed(int64(index+1)*0x60a17f3d + 7)
	blackHoleRNG.Seed(int64(index+1)*0x63b10c7 + 11)
	resetAutoPaddleHitPlan()
	buildBricksFromLevel(levels[index], index)
	currentLevelIndex = index
	beginReplayRecording(index)
	if autoPaddleEnabled {
		waitingForStart = false
	}
	syncRenderInterpolation()
	saveCurrentLevel()
	log("Level " + strconv.Itoa(index+1) + " started")
}

// ---- Jump to a specific level (cheat) ----
func jumpToLevel(index int) {
	if index < 0 {
		index = 0
	}
	if index >= len(levels) {
		index = len(levels) - 1
	}
	if index > activeUnlockedLimit() {
		showStatus("Level locked", 2.0)
		return
	}

	lives = defaultLives
	score = 0

	gameOver = false
	win = false
	if index != 0 {
		cancelFullGameRun()
	}
	startLevel(index)
}

// ---- Reset game ----
func resetGame() {
	lives = defaultLives
	score = 0
	gameOver = false
	win = false
	paused = false
	leftPressed = false
	rightPressed = false
	unlockFrontier = loadUnlockFrontier()
	refreshHighestUnlockedLevel()
	startLevel(loadSavedLevel())
}

// ---- Retry the level after losing all lives ----
func retryCurrentLevel() {
	lives = defaultLives
	score = 0
	gameOver = false
	win = false
	paused = false
	leftPressed = false
	rightPressed = false
	paddle.vx = 0
	if currentLevelIndex != 0 {
		cancelFullGameRun()
	}
	startLevel(currentLevelIndex)
}

// ---- Magnetic breakable bricks ----
func magnetIsActive() bool {
	return magnetPowerActive || (levelMagnetActive != magnetCheat)
}

func applyBrickMagnetism(b *Ball, dt float64) {
	if !magnetIsActive() {
		return
	}

	nearestIndex := -1
	nearestDistanceSquared := magnetRange * magnetRange
	for i := range bricks {
		br := &bricks[i]
		if !br.alive || br.unbreakable {
			continue
		}
		dx := br.x + br.w/2 - b.x
		dy := br.y + br.h/2 - b.y
		distanceSquared := dx*dx + dy*dy
		if distanceSquared > 1 && distanceSquared < nearestDistanceSquared {
			nearestIndex = i
			nearestDistanceSquared = distanceSquared
		}
	}
	if nearestIndex < 0 {
		return
	}

	nearest := &bricks[nearestIndex]
	dx := nearest.x + nearest.w/2 - b.x
	dy := nearest.y + nearest.h/2 - b.y
	nearestDistance := math.Sqrt(nearestDistanceSquared)
	falloff := 1 - nearestDistance/magnetRange
	acceleration := magnetStrength * falloff
	b.vx += dx / nearestDistance * acceleration * dt
	b.vy += dy / nearestDistance * acceleration * dt
}

// ---- Influencer area damage ----
func destroyBricksInRadius(ballX, ballY, radius, impactSpeed float64) {
	radiusSquared := radius * radius
	for i := range bricks {
		br := &bricks[i]
		if !br.alive || br.unbreakable {
			continue
		}
		closestX := math.Max(br.x, math.Min(ballX, br.x+br.w))
		closestY := math.Max(br.y, math.Min(ballY, br.y+br.h))
		dx := ballX - closestX
		dy := ballY - closestY
		if dx*dx+dy*dy <= radiusSquared && destroyBrick(br, impactSpeed) {
			hitX := brickCenterX(br)
			if br.magic {
				playMagic(impactSpeed, hitX)
			} else {
				playBrickBreak(impactSpeed, hitX)
			}
		}
	}
}

func candidateBrickIndices(b *Ball, indices []int) []int {
	indices = indices[:0]
	if gridRows <= 0 || gridCols <= 0 || gridCellWidth <= 0 || gridCellHeight <= 0 {
		for i := range bricks {
			indices = append(indices, i)
		}
		return indices
	}

	minCol := int(math.Floor((b.x - b.r - gridOffsetLeft) / gridCellWidth))
	maxCol := int(math.Floor((b.x + b.r - gridOffsetLeft) / gridCellWidth))
	minRow := int(math.Floor((b.y - b.r - gridOffsetTop) / gridCellHeight))
	maxRow := int(math.Floor((b.y + b.r - gridOffsetTop) / gridCellHeight))
	minCol, maxCol = minCol-1, maxCol+1
	minRow, maxRow = minRow-1, maxRow+1
	if minCol < 0 {
		minCol = 0
	}
	if minRow < 0 {
		minRow = 0
	}
	if maxCol >= gridCols {
		maxCol = gridCols - 1
	}
	if maxRow >= gridRows {
		maxRow = gridRows - 1
	}

	for row := minRow; row <= maxRow; row++ {
		for col := minCol; col <= maxCol; col++ {
			indices = append(indices, brickGrid[gridKey(row, col)]...)
		}
	}
	return indices
}

func playTilt() {
	scheduleTone("triangle", varyFreq(130, 0.05), varyFreq(360, 0.05), 0.12, 0.14, 0)
	scheduleTone("sine", varyFreq(70, 0.04), varyFreq(115, 0.04), 0.18, 0.08, 0.02)
}

func applyTilt(b *Ball) {
	physics := &physicsConfig
	sideMin, sideMax := physics.tiltSideMin, physics.tiltSideMax
	if sideMax < sideMin {
		sideMin, sideMax = sideMax, sideMin
	}
	sideSpeed := sideMin
	if sideMax > sideMin {
		sideSpeed += rand.Float64() * (sideMax - sideMin)
	}

	if rand.Intn(2) == 0 {
		sideSpeed = -sideSpeed
	}

	b.vx = sideSpeed
	b.vy = -physics.tiltUpSpeed
	b.omega += (rand.Float64()*2 - 1) * 8
	b.stuckTimer = 0
	beginBallRescueAttempt(b, orbitAxisNone)

	showStatus("TILT!", 1.5)
	playTilt()
}

func ballPastBelowFloorLimit(b *Ball) bool {
	if b == nil {
		return false
	}
	return b.y+b.r > canvasHeight+math.Max(0, ballBelowFloorGracePixels)
}

func updateStuckDetector(b *Ball, dt float64) {
	physics := &physicsConfig
	if b.y+b.r > canvasHeight {
		b.stuckTimer = 0
		return
	}

	speed := math.Hypot(b.vx, b.vy)
	if speed < physics.stuckSpeedThreshold {
		b.stuckTimer += dt
		if b.stuckTimer >= physics.stuckDuration {
			applyTilt(b)
		}
	} else {
		b.stuckTimer = 0
	}
}

func nearestBreakableBrickIndex(x, y float64) int {
	nearest := -1
	nearestDistanceSquared := zapperRange * zapperRange

	for i := range bricks {
		br := &bricks[i]
		if !br.alive || br.unbreakable {
			continue
		}

		dx := br.x + br.w/2 - x
		dy := br.y + br.h/2 - y
		distanceSquared := dx*dx + dy*dy

		if distanceSquared < nearestDistanceSquared {
			nearest = i
			nearestDistanceSquared = distanceSquared
		}
	}

	return nearest
}

func zapperIsActive() bool {
	threshold := zapperThreshold()
	autoActive := threshold > 0 &&
		remainingBreakableBricks > 0 &&
		remainingBreakableBricks <= threshold

	return autoActive ||
		zapperPowerActive ||
		(levelZapperActive != zapperCheat)
}

func zapperThreshold() int {
	if initialBreakableBricks <= 0 {
		return 0
	}

	threshold := int(math.Ceil(float64(initialBreakableBricks) * 0.03))
	if threshold < 1 {
		threshold = 1
	}
	return threshold
}

func updateOneZapper(
	b *Ball,
	active bool,
	targetIndex *int,
	hitTimer *float64,
	dt float64,
) {
	if !active {
		*targetIndex = -1
		*hitTimer = 0
		return
	}

	target := nearestBreakableBrickIndex(b.x, b.y)
	if target < 0 {
		*targetIndex = -1
		*hitTimer = 0
		return
	}

	if target != *targetIndex {
		*targetIndex = target
		*hitTimer = 0
	}

	*hitTimer += dt
	if *hitTimer < defaultZapperHitTime {
		return
	}

	br := &bricks[*targetIndex]
	if destroyBrick(br, 0) {
		playZapperDestroyedBrick(br)
		playZapper(brickCenterX(br))
	}

	*targetIndex = -1
	*hitTimer = 0
}

func updateZapper(dt float64) {
	if !zapperIsActive() {
		resetAllBallZapperStates()
		return
	}

	forEachActiveBall(func(b *Ball, _ int) {
		updateOneZapper(
			b,
			true,
			&b.zapperTargetIndex,
			&b.zapperHitTimer,
			dt,
		)
	})
}

// One brick contact. We still choose a single classic axis
// normal, but group simultaneous overlaps so adjacent bricks do not bounce the
// ball multiple times in one substep.
type brickContact struct {
	index          int
	nx, ny         float64
	axisNX, axisNY float64
	penetration    float64
	impact         float64
	swept          bool
	time           float64
	cornerApplied  bool
}

func sweptPointAABB(
	previousX, previousY, currentX, currentY,
	minX, minY, maxX, maxY float64,
) (bool, float64, float64, float64) {
	dx := currentX - previousX
	dy := currentY - previousY

	axisTimes := func(position, delta, minimum, maximum float64) (float64, float64) {
		if delta > 0 {
			return (minimum - position) / delta, (maximum - position) / delta
		}
		if delta < 0 {
			return (maximum - position) / delta, (minimum - position) / delta
		}
		if position < minimum || position > maximum {
			return math.Inf(1), math.Inf(-1)
		}
		return math.Inf(-1), math.Inf(1)
	}

	xEntry, xExit := axisTimes(previousX, dx, minX, maxX)
	yEntry, yExit := axisTimes(previousY, dy, minY, maxY)
	entry := math.Max(xEntry, yEntry)
	exit := math.Min(xExit, yExit)
	if entry > exit || exit < 0 || entry < 0 || entry > 1 {
		return false, 0, 0, 0
	}

	nx, ny := 0.0, 0.0
	if math.Abs(xEntry-yEntry) <= 0.000001 {
		if math.Abs(dy) >= math.Abs(dx) {
			ny = -sign(dy)
		} else {
			nx = -sign(dx)
		}
	} else if xEntry > yEntry {
		nx = -sign(dx)
	} else {
		ny = -sign(dy)
	}
	return true, nx, ny, entry
}

func penetrationForBrickNormal(b *Ball, br *brick, nx, ny float64) float64 {
	switch {
	case nx < 0:
		return math.Max(0, b.x-(br.x-b.r))
	case nx > 0:
		return math.Max(0, br.x+br.w+b.r-b.x)
	case ny < 0:
		return math.Max(0, b.y-(br.y-b.r))
	default:
		return math.Max(0, br.y+br.h+b.r-b.y)
	}
}

func pointIntervalDistance(value, minimum, maximum float64) float64 {
	if value < minimum {
		return minimum - value
	}
	if value > maximum {
		return value - maximum
	}
	return 0
}

// brickCornerGapPassable decides whether the two outward sides of one specific
// brick corner are genuinely exposed to the ball. Two neighboring bricks whose
// gap is narrower than the ball diameter have overlapping ball-centre exclusion
// zones, so their facing corners should behave like one continuous surface.
// Once the gap is wide enough for the current ball to pass, those corners become
// real corners again. Destroyed bricks are ignored, so a newly opened gap takes
// effect immediately.
func brickCornerGapPassable(brickIndex int, cornerX, cornerY, sideX, sideY, radius float64) bool {
	if brickIndex < 0 || brickIndex >= len(bricks) || radius <= 0 {
		return true
	}
	br := &bricks[brickIndex]
	clearance := math.Max(0, physicsConfig.collisionSlop)
	requiredGap := 2*radius + 2*clearance
	perpendicularReach := radius + clearance
	const epsilon = 1e-6

	// Only bricks whose grid cells overlap the small region around this corner can
	// possibly close either outward gap. This replaces the previous full-brick scan
	// on every qualifying corner contact while preserving the exact gap tests below.
	// If the spatial grid is unavailable, retain the old full scan as a safe fallback.
	minX, maxX := cornerX-perpendicularReach, cornerX+perpendicularReach
	minY, maxY := cornerY-perpendicularReach, cornerY+perpendicularReach
	if sideX > 0 {
		maxX = cornerX + requiredGap
	} else if sideX < 0 {
		minX = cornerX - requiredGap
	}
	if sideY > 0 {
		maxY = cornerY + requiredGap
	} else if sideY < 0 {
		minY = cornerY - requiredGap
	}

	checkBrick := func(i int) bool {
		if i == brickIndex || i < 0 || i >= len(bricks) || !bricks[i].alive {
			return false
		}
		other := &bricks[i]

		if sideX > 0 && other.x >= br.x+br.w-epsilon {
			gap := other.x - (br.x + br.w)
			if gap < requiredGap &&
				pointIntervalDistance(cornerY, other.y, other.y+other.h) <= perpendicularReach {
				return true
			}
		} else if sideX < 0 && other.x+other.w <= br.x+epsilon {
			gap := br.x - (other.x + other.w)
			if gap < requiredGap &&
				pointIntervalDistance(cornerY, other.y, other.y+other.h) <= perpendicularReach {
				return true
			}
		}

		if sideY > 0 && other.y >= br.y+br.h-epsilon {
			gap := other.y - (br.y + br.h)
			if gap < requiredGap &&
				pointIntervalDistance(cornerX, other.x, other.x+other.w) <= perpendicularReach {
				return true
			}
		} else if sideY < 0 && other.y+other.h <= br.y+epsilon {
			gap := br.y - (other.y + other.h)
			if gap < requiredGap &&
				pointIntervalDistance(cornerX, other.x, other.x+other.w) <= perpendicularReach {
				return true
			}
		}
		return false
	}

	if gridRows <= 0 || gridCols <= 0 || gridCellWidth <= 0 || gridCellHeight <= 0 || brickGrid == nil {
		for i := range bricks {
			if checkBrick(i) {
				return false
			}
		}
		return true
	}

	// A brick can overlap the search region even when its cell origin lies up to
	// one brick width/height before the region, so include that extent when mapping
	// world coordinates back to grid cells.
	minCol := int(math.Ceil((minX - br.w - gridOffsetLeft - epsilon) / gridCellWidth))
	maxCol := int(math.Floor((maxX - gridOffsetLeft + epsilon) / gridCellWidth))
	minRow := int(math.Ceil((minY - br.h - gridOffsetTop - epsilon) / gridCellHeight))
	maxRow := int(math.Floor((maxY - gridOffsetTop + epsilon) / gridCellHeight))

	if minCol < 0 {
		minCol = 0
	}
	if minRow < 0 {
		minRow = 0
	}
	if maxCol >= gridCols {
		maxCol = gridCols - 1
	}
	if maxRow >= gridRows {
		maxRow = gridRows - 1
	}
	if minCol > maxCol || minRow > maxRow {
		return true
	}

	for row := minRow; row <= maxRow; row++ {
		for col := minCol; col <= maxCol; col++ {
			for _, i := range brickGrid[gridKey(row, col)] {
				if checkBrick(i) {
					return false
				}
			}
		}
	}
	return true
}

// brickCornerResponse makes collision geometry follow the same rounded corners
// that are drawn with roundRect(..., brickRadius). The rounded-rectangle corner
// arc is expanded by the ball radius, so a visible curved hit produces a radial
// normal rather than requiring the ball centre to reach the old mathematical
// square corner. The final bool reports whether the broad-phase AABB overlap is
// a real rounded-rectangle contact; false lets the ball pass through the visual
// cut-away outside a rounded corner.
func brickCornerResponse(b *Ball, brickIndex int, br *brick, axisNX, axisNY, axisPenetration float64) (float64, float64, float64, bool, bool) {
	// Canvas roundRect effectively cannot have a corner radius larger than half
	// the shorter side. Clamp physics the same way so level geometry stays sane.
	roundRadius := clampFloat(brickRadius, 0, math.Min(br.w, br.h)/2)
	innerLeft := br.x + roundRadius
	innerRight := br.x + br.w - roundRadius
	innerTop := br.y + roundRadius
	innerBottom := br.y + br.h - roundRadius

	sideX := 0.0
	cornerCenterX := b.x
	outerCornerX := b.x
	switch {
	case b.x < innerLeft:
		sideX = -1
		cornerCenterX = innerLeft
		outerCornerX = br.x
	case b.x > innerRight:
		sideX = 1
		cornerCenterX = innerRight
		outerCornerX = br.x + br.w
	}

	sideY := 0.0
	cornerCenterY := b.y
	outerCornerY := b.y
	switch {
	case b.y < innerTop:
		sideY = -1
		cornerCenterY = innerTop
		outerCornerY = br.y
	case b.y > innerBottom:
		sideY = 1
		cornerCenterY = innerBottom
		outerCornerY = br.y + br.h
	}

	// If only one axis lies beyond the inner rounded core, this is a normal flat
	// top/bottom/side hit, not a curved-corner hit.
	if sideX == 0 || sideY == 0 {
		return axisNX, axisNY, axisPenetration, false, true
	}

	dx := b.x - cornerCenterX
	dy := b.y - cornerCenterY
	distanceSquared := dx*dx + dy*dy
	combinedRadius := roundRadius + b.r

	// The expanded AABB used by the broad phase includes the empty cut-away around
	// a rounded corner. Reject that false overlap so physics matches the drawing.
	if distanceSquared >= combinedRadius*combinedRadius {
		return axisNX, axisNY, axisPenetration, false, false
	}
	if distanceSquared <= 1e-12 {
		return axisNX, axisNY, axisPenetration, false, true
	}

	if !physicsConfig.cornerPhysicsEnabled {
		return axisNX, axisNY, axisPenetration, false, true
	}
	amount := clampFloat(physicsConfig.cornerPhysicsAmount, 0, 1)
	if amount <= 0 {
		return axisNX, axisNY, axisPenetration, false, true
	}

	if !physicsConfig.cornerPhysicsAllBricks &&
		!brickCornerGapPassable(brickIndex, outerCornerX, outerCornerY, sideX, sideY, b.r) {
		return axisNX, axisNY, axisPenetration, false, true
	}

	distance := math.Sqrt(distanceSquared)
	cornerNX := dx / distance
	cornerNY := dy / distance
	nx := lerpFloat(axisNX, cornerNX, amount)
	ny := lerpFloat(axisNY, cornerNY, amount)
	length := math.Hypot(nx, ny)
	if length <= 1e-12 {
		return axisNX, axisNY, axisPenetration, false, true
	}
	nx /= length
	ny /= length

	cornerPenetration := math.Max(0, combinedRadius-distance)
	penetration := lerpFloat(axisPenetration, cornerPenetration, amount)
	return nx, ny, penetration, true, true
}

func flushCornerPhysicsDebugLogs() {
	if len(cornerPhysicsDebugPending) == 0 {
		return
	}
	for _, event := range cornerPhysicsDebugPending {
		log(fmt.Sprintf(
			"%03d CORNER HIT row=%d col=%d amount=%.2f normal=(%.3f, %.3f) impact=%.1f",
			event.count, event.row, event.col, event.amount, event.nx, event.ny, event.impact,
		))
	}
	cornerPhysicsDebugPending = cornerPhysicsDebugPending[:0]
}

func findBrickContacts(b *Ball, previousX, previousY float64, contacts []brickContact) []brickContact {
	contacts = contacts[:0]
	// Most queries fit on the stack. append still handles unusually dense grids
	// without truncating candidates or changing their traversal order.
	var candidateStorage [32]int
	for _, i := range candidateBrickIndices(b, candidateStorage[:0]) {
		br := &bricks[i]
		if !br.alive ||
			b.x+b.r <= br.x || b.x-b.r >= br.x+br.w ||
			b.y+b.r <= br.y || b.y-b.r >= br.y+br.h {
			continue
		}

		leftPenetration := b.x + b.r - br.x
		rightPenetration := br.x + br.w - (b.x - b.r)
		topPenetration := b.y + b.r - br.y
		bottomPenetration := br.y + br.h - (b.y - b.r)

		axisNX, axisNY := -1.0, 0.0
		penetration := leftPenetration
		if rightPenetration < penetration {
			axisNX, axisNY = 1, 0
			penetration = rightPenetration
		}
		if topPenetration < penetration {
			axisNX, axisNY = 0, -1
			penetration = topPenetration
		}
		if bottomPenetration < penetration {
			axisNX, axisNY = 0, 1
			penetration = bottomPenetration
		}

		swept, sweptNX, sweptNY, hitTime := sweptPointAABB(
			previousX, previousY, b.x, b.y,
			br.x-b.r, br.y-b.r, br.x+br.w+b.r, br.y+br.h+b.r,
		)
		if swept {
			axisNX, axisNY = sweptNX, sweptNY
			penetration = penetrationForBrickNormal(b, br, axisNX, axisNY)
		}

		// Match the visible rounded brick corner without reintroducing seam bounces.
		// cornerPhysicsAmount=0 is the classic axis behavior; 1 uses the full radial
		// roundRect-corner normal. The tiny visual brick tilt is applied last.
		responseNX, responseNY, penetration, cornerApplied, contactValid := brickCornerResponse(
			b, i, br, axisNX, axisNY, penetration,
		)
		if !contactValid {
			continue
		}
		nx, ny := rotateVector(responseNX, responseNY, br.tiltRadians)
		axisComponent := math.Abs(nx*responseNX + ny*responseNY)
		if axisComponent > 0.000001 {
			penetration /= axisComponent
		}

		contacts = append(contacts, brickContact{
			index:         i,
			nx:            nx,
			ny:            ny,
			axisNX:        axisNX,
			axisNY:        axisNY,
			penetration:   math.Max(0, penetration),
			impact:        math.Max(0, -(b.vx*nx + b.vy*ny)),
			swept:         swept,
			time:          hitTime,
			cornerApplied: cornerApplied,
		})
	}
	return contacts
}

func handleBrickCollisions(b *Ball, isPrimary bool, previousX, previousY float64) {
	var contactStorage [8]brickContact
	contacts := findBrickContacts(b, previousX, previousY, contactStorage[:0])
	if len(contacts) == 0 {
		return
	}

	// A chain is the number of bricks destroyed as a consequence of one physical
	// ball-to-brick contact event. This naturally includes immediate Nuke/Influencer
	// secondary destruction, but not unrelated Zapper kills that happen later.
	scoreBeforeHit := score
	defer func() {
		chain := score - scoreBeforeHit
		if chain > levelRunStats.bestChain {
			levelRunStats.bestChain = chain
		}
	}()

	if passActive {
		for _, contact := range contacts {
			br := &bricks[contact.index]
			featureActivated := false
			if br.magic && isPrimary {
				featureActivated = activatePowerUpWithBrick(br, contact.impact)
			}
			if destroyBrick(br, contact.impact) {
				hitX := brickCenterX(br)
				if br.magic {
					if !featureActivated {
						playMagic(contact.impact, hitX)
					}
				} else {
					playBrickBreak(contact.impact, hitX)
				}
				if influencerActive {
					destroyBricksInRadius(b.x, b.y, b.r*influencerMultiplier, contact.impact)
				}
			}
		}
		return
	}

	// Swept entry normals win over overlap-only normals. This prevents a ball
	// approaching a flat row from above or below from mistaking the internal
	// seam between two adjacent bricks for alternating left/right walls.
	bestIndex := -1
	for i, contact := range contacts {
		if !contact.swept {
			continue
		}
		if bestIndex < 0 || contact.time < contacts[bestIndex].time-0.000001 ||
			(math.Abs(contact.time-contacts[bestIndex].time) <= 0.000001 && contact.impact > contacts[bestIndex].impact) {
			bestIndex = i
		}
	}

	if bestIndex < 0 {
		hasLeft, hasRight, hasVertical := false, false, false
		for _, contact := range contacts {
			hasLeft = hasLeft || contact.axisNX < 0
			hasRight = hasRight || contact.axisNX > 0
			hasVertical = hasVertical || contact.axisNY != 0
		}
		ignoreInternalHorizontalSeam := hasLeft && hasRight && hasVertical
		for i, contact := range contacts {
			if ignoreInternalHorizontalSeam && contact.axisNX != 0 {
				continue
			}
			if bestIndex < 0 || contact.impact > contacts[bestIndex].impact+0.001 ||
				(math.Abs(contact.impact-contacts[bestIndex].impact) <= 0.001 && contact.penetration > contacts[bestIndex].penetration) {
				bestIndex = i
			}
		}
	}
	if bestIndex < 0 {
		bestIndex = 0
	}
	best := contacts[bestIndex]

	if enableCornerPhysicsDebug && best.cornerApplied {
		cornerPhysicsDebugHitCount++
		br := &bricks[best.index]
		cornerPhysicsDebugPending = append(cornerPhysicsDebugPending, cornerPhysicsDebugEvent{
			count:  cornerPhysicsDebugHitCount,
			row:    br.row,
			col:    br.col,
			amount: clampFloat(physicsConfig.cornerPhysicsAmount, 0, 1),
			nx:     best.nx,
			ny:     best.ny,
			impact: best.impact,
		})
	}

	b.x += best.nx * (best.penetration + physicsConfig.collisionSlop)
	b.y += best.ny * (best.penetration + physicsConfig.collisionSlop)
	bestWasUnbreakable := bricks[best.index].unbreakable
	frictionScale := physicsConfig.brickFrictionScale
	if bestWasUnbreakable {
		frictionScale = physicsConfig.unbreakableFrictionScale
	}
	resolveCollisionDebug(b, best.nx, best.ny, 0, 0, frictionScale, "BRICK", isPrimary)
	if bestWasUnbreakable && best.impact >= physicsConfig.orbitMinimumHitSpeed {
		recordUnbreakableOrbitHit(b)
	}

	hitUnbreakable := false
	destroyedNormal := false
	for _, contact := range contacts {
		br := &bricks[contact.index]
		if !br.alive {
			continue
		}
		if br.unbreakable {
			hitUnbreakable = true
			continue
		}
		if br.magic {
			featureActivated := false
			if isPrimary {
				featureActivated = activatePowerUpWithBrick(br, contact.impact)
			}
			if destroyBrick(br, contact.impact) && !featureActivated {
				playMagic(contact.impact, brickCenterX(br))
			}
			if influencerActive {
				destroyBricksInRadius(b.x, b.y, b.r*influencerMultiplier, contact.impact)
			}
			continue
		}
		if destroyBrick(br, contact.impact) {
			destroyedNormal = true
			playBrickBreak(contact.impact, brickCenterX(br))
		}
		if influencerActive {
			destroyBricksInRadius(b.x, b.y, b.r*influencerMultiplier, contact.impact)
		}
	}

	if hitUnbreakable {
		hitX := brickCenterX(&bricks[best.index])
		playImpactSound(b, best.impact, func() { playUnbreakable(hitX) })
	}
	if destroyedNormal && physicsConfig.brickBoost != 0 {
		b.vy -= physicsConfig.brickBoost
		if math.Abs(b.vy) > physicsConfig.maxSpeed {
			b.vy = math.Copysign(physicsConfig.maxSpeed, b.vy)
		}
		if math.Abs(b.vx) > physicsConfig.maxSpeed {
			b.vx = math.Copysign(physicsConfig.maxSpeed, b.vx)
		}
	}
}

// ---- Update a single ball ----
func updateBallStep(b *Ball, dt float64, isPrimary bool) {
	physics := &physicsConfig
	if b.soundCooldown > 0 {
		b.soundCooldown = math.Max(0, b.soundCooldown-dt)
	}
	if b.ballBallSoundCooldown > 0 {
		b.ballBallSoundCooldown = math.Max(0, b.ballBallSoundCooldown-dt)
	}

	b.vy += currentGravity * dt
	applyBrickMagnetism(b, dt)

	if blackHoleActive {
		dx := blackHoleX - b.x
		dy := blackHoleY - b.y
		dist := math.Hypot(dx, dy)
		if dist > 1.0 {
			b.vx += (dx / dist) * blackHoleStrength * dt
			b.vy += (dy / dist) * blackHoleStrength * dt
		} else {
			b.vx += (rand.Float64() - 0.5) * 10.0
			b.vy += (rand.Float64() - 0.5) * 10.0
		}
	}

	// Magnus effect: spin bends the flight path perpendicular to velocity.
	magnusAx := -b.vy * b.omega * physics.magnusCoefficient
	magnusAy := b.vx * b.omega * physics.magnusCoefficient
	magnusMagnitude := math.Hypot(magnusAx, magnusAy)
	maxMagnusAcceleration := math.Max(100.0, physics.maxSpeed*physics.magnusAccelerationScale)
	if magnusMagnitude > maxMagnusAcceleration {
		scale := maxMagnusAcceleration / magnusMagnitude
		magnusAx *= scale
		magnusAy *= scale
	}
	b.vx += magnusAx * dt
	b.vy += magnusAy * dt

	airDamping := math.Exp(-physics.airDrag * dt)
	spinDamping := math.Exp(-physics.spinDrag * dt)
	b.vx *= airDamping
	b.vy *= airDamping
	b.omega *= spinDamping
	applyOverspeedDrag(b, dt, physics)

	previousX, previousY := b.x, b.y
	b.x += b.vx * dt
	b.y += b.vy * dt
	b.angle += b.omega * dt

	// Walls.
	if b.x-b.r < 0 {
		impactSpeed := math.Max(0, -b.vx)
		b.x = b.r + physics.collisionSlop
		nx, ny := roughWallNormal(1, 0, b.y, canvasHeight, wallNoiseIDLeft, physics.wallSideTiltDegrees)
		resolveCollisionDebug(b, nx, ny, 0, 0, physics.wallFrictionScale, "WALL LEFT", isPrimary)
		playImpactSound(b, impactSpeed, playWallHit)
	}
	if b.x+b.r > canvasWidth {
		impactSpeed := math.Max(0, b.vx)
		b.x = canvasWidth - b.r - physics.collisionSlop
		nx, ny := roughWallNormal(-1, 0, b.y, canvasHeight, wallNoiseIDRight, physics.wallSideTiltDegrees)
		resolveCollisionDebug(b, nx, ny, 0, 0, physics.wallFrictionScale, "WALL RIGHT", isPrimary)
		playImpactSound(b, impactSpeed, playWallHit)
	}
	if b.y-b.r < 0 {
		impactSpeed := math.Max(0, -b.vy)
		b.y = b.r + physics.collisionSlop
		nx, ny := roughWallNormal(0, 1, b.x, canvasWidth, wallNoiseIDTop, physics.wallTopTiltDegrees)
		resolveCollisionDebug(b, nx, ny, 0, 0, physics.wallFrictionScale, "WALL TOP", isPrimary)
		playImpactSound(b, impactSpeed, playWallHit)
	}
	if ballPastBelowFloorLimit(b) {
		return
	}

	// Paddle.
	pLeft, pRight := paddle.x, paddle.x+paddle.w
	pTop, pBottom := paddle.y, paddle.y+paddle.h
	if b.vy > 0 &&
		b.x+b.r > pLeft && b.x-b.r < pRight &&
		b.y+b.r > pTop && b.y+b.r < pBottom {
		recordIncomingCollisionSpeed(math.Hypot(b.vx, b.vy))
		b.y = pTop - b.r - physics.collisionSlop
		resetFastOrbitState(b)
		if autoPaddleEnabled {
			autoPaddleNeedsNewHitOffset = true
			autoPaddleTargetBall = 0
		}

		spinBeforePaddle := b.omega
		incomingPaddleAngle := velocityAngleDegrees(b.vx, b.vy)
		hitPos := clampFloat((b.x-pLeft)/paddle.w, 0, 1)
		angle := (hitPos - 0.5) * 2.0 * (80.0 * math.Pi / 180.0)
		speed := math.Max(100, math.Hypot(b.vx, b.vy))
		speed = math.Min(speed*physics.restitution+physics.paddleBoost, physics.maxSpeed)

		incomingVx := b.vx
		incomingVy := b.vy
		b.vx = speed*math.Sin(angle) + incomingVx*0.15
		b.vy = -speed * math.Cos(angle)

		effectivePaddleVx := effectivePaddleSpinVelocity()
		relativeSlip := b.vx - b.omega*b.r - effectivePaddleVx
		normalDeltaSpeed := math.Abs(b.vy - incomingVy)
		effectivePaddleFriction := math.Max(
			physics.frictionCoeff*physics.paddleFrictionScale,
			physics.minimumPaddleGrip,
		)
		maxFrictionDelta := effectivePaddleFriction * normalDeltaSpeed
		desiredDeltaVx := -relativeSlip / 3.0
		deltaVx := clampFloat(desiredDeltaVx, -maxFrictionDelta, maxFrictionDelta)
		b.vx += deltaVx
		b.omega -= 2 * deltaVx / b.r
		b.omega += -effectivePaddleVx * physics.paddleSpinTransfer / math.Max(b.r, 1)

		preferredDirection := incomingVx
		if preferredDirection == 0 {
			preferredDirection = paddle.vx
		}
		if preferredDirection == 0 {
			preferredDirection = hitPos - 0.5
		}

		noSpinPaddle := *b
		noSpinPaddle.omega = 0
		noSpinPaddle.vx -= deltaVx
		noSpinRelativeSlip := noSpinPaddle.vx - effectivePaddleVx
		noSpinDesiredDeltaVx := -noSpinRelativeSlip / 3.0
		noSpinDeltaVx := clampFloat(noSpinDesiredDeltaVx, -maxFrictionDelta, maxFrictionDelta)
		noSpinPaddle.vx += noSpinDeltaVx
		preventVerticalLock(&noSpinPaddle, preferredDirection)
		clampBallToPhysicsSettings(&noSpinPaddle, physics)

		preventVerticalLock(b, preferredDirection)
		clampBallToPhysicsSettings(b, physics)
		recordMeasuredBallSpeed(b)
		recordMeasuredBallSpin(b)
		maybeShowHighSpin(spinBeforePaddle, b.omega)
		if isPrimary {
			recordLastPaddleSpinDebug(true, spinBeforePaddle, b.omega)
			recordLastCollisionDebug(
				"PADDLE", incomingPaddleAngle, velocityAngleDegrees(b.vx, b.vy),
				velocityAngleDegrees(noSpinPaddle.vx, noSpinPaddle.vy),
				spinBeforePaddle, b.omega, deltaVx, noSpinDeltaVx, b.r,
			)
		}
		playImpactSound(b, math.Abs(incomingVy), playPaddleHit)
	}

	handleBrickCollisions(b, isPrimary, previousX, previousY)
	updateFastOrbitDetector(b, dt)
	updateStuckDetector(b, dt)
	updateBallRescueAttempt(b, dt)
}

// Physics subdivides fast movement so a ball cannot skip through thin
// bricks between frames.  still performs exactly one step.
// Subdivide exceptionally fast movement so the ball cannot skip thin bricks.
func updateBallAdaptive(b *Ball, dt float64, isPrimary bool) {
	if dt <= 0 {
		return
	}

	speed := math.Hypot(b.vx, b.vy)
	maxTravelPerStep := math.Max(b.r, 6.0)
	steps := int(math.Ceil(speed * dt / maxTravelPerStep))
	if steps < 1 {
		steps = 1
	}
	if steps > 6 {
		steps = 6
	}

	stepDT := dt / float64(steps)
	for step := 0; step < steps; step++ {
		updateBallStep(b, stepDT, isPrimary)
		if ballPastBelowFloorLimit(b) {
			break
		}
	}
}

func updateBall(b *Ball, dt float64, isPrimary bool) {
	if b == nil {
		return
	}
	if b.ballCollisionGrace > 0 {
		b.ballCollisionGrace = math.Max(0, b.ballCollisionGrace-dt)
	}
	updateBallAdaptive(b, dt, isPrimary)
	recordMeasuredBallSpeed(b)
	recordMeasuredBallSpin(b)
}

// Equal-mass circle collision. With the default five-ball cap there are at most
// ten pairs per fixed step, so this remains a tiny part of the 240 Hz physics load.
func resolveBallBallCollision(a, b *Ball) bool {
	if a == nil || b == nil || a == b || !defaultBallBallCollisions ||
		a.ballCollisionGrace > 0 || b.ballCollisionGrace > 0 {
		return false
	}

	combinedRadius := a.r + b.r
	dx := b.x - a.x
	dy := b.y - a.y
	distanceSquared := dx*dx + dy*dy
	if distanceSquared > combinedRadius*combinedRadius {
		return false
	}

	distance := math.Sqrt(math.Max(distanceSquared, 0))
	nx, ny := 1.0, 0.0
	if distance > 1e-9 {
		nx = dx / distance
		ny = dy / distance
	} else {
		// Coincident centers can happen immediately after a multi-ball spawn. Use
		// relative motion to choose a deterministic separating normal.
		rvx := b.vx - a.vx
		rvy := b.vy - a.vy
		relativeSpeed := math.Hypot(rvx, rvy)
		if relativeSpeed > 1e-9 {
			nx = -rvx / relativeSpeed
			ny = -rvy / relativeSpeed
		}
	}

	overlap := combinedRadius - distance
	if overlap > 0 {
		correction := overlap*0.5 + physicsConfig.collisionSlop*0.5
		a.x -= nx * correction
		a.y -= ny * correction
		b.x += nx * correction
		b.y += ny * correction
	}

	relativeVx := b.vx - a.vx
	relativeVy := b.vy - a.vy
	normalSpeed := relativeVx*nx + relativeVy*ny
	if normalSpeed >= 0 {
		return true
	}

	impactSpeed := -normalSpeed
	restitution := clampFloat(defaultBallBallRestitution, 0, 1.5)
	impulse := (1 + restitution) * impactSpeed / 2.0
	a.vx -= impulse * nx
	a.vy -= impulse * ny
	b.vx += impulse * nx
	b.vy += impulse * ny

	if defaultBallBallSoundEnabled &&
		impactSpeed >= defaultBallBallSoundMinSpeed &&
		a.ballBallSoundCooldown <= 0 && b.ballBallSoundCooldown <= 0 {
		playBallBallHit((a.x+b.x)*0.5, impactSpeed)
		a.ballBallSoundCooldown = defaultBallBallSoundCooldown
		b.ballBallSoundCooldown = defaultBallBallSoundCooldown
	}

	recordMeasuredBallSpeed(a)
	recordMeasuredBallSpeed(b)
	return true
}

func resolveBallBallCollisions() {
	if !defaultBallBallCollisions || activeBallCount() < 2 {
		return
	}
	count := activeBallCount()
	for i := 0; i < count; i++ {
		a := activeBallAt(i)
		if a == nil || ballPastBelowFloorLimit(a) {
			continue
		}
		for j := i + 1; j < count; j++ {
			b := activeBallAt(j)
			if b == nil || ballPastBelowFloorLimit(b) {
				continue
			}
			resolveBallBallCollision(a, b)
		}
	}
}

// Remove balls that fell below the floor. A surviving extra ball is promoted to
// ball 1 when necessary, preserving the old two-ball rule that a life is lost
// only after every active ball has gone. The promoted ball becomes the prize taker.
func compactLostBalls() {
	primaryLost := ballPastBelowFloorLimit(&ball)
	originalExtraCount := len(extraBalls)
	write := 0
	for i := range extraBalls {
		if ballPastBelowFloorLimit(&extraBalls[i]) {
			continue
		}
		if write != i {
			extraBalls[write] = extraBalls[i]
		}
		write++
	}
	extraBalls = extraBalls[:write]

	changed := len(extraBalls) != originalExtraCount
	if primaryLost {
		if len(extraBalls) == 0 {
			loseLife()
			return
		}
		ball = extraBalls[0]
		copy(extraBalls, extraBalls[1:])
		extraBalls = extraBalls[:len(extraBalls)-1]
		changed = true
	}

	if changed {
		resetAutoPaddleHitPlan()
		syncRenderInterpolation()
	}
}

func loseLife() {
	if gameOver || levelAdvancePending || remainingBreakableBricks == 0 || lives <= 0 {
		lives = 0
		return
	}

	lives--
	if lives < 0 {
		lives = 0
	}
	levelRunStats.ballsLost++

	playDie()

	if lives == 0 {
		gameOver = true
		win = false
		cancelFullGameRun()
		playGameOver()
	} else {
		resetBalls()
	}
}

const mobileControlStorageKey = "breakout.mobileControlMode"

func validMobileControlMode(mode string) bool {
	switch mode {
	case "vertical", "tilt", "follow", "two-thumb":
		return true
	default:
		return false
	}
}

func resetMobileControlState() {
	touchControlActive = false
	touchPointerID = -1
	mobileLeftHeld = false
	mobileRightHeld = false
	mobileLeftPointerID = -1
	mobileRightPointerID = -1
	leftPressed = false
	rightPressed = false
	paddle.vx = 0
}

func saveMobileControlMode() {
	defer func() { _ = recover() }()
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return
	}
	storage.Call("setItem", mobileControlStorageKey, mobileControlMode)
}

func selectMobileControlMode(mode string) {
	if !validMobileControlMode(mode) {
		return
	}

	mobileControlMode = mode
	resetMobileControlState()
	saveMobileControlMode()

	// Let the HTML page own the Screen Wake Lock lifecycle.
	wakeLockCallback := js.Global().Get("breakoutSetTiltWakeLock")
	if wakeLockCallback.Type() == js.TypeFunction {
		wakeLockCallback.Invoke(mode == "tilt")
	}

	if mode == "tilt" {
		phoneTiltAvailable = false
		recalibratePhoneTilt()
	}
}

func hideMobileControlSelector() {
	overlay := doc.Call("getElementById", "mobileControlSelector")
	if !overlay.IsUndefined() && !overlay.IsNull() {
		overlay.Get("style").Set("display", "none")
	}
	mobileSelectorVisible = false
}

func showMobileControlSelector() {
	if !mobileControlsEnabled {
		return
	}

	overlay := doc.Call("getElementById", "mobileControlSelector")
	if overlay.IsUndefined() || overlay.IsNull() {
		return
	}

	resetMobileControlState()
	mobileSelectorVisible = true
	overlay.Get("style").Set("display", "flex")
}

func setupMobileControlSelector() {
	window := js.Global().Get("window")
	navigator := js.Global().Get("navigator")

	maxTouchPoints := 0
	if !navigator.IsUndefined() && !navigator.IsNull() {
		value := navigator.Get("maxTouchPoints")
		if value.Type() == js.TypeNumber {
			maxTouchPoints = value.Int()
		}
	}

	coarsePointer := false
	matchMedia := window.Get("matchMedia")
	if matchMedia.Type() == js.TypeFunction {
		coarsePointer = window.Call("matchMedia", "(pointer: coarse)").Get("matches").Bool()
	}

	width := window.Get("innerWidth").Int()
	height := window.Get("innerHeight").Int()
	mobileControlsEnabled = (maxTouchPoints > 0 || coarsePointer) &&
		(width <= 1200 || height <= 800)

	if !mobileControlsEnabled {
		return
	}

	style := doc.Call("createElement", "style")
	style.Set("textContent", `
#mobileControlSelector {
	position: fixed;
	inset: 0;
	z-index: 100000;
	display: none;
	align-items: flex-start;
	justify-content: center;
	box-sizing: border-box;
	padding:
		max(18px, env(safe-area-inset-top))
		max(18px, env(safe-area-inset-right))
		max(18px, env(safe-area-inset-bottom))
		max(18px, env(safe-area-inset-left));
	overflow-x: hidden;
	overflow-y: auto;
	-webkit-overflow-scrolling: touch;
	overscroll-behavior: contain;
	background: rgba(8, 10, 24, 0.94);
	font-family: GameFont, monospace;
}
#mobileControlPanel {
	box-sizing: border-box;
	width: min(620px, 96vw);
	max-width: 100%;
	margin: auto 0;
	padding: 20px;
	border: 1px solid rgba(255,255,255,0.22);
	border-radius: 12px;
	background: #16213e;
	box-shadow: 0 12px 40px rgba(0,0,0,0.45);
	color: white;
}
#mobileControlPanel h2 {
	margin: 0 0 8px;
	text-align: center;
	font-size: 24px;
}
#mobileControlPanel p {
	margin: 0 0 16px;
	text-align: center;
	color: rgba(255,255,255,0.68);
	font-size: 14px;
}
.mobileControlChoice {
	display: block;
	width: 100%;
	margin: 9px 0;
	padding: 12px 14px;
	border: 1px solid rgba(255,255,255,0.18);
	border-radius: 8px;
	background: rgba(255,255,255,0.07);
	color: white;
	text-align: left;
	font: 700 16px/1.25 GameFont, monospace;
	touch-action: manipulation;
}
.mobileControlChoice small {
	display: block;
	margin-top: 4px;
	color: rgba(255,255,255,0.60);
	font: 700 12px/1.3 GameFont, monospace;
}
.mobileControlChoice:active {
	background: rgba(255,255,255,0.18);
}
`)
	doc.Get("head").Call("appendChild", style)

	overlay := doc.Call("createElement", "div")
	overlay.Set("id", "mobileControlSelector")
	overlay.Set("innerHTML", `
<div id="mobileControlPanel">
	<h2>Choose controls</h2>
	<p>You can change this later with the Controls button.</p>
	<button class="mobileControlChoice" data-mode="vertical">
		Up / down drag
		<small>Slide one finger vertically. Down moves right; up moves left.</small>
	</button>
	<button class="mobileControlChoice" data-mode="tilt">
		Phone tilt
		<small>Rotate the phone left or right. iPhone/iPad will ask for motion permission.</small>
	</button>
	<button class="mobileControlChoice" data-mode="follow">
		Follow finger
		<small>The paddle follows one finger's horizontal position.</small>
	</button>
	<button class="mobileControlChoice" data-mode="two-thumb">
		Two thumbs
		<small>Hold the left or right half of the screen to move.</small>
	</button>
</div>
`)
	doc.Get("body").Call("appendChild", overlay)

	settingsButton := doc.Call("getElementById", "mobileControlModeButton")
	if !settingsButton.IsUndefined() && !settingsButton.IsNull() {
		settingsCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("preventDefault")
				args[0].Call("stopPropagation")
			}
			showMobileControlSelector()
			return nil
		})
		mobileControlCallbacks = append(mobileControlCallbacks, settingsCallback)
		settingsButton.Call("addEventListener", "pointerdown", settingsCallback)
	}

	buttons := overlay.Call("querySelectorAll", ".mobileControlChoice")
	for i := 0; i < buttons.Get("length").Int(); i++ {
		button := buttons.Index(i)
		mode := button.Get("dataset").Get("mode").String()

		callback := js.FuncOf(func(selectedMode string) func(js.Value, []js.Value) interface{} {
			return func(this js.Value, args []js.Value) interface{} {
				if len(args) > 0 {
					args[0].Call("preventDefault")
					args[0].Call("stopPropagation")
				}
				selectMobileControlMode(selectedMode)
				if selectedMode == "tilt" {
					// iOS requires the permission request to run directly from a
					// completed user gesture. A click is more reliable than pointerdown.
					requestPhoneTiltPermission()
				}
				hideMobileControlSelector()
				return nil
			}
		}(mode))

		mobileControlCallbacks = append(mobileControlCallbacks, callback)
		button.Call("addEventListener", "click", callback)
	}

	savedMode := ""
	func() {
		defer func() { _ = recover() }()
		storage := js.Global().Get("localStorage")
		if storage.IsUndefined() || storage.IsNull() {
			return
		}
		value := storage.Call("getItem", mobileControlStorageKey)
		if !value.IsUndefined() && !value.IsNull() {
			savedMode = value.String()
		}
	}()

	if validMobileControlMode(savedMode) {
		selectMobileControlMode(savedMode)
		if savedMode == "tilt" {
			// iOS permission cannot be requested during startup. Show the chooser
			// again so a completed click can grant or refresh motion access.
			showMobileControlSelector()
		}
	} else {
		showMobileControlSelector()
	}
}

// ---- Phone/tablet tilt paddle control ----
func screenOrientationAngle() int {
	screen := js.Global().Get("screen")
	if screen.IsUndefined() || screen.IsNull() {
		return 0
	}

	orientation := screen.Get("orientation")
	if !orientation.IsUndefined() && !orientation.IsNull() {
		angle := orientation.Get("angle")
		if angle.Type() == js.TypeNumber {
			return angle.Int()
		}
	}

	legacy := js.Global().Get("orientation")
	if legacy.Type() == js.TypeNumber {
		return legacy.Int()
	}

	return 0
}

func orientationTiltValue(event js.Value) (float64, bool) {
	betaValue := event.Get("beta")
	gammaValue := event.Get("gamma")
	if betaValue.Type() != js.TypeNumber || gammaValue.Type() != js.TypeNumber {
		return 0, false
	}

	beta := betaValue.Float()
	gamma := gammaValue.Float()

	switch ((screenOrientationAngle() % 360) + 360) % 360 {
	case 90:
		return beta, true
	case 270:
		return -beta, true
	default:
		return gamma, true
	}
}

func installPhoneTiltListener() {
	if phoneTiltListenerSet {
		return
	}
	phoneTiltListenerSet = true

	deviceOrientationCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}

		value, ok := orientationTiltValue(args[0])
		if !ok || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil
		}

		if !phoneTiltCalibrated {
			phoneTiltNeutral = value
			phoneTiltValue = 0
			phoneTiltCalibrated = true
		} else {
			phoneTiltValue = value - phoneTiltNeutral
		}

		phoneTiltAvailable = true
		return nil
	})

	js.Global().Call("addEventListener", "deviceorientation", deviceOrientationCallback)
}

func requestPhoneTiltPermission() {
	window := js.Global().Get("window")
	secureContext := window.Get("isSecureContext")
	if secureContext.Type() == js.TypeBoolean && !secureContext.Bool() {
		phoneTiltPermissionAsked = false
		showStatus("Tilt needs HTTPS", 3.0)
		return
	}

	orientationEvent := window.Get("DeviceOrientationEvent")
	if orientationEvent.IsUndefined() || orientationEvent.IsNull() {
		orientationEvent = js.Global().Get("DeviceOrientationEvent")
	}
	if orientationEvent.IsUndefined() || orientationEvent.IsNull() {
		showStatus("Tilt unavailable", 2.0)
		return
	}

	requestPermission := orientationEvent.Get("requestPermission")
	if requestPermission.Type() != js.TypeFunction {
		installPhoneTiltListener()
		recalibratePhoneTilt()
		showStatus("Tilt ready", 1.5)
		return
	}

	if phoneTiltListenerSet {
		recalibratePhoneTilt()
		showStatus("Tilt recalibrated", 1.5)
		return
	}
	if phoneTiltPermissionAsked {
		return
	}

	phoneTiltPermissionAsked = true
	promise := js.Undefined()
	callSucceeded := false
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				phoneTiltPermissionAsked = false
				log("Tilt permission request failed: " + fmt.Sprint(recovered))
			}
		}()
		promise = orientationEvent.Call("requestPermission")
		callSucceeded = true
	}()
	if !callSucceeded || promise.IsUndefined() || promise.IsNull() || promise.Type() != js.TypeObject {
		phoneTiltPermissionAsked = false
		showStatus("Tap again to allow tilt", 2.0)
		return
	}

	granted := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 && strings.EqualFold(args[0].String(), "granted") {
			installPhoneTiltListener()
			recalibratePhoneTilt()
			showStatus("Tilt ready", 1.5)
		} else {
			phoneTiltPermissionAsked = false
			showStatus("Tilt permission denied", 2.0)
		}
		return nil
	})
	rejected := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		phoneTiltPermissionAsked = false
		reason := "request rejected"
		if len(args) > 0 {
			reason = fmt.Sprint(args[0])
		}
		log("Tilt permission rejected: " + reason)
		showStatus("Tap again to allow tilt", 2.0)
		return nil
	})

	js.Global().Set("__breakoutTiltGranted", granted)
	js.Global().Set("__breakoutTiltRejected", rejected)
	promise.Call("then", granted).Call("catch", rejected)
}

func recalibratePhoneTilt() {
	phoneTiltCalibrated = false
	phoneTiltValue = 0
}

func applyPhoneTiltControl(dt float64) bool {
	if !phoneTiltAvailable || !phoneTiltCalibrated || dt <= 0 {
		return false
	}

	tilt := phoneTiltValue
	if math.Abs(tilt) <= defaultPhoneTiltDeadZone {
		tilt = 0
	} else {
		tilt = math.Copysign(math.Abs(tilt)-defaultPhoneTiltDeadZone, tilt)
	}

	usableRange := defaultPhoneTiltMaxAngle - defaultPhoneTiltDeadZone
	if usableRange <= 0 {
		return false
	}

	normalized := tilt / usableRange
	if normalized < -1 {
		normalized = -1
	}
	if normalized > 1 {
		normalized = 1
	}

	availableWidth := canvasWidth - paddle.w
	phoneTiltTargetX = (normalized + 1) * 0.5 * availableWidth

	oldX := paddle.x
	smoothing := 1 - math.Exp(-defaultPhoneTiltSmoothing*dt)
	paddle.x += (phoneTiltTargetX - paddle.x) * smoothing
	paddle.vx = (paddle.x - oldX) / dt

	return true
}

// applyMousePaddleControl places the paddle at the latest mouse target in one
// physics step. The shared post-control velocity measurement below still derives
// paddle.vx from the real position delta, so paddle-to-ball spin transfer remains
// active even though the visible position no longer eases toward the cursor.
func applyMousePaddleControl(dt float64) bool {
	if !mouseControlActive || dt <= 0 {
		return false
	}

	maxX := math.Max(0, canvasWidth-paddle.w)
	mousePaddleTargetX = clampFloat(mousePaddleTargetX, 0, maxX)
	paddle.x = mousePaddleTargetX
	return true
}

func normalizedGamepadAxis(value float64) float64 {
	value = clampFloat(value, -1, 1)
	magnitude := math.Abs(value)
	if magnitude <= defaultGamepadDeadZone {
		return 0
	}

	usableRange := 1 - defaultGamepadDeadZone
	if usableRange <= 0 {
		return math.Copysign(1, value)
	}

	normalized := (magnitude - defaultGamepadDeadZone) / usableRange
	return math.Copysign(clampFloat(normalized, 0, 1), value)
}

func gamepadButtonPressed(button js.Value) bool {
	if button.IsUndefined() || button.IsNull() {
		return false
	}

	if button.Type() == js.TypeNumber {
		return button.Float() >= 0.5
	}

	pressed := button.Get("pressed")
	if !pressed.IsUndefined() && !pressed.IsNull() && pressed.Type() == js.TypeBoolean {
		return pressed.Bool()
	}

	value := button.Get("value")
	return !value.IsUndefined() && !value.IsNull() && value.Type() == js.TypeNumber && value.Float() >= 0.5
}

func handleGamepadFirePress() {
	if physicsEditorVisible || gameOver {
		return
	}
	if levelAdvancePending {
		advanceFromLevelComplete()
		return
	}

	// B0 starts a level from its READY/title screen. Gamepad polling is not a
	// guaranteed browser user-activation event, but attempting to resume audio is
	// harmless and works in browsers that accept gamepad input for audio unlock.
	if waitingForStart {
		waitingForStart = false
		paused = false
		leftPressed = false
		rightPressed = false
		mobileLeftHeld = false
		mobileRightHeld = false
		touchControlActive = false
		paddle.vx = 0
		resetPaddleSpinHistory()
		syncRenderInterpolation()
		unlockAudioFromGesture()
		showStatus("Go!", 1.0)
		return
	}

	paused = !paused
	leftPressed = false
	rightPressed = false
	mobileLeftHeld = false
	mobileRightHeld = false
	touchControlActive = false
	paddle.vx = 0
	resetPaddleSpinHistory()
	syncRenderInterpolation()
}

func keepGamepadFullscreenCallback(callback js.Func) {
	gamepadFullscreenCallbacks = append(gamepadFullscreenCallbacks, callback)
}

func trackFullscreenPromise(promise js.Value, action string) {
	if promise.IsUndefined() || promise.IsNull() || promise.Type() != js.TypeObject {
		return
	}
	catchMethod := promise.Get("catch")
	if catchMethod.IsUndefined() || catchMethod.IsNull() || catchMethod.Type() != js.TypeFunction {
		return
	}

	failureCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		reason := "request rejected"
		if len(args) > 0 {
			reason = fmt.Sprint(args[0])
		}
		log("Fullscreen " + action + " failed: " + reason)
		if appleMobileBrowser() {
			showIOSStandaloneHint()
		}
		return nil
	})
	keepGamepadFullscreenCallback(failureCallback)
	promise.Call("catch", failureCallback)
}

func fullscreenElement() js.Value {
	element := doc.Get("fullscreenElement")
	if !element.IsUndefined() && !element.IsNull() {
		return element
	}
	element = doc.Get("webkitFullscreenElement")
	if !element.IsUndefined() && !element.IsNull() {
		return element
	}
	return js.Null()
}

func callFullscreenMethod(target js.Value, standardMethod, webkitMethod, action string) bool {
	if target.IsUndefined() || target.IsNull() {
		return false
	}

	method := standardMethod
	fn := target.Get(method)
	if fn.IsUndefined() || fn.IsNull() || fn.Type() != js.TypeFunction {
		method = webkitMethod
		fn = target.Get(method)
	}
	if fn.IsUndefined() || fn.IsNull() || fn.Type() != js.TypeFunction {
		return false
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			log("Fullscreen " + action + " panic: " + fmt.Sprint(recovered))
			showStatus("Fullscreen unavailable", 2.0)
		}
	}()
	promise := target.Call(method)
	trackFullscreenPromise(promise, action)
	return true
}

func handleGamepadFullscreenPress() {
	// Fullscreen the document rather than only the canvas so the physics tuner and
	// any surrounding controls remain visible. B1 is edge-triggered in the poller.
	if !fullscreenElement().IsNull() {
		if !callFullscreenMethod(doc, "exitFullscreen", "webkitExitFullscreen", "exit") {
			showStatus("Fullscreen unavailable", 2.0)
		}
		return
	}

	target := doc.Get("documentElement")
	if !callFullscreenMethod(target, "requestFullscreen", "webkitRequestFullscreen", "request") {
		if appleMobileBrowser() {
			showIOSStandaloneHint()
		} else {
			showStatus("Fullscreen unavailable", 2.0)
		}
	}
}

// pollGamepadInput reads the first connected controller with axis 0 available.
// This intentionally uses the raw mapping reported by the Vivanco USB device:
// axis 0 moves the paddle, B0 starts/toggles pause, and B1 toggles fullscreen.
func pollGamepadInput() {
	defer func() {
		if recovered := recover(); recovered != nil {
			gamepadAxisX = 0
		}
	}()

	navigator := js.Global().Get("navigator")
	getGamepads := navigator.Get("getGamepads")
	if getGamepads.IsUndefined() || getGamepads.IsNull() {
		gamepadAxisX = 0
		gamepadFireWasPressed = false
		gamepadFullscreenWasPressed = false
		return
	}

	gamepads := navigator.Call("getGamepads")
	selected := js.Null()
	selectedIndex := -1
	for index := 0; index < gamepads.Length(); index++ {
		gamepad := gamepads.Index(index)
		if gamepad.IsUndefined() || gamepad.IsNull() {
			continue
		}

		axes := gamepad.Get("axes")
		if axes.IsUndefined() || axes.IsNull() || axes.Length() <= defaultGamepadHorizontalAxis {
			continue
		}

		selected = gamepad
		selectedIndex = index
		break
	}

	if selectedIndex < 0 {
		if gamepadActiveIndex >= 0 {
			log("Gamepad disconnected")
		}
		gamepadActiveIndex = -1
		gamepadActiveID = ""
		gamepadAxisX = 0
		gamepadFireWasPressed = false
		gamepadFullscreenWasPressed = false
		return
	}

	id := selected.Get("id").String()
	if selectedIndex != gamepadActiveIndex || id != gamepadActiveID {
		gamepadActiveIndex = selectedIndex
		gamepadActiveID = id
		log(fmt.Sprintf("Gamepad connected: index=%d id=%s axis=%d fire=B%d fullscreen=B%d",
			selectedIndex, id, defaultGamepadHorizontalAxis, defaultGamepadFireButton,
			defaultGamepadFullscreenButton))
	}

	axisValue := selected.Get("axes").Index(defaultGamepadHorizontalAxis).Float()
	gamepadAxisX = normalizedGamepadAxis(axisValue)
	if gamepadAxisX != 0 {
		// Moving the stick takes control from the absolute-position mouse path.
		mouseControlActive = false
	}

	firePressed := false
	fullscreenPressed := false
	buttons := selected.Get("buttons")
	if !buttons.IsUndefined() && !buttons.IsNull() {
		if buttons.Length() > defaultGamepadFireButton {
			firePressed = gamepadButtonPressed(buttons.Index(defaultGamepadFireButton))
		}
		if buttons.Length() > defaultGamepadFullscreenButton {
			fullscreenPressed = gamepadButtonPressed(buttons.Index(defaultGamepadFullscreenButton))
		}
	}
	if firePressed && !gamepadFireWasPressed {
		handleGamepadFirePress()
	}
	if fullscreenPressed && !gamepadFullscreenWasPressed {
		handleGamepadFullscreenPress()
	}
	gamepadFireWasPressed = firePressed
	gamepadFullscreenWasPressed = fullscreenPressed
}

func clampDebrisSpeed(fragment *debrisFragment) {
	if fragment == nil || debrisMaxSpeed <= 0 {
		return
	}
	speed := math.Hypot(fragment.vx, fragment.vy)
	if speed <= debrisMaxSpeed || speed <= 0 {
		return
	}
	scale := debrisMaxSpeed / speed
	fragment.vx *= scale
	fragment.vy *= scale
}

func nearestMagneticBrickForDebris(x, y float64) (float64, float64, float64, bool) {
	if magnetRange <= 0 {
		return 0, 0, 0, false
	}

	bestDistanceSquared := magnetRange * magnetRange
	bestX, bestY := 0.0, 0.0
	found := false
	consider := func(index int) {
		if index < 0 || index >= len(bricks) {
			return
		}
		br := &bricks[index]
		if !br.alive || br.unbreakable {
			return
		}
		centerX := br.x + br.w/2
		centerY := br.y + br.h/2
		dx := centerX - x
		dy := centerY - y
		distanceSquared := dx*dx + dy*dy
		if distanceSquared > 1 && distanceSquared < bestDistanceSquared {
			bestDistanceSquared = distanceSquared
			bestX, bestY = centerX, centerY
			found = true
		}
	}

	if gridRows <= 0 || gridCols <= 0 || gridCellWidth <= 0 || gridCellHeight <= 0 {
		for i := range bricks {
			consider(i)
		}
	} else {
		minCol := int(math.Floor((x - magnetRange - gridOffsetLeft) / gridCellWidth))
		maxCol := int(math.Floor((x + magnetRange - gridOffsetLeft) / gridCellWidth))
		minRow := int(math.Floor((y - magnetRange - gridOffsetTop) / gridCellHeight))
		maxRow := int(math.Floor((y + magnetRange - gridOffsetTop) / gridCellHeight))
		minCol = max(0, minCol)
		maxCol = min(gridCols-1, maxCol)
		minRow = max(0, minRow)
		maxRow = min(gridRows-1, maxRow)
		direct := len(brickGridCells) == gridRows*gridCols
		for row := minRow; row <= maxRow; row++ {
			for col := minCol; col <= maxCol; col++ {
				key := gridKey(row, col)
				var indices []int
				if direct {
					indices = brickGridCells[key]
				} else {
					indices = brickGrid[key]
				}
				for _, index := range indices {
					consider(index)
				}
			}
		}
	}

	if !found {
		return 0, 0, 0, false
	}
	return bestX, bestY, math.Sqrt(bestDistanceSquared), true
}

func applyDebrisFields(fragment *debrisFragment, dt, linearDamping, angularDamping float64, applyMagnet bool) {
	fragment.vy += currentGravity * debrisGravityScale * dt

	if blackHoleActive && debrisFieldScale > 0 {
		dx := blackHoleX - fragment.x
		dy := blackHoleY - fragment.y
		distance := math.Hypot(dx, dy)
		if distance > 1 {
			acceleration := blackHoleStrength * debrisFieldScale
			fragment.vx += dx / distance * acceleration * dt
			fragment.vy += dy / distance * acceleration * dt
		} else {
			// Avoid a zero-length normal without consuming gameplay randomness.
			direction := sign(fragment.omega)
			if direction == 0 {
				direction = 1
			}
			fragment.vx += direction * blackHoleStrength * debrisFieldScale * 0.05 * dt
		}
	}

	if applyMagnet && magnetIsActive() && debrisMagnetScale > 0 {
		if targetX, targetY, distance, found := nearestMagneticBrickForDebris(fragment.x, fragment.y); found {
			dx := targetX - fragment.x
			dy := targetY - fragment.y
			falloff := clampFloat(1-distance/math.Max(1, magnetRange), 0, 1)
			// Nearest-brick lookup is staggered across four fixed steps. Multiplying
			// the impulse by four preserves the average 240 Hz force while reducing
			// the expensive search work to 60 Hz per fragment.
			acceleration := magnetStrength * debrisMagnetScale * falloff
			fragment.vx += dx / distance * acceleration * dt * 4
			fragment.vy += dy / distance * acceleration * dt * 4
		}
	}

	fragment.vx *= linearDamping
	fragment.vy *= linearDamping
	fragment.omega *= angularDamping
	if math.Abs(fragment.omega) < debrisAngularStopSpeed {
		fragment.omega = 0
	}
	clampDebrisSpeed(fragment)
}

func circleAABBContact(
	x, y, radius,
	left, top, right, bottom float64,
) (bool, float64, float64, float64) {
	closestX := clampFloat(x, left, right)
	closestY := clampFloat(y, top, bottom)
	dx := x - closestX
	dy := y - closestY
	distanceSquared := dx*dx + dy*dy
	if distanceSquared > 0 {
		if distanceSquared >= radius*radius {
			return false, 0, 0, 0
		}
		distance := math.Sqrt(distanceSquared)
		return true, dx / distance, dy / distance, radius - distance
	}

	// The centre is inside the box. Push toward the nearest side.
	leftDistance := x - left
	rightDistance := right - x
	topDistance := y - top
	bottomDistance := bottom - y
	nx, ny := -1.0, 0.0
	nearest := leftDistance
	if rightDistance < nearest {
		nx, ny = 1, 0
		nearest = rightDistance
	}
	if topDistance < nearest {
		nx, ny = 0, -1
		nearest = topDistance
	}
	if bottomDistance < nearest {
		nx, ny = 0, 1
		nearest = bottomDistance
	}
	return true, nx, ny, radius + math.Max(0, nearest)
}

func resolveDebrisSurfaceCollision(
	fragment *debrisFragment,
	nx, ny, penetration, surfaceVx, surfaceVy float64,
) {
	fragment.x += nx * (penetration + physicsConfig.collisionSlop)
	fragment.y += ny * (penetration + physicsConfig.collisionSlop)

	relativeVx := fragment.vx - surfaceVx
	relativeVy := fragment.vy - surfaceVy
	normalVelocity := relativeVx*nx + relativeVy*ny
	if normalVelocity >= 0 {
		return
	}

	fragment.vx -= (1 + debrisRestitution) * normalVelocity * nx
	fragment.vy -= (1 + debrisRestitution) * normalVelocity * ny

	tangentX, tangentY := -ny, nx
	tangentVelocity := (fragment.vx-surfaceVx)*tangentX + (fragment.vy-surfaceVy)*tangentY
	frictionDelta := -tangentVelocity * debrisFriction
	fragment.vx += frictionDelta * tangentX
	fragment.vy += frictionDelta * tangentY
	if fragment.radius > 0 {
		fragment.omega -= frictionDelta / fragment.radius * 0.35
	}
	clampDebrisSpeed(fragment)
}

func collideDebrisWithWalls(fragment *debrisFragment) {
	if fragment.x-fragment.radius < 0 {
		resolveDebrisSurfaceCollision(fragment, 1, 0, fragment.radius-fragment.x, 0, 0)
	}
	if fragment.x+fragment.radius > canvasWidth {
		resolveDebrisSurfaceCollision(fragment, -1, 0, fragment.x+fragment.radius-canvasWidth, 0, 0)
	}
	if fragment.y-fragment.radius < 0 {
		resolveDebrisSurfaceCollision(fragment, 0, 1, fragment.radius-fragment.y, 0, 0)
	}
	// There is intentionally no bottom wall. Fragments fall out and are removed.
}

func collideDebrisWithPaddle(fragment *debrisFragment) {
	hit, nx, ny, penetration := circleAABBContact(
		fragment.x, fragment.y, fragment.radius,
		paddle.x, paddle.y, paddle.x+paddle.w, paddle.y+paddle.h,
	)
	if !hit {
		return
	}
	resolveDebrisSurfaceCollision(fragment, nx, ny, penetration, paddle.vx, 0)
}

func collideDebrisWithBall(fragment *debrisFragment, b *Ball) {
	if b == nil || b.r <= 0 || b.y-b.r > canvasHeight {
		return
	}
	dx := b.x - fragment.x
	dy := b.y - fragment.y
	minimumDistance := b.r + fragment.radius
	distanceSquared := dx*dx + dy*dy
	if distanceSquared >= minimumDistance*minimumDistance {
		return
	}

	distance := math.Sqrt(distanceSquared)
	nx, ny := 0.0, -1.0
	if distance > 0.000001 {
		nx, ny = dx/distance, dy/distance
	} else {
		relativeLength := math.Hypot(b.vx-fragment.vx, b.vy-fragment.vy)
		if relativeLength > 0.000001 {
			nx = (b.vx - fragment.vx) / relativeLength
			ny = (b.vy - fragment.vy) / relativeLength
		}
	}

	penetration := minimumDistance - distance
	// The small fragment takes most of the positional correction so the ball does
	// not visibly jump when it touches a cloud of debris.
	fragment.x -= nx * penetration * 0.82
	fragment.y -= ny * penetration * 0.82
	b.x += nx * penetration * 0.18
	b.y += ny * penetration * 0.18

	relativeVx := b.vx - fragment.vx
	relativeVy := b.vy - fragment.vy
	normalVelocity := relativeVx*nx + relativeVy*ny
	if normalVelocity >= 0 {
		return
	}

	fragmentMass := math.Max(0.05, fragment.mass)
	ballMass := 1.0
	restitution := math.Min(physicsConfig.restitution, debrisRestitution)
	impulse := -(1 + restitution) * normalVelocity / (1/ballMass + 1/fragmentMass)
	ballImpulse := impulse * debrisBallInfluence
	fragmentImpulse := impulse

	b.vx += nx * ballImpulse / ballMass
	b.vy += ny * ballImpulse / ballMass
	fragment.vx -= nx * fragmentImpulse / fragmentMass
	fragment.vy -= ny * fragmentImpulse / fragmentMass

	// Relative tangential motion gives the ball a restrained spin nudge. The main
	// direction change remains the speed-dependent normal impulse above.
	tangentX, tangentY := -ny, nx
	tangentVelocity := relativeVx*tangentX + relativeVy*tangentY
	if b.r > 0 {
		b.omega += tangentVelocity * debrisBallInfluence * 0.06 / b.r
		b.omega = clampFloat(b.omega, -physicsConfig.maxSpin, physicsConfig.maxSpin)
	}

	maximumBallSpeed := math.Max(physicsConfig.maxSpeed, physicsConfig.maxSpeed*1.35)
	ballSpeed := math.Hypot(b.vx, b.vy)
	if maximumBallSpeed > 0 && ballSpeed > maximumBallSpeed {
		scale := maximumBallSpeed / ballSpeed
		b.vx *= scale
		b.vy *= scale
	}
	clampDebrisSpeed(fragment)
	fragment.omega -= tangentVelocity * 0.025 / math.Max(1, fragment.radius)
	resetFastOrbitCandidate(b)
	recordMeasuredBallSpeed(b)
	recordMeasuredBallSpin(b)
}

func collideDebrisWithLivingBricks(fragment *debrisFragment) {
	if fragment.age < debrisBrickCollisionDelay {
		return
	}

	collided := false
	consider := func(index int) {
		if collided || index < 0 || index >= len(bricks) {
			return
		}
		br := &bricks[index]
		if !br.alive {
			return
		}
		hit, nx, ny, penetration := circleAABBContact(
			fragment.x, fragment.y, fragment.radius,
			br.x, br.y, br.x+br.w, br.y+br.h,
		)
		if !hit {
			return
		}
		resolveDebrisSurfaceCollision(fragment, nx, ny, penetration, 0, 0)
		collided = true
	}

	if gridRows <= 0 || gridCols <= 0 || gridCellWidth <= 0 || gridCellHeight <= 0 {
		for i := range bricks {
			consider(i)
			if collided {
				return
			}
		}
		return
	}

	minCol := int(math.Floor((fragment.x-fragment.radius-gridOffsetLeft)/gridCellWidth)) - 1
	maxCol := int(math.Floor((fragment.x+fragment.radius-gridOffsetLeft)/gridCellWidth)) + 1
	minRow := int(math.Floor((fragment.y-fragment.radius-gridOffsetTop)/gridCellHeight)) - 1
	maxRow := int(math.Floor((fragment.y+fragment.radius-gridOffsetTop)/gridCellHeight)) + 1
	minCol = max(0, minCol)
	maxCol = min(gridCols-1, maxCol)
	minRow = max(0, minRow)
	maxRow = min(gridRows-1, maxRow)
	for row := minRow; row <= maxRow && !collided; row++ {
		for col := minCol; col <= maxCol && !collided; col++ {
			for _, index := range brickGrid[gridKey(row, col)] {
				consider(index)
				if collided {
					break
				}
			}
		}
	}
}

func debrisIsOffscreen(fragment *debrisFragment) bool {
	margin := debrisOffscreenMargin
	return fragment.y-fragment.radius > canvasHeight+margin ||
		fragment.x+fragment.radius < -margin ||
		fragment.x-fragment.radius > canvasWidth+margin ||
		fragment.y+fragment.radius < -margin
}

func updateBrickDebris(dt float64) {
	if len(brickDebris) == 0 || dt <= 0 {
		return
	}

	debrisFieldTick++
	// These factors are identical for every fragment in this fixed step.
	linearDamping := math.Exp(-debrisAirDrag * dt)
	angularDamping := math.Exp(-debrisAngularDrag * dt)
	kept := 0
	for i := range brickDebris {
		fragment := &brickDebris[i]
		fragment.previousX = fragment.x
		fragment.previousY = fragment.y
		fragment.previousAngle = fragment.angle
		fragment.age += dt
		if fragment.age >= fragment.lifetime {
			continue
		}

		applyDebrisFields(fragment, dt, linearDamping, angularDamping, (debrisFieldTick+i)%4 == 0)
		fragment.x += fragment.vx * dt
		fragment.y += fragment.vy * dt
		fragment.angle += fragment.omega * dt

		collideDebrisWithWalls(fragment)
		collideDebrisWithPaddle(fragment)
		collideDebrisWithLivingBricks(fragment)
		forEachActiveBall(func(b *Ball, _ int) {
			collideDebrisWithBall(fragment, b)
		})

		if debrisIsOffscreen(fragment) {
			continue
		}
		// Keep the original index for field staggering and preserve fragment order.
		// Only copy a survivor when an earlier fragment has actually been removed.
		if kept != i {
			brickDebris[kept] = *fragment
		}
		kept++
	}
	clear(brickDebris[kept:])
	brickDebris = brickDebris[:kept]
}

type debrisRGB struct {
	r, g, b float64
}

func parseResolvedDebrisColor(value string) (debrisRGB, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if strings.HasPrefix(value, "#") {
		hex := strings.TrimPrefix(value, "#")
		if len(hex) == 3 || len(hex) == 4 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		if len(hex) == 6 || len(hex) == 8 {
			n, err := strconv.ParseUint(hex[:6], 16, 24)
			if err == nil {
				return debrisRGB{
					r: float64((n >> 16) & 0xff),
					g: float64((n >> 8) & 0xff),
					b: float64(n & 0xff),
				}, true
			}
		}
	}
	if strings.HasPrefix(value, "rgb(") || strings.HasPrefix(value, "rgba(") {
		start := strings.IndexByte(value, '(')
		end := strings.LastIndexByte(value, ')')
		if start >= 0 && end > start {
			parts := strings.Split(value[start+1:end], ",")
			if len(parts) >= 3 {
				components := [3]float64{}
				for i := 0; i < 3; i++ {
					part := strings.TrimSpace(parts[i])
					if strings.HasSuffix(part, "%") {
						percent, err := strconv.ParseFloat(strings.TrimSuffix(part, "%"), 64)
						if err != nil {
							return debrisRGB{}, false
						}
						components[i] = clampFloat(percent, 0, 100) * 2.55
					} else {
						component, err := strconv.ParseFloat(part, 64)
						if err != nil {
							return debrisRGB{}, false
						}
						components[i] = clampFloat(component, 0, 255)
					}
				}
				return debrisRGB{r: components[0], g: components[1], b: components[2]}, true
			}
		}
	}
	return debrisRGB{}, false
}

func resolveDebrisCSSColor(value string) (debrisRGB, bool) {
	if ctx.IsUndefined() || ctx.IsNull() {
		return parseResolvedDebrisColor(value)
	}
	previous := ctx.Get("fillStyle").String()
	ctx.Set("fillStyle", "#010203")
	ctx.Set("fillStyle", value)
	resolved := ctx.Get("fillStyle").String()
	ctx.Set("fillStyle", previous)
	return parseResolvedDebrisColor(resolved)
}

func opaqueDebrisPalette(fillColor, backgroundColor string) []string {
	key := fillColor + "\x00" + backgroundColor
	palette, ok := debrisOpaqueColorCache[key]
	if ok {
		return palette
	}

	fill, fillOK := resolveDebrisCSSColor(fillColor)
	background, backgroundOK := resolveDebrisCSSColor(backgroundColor)
	palette = make([]string, 256)
	if fillOK && backgroundOK {
		for i := 0; i < 256; i++ {
			a := float64(i) / 255.0
			r := int(math.Round(background.r + (fill.r-background.r)*a))
			g := int(math.Round(background.g + (fill.g-background.g)*a))
			b := int(math.Round(background.b + (fill.b-background.b)*a))
			palette[i] = "rgb(" + strconv.Itoa(r) + "," + strconv.Itoa(g) + "," + strconv.Itoa(b) + ")"
		}
	} else {
		for i := 0; i < 256; i++ {
			percent := strconv.FormatFloat(float64(i)*100.0/255.0, 'f', 2, 64)
			palette[i] = "color-mix(in srgb, " + fillColor + " " + percent + "%, " + backgroundColor + ")"
		}
	}
	debrisOpaqueColorCache[key] = palette
	return palette
}

func opaqueDebrisColorFromPalette(palette []string, opacity float64) string {
	if len(palette) == 0 {
		return "#000000"
	}
	if len(palette) != 256 {
		return palette[0]
	}
	level := int(math.Round(clampFloat(opacity, 0, 1) * 255))
	return palette[level]
}

func traceDebrisShape(target js.Value, fragment *debrisFragment) {
	if fragment.circle {
		target.Call("arc", 0, 0, fragment.radius, 0, 2*math.Pi)
	} else if fragment.roundness <= 0.001 {
		target.Call("moveTo", fragment.points[0], fragment.points[1])
		for point := 1; point < fragment.pointCount; point++ {
			target.Call("lineTo", fragment.points[point*2], fragment.points[point*2+1])
		}
	} else {
		cornerFraction := 0.08 + clampFloat(fragment.roundness, 0, 1)*0.34
		last := fragment.pointCount - 1
		startX := fragment.points[last*2] + (fragment.points[0]-fragment.points[last*2])*(1-cornerFraction)
		startY := fragment.points[last*2+1] + (fragment.points[1]-fragment.points[last*2+1])*(1-cornerFraction)
		target.Call("moveTo", startX, startY)
		for point := 0; point < fragment.pointCount; point++ {
			next := (point + 1) % fragment.pointCount
			currentX := fragment.points[point*2]
			currentY := fragment.points[point*2+1]
			outX := currentX + (fragment.points[next*2]-currentX)*cornerFraction
			outY := currentY + (fragment.points[next*2+1]-currentY)*cornerFraction
			target.Call("quadraticCurveTo", currentX, currentY, outX, outY)
		}
	}
	target.Call("closePath")
}

func ensureDebrisPath2DSupport() bool {
	if debrisPath2DChecked {
		return debrisPath2DSupported
	}
	debrisPath2DChecked = true
	if !defaultDebrisUsePath2DCache {
		return false
	}
	constructor := js.Global().Get("Path2D")
	if constructor.IsUndefined() || constructor.IsNull() || constructor.Type() != js.TypeFunction {
		return false
	}
	debrisPath2DConstructor = constructor
	debrisPath2DSupported = true
	return true
}

func rebuildDebrisRenderPath(fragment *debrisFragment) {
	if fragment == nil {
		return
	}
	fragment.renderPathReady = false
	if (!fragment.circle && fragment.pointCount < 3) || !ensureDebrisPath2DSupport() {
		return
	}
	defer func() {
		if recover() != nil {
			debrisPath2DSupported = false
			fragment.renderPathReady = false
		}
	}()
	path := debrisPath2DConstructor.New()
	traceDebrisShape(path, fragment)
	fragment.renderPath = path
	fragment.renderPathReady = true
}

func updateDebrisAdaptiveRenderStride() {
	if !defaultDebrisAdaptiveRendering || defaultDebrisAdaptiveMaxStride <= 1 {
		debrisAdaptiveRenderStride = 1
		return
	}
	if fpsCurrent <= 0 {
		return
	}

	// Learn the machine/display ceiling only while no debris is present. This keeps
	// a genuine 30 Hz display from being mistaken for a slow 60 Hz machine.
	if len(brickDebris) == 0 {
		if debrisAdaptiveBaselineFPS <= 0 {
			debrisAdaptiveBaselineFPS = fpsCurrent
		} else {
			debrisAdaptiveBaselineFPS = debrisAdaptiveBaselineFPS*0.85 + fpsCurrent*0.15
		}
		debrisAdaptiveRenderStride = 1
		return
	}

	baseline := debrisAdaptiveBaselineFPS
	if baseline <= 0 {
		baseline = defaultDebrisAdaptiveTargetFPS
	}
	target := math.Min(defaultDebrisAdaptiveTargetFPS, baseline*0.92)
	target = math.Max(20, target)
	ratio := fpsCurrent / target
	maximumStride := max(1, defaultDebrisAdaptiveMaxStride)

	switch {
	case ratio < 0.65:
		debrisAdaptiveRenderStride = maximumStride
	case ratio < 0.85:
		debrisAdaptiveRenderStride = min(2, maximumStride)
	case ratio >= 0.95:
		debrisAdaptiveRenderStride = 1
	case ratio >= 0.90 && debrisAdaptiveRenderStride > 2:
		debrisAdaptiveRenderStride = 2
	}
}

func shouldDrawDebrisFragment(fragment *debrisFragment, speedSquared float64) bool {
	stride := debrisAdaptiveRenderStride
	if stride <= 1 || !defaultDebrisAdaptiveRendering {
		return true
	}
	if fragment.age < defaultDebrisAdaptiveFreshSeconds {
		return true
	}
	alwaysDrawSpeedSquared := defaultDebrisAdaptiveAlwaysDrawSpeed * defaultDebrisAdaptiveAlwaysDrawSpeed
	if speedSquared >= alwaysDrawSpeedSquared {
		return true
	}
	return fragment.renderID%uint32(stride) == 0
}

func debrisOpacity(fragment *debrisFragment) float64 {
	remaining := fragment.lifetime - fragment.age
	if remaining <= 0 {
		return 0
	}

	opacity := fragment.startOpacity
	if debrisFlashDuration > 0 && fragment.age < debrisFlashDuration {
		t := clampFloat(fragment.age/debrisFlashDuration, 0, 1)
		t = t * t * (3 - 2*t)
		opacity = debrisFlashOpacity + (fragment.startOpacity-debrisFlashOpacity)*t
	}
	if debrisFadeDuration > 0 && remaining < debrisFadeDuration {
		opacity *= clampFloat(remaining/debrisFadeDuration, 0, 1)
	}
	return clampFloat(opacity, 0, 1)
}

func drawBrickDebris(alpha float64, frontLayer bool) {
	if len(brickDebris) == 0 {
		return
	}
	alpha = clampFloat(alpha, 0, 1)
	thresholdSquared := debrisFrontLayerSpeed * debrisFrontLayerSpeed
	ctx.Call("save")
	for i := range brickDebris {
		fragment := &brickDebris[i]
		speedSquared := fragment.vx*fragment.vx + fragment.vy*fragment.vy
		fragmentFrontLayer := debrisFrontLayerSpeed <= 0 || speedSquared >= thresholdSquared
		if fragmentFrontLayer != frontLayer {
			continue
		}
		if !shouldDrawDebrisFragment(fragment, speedSquared) {
			debrisSkippedLastFrame++
			continue
		}
		opacity := debrisOpacity(fragment)
		if opacity <= 0 || (!fragment.circle && fragment.pointCount < 3) {
			continue
		}
		x := lerpFloat(fragment.previousX, fragment.x, alpha)
		y := lerpFloat(fragment.previousY, fragment.y, alpha)
		angle := lerpFloat(fragment.previousAngle, fragment.angle, alpha)
		cosAngle := 1.0
		sinAngle := 0.0
		if !fragment.circle {
			cosAngle = math.Cos(angle)
			sinAngle = math.Sin(angle)
		}

		fillStyle := opaqueDebrisColorFromPalette(fragment.renderColorPalette, opacity)
		// Send one compact shape instance instead of replaying every path vertex
		// through JSON. Native Qt caches the immutable outline by render ID.
		ctx.Call("debris", fragment.renderID, x, y, cosAngle, sinAngle, fillStyle,
			fragment.radius, fragment.circle, fragment.roundness, fragment.points[:fragment.pointCount*2])
		debrisRenderedLastFrame++
	}
	ctx.Call("restore")
}

func reflectCoordinate(value, minimum, maximum float64) float64 {
	width := maximum - minimum
	if width <= 0 {
		return minimum
	}
	period := 2 * width
	position := math.Mod(value-minimum, period)
	if position < 0 {
		position += period
	}
	if position > width {
		position = period - position
	}
	return minimum + position
}

func predictedPaddleCrossing(b *Ball) (x, seconds float64, ok bool) {
	if b == nil {
		return 0, 0, false
	}
	targetY := paddle.y - b.r
	c := b.y - targetY
	var candidates [2]float64
	candidateCount := 0
	if math.Abs(currentGravity) < 1e-9 {
		if math.Abs(b.vy) < 1e-9 {
			return 0, 0, false
		}
		t := -c / b.vy
		if t > 0 {
			candidates[candidateCount] = t
			candidateCount++
		}
	} else {
		a := 0.5 * currentGravity
		discriminant := b.vy*b.vy - 4*a*c
		if discriminant < 0 {
			return 0, 0, false
		}
		root := math.Sqrt(discriminant)
		for _, t := range []float64{(-b.vy - root) / (2 * a), (-b.vy + root) / (2 * a)} {
			if t > 0 {
				candidates[candidateCount] = t
				candidateCount++
			}
		}
	}
	if candidateCount == 0 {
		return 0, 0, false
	}
	seconds = candidates[0]
	for i := 1; i < candidateCount; i++ {
		if candidates[i] < seconds {
			seconds = candidates[i]
		}
	}
	if seconds <= 0 || seconds > defaultAutoPaddlePredictionMaxSeconds {
		return 0, 0, false
	}
	unfoldedX := b.x + b.vx*seconds
	x = reflectCoordinate(unfoldedX, b.r, canvasWidth-b.r)
	return x, seconds, true
}

func resetAutoPaddleHitPlan() {
	autoPaddleHitOffset = 0
	autoPaddleTargetBall = 0
	autoPaddleNeedsNewHitOffset = true
}

func chooseAutoPaddleHitOffset() float64 {
	maximum := clampFloat(autoPaddleHitVariation, 0, 0.90)
	if maximum <= 0 {
		return 0
	}
	return (autoPaddleRNG.Float64()*2 - 1) * maximum
}

func autoPaddleAimX() float64 {
	bestTime := math.Inf(1)
	bestX := ball.x
	bestBall := 0
	consider := func(b *Ball, ballIndex int) {
		if x, seconds, ok := predictedPaddleCrossing(b); ok && seconds < bestTime {
			bestTime = seconds
			bestX = x
			bestBall = ballIndex
		}
	}
	forEachActiveBall(func(b *Ball, index int) {
		consider(b, index+1)
	})

	if bestBall == 0 {
		lowestY := math.Inf(-1)
		forEachActiveBall(func(b *Ball, index int) {
			if b.y > lowestY {
				lowestY = b.y
				bestX = b.x
				bestBall = index + 1
			}
		})
	}

	if autoPaddleNeedsNewHitOffset || autoPaddleTargetBall != bestBall {
		autoPaddleHitOffset = chooseAutoPaddleHitOffset()
		autoPaddleTargetBall = bestBall
		autoPaddleNeedsNewHitOffset = false
	}

	// Offset is normalized to half the paddle width: -1 aims at the far left,
	// +1 at the far right. The configured 0.90 maximum leaves a safety margin.
	desiredHitPosition := 0.5 + autoPaddleHitOffset*0.5
	targetX := bestX - desiredHitPosition*paddle.w
	return clampFloat(targetX, 0, math.Max(0, canvasWidth-paddle.w))
}

func applyAutoPaddleControl(dt float64) bool {
	if !autoPaddleEnabled || dt <= 0 {
		return false
	}
	autoPaddleTargetX = autoPaddleAimX()
	errorX := autoPaddleTargetX - paddle.x
	targetSpeed := clampFloat(errorX*10, -defaultAutoPaddleMaxSpeed, defaultAutoPaddleMaxSpeed)
	if math.Abs(errorX) <= defaultAutoPaddleDeadZone {
		targetSpeed = 0
	}
	changeRate := defaultAutoPaddleAcceleration
	if targetSpeed == 0 || math.Signbit(targetSpeed) != math.Signbit(paddle.vx) {
		changeRate = defaultAutoPaddleBraking
	}
	paddle.vx = moveToward(paddle.vx, targetSpeed, changeRate*dt)
	paddle.x += paddle.vx * dt
	return true
}

func refreshAutoPaddleControl() {
	if !physicsEditorAutoPaddleCheck.IsUndefined() && !physicsEditorAutoPaddleCheck.IsNull() {
		physicsEditorAutoPaddleCheck.Set("checked", autoPaddleEnabled)
	}
}

func setAutoPaddleEnabled(enabled bool, announce bool) {
	autoPaddleEnabled = enabled
	autoDemoWait = 0
	if enabled {
		markLevelRunAssisted()
	}
	resetAutoPaddleHitPlan()
	if enabled && waitingForStart && !gameOver {
		waitingForStart = false
		paused = false
	}
	if enabled && (levelAdvancePending || gameOver) {
		paused = false // Enabling demo on a result screen starts its countdown.
	}
	leftPressed = false
	rightPressed = false
	mobileLeftHeld = false
	mobileRightHeld = false
	touchControlActive = false
	mouseControlActive = false
	gamepadAxisX = 0
	paddle.vx = 0
	resetPaddleSpinHistory()
	refreshAutoPaddleControl()
	if announce {
		if enabled {
			showStatus("Auto paddle / demo ON", 2.0)
		} else {
			showStatus("Auto paddle / demo OFF", 2.0)
		}
	}
}

func toggleAutoPaddle() {
	setAutoPaddleEnabled(!autoPaddleEnabled, true)
}

// ---- Update (main loop) ----
func update(dt float64) {
	if updateAutoDemo(dt) {
		return
	}
	if gameOver || paused || waitingForStart {
		resetPaddleSpinHistory()
		return
	}

	if levelAdvancePending {
		// Stay on the results screen until the player explicitly continues.
		// Debris may keep animating, but the frozen completion stats do not change.
		updateBrickDebris(dt)
		return
	}

	// Active-play time excludes READY, pause, and the level-complete screen.
	updateMagicBrickFlip(dt)
	trackFullGameSpeed()
	levelRunStats.elapsedSeconds += dt / gameSpeedScale()

	// Keyboard and two-thumb controls use acceleration rather than jumping
	// immediately to one fixed speed. Short taps make small corrections;
	// holding a direction quickly reaches the higher cross-screen speed.
	digitalControl := false
	digitalDirection := 0.0

	if leftPressed || rightPressed {
		digitalControl = true
		if leftPressed && !rightPressed {
			digitalDirection = -1
		} else if rightPressed && !leftPressed {
			digitalDirection = 1
		}
	} else if mobileControlsEnabled && mobileControlMode == "two-thumb" {
		digitalControl = true
		if mobileLeftHeld && !mobileRightHeld {
			digitalDirection = -1
		} else if mobileRightHeld && !mobileLeftHeld {
			digitalDirection = 1
		}
	}

	if autoPaddleEnabled {
		applyAutoPaddleControl(dt)
	} else if gamepadAxisX != 0 {
		// Analog joystick movement is immediate and proportional to stick travel.
		// The resulting paddle velocity continues through the normal spin history.
		paddle.vx = gamepadAxisX * defaultDigitalPaddleMaxSpeed
		paddle.x += paddle.vx * dt
	} else if digitalControl {
		targetSpeed := digitalDirection * defaultDigitalPaddleMaxSpeed
		changeRate := defaultDigitalPaddleAcceleration
		if digitalDirection == 0 {
			changeRate = defaultDigitalPaddleBraking
		}
		paddle.vx = moveToward(paddle.vx, targetSpeed, changeRate*dt)
		paddle.x += paddle.vx * dt
	} else if applyMousePaddleControl(dt) {
		// Mouse position is direct; velocity is measured below for spin transfer.
	} else if mobileControlsEnabled {
		switch mobileControlMode {
		case "vertical", "follow":
			// These modes position the paddle directly from pointer events.
			if !touchControlActive {
				paddle.vx = 0
			}
		case "tilt":
			if !applyPhoneTiltControl(dt) {
				paddle.vx = 0
			}
		default:
			paddle.vx = 0
		}
	} else {
		// Brake keyboard movement quickly after the key is released.
		paddle.vx = moveToward(paddle.vx, 0, defaultDigitalPaddleBraking*dt)
		paddle.x += paddle.vx * dt
	}
	if paddle.x < 0 {
		paddle.x = 0
	}
	if paddle.x+paddle.w > canvasWidth {
		paddle.x = canvasWidth - paddle.w
	}

	if dt > 0 {
		paddle.vx = (paddle.x - paddlePreviousX) / dt
		if mouseControlActive && physicsConfig.mousePaddleSpinVelocityLimit > 0 {
			paddle.vx = clampFloat(
				paddle.vx,
				-physicsConfig.mousePaddleSpinVelocityLimit,
				physicsConfig.mousePaddleSpinVelocityLimit,
			)
		}
	} else {
		paddle.vx = 0
	}
	paddlePreviousX = paddle.x
	updatePaddleSpinHistory(dt)

	// ---- Independent power-up timers ----
	gravityChanged := false
	if lowGravityActive {
		lowGravityTimer -= dt
		if lowGravityTimer <= 0 {
			stopMagicFeatureVoice("lowgravity")
			lowGravityActive = false
			lowGravityTimer = 0
			gravityChanged = true
		}
	}
	if reverseGravityActive {
		reverseGravityTimer -= dt
		if reverseGravityTimer <= 0 {
			stopMagicFeatureVoice("reversegravity")
			reverseGravityActive = false
			reverseGravityTimer = 0
			gravityChanged = true
		}
	}
	if passActive {
		passTimer -= dt
		if passTimer <= 0 {
			stopMagicFeatureVoice("passthrough")
			passActive = false
			passTimer = 0
		}
	}
	if magnetPowerActive {
		magnetPowerTimer -= dt
		if magnetPowerTimer <= 0 {
			stopMagicFeatureVoice("magnet")
			magnetPowerActive = false
			magnetPowerTimer = 0
		}
	}
	if zapperPowerActive {
		zapperPowerTimer -= dt
		if zapperPowerTimer <= 0 {
			stopMagicFeatureVoice("zapper")
			zapperPowerActive = false
			zapperPowerTimer = 0
			resetAllBallZapperStates()
		}
	}
	if bigPaddleActive {
		if bigPaddleTimer > 0 {
			bigPaddleTimer -= dt
			if bigPaddleTimer <= 0 {
				bigPaddleTimer = 0
				bigPaddleExpiryPending = true
				stopMagicFeatureVoice("bigpaddle")
			}
		}
		if bigPaddleExpiryPending && allActiveBallsMovingUp() {
			bigPaddleActive = false
			bigPaddleExpiryPending = false
			setPaddleSize(paddleWidth, paddleHeight)
		}
	}

	if blackHoleActive {
		blackHolePathElapsed = math.Min(blackHolePathDuration, blackHolePathElapsed+dt)
		updateBlackHolePathPosition()

		blackHoleTimer -= dt
		if blackHoleTimer <= 0 {
			stopMagicFeatureVoice("blackhole")
			blackHoleActive = false
			blackHoleTimer = 0
			resetBlackHolePath()
			gravityChanged = true
		}
	}

	if influencerActive {
		influencerTimer -= dt
		if influencerTimer <= 0 {
			stopMagicFeatureVoice("influencer")
			influencerActive = false
			influencerTimer = 0
		}
	}

	if gravityChanged {
		refreshCurrentGravity()
	}

	updateZapper(dt)

	for i := len(statusMessages) - 1; i >= 0; i-- {
		statusMessages[i].timer -= dt
		if statusMessages[i].timer <= 0 {
			statusMessages = append(statusMessages[:i], statusMessages[i+1:]...)
		}
	}

	// Debris is integrated before the balls so shard impulses affect the balls'
	// velocity during this same fixed step. Debris spawned by a ball impact begins
	// moving on the following step, which avoids source-brick self-collisions.
	updateBrickDebris(dt)

	// Ball 1 is the only prize-taking ball. Extra balls get the same physics but
	// pass isPrimary=false, so magic bricks they hit are destroyed without activating
	// another prize. A prize spawned by ball 1 may append a ball during this step.
	updateBall(&ball, dt, true)
	for i := range extraBalls {
		updateBall(&extraBalls[i], dt, false)
	}

	resolveBallBallCollisions()
	compactLostBalls()
	// Freeze run stats and hold the results screen until the player continues.
	if !gameOver && !levelAdvancePending && remainingBreakableBricks == 0 {
		recordLevelCompletionStats()
		recordFullGameLevelCompletion()
		lives++
		unlockNextLevel()
		levelAdvancePending = true
		levelCompleteTimer = 0
		debugOverlayVisible = false
		physicsOverlayVisible = false
		statsOverlayVisible = false
		leftPressed = false
		rightPressed = false
		touchControlActive = false
		paddle.vx = 0
		resetAllBallZapperStates()
		playLevelComplete()
		showStatus("Level complete! +1 life", 3.0)
	}

	recordReplayPhysicsStep(dt)
}

func advanceFromLevelComplete() {
	if !levelAdvancePending {
		return
	}
	levelAdvancePending = false
	nextLevel := currentLevelIndex + 1
	if nextLevel >= len(levels) {
		gameOver = true
		win = true
		playYouWin()
		return
	}
	startLevel(nextLevel)
}

// ---- Reset balls on life lost ----
func resetBalls() {
	ball.x, ball.y = startBallX, startBallY
	ball.vx, ball.vy = startBallVx, startBallVy
	ball.omega, ball.angle = 0, 0
	ball.stuckTimer = 0
	ball.soundCooldown = 0
	ball.ballBallSoundCooldown = 0
	ball.ballCollisionGrace = 0
	resetFastOrbitState(&ball)
	resetBallRescueState(&ball, true)
	resetBallZapperState(&ball)
	extraBalls = extraBalls[:0]
	paddle.x = (canvasWidth - paddle.w) / 2
	paddle.vx = 0
	paddlePreviousX = paddle.x
	resetPaddleSpinHistory()
	brickDebris = brickDebris[:0]
	debrisFieldTick = 0
	mouseControlActive = false
	mousePaddleTargetX = paddle.x
	clearLastPaddleSpinDebug()
	clearLastCollisionDebug()
	resetAutoPaddleHitPlan()
	syncRenderInterpolation()
}

func ensureBrickCanvas() {
	if brickCanvas.IsUndefined() || brickCanvas.IsNull() {
		brickCanvas = doc.Call("createElement", "canvas")
		brickCtx = brickCanvas.Call("getContext", "2d")
	}
	if brickCanvas.Get("width").Int() != int(canvasWidth) {
		brickCanvas.Set("width", int(canvasWidth))
	}
	if brickCanvas.Get("height").Int() != int(canvasHeight) {
		brickCanvas.Set("height", int(canvasHeight))
	}
}

func drawBrickGroup(target js.Value, kind int) {
	switch kind {
	case 0: // normal
		target.Set("fillStyle", palette[2])
		target.Set("strokeStyle", brickStrokeColor)
		target.Set("lineWidth", 1)
		target.Call("setLineDash", []interface{}{})
	case 1: // magic
		target.Set("fillStyle", magicColor)
		target.Set("strokeStyle", magicStrokeColor)
		target.Set("lineWidth", 2)
		target.Call("setLineDash", []interface{}{})
	case 2: // unbreakable
		target.Set("fillStyle", palette[6])
		target.Set("strokeStyle", unbreakableStrokeColor)
		target.Set("lineWidth", 2)
		target.Call("setLineDash", []interface{}{})
	}

	for i := range bricks {
		br := &bricks[i]
		if !br.alive {
			continue
		}
		matches := (kind == 0 && !br.magic && !br.unbreakable) ||
			(kind == 1 && br.magic) || (kind == 2 && br.unbreakable)
		if !matches {
			continue
		}
		target.Call("save")
		target.Call("beginPath")
		if physicsConfig.drawBrickTilt {
			centerX := br.x + br.w/2
			centerY := br.y + br.h/2
			target.Call("translate", centerX, centerY)
			target.Call("rotate", br.tiltRadians)
			target.Call("roundRect", -br.w/2, -br.h/2, br.w, br.h, brickRadius)
		} else {
			target.Call("roundRect", br.x, br.y, br.w, br.h, brickRadius)
		}
		target.Call("fill")
		target.Call("stroke")
		target.Call("restore")
	}
	target.Call("setLineDash", []interface{}{})
}

func rebuildBrickCanvas() {
	ensureBrickCanvas()
	brickCtx.Call("clearRect", 0, 0, canvasWidth, canvasHeight)
	if !drawBrickCanvasBatch() {
		drawBrickGroup(brickCtx, 0)
		drawBrickGroup(brickCtx, 1)
		drawBrickGroup(brickCtx, 2)
	}
	bricksDirty = false
}

// ---- Draw ----
func drawOneZapperBolt(
	b *Ball,
	targetIndex int,
	strokeColor string,
	shadowColor string,
) {
	if targetIndex < 0 || targetIndex >= len(bricks) {
		return
	}

	br := &bricks[targetIndex]
	if !br.alive || br.unbreakable {
		return
	}

	startX, startY := b.x, b.y
	endX := br.x + br.w/2
	endY := br.y + br.h/2

	const segments = 12

	ctx.Call("save")
	ctx.Set("strokeStyle", strokeColor)
	ctx.Set("lineWidth", 3)
	ctx.Set("shadowColor", shadowColor)
	ctx.Set("shadowBlur", 12)
	ctx.Call("beginPath")
	ctx.Call("moveTo", startX, startY)

	for i := 1; i < segments; i++ {
		t := float64(i) / float64(segments)
		x := startX + (endX-startX)*t
		y := startY + (endY-startY)*t

		dx := endX - startX
		dy := endY - startY
		length := math.Sqrt(dx*dx + dy*dy)
		if length > 0 {
			nx := -dy / length
			ny := dx / length
			jitter := (rand.Float64()*2 - 1) * 14
			x += nx * jitter
			y += ny * jitter
		}

		ctx.Call("lineTo", x, y)
	}

	ctx.Call("lineTo", endX, endY)
	ctx.Call("stroke")
	ctx.Call("restore")
}

func drawZapperBolts() {
	if !zapperIsActive() {
		return
	}

	forEachActiveBall(func(b *Ball, index int) {
		strokeColor, shadowColor := "#e8fbff", "#58d9ff"
		if index%2 == 1 {
			strokeColor, shadowColor = "#f3e8ff", "#b56cff"
		}
		drawOneZapperBolt(b, b.zapperTargetIndex, strokeColor, shadowColor)
	})
}

func physicsFloatSettingValue(settings *physicsSettings, key string) float64 {
	switch key {
	case "gravity":
		return settings.gravity
	case "restitution":
		return settings.restitution
	case "frictionCoeff":
		return settings.frictionCoeff
	case "paddleBoost":
		return settings.paddleBoost
	case "brickBoost":
		return settings.brickBoost
	case "maxSpeed":
		return settings.maxSpeed
	case "maxSpin":
		return settings.maxSpin
	case "stuckSpeedThreshold":
		return settings.stuckSpeedThreshold
	case "stuckDuration":
		return settings.stuckDuration
	case "tiltUpSpeed":
		return settings.tiltUpSpeed
	case "tiltSideMin":
		return settings.tiltSideMin
	case "tiltSideMax":
		return settings.tiltSideMax
	case "magnusCoefficient":
		return settings.magnusCoefficient
	case "magnusAccelerationScale":
		return settings.magnusAccelerationScale
	case "spinDrag":
		return settings.spinDrag
	case "airDrag":
		return settings.airDrag
	case "wallFrictionScale":
		return settings.wallFrictionScale
	case "brickFrictionScale":
		return settings.brickFrictionScale
	case "unbreakableFrictionScale":
		return settings.unbreakableFrictionScale
	case "paddleFrictionScale":
		return settings.paddleFrictionScale
	case "paddleSpinTransfer":
		return settings.paddleSpinTransfer
	case "collisionSpinCoupling":
		return settings.collisionSpinCoupling
	case "minimumCollisionGrip":
		return settings.minimumCollisionGrip
	case "minimumPaddleGrip":
		return settings.minimumPaddleGrip
	case "collisionSlop":
		return settings.collisionSlop
	case "paddleSpinGraceSeconds":
		return settings.paddleSpinGraceSeconds
	case "mousePaddleSpinVelocityLimit":
		return settings.mousePaddleSpinVelocityLimit
	case "overspeedHalfLife":
		return settings.overspeedHalfLife
	case "wallNoiseCellSize":
		return settings.wallNoiseCellSize
	case "wallSideTiltDegrees":
		return settings.wallSideTiltDegrees
	case "wallTopTiltDegrees":
		return settings.wallTopTiltDegrees
	case "wallCornerFadeDistance":
		return settings.wallCornerFadeDistance
	case "cornerPhysicsAmount":
		return settings.cornerPhysicsAmount
	case "brickTiltMinDegrees":
		return settings.brickTiltMinDegrees
	case "brickTiltMaxDegrees":
		return settings.brickTiltMaxDegrees
	case "orbitMinimumSpeed":
		return settings.orbitMinimumSpeed
	case "orbitMinimumHitSpeed":
		return settings.orbitMinimumHitSpeed
	case "orbitMinorSpeedRatio":
		return settings.orbitMinorSpeedRatio
	case "orbitMinorSpeedFloor":
		return settings.orbitMinorSpeedFloor
	case "orbitDetectionWindow":
		return settings.orbitDetectionWindow
	case "orbitMaximumMinorProgress":
		return settings.orbitMaximumMinorProgress
	case "orbitHitCooldown":
		return settings.orbitHitCooldown
	case "orbitEscapeSpeed":
		return settings.orbitEscapeSpeed
	case "orbitEscapeDuration":
		return settings.orbitEscapeDuration
	}
	return 0
}

func parsePhysicsFloatConfigKey(key string) (field string, ok bool) {
	fields := []string{
		"gravity", "restitution", "frictionCoeff", "paddleBoost", "brickBoost",
		"maxSpeed", "maxSpin", "stuckSpeedThreshold", "stuckDuration",
		"tiltUpSpeed", "tiltSideMin", "tiltSideMax",
		"magnusCoefficient", "magnusAccelerationScale", "spinDrag", "airDrag",
		"wallFrictionScale", "brickFrictionScale", "unbreakableFrictionScale",
		"paddleFrictionScale", "paddleSpinTransfer", "collisionSpinCoupling",
		"minimumCollisionGrip", "minimumPaddleGrip", "collisionSlop",
		"paddleSpinGraceSeconds", "mousePaddleSpinVelocityLimit", "overspeedHalfLife",
		"wallNoiseCellSize", "wallSideTiltDegrees", "wallTopTiltDegrees",
		"wallCornerFadeDistance", "cornerPhysicsAmount", "brickTiltMinDegrees", "brickTiltMaxDegrees",
		"orbitMinimumSpeed", "orbitMinimumHitSpeed", "orbitMinorSpeedRatio",
		"orbitMinorSpeedFloor", "orbitDetectionWindow", "orbitMaximumMinorProgress",
		"orbitHitCooldown", "orbitEscapeSpeed", "orbitEscapeDuration",
	}
	for _, candidate := range fields {
		if key == candidate {
			return candidate, true
		}
	}
	return "", false
}

func validPhysicsFloatSetting(key string, value float64) bool {
	switch key {
	case "restitution", "cornerPhysicsAmount":
		return value >= 0 && value <= 1
	case "maxSpeed", "maxSpin":
		return value > 0
	case "gravity", "paddleBoost", "brickBoost":
		return true
	default:
		return value >= 0
	}
}

func configState(key string) (effective, defaultValue, kind string, ok bool) {
	defaults := defaultPhysicsSettings()
	if field, physicsKey := parsePhysicsFloatConfigKey(key); physicsKey {
		return formatConfigFloat(physicsFloatSettingValue(&physicsConfig, field)),
			formatConfigFloat(physicsFloatSettingValue(&defaults, field)), "float", true
	}
	if key == "orbitRequiredHits" {
		return strconv.Itoa(physicsConfig.orbitRequiredHits),
			strconv.Itoa(defaults.orbitRequiredHits), "int", true
	}
	if key == "drawBrickTilt" {
		return strconv.FormatBool(physicsConfig.drawBrickTilt),
			strconv.FormatBool(defaults.drawBrickTilt), "bool", true
	}
	if key == "cornerPhysics" || key == "cornerPhysicsEnabled" {
		return strconv.FormatBool(physicsConfig.cornerPhysicsEnabled),
			strconv.FormatBool(defaults.cornerPhysicsEnabled), "bool", true
	}
	if key == "cornerPhysicsAllBricks" || key == "allBrickCorners" {
		return strconv.FormatBool(physicsConfig.cornerPhysicsAllBricks),
			strconv.FormatBool(defaults.cornerPhysicsAllBricks), "bool", true
	}
	switch key {
	case "audioRoom":
		return currentAudioRoom, "none", "string", true
	case "audioRoomDry":
		return formatConfigFloat(currentAudioRoomDry), "-1", "float", true
	case "backgroundColor":
		return palette[0], defaultPalette[0], "string", true
	case "paddleColor":
		return palette[1], defaultPalette[1], "string", true
	case "brickColor":
		return palette[2], defaultPalette[2], "string", true
	case "ballColor":
		return palette[3], defaultPalette[3], "string", true
	case "textColor":
		return palette[4], defaultPalette[4], "string", true
	case "spinColor":
		return palette[5], defaultPalette[5], "string", true
	case "unbreakableColor":
		return palette[6], defaultPalette[6], "string", true
	case "secondBallColor":
		return palette[7], defaultPalette[7], "string", true
	case "secondBallSpinColor":
		return palette[8], defaultPalette[8], "string", true
	case "magicColor":
		return magicColor, defaultMagicColor, "string", true
	case "magicStrokeColor":
		return magicStrokeColor, defaultMagicStrokeColor, "string", true
	case "unbreakableStrokeColor":
		return unbreakableStrokeColor, defaultUnbreakableStrokeColor, "string", true
	case "brickStrokeColor":
		return brickStrokeColor, defaultBrickStrokeColor, "string", true
	case "powerUpDuration":
		return formatConfigFloat(powerUpDuration), formatConfigFloat(defaultPowerUpDuration), "float", true
	case "magnet":
		return strconv.FormatBool(levelMagnetActive), "false", "bool", true
	case "zapper":
		return strconv.FormatBool(levelZapperActive), "false", "bool", true
	case "blackHoleStrength":
		return formatConfigFloat(blackHoleStrength), formatConfigFloat(defaultBlackHoleStrength), "float", true
	case "blackHoleRange":
		return formatConfigFloat(blackHoleRange), formatConfigFloat(defaultBlackHoleRange), "float", true
	case "blackHolePathVerticalRange":
		return formatConfigFloat(blackHolePathVerticalRange), formatConfigFloat(defaultBlackHolePathVerticalRange), "float", true
	case "blackHolePathCenterYOffset":
		return formatConfigFloat(blackHolePathCenterYOffset), formatConfigFloat(defaultBlackHolePathCenterYOffset), "float", true
	case "blackHolePathHorizontalCyclesMin":
		return formatConfigFloat(blackHolePathHorizontalCyclesMin), formatConfigFloat(defaultBlackHolePathHorizontalCyclesMin), "float", true
	case "blackHolePathHorizontalCyclesMax":
		return formatConfigFloat(blackHolePathHorizontalCyclesMax), formatConfigFloat(defaultBlackHolePathHorizontalCyclesMax), "float", true
	case "blackHolePathVerticalCyclesMin":
		return formatConfigFloat(blackHolePathVerticalCyclesMin), formatConfigFloat(defaultBlackHolePathVerticalCyclesMin), "float", true
	case "blackHolePathVerticalCyclesMax":
		return formatConfigFloat(blackHolePathVerticalCyclesMax), formatConfigFloat(defaultBlackHolePathVerticalCyclesMax), "float", true
	case "blackHolePathWobble":
		return formatConfigFloat(blackHolePathWobble), formatConfigFloat(defaultBlackHolePathWobble), "float", true
	case "magnetStrength":
		return formatConfigFloat(magnetStrength), formatConfigFloat(defaultMagnetStrength), "float", true
	case "magnetRange":
		return formatConfigFloat(magnetRange), formatConfigFloat(defaultMagnetRange), "float", true
	case "influencerMultiplier":
		return formatConfigFloat(influencerMultiplier), formatConfigFloat(defaultInfluencerMultiplier), "float", true
	case "zapperRange":
		return formatConfigFloat(zapperRange), formatConfigFloat(defaultZapperRange), "float", true
	case "brickOffsetTop":
		return formatConfigFloat(brickOffsetTop), formatConfigFloat(defaultBrickOffsetTop), "float", true
	case "brickWidth", "defaultBrickWidth":
		return formatConfigFloat(brickWidth), formatConfigFloat(defaultBrickWidth), "float", true
	case "brickHeight", "defaultBrickHeight":
		return formatConfigFloat(brickHeight), formatConfigFloat(defaultBrickHeight), "float", true
	case "brickPadding", "defaultBrickPadding":
		return formatConfigFloat(brickPadding), formatConfigFloat(defaultBrickPadding), "float", true
	case "brickRows":
		return strconv.Itoa(brickRows), strconv.Itoa(defaultBrickRows), "int", true
	case "brickCols":
		return strconv.Itoa(brickCols), strconv.Itoa(defaultBrickCols), "int", true
	case "ballX":
		return formatConfigFloat(startBallX), formatConfigFloat(defaultStartBallX), "float", true
	case "ballY":
		return formatConfigFloat(startBallY), formatConfigFloat(defaultStartBallY), "float", true
	case "ballVx":
		return formatConfigFloat(startBallVx), formatConfigFloat(defaultStartBallVx), "float", true
	case "ballVy":
		return formatConfigFloat(startBallVy), formatConfigFloat(defaultStartBallVy), "float", true
	case "ballRadius":
		return formatConfigFloat(ballRadius), formatConfigFloat(defaultBallRadius), "float", true
	case "enableLowGravity":
		return strconv.FormatBool(enableLowGravity), strconv.FormatBool(defaultEnableLowGravity), "bool", true
	case "enablePassThrough":
		return strconv.FormatBool(enablePassThrough), strconv.FormatBool(defaultEnablePassThrough), "bool", true
	case "enableNuke":
		return strconv.FormatBool(enableNuke), strconv.FormatBool(defaultEnableNuke), "bool", true
	case "enableReverseGravity":
		return strconv.FormatBool(enableReverseGravity), strconv.FormatBool(defaultEnableReverseGravity), "bool", true
	case "enableDualBalls":
		return strconv.FormatBool(enableDualBalls), strconv.FormatBool(defaultEnableDualBalls), "bool", true
	case "enableBlackHole":
		return strconv.FormatBool(enableBlackHole), strconv.FormatBool(defaultEnableBlackHole), "bool", true
	case "enableMagnet":
		return strconv.FormatBool(enableMagnet), strconv.FormatBool(defaultEnableMagnet), "bool", true
	case "enableInfluencer":
		return strconv.FormatBool(enableInfluencer), strconv.FormatBool(defaultEnableInfluencer), "bool", true
	case "enableZapper":
		return strconv.FormatBool(enableZapper), strconv.FormatBool(defaultEnableZapper), "bool", true
	case "enableBreakUnbreakable":
		return strconv.FormatBool(enableBreakUnbreakable), strconv.FormatBool(defaultEnableBreakUnbreakable), "bool", true
	case "enableBigPaddle":
		return strconv.FormatBool(enableBigPaddle), strconv.FormatBool(defaultEnableBigPaddle), "bool", true
	case "enableBrickFlip":
		return strconv.FormatBool(enableBrickFlip), strconv.FormatBool(defaultEnableBrickFlip), "bool", true
	case "paddleWidth":
		return formatConfigFloat(paddleWidth), formatConfigFloat(defaultPaddleWidth), "float", true
	case "paddleHeight":
		return formatConfigFloat(paddleHeight), formatConfigFloat(defaultPaddleHeight), "float", true
	case "autoPaddleHitVariation":
		return formatConfigFloat(autoPaddleHitVariation), formatConfigFloat(defaultAutoPaddleHitVariation), "float", true
	case "debrisShapeMode":
		return debrisShapeMode, defaultDebrisShapeMode, "string", true
	case "debrisSizeScale":
		return formatConfigFloat(debrisSizeScale), formatConfigFloat(defaultDebrisSizeScale), "float", true
	case "debrisLifetimeVariationPercent":
		return formatConfigFloat(debrisLifetimeVariationPercent), formatConfigFloat(defaultDebrisLifetimeVariationPercent), "float", true
	case "debrisFrontLayerSpeed":
		return formatConfigFloat(debrisFrontLayerSpeed), formatConfigFloat(defaultDebrisFrontLayerSpeed), "float", true
	default:
		return "", "", "", false
	}
}

func formatConfigFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func configValuesEqual(kind, a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	switch kind {
	case "float":
		af, errA := strconv.ParseFloat(a, 64)
		bf, errB := strconv.ParseFloat(b, 64)
		return errA == nil && errB == nil && math.Abs(af-bf) <= 1e-9
	case "int":
		ai, errA := strconv.Atoi(a)
		bi, errB := strconv.Atoi(b)
		return errA == nil && errB == nil && ai == bi
	case "bool":
		ab, errA := strconv.ParseBool(a)
		bb, errB := strconv.ParseBool(b)
		return errA == nil && errB == nil && ab == bb
	default:
		return a == b
	}
}

func isPaletteConfigKey(key string) bool {
	return strings.HasSuffix(key, "Color")
}

func currentLevelOverrideSections() (otherLines, paletteLines []string) {
	if currentLevelIndex < 0 || currentLevelIndex >= len(levels) {
		return nil, nil
	}
	config := levels[currentLevelIndex].config
	keys := make([]string, 0, len(config))
	for key := range config {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		effective, defaultValue, kind, accepted := configState(key)
		if !accepted {
			continue
		}
		// Only list values that were successfully applied and differ from the
		// corresponding built-in default.
		if !configValuesEqual(kind, config[key], effective) {
			continue
		}
		if configValuesEqual(kind, effective, defaultValue) {
			continue
		}

		line := key + "=" + effective
		if isPaletteConfigKey(key) {
			paletteLines = append(paletteLines, line)
		} else {
			otherLines = append(otherLines, line)
		}
	}
	return otherLines, paletteLines
}

func debugOverlayLines() []string {
	devState := "OFF"
	if devAllLevelsUnlocked {
		devState = "ON"
	}

	lines := []string{
		"BUILD " + buildID,
		"LEVEL " + strconv.Itoa(currentLevelIndex+1) + "/" + strconv.Itoa(len(levels)),
		"UNLOCKED THROUGH " + strconv.Itoa(highestUnlockedLevel+1),
		"DEV ALL LEVELS " + devState,
		"LEVEL OVERRIDES:",
	}

	otherLines, paletteLines := currentLevelOverrideSections()
	if len(otherLines) == 0 && len(paletteLines) == 0 {
		return append(lines, "(none)")
	}

	lines = append(lines, otherLines...)
	if len(paletteLines) > 0 {
		lines = append(lines, "", "PALETTE / COLORS:")
		lines = append(lines, paletteLines...)
	}
	return lines
}
func debrisPathCacheState() string {
	if !defaultDebrisUsePath2DCache {
		return "DISABLED"
	}
	if !debrisPath2DChecked {
		return "READY"
	}
	if debrisPath2DSupported {
		return "ON"
	}
	return "FALLBACK"
}

func physicsOverlayLines() []string {
	status := "OK"
	if physicsWarningTimer > 0 {
		status = "WARNING: PHYSICS COULD NOT KEEP UP"
	}
	rescueGate := "BLOCKED"
	if ballRescuePerformanceHealthy() {
		rescueGate = "OK"
	}

	autoPaddleState := "OFF"
	if autoPaddleEnabled {
		autoPaddleState = "ON"
	}
	debrisMasterState := "OFF"
	if debrisMasterEnabled {
		debrisMasterState = "ON"
	}
	debrisLevelState := "OFF"
	if debrisEnabled {
		debrisLevelState = "ON"
	}
	mouseLockState := "UNAVAILABLE"
	if pointerLockSupported() {
		mouseLockState = "READY"
		if mousePointerLocked {
			mouseLockState = "LOCKED"
		}
	}
	tiltState := "WAITING"
	if phoneTiltListenerSet && phoneTiltAvailable {
		tiltState = "ACTIVE"
	} else if phoneTiltListenerSet {
		tiltState = "READY"
	}

	lines := []string{
		"BUILD " + buildID,
		"AUTO PADDLE        " + autoPaddleState,
		"AUTO HIT OFFSET    " + fmt.Sprintf("%+.2f / +/-%.2f", autoPaddleHitOffset, autoPaddleHitVariation),
		"PHYSICS FIXED STEP " + fmt.Sprintf("%.0f Hz / %.3f ms", physicsStepHz, physicsStepSeconds*1000),
		"MAX TRAVEL / TICK  " + fmt.Sprintf("%.2f px", physicsConfig.maxSpeed*physicsStepSeconds),
		"PHYSICS ACTUAL     " + fmt.Sprintf("%.1f Hz", physicsStepRateCurrent),
		"SIMULATION REALTIME " + fmt.Sprintf("%.1f%%", physicsRealtimePercent),
		"PHYSICS COMPUTE LOAD " + fmt.Sprintf("%.1f%%", physicsComputeLoad),
		"RENDER FPS          " + fmt.Sprintf("%.1f", fpsCurrent),
		"RENDER FPS LOWEST   " + formatLowestRenderFPS(),
		"RENDER INTERP       ON / alpha " + fmt.Sprintf("%.3f", renderInterpolationAlpha),
		"STEPS LAST FRAME    " + strconv.Itoa(physicsLastFrameSteps),
		"STEPS PEAK FRAME    " + strconv.Itoa(physicsPeakFrameSteps),
		"CATCH-UP LIMIT      " + strconv.Itoa(physicsMaxCatchUpSteps),
		"DROPPED SIM TIME    " + fmt.Sprintf("%.4f s", physicsDroppedTimeTotal),
		"STATUS " + status,
		"MOUSE CAPTURE      " + mouseLockState + " (left lock / right release)",
		"PHONE TILT         " + tiltState,
		"DEBRIS MASTER      " + debrisMasterState + " (R, saved)",
		"DEBRIS LEVEL       " + debrisLevelState,
		"DEBRIS             " + strconv.Itoa(len(brickDebris)) + "/" + strconv.Itoa(debrisMaxActivePieces),
		"DEBRIS SHAPE       " + strings.ToUpper(debrisShapeMode),
		"DEBRIS DRAWN       " + strconv.Itoa(debrisRenderedLastFrame) + " / skipped " + strconv.Itoa(debrisSkippedLastFrame),
		"DEBRIS RENDER      1/" + strconv.Itoa(debrisAdaptiveRenderStride) + " old slow shards",
		"DEBRIS BASELINE    " + fmt.Sprintf("%.1f FPS", debrisAdaptiveBaselineFPS),
		"DEBRIS PATH CACHE  " + debrisPathCacheState(),
		"RESCUE PERF GATE    " + rescueGate,
		"FLOOR GRACE         " + fmt.Sprintf("%.0f px", ballBelowFloorGracePixels),
		"BALLS ACTIVE        " + strconv.Itoa(activeBallCount()) + "/" + strconv.Itoa(maxConfiguredBalls()),
		"BALL COLLISIONS     " + map[bool]string{true: "ON", false: "OFF"}[defaultBallBallCollisions] +
			" / " + strconv.Itoa(activeBallCount()*(activeBallCount()-1)/2) + " pairs",
		"",
	}
	forEachActiveBall(func(b *Ball, index int) {
		ballNumber := strconv.Itoa(index + 1)
		lines = append(lines,
			"RESCUE FAILURES B"+ballNumber+" "+strconv.Itoa(b.rescueFailureCount)+"/"+strconv.Itoa(ballRescueFailureLimit),
			"BALL "+ballNumber+" SPEED "+fmt.Sprintf("%.2f", math.Hypot(b.vx, b.vy))+
				" (max "+strconv.FormatFloat(physicsConfig.maxSpeed, 'f', -1, 64)+")",
			"BALL "+ballNumber+" SPIN  "+fmt.Sprintf("%+.2f", b.omega)+
				" (max "+strconv.FormatFloat(physicsConfig.maxSpin, 'f', -1, 64)+")",
		)
	})
	if lastPaddleSpinValid {
		lines = append(lines,
			"",
			"LAST PADDLE HIT BALL "+strconv.Itoa(lastPaddleSpinBall),
			"SPIN BEFORE "+fmt.Sprintf("%+.2f", lastPaddleSpinBefore),
			"SPIN ADDED  "+fmt.Sprintf("%+.2f", lastPaddleSpinAdded),
			"SPIN AFTER  "+fmt.Sprintf("%+.2f", lastPaddleSpinAfter),
		)
	}
	if lastCollisionValid {
		lines = append(lines,
			"",
			"LAST BALL 1 COLLISION "+lastCollisionSurface,
			"ANGLE IN          "+fmt.Sprintf("%+.2f deg", lastCollisionIncomingAngle),
			"ANGLE OUT         "+fmt.Sprintf("%+.2f deg", lastCollisionOutgoingAngle),
			"ANGLE CHANGE      "+fmt.Sprintf("%+.2f deg", lastCollisionAngleChange),
			"SPIN ANGLE EFFECT "+fmt.Sprintf("%+.2f deg", lastCollisionSpinAngleEffect),
			"HIT SPIN BEFORE   "+fmt.Sprintf("%+.2f", lastCollisionSpinBefore),
			"HIT SPIN CHANGE   "+fmt.Sprintf("%+.2f", lastCollisionSpinChange),
			"HIT SPIN AFTER    "+fmt.Sprintf("%+.2f", lastCollisionSpinAfter),
			"CONTACT SPIN SPEED "+fmt.Sprintf("%+.2f", lastCollisionContactSpinSurfaceSpeed),
			"IMPULSE WITH SPIN "+fmt.Sprintf("%+.2f", lastCollisionTangentialImpulse),
			"IMPULSE NO SPIN   "+fmt.Sprintf("%+.2f", lastCollisionNoSpinImpulse),
			"SPIN IMPULSE EFFECT "+fmt.Sprintf("%+.2f", lastCollisionSpinImpulseEffect),
		)
	}
	return lines
}

func physicsOverlayReport() string {
	return strings.Join(physicsOverlayLines(), "\n")
}

func debugOverlayGeometry(lineCount int) (panelX, panelY, panelWidth, panelHeight float64, maxRows int) {
	const lineHeight = 18.0
	const padding = 12.0
	const columnWidth = 420.0

	maxRows = int((canvasHeight - padding*4) / lineHeight)
	if maxRows < 1 {
		maxRows = 1
	}
	columns := (lineCount + maxRows - 1) / maxRows
	if columns < 1 {
		columns = 1
	}
	panelWidth = float64(columns)*columnWidth + padding*2
	rows := lineCount
	if rows > maxRows {
		rows = maxRows
	}
	panelHeight = float64(rows)*lineHeight + padding*2
	panelX = canvasWidth - panelWidth - 10
	panelY = canvasHeight - panelHeight - 10
	if panelX < 10 {
		panelX = 10
	}
	if panelY < 10 {
		panelY = 10
	}
	return panelX, panelY, panelWidth, panelHeight, maxRows
}

func physicsOverlayGeometry(lineCount int) (panelX, panelY, panelWidth, panelHeight float64, maxRows int) {
	return debugOverlayGeometry(lineCount)
}

func debugOverlayReport() string {
	return strings.Join(debugOverlayLines(), "\n")
}

func statsOverlayLines() []string {
	run := levelRunStats
	if levelCompletedStatsValid && levelAdvancePending {
		run = levelCompletedStats
	}
	eligibility := "YES"
	if !levelRunRecordEligible {
		eligibility = "NO (ASSISTED)"
		if run.mixedSpeed {
			eligibility = "NO (MIXED SPEED)"
		}
	}
	wasChain := "--"
	if levelPreviousBests.chainFound {
		wasChain = strconv.Itoa(levelPreviousBests.bestChain) + " bricks"
	}
	bestChain := "--"
	if levelCurrentBests.chainFound {
		bestChain = strconv.Itoa(levelCurrentBests.bestChain) + " bricks"
	}
	wasBallsLost := formatBestInt(levelPreviousBests.ballsLost, levelPreviousBests.ballsLostFound)
	bestBallsLost := formatBestInt(levelCurrentBests.ballsLost, levelCurrentBests.ballsLostFound)
	newTag := func(isNew bool) string {
		if isNew {
			return " NEW BEST!"
		}
		return ""
	}
	lines := []string{
		"LEVEL RECORDS - LEVEL " + strconv.Itoa(currentLevelIndex+1),
		"",
		fmt.Sprintf("%-16s %-14s %-14s %-14s", "", "THIS RUN", "WAS", "BEST"),
		fmt.Sprintf("%-16s %-14s %-14s %-14s%s", "TIME", formatLevelTimeSeconds(run.elapsedSeconds), formatBestTime(levelPreviousBests), formatBestTime(levelCurrentBests), newTag(levelCompletionNewRecords.time)),
		fmt.Sprintf("%-16s %-14s %-14s %-14s%s", "FASTEST BALL", fmt.Sprintf("%.0f", run.fastestSpeed), formatBestInt(levelPreviousBests.fastestSpeed, levelPreviousBests.speedFound), formatBestInt(levelCurrentBests.fastestSpeed, levelCurrentBests.speedFound), newTag(levelCompletionNewRecords.speed)),
		fmt.Sprintf("%-16s %-14s %-14s %-14s%s", "FASTEST SPIN", fmt.Sprintf("%.0f", run.fastestSpin), formatBestInt(levelPreviousBests.fastestSpin, levelPreviousBests.spinFound), formatBestInt(levelCurrentBests.fastestSpin, levelCurrentBests.spinFound), newTag(levelCompletionNewRecords.spin)),
		fmt.Sprintf("%-16s %-14s %-14s %-14s%s", "BEST CHAIN", strconv.Itoa(run.bestChain)+" bricks", wasChain, bestChain, newTag(levelCompletionNewRecords.chain)),
		fmt.Sprintf("%-16s %-14s %-14s %-14s%s", "BALLS LOST", strconv.Itoa(run.ballsLost), wasBallsLost, bestBallsLost, newTag(levelCompletionNewRecords.ballsLost)),
		"",
		"RECORD ELIGIBLE " + eligibility,
	}
	return lines
}

func statsOverlayReport() string {
	return strings.Join(statsOverlayLines(), "\n")
}

func statsOverlayGeometry(lineCount int) (panelX, panelY, panelWidth, panelHeight float64, maxRows int) {
	return debugOverlayGeometry(lineCount)
}

func formatLowestRenderFPS() string {
	if fpsLowest <= 0 {
		return "--"
	}
	return fmt.Sprintf("%.1f", fpsLowest)
}

// Start a fresh gameplay-only minimum, preserving the current-FPS sampler.
func resetLowestRenderFPS() {
	fpsLowest = 0
	fpsVisibleSamplesSeen = 0
	fpsLowestElapsed = 0
	fpsLowestFrames = 0
	fpsLowestPlaying = false
	showStatus("Lowest FPS reset", 1.5)
}

func sampleLowestRenderFPS(dt float64, playing bool) {
	if !playing || dt <= 0 || dt >= 1 {
		fpsLowestElapsed, fpsLowestFrames, fpsVisibleSamplesSeen = 0, 0, 0
		fpsLowestPlaying = false
		return
	}
	// The first interval may still include the title screen or paused time.
	if !fpsLowestPlaying {
		fpsLowestPlaying = true
		return
	}
	fpsLowestElapsed += dt
	fpsLowestFrames++
	if fpsLowestElapsed < renderFPSSampleWindowSeconds {
		return
	}
	sample := float64(fpsLowestFrames) / fpsLowestElapsed
	if fpsVisibleSamplesSeen >= renderFPSLowestWarmupSamples && (fpsLowest == 0 || sample < fpsLowest) {
		fpsLowest = sample
	}
	fpsVisibleSamplesSeen++
	fpsLowestElapsed, fpsLowestFrames = 0, 0
}

func pageIsVisibleForFPS() bool {
	hidden := doc.Get("hidden")
	return hidden.IsUndefined() || hidden.IsNull() || !hidden.Bool()
}

// P cycles physics diagnostics -> level/config diagnostics -> level records -> off.
// I remains an alias for the same cycle so older muscle memory still works.
func cycleDiagnosticsOverlay() {
	if physicsOverlayVisible {
		physicsOverlayVisible = false
		debugOverlayVisible = true
		statsOverlayVisible = false
		return
	}
	if debugOverlayVisible {
		debugOverlayVisible = false
		statsOverlayVisible = true
		return
	}
	if statsOverlayVisible {
		statsOverlayVisible = false
		return
	}
	physicsOverlayVisible = true
	debugOverlayVisible = false
	statsOverlayVisible = false
}

func pointerCanvasPosition(e js.Value) (x, y float64, ok bool) {
	rect := canvas.Call("getBoundingClientRect")
	width := rect.Get("width").Float()
	height := rect.Get("height").Float()
	if width <= 0 || height <= 0 {
		return 0, 0, false
	}

	x = (e.Get("clientX").Float() - rect.Get("left").Float()) * canvasWidth / width
	y = (e.Get("clientY").Float() - rect.Get("top").Float()) * canvasHeight / height
	return x, y, true
}

func overlayReportAtPointer(e js.Value) (string, bool) {
	x, y, ok := pointerCanvasPosition(e)
	if !ok {
		return "", false
	}
	if debugOverlayVisible {
		panelX, panelY, panelWidth, panelHeight, _ := debugOverlayGeometry(len(debugOverlayLines()))
		if x >= panelX && x <= panelX+panelWidth && y >= panelY && y <= panelY+panelHeight {
			return debugOverlayReport(), true
		}
	}
	if physicsOverlayVisible {
		panelX, panelY, panelWidth, panelHeight, _ := physicsOverlayGeometry(len(physicsOverlayLines()))
		if x >= panelX && x <= panelX+panelWidth && y >= panelY && y <= panelY+panelHeight {
			return physicsOverlayReport(), true
		}
	}
	if statsOverlayVisible {
		panelX, panelY, panelWidth, panelHeight, _ := statsOverlayGeometry(len(statsOverlayLines()))
		if x >= panelX && x <= panelX+panelWidth && y >= panelY && y <= panelY+panelHeight {
			return statsOverlayReport(), true
		}
	}
	return "", false
}

func copyTextToClipboard(text string) bool {
	nativeClipboard = text
	return true
}

func keepPhysicsEditorCallback(callback js.Func) {
	physicsEditorCallbacks = append(physicsEditorCallbacks, callback)
}

func setStyle(element js.Value, property, value string) {
	element.Get("style").Set(property, value)
}

func formatAutoPaddleSliderValue(spec autoPaddleSliderSpec, value float64) string {
	return strconv.FormatFloat(value, 'f', spec.precision, 64)
}

func autoPaddleEditorValue(key string) float64 {
	switch key {
	case "autoPaddleHitVariation":
		return autoPaddleHitVariation
	}
	return 0
}

func defaultAutoPaddleEditorValue(key string) float64 {
	switch key {
	case "autoPaddleHitVariation":
		return defaultAutoPaddleHitVariation
	}
	return 0
}

func applyAutoPaddleEditorValue(spec autoPaddleSliderSpec, value float64) {
	value = clampFloat(value, spec.min, spec.max)
	switch spec.key {
	case "autoPaddleHitVariation":
		autoPaddleHitVariation = value
		resetAutoPaddleHitPlan()
	}
	refreshAutoPaddleEditorControls()
}

func refreshAutoPaddleEditorControls() {
	for _, spec := range autoPaddleEditorSliderSpecs {
		input, inputOK := physicsEditorInputs[spec.key]
		label, labelOK := physicsEditorValueLabels[spec.key]
		if !inputOK || !labelOK || input.IsUndefined() || input.IsNull() {
			continue
		}
		value := autoPaddleEditorValue(spec.key)
		text := formatAutoPaddleSliderValue(spec, value)
		input.Set("value", text)
		label.Set("textContent", text)
	}
}

func formatDebrisSliderValue(spec debrisSliderSpec, value float64) string {
	return strconv.FormatFloat(value, 'f', spec.precision, 64)
}

func debrisEditorValue(key string) float64 {
	switch key {
	case "debrisPiecesMin":
		return float64(debrisPiecesMin)
	case "debrisPiecesMax":
		return float64(debrisPiecesMax)
	case "debrisMaxActivePieces":
		return float64(debrisMaxActivePieces)
	case "debrisLifetime":
		return debrisLifetime
	case "debrisLifetimeVariationPercent":
		return debrisLifetimeVariationPercent
	case "debrisFadeDuration":
		return debrisFadeDuration
	case "debrisStartOpacity":
		return debrisStartOpacity
	case "debrisStartOpacityVariation":
		return debrisStartOpacityVariation
	case "debrisFlashDuration":
		return debrisFlashDuration
	case "debrisFlashOpacity":
		return debrisFlashOpacity
	case "debrisImpactSpeedFactor":
		return debrisImpactSpeedFactor
	case "debrisBallPieceChance":
		return debrisBallPieceChance
	case "debrisTrianglePieceChance":
		return debrisTrianglePieceChance
	case "debrisStarPieceChance":
		return debrisStarPieceChance
	case "debrisStarPointsMin":
		return float64(debrisStarPointsMin)
	case "debrisStarPointsMax":
		return float64(debrisStarPointsMax)
	case "debrisGlassPieceChance":
		return debrisGlassPieceChance
	case "debrisGlassCornersMin":
		return float64(debrisGlassCornersMin)
	case "debrisGlassCornersMax":
		return float64(debrisGlassCornersMax)
	case "debrisSliverPieceChance":
		return debrisSliverPieceChance
	case "debrisMaxChunkAspectRatio":
		return debrisMaxChunkAspectRatio
	case "debrisSizeScale":
		return debrisSizeScale
	case "debrisBrickCollisionDelay":
		return debrisBrickCollisionDelay
	case "debrisGravityScale":
		return debrisGravityScale
	case "debrisAirDrag":
		return debrisAirDrag
	case "debrisRestitution":
		return debrisRestitution
	case "debrisFriction":
		return debrisFriction
	case "debrisExplosionSpeedMin":
		return debrisExplosionSpeedMin
	case "debrisExplosionSpeedMax":
		return debrisExplosionSpeedMax
	case "debrisAngularSpeedMin":
		return debrisAngularSpeedMin
	case "debrisAngularSpeedMax":
		return debrisAngularSpeedMax
	case "debrisAngularDrag":
		return debrisAngularDrag
	case "debrisAngularStopSpeed":
		return debrisAngularStopSpeed
	case "debrisBallInfluence":
		return debrisBallInfluence
	case "debrisFieldScale":
		return debrisFieldScale
	case "debrisMagnetScale":
		return debrisMagnetScale
	case "debrisMaxSpeed":
		return debrisMaxSpeed
	case "debrisFrontLayerSpeed":
		return debrisFrontLayerSpeed
	case "debrisOffscreenMargin":
		return debrisOffscreenMargin
	}
	return 0
}

func setDebrisEditorRawValue(key string, value float64) {
	switch key {
	case "debrisPiecesMin":
		debrisPiecesMin = int(math.Round(value))
	case "debrisPiecesMax":
		debrisPiecesMax = int(math.Round(value))
	case "debrisMaxActivePieces":
		debrisMaxActivePieces = int(math.Round(value))
	case "debrisLifetime":
		debrisLifetime = value
	case "debrisLifetimeVariationPercent":
		debrisLifetimeVariationPercent = value
	case "debrisFadeDuration":
		debrisFadeDuration = value
	case "debrisStartOpacity":
		debrisStartOpacity = value
	case "debrisStartOpacityVariation":
		debrisStartOpacityVariation = value
	case "debrisFlashDuration":
		debrisFlashDuration = value
	case "debrisFlashOpacity":
		debrisFlashOpacity = value
	case "debrisImpactSpeedFactor":
		debrisImpactSpeedFactor = value
	case "debrisBallPieceChance":
		debrisBallPieceChance = value
	case "debrisTrianglePieceChance":
		debrisTrianglePieceChance = value
	case "debrisStarPieceChance":
		debrisStarPieceChance = value
	case "debrisStarPointsMin":
		debrisStarPointsMin = int(math.Round(value))
	case "debrisStarPointsMax":
		debrisStarPointsMax = int(math.Round(value))
	case "debrisGlassPieceChance":
		debrisGlassPieceChance = value
	case "debrisGlassCornersMin":
		debrisGlassCornersMin = int(math.Round(value))
	case "debrisGlassCornersMax":
		debrisGlassCornersMax = int(math.Round(value))
	case "debrisSliverPieceChance":
		debrisSliverPieceChance = value
	case "debrisMaxChunkAspectRatio":
		debrisMaxChunkAspectRatio = value
	case "debrisSizeScale":
		debrisSizeScale = value
	case "debrisBrickCollisionDelay":
		debrisBrickCollisionDelay = value
	case "debrisGravityScale":
		debrisGravityScale = value
	case "debrisAirDrag":
		debrisAirDrag = value
	case "debrisRestitution":
		debrisRestitution = value
	case "debrisFriction":
		debrisFriction = value
	case "debrisExplosionSpeedMin":
		debrisExplosionSpeedMin = value
	case "debrisExplosionSpeedMax":
		debrisExplosionSpeedMax = value
	case "debrisAngularSpeedMin":
		debrisAngularSpeedMin = value
	case "debrisAngularSpeedMax":
		debrisAngularSpeedMax = value
	case "debrisAngularDrag":
		debrisAngularDrag = value
	case "debrisAngularStopSpeed":
		debrisAngularStopSpeed = value
	case "debrisBallInfluence":
		debrisBallInfluence = value
	case "debrisFieldScale":
		debrisFieldScale = value
	case "debrisMagnetScale":
		debrisMagnetScale = value
	case "debrisMaxSpeed":
		debrisMaxSpeed = value
	case "debrisFrontLayerSpeed":
		debrisFrontLayerSpeed = value
	case "debrisOffscreenMargin":
		debrisOffscreenMargin = value
	}
}

func captureDebrisEditorSnapshot() debrisEditorSnapshot {
	values := make(map[string]float64, len(debrisEditorSliderSpecs))
	for _, spec := range debrisEditorSliderSpecs {
		values[spec.key] = debrisEditorValue(spec.key)
	}
	return debrisEditorSnapshot{enabled: debrisEnabled, values: values}
}

func defaultDebrisEditorValue(key string) float64 {
	switch key {
	case "debrisPiecesMin":
		return float64(defaultDebrisPiecesMin)
	case "debrisPiecesMax":
		return float64(defaultDebrisPiecesMax)
	case "debrisMaxActivePieces":
		return float64(defaultDebrisMaxActivePieces)
	case "debrisLifetime":
		return defaultDebrisLifetime
	case "debrisLifetimeVariationPercent":
		return defaultDebrisLifetimeVariationPercent
	case "debrisFadeDuration":
		return defaultDebrisFadeDuration
	case "debrisStartOpacity":
		return defaultDebrisStartOpacity
	case "debrisStartOpacityVariation":
		return defaultDebrisStartOpacityVariation
	case "debrisFlashDuration":
		return defaultDebrisFlashDuration
	case "debrisFlashOpacity":
		return defaultDebrisFlashOpacity
	case "debrisImpactSpeedFactor":
		return defaultDebrisImpactSpeedFactor
	case "debrisBallPieceChance":
		return defaultDebrisBallPieceChance
	case "debrisTrianglePieceChance":
		return defaultDebrisTrianglePieceChance
	case "debrisStarPieceChance":
		return defaultDebrisStarPieceChance
	case "debrisStarPointsMin":
		return float64(defaultDebrisStarPointsMin)
	case "debrisStarPointsMax":
		return float64(defaultDebrisStarPointsMax)
	case "debrisGlassPieceChance":
		return defaultDebrisGlassPieceChance
	case "debrisGlassCornersMin":
		return float64(defaultDebrisGlassCornersMin)
	case "debrisGlassCornersMax":
		return float64(defaultDebrisGlassCornersMax)
	case "debrisSliverPieceChance":
		return defaultDebrisSliverPieceChance
	case "debrisMaxChunkAspectRatio":
		return defaultDebrisMaxChunkAspectRatio
	case "debrisSizeScale":
		return defaultDebrisSizeScale
	case "debrisBrickCollisionDelay":
		return defaultDebrisBrickCollisionDelay
	case "debrisGravityScale":
		return defaultDebrisGravityScale
	case "debrisAirDrag":
		return defaultDebrisAirDrag
	case "debrisRestitution":
		return defaultDebrisRestitution
	case "debrisFriction":
		return defaultDebrisFriction
	case "debrisExplosionSpeedMin":
		return defaultDebrisExplosionSpeedMin
	case "debrisExplosionSpeedMax":
		return defaultDebrisExplosionSpeedMax
	case "debrisAngularSpeedMin":
		return defaultDebrisAngularSpeedMin
	case "debrisAngularSpeedMax":
		return defaultDebrisAngularSpeedMax
	case "debrisAngularDrag":
		return defaultDebrisAngularDrag
	case "debrisAngularStopSpeed":
		return defaultDebrisAngularStopSpeed
	case "debrisBallInfluence":
		return defaultDebrisBallInfluence
	case "debrisFieldScale":
		return defaultDebrisFieldScale
	case "debrisMagnetScale":
		return defaultDebrisMagnetScale
	case "debrisMaxSpeed":
		return defaultDebrisMaxSpeed
	case "debrisFrontLayerSpeed":
		return defaultDebrisFrontLayerSpeed
	case "debrisOffscreenMargin":
		return defaultDebrisOffscreenMargin
	}
	return 0
}

func defaultDebrisEditorSnapshot() debrisEditorSnapshot {
	values := make(map[string]float64, len(debrisEditorSliderSpecs))
	for _, spec := range debrisEditorSliderSpecs {
		values[spec.key] = defaultDebrisEditorValue(spec.key)
	}
	return debrisEditorSnapshot{enabled: defaultDebrisEnabled, values: values}
}

func applyDebrisEditorSnapshot(snapshot debrisEditorSnapshot) {
	oldLifetime := debrisLifetime
	oldOpacity := debrisStartOpacity
	oldSize := debrisSizeScale
	debrisEnabled = snapshot.enabled
	for _, spec := range debrisEditorSliderSpecs {
		if value, ok := snapshot.values[spec.key]; ok {
			setDebrisEditorRawValue(spec.key, value)
		}
	}
	normalizeDebrisSettings()
	if !debrisEnabled {
		brickDebris = brickDebris[:0]
		return
	}
	if oldLifetime > 0 && debrisLifetime != oldLifetime {
		for i := range brickDebris {
			progress := clampFloat(brickDebris[i].age/brickDebris[i].lifetime, 0, 0.999)
			variation := brickDebris[i].lifetime / oldLifetime
			brickDebris[i].lifetime = math.Max(0.05, debrisLifetime*variation)
			brickDebris[i].age = progress * brickDebris[i].lifetime
		}
	}
	if debrisStartOpacity != oldOpacity {
		delta := debrisStartOpacity - oldOpacity
		for i := range brickDebris {
			brickDebris[i].startOpacity = clampFloat(brickDebris[i].startOpacity+delta, 0, 1)
		}
	}
	if oldSize > 0 && debrisSizeScale != oldSize {
		ratio := debrisSizeScale / oldSize
		for i := range brickDebris {
			for point := 0; point < brickDebris[i].pointCount; point++ {
				brickDebris[i].points[point*2] *= ratio
				brickDebris[i].points[point*2+1] *= ratio
			}
			brickDebris[i].radius *= ratio
			brickDebris[i].mass *= ratio * ratio
			rebuildDebrisRenderPath(&brickDebris[i])
		}
	}
	for i := range brickDebris {
		speed := math.Hypot(brickDebris[i].vx, brickDebris[i].vy)
		if debrisMaxSpeed > 0 && speed > debrisMaxSpeed {
			scale := debrisMaxSpeed / speed
			brickDebris[i].vx *= scale
			brickDebris[i].vy *= scale
		}
		brickDebris[i].omega = clampFloat(brickDebris[i].omega, -debrisAngularSpeedMax, debrisAngularSpeedMax)
	}
	if len(brickDebris) > debrisMaxActivePieces {
		brickDebris = brickDebris[len(brickDebris)-debrisMaxActivePieces:]
	}
}

func refreshDebrisEditorControls() {
	if !physicsEditorDebrisCheckbox.IsUndefined() && !physicsEditorDebrisCheckbox.IsNull() {
		physicsEditorDebrisCheckbox.Set("checked", debrisEnabled)
	}
	for _, spec := range debrisEditorSliderSpecs {
		input, inputOK := physicsEditorInputs[spec.key]
		label, labelOK := physicsEditorValueLabels[spec.key]
		if !inputOK || !labelOK || input.IsUndefined() || input.IsNull() {
			continue
		}
		value := debrisEditorValue(spec.key)
		text := formatDebrisSliderValue(spec, value)
		input.Set("value", text)
		label.Set("textContent", text)
	}
}

func applyDebrisEditorValue(spec debrisSliderSpec, value float64) {
	value = clampFloat(value, spec.min, spec.max)
	oldLifetime := debrisLifetime
	oldOpacity := debrisStartOpacity
	oldSize := debrisSizeScale
	setDebrisEditorRawValue(spec.key, value)
	normalizeDebrisSettings()

	switch spec.key {
	case "debrisLifetime":
		if oldLifetime > 0 {
			for i := range brickDebris {
				progress := clampFloat(brickDebris[i].age/brickDebris[i].lifetime, 0, 0.999)
				variation := brickDebris[i].lifetime / oldLifetime
				brickDebris[i].lifetime = math.Max(0.05, debrisLifetime*variation)
				brickDebris[i].age = progress * brickDebris[i].lifetime
			}
		}
	case "debrisStartOpacity":
		delta := debrisStartOpacity - oldOpacity
		for i := range brickDebris {
			brickDebris[i].startOpacity = clampFloat(brickDebris[i].startOpacity+delta, 0, 1)
		}
	case "debrisSizeScale":
		if oldSize > 0 {
			ratio := debrisSizeScale / oldSize
			for i := range brickDebris {
				for p := 0; p < brickDebris[i].pointCount; p++ {
					brickDebris[i].points[p*2] *= ratio
					brickDebris[i].points[p*2+1] *= ratio
				}
				brickDebris[i].radius *= ratio
				brickDebris[i].mass *= ratio * ratio
				rebuildDebrisRenderPath(&brickDebris[i])
			}
		}
	case "debrisMaxSpeed":
		for i := range brickDebris {
			speed := math.Hypot(brickDebris[i].vx, brickDebris[i].vy)
			if debrisMaxSpeed > 0 && speed > debrisMaxSpeed {
				scale := debrisMaxSpeed / speed
				brickDebris[i].vx *= scale
				brickDebris[i].vy *= scale
			}
		}
	case "debrisAngularSpeedMax":
		for i := range brickDebris {
			brickDebris[i].omega = clampFloat(brickDebris[i].omega, -debrisAngularSpeedMax, debrisAngularSpeedMax)
		}
	case "debrisMaxActivePieces":
		if len(brickDebris) > debrisMaxActivePieces {
			brickDebris = brickDebris[len(brickDebris)-debrisMaxActivePieces:]
		}
	}
	refreshDebrisEditorControls()
}

func formatPhysicsSliderValue(spec physicsSliderSpec, value float64) string {
	return strconv.FormatFloat(value, 'f', spec.precision, 64)
}

func physicsEditorSpecValue(spec physicsSliderSpec) float64 {
	return physicsFloatSettingValue(&physicsConfig, spec.key)
}

func refreshPhysicsEditorControls() {
	if !physicsEditorCornerPhysicsCheckbox.IsUndefined() && !physicsEditorCornerPhysicsCheckbox.IsNull() {
		physicsEditorCornerPhysicsCheckbox.Set("checked", physicsConfig.cornerPhysicsEnabled)
	}
	if !physicsEditorCornerPhysicsAllBricksCheckbox.IsUndefined() && !physicsEditorCornerPhysicsAllBricksCheckbox.IsNull() {
		physicsEditorCornerPhysicsAllBricksCheckbox.Set("checked", physicsConfig.cornerPhysicsAllBricks)
	}
	for _, spec := range physicsEditorSliderSpecs {
		input, inputOK := physicsEditorInputs[spec.key]
		label, labelOK := physicsEditorValueLabels[spec.key]
		if !inputOK || !labelOK || input.IsUndefined() || input.IsNull() {
			continue
		}
		value := physicsEditorSpecValue(spec)
		text := formatPhysicsSliderValue(spec, value)
		input.Set("value", text)
		label.Set("textContent", text)
	}
	refreshDebrisEditorControls()
	refreshAutoPaddleEditorControls()
	refreshAutoPaddleControl()
	if !physicsEditorLiveCheckbox.IsUndefined() && !physicsEditorLiveCheckbox.IsNull() {
		physicsEditorLiveCheckbox.Set("checked", physicsEditorLiveSimulation)
	}
}

func recomputeBrickTilts() {
	first, last, _ := brickVerticalBounds()
	for i := range bricks {
		row := bricks[i].row
		if bricksVerticalFlipped {
			row = first + last - row
		}
		bricks[i].tiltRadians = brickMicroTiltRadians(currentLevelIndex, row, bricks[i].col)
		if bricksVerticalFlipped {
			bricks[i].tiltRadians = -bricks[i].tiltRadians
		}
	}
	bricksDirty = true
}

func applyPhysicsEditorValue(spec physicsSliderSpec, value float64) {
	value = clampFloat(value, spec.min, spec.max)
	if !validPhysicsFloatSetting(spec.key, value) {
		return
	}
	setPhysicsFloatSetting(&physicsConfig, spec.key, value)

	switch spec.key {
	case "gravity":
		refreshCurrentGravity()
	case "maxSpin":
		forEachActiveBall(func(b *Ball, _ int) {
			b.omega = clampFloat(b.omega, -physicsConfig.maxSpin, physicsConfig.maxSpin)
		})
	case "brickTiltMinDegrees", "brickTiltMaxDegrees":
		recomputeBrickTilts()
	}
}

func physicsEditorLevelText() string {
	lines := make([]string, 0, len(autoPaddleEditorSliderSpecs)+len(physicsEditorSliderSpecs)+len(debrisEditorSliderSpecs)+2)
	for _, spec := range autoPaddleEditorSliderSpecs {
		lines = append(lines, spec.key+"="+formatAutoPaddleSliderValue(spec, autoPaddleEditorValue(spec.key)))
	}
	lines = append(lines, "cornerPhysicsEnabled="+strconv.FormatBool(physicsConfig.cornerPhysicsEnabled))
	lines = append(lines, "cornerPhysicsAllBricks="+strconv.FormatBool(physicsConfig.cornerPhysicsAllBricks))
	for _, spec := range physicsEditorSliderSpecs {
		value := physicsEditorSpecValue(spec)
		lines = append(lines, spec.key+"="+formatPhysicsSliderValue(spec, value))
	}
	lines = append(lines, "debris="+strconv.FormatBool(debrisEnabled))
	for _, spec := range debrisEditorSliderSpecs {
		lines = append(lines, spec.key+"="+formatDebrisSliderValue(spec, debrisEditorValue(spec.key)))
	}
	return strings.Join(lines, "\n")
}

func physicsEditorConfigText() string {
	lines := []string{"// Auto-paddle, physics and debris values copied from the in-game E panel."}
	for _, spec := range autoPaddleEditorSliderSpecs {
		lines = append(lines, spec.configName+" = "+formatAutoPaddleSliderValue(spec, autoPaddleEditorValue(spec.key)))
	}
	lines = append(lines, "defaultPhysicsCornerPhysicsEnabled = "+strconv.FormatBool(physicsConfig.cornerPhysicsEnabled))
	lines = append(lines, "defaultPhysicsCornerPhysicsAllBricks = "+strconv.FormatBool(physicsConfig.cornerPhysicsAllBricks))
	for _, spec := range physicsEditorSliderSpecs {
		value := physicsEditorSpecValue(spec)
		lines = append(lines, spec.configName+" = "+formatPhysicsSliderValue(spec, value))
	}
	lines = append(lines, "defaultDebrisEnabled = "+strconv.FormatBool(debrisEnabled))
	for _, spec := range debrisEditorSliderSpecs {
		lines = append(lines, spec.configName+" = "+formatDebrisSliderValue(spec, debrisEditorValue(spec.key)))
	}
	return strings.Join(lines, "\n")
}

func physicsEditorClipboardText() string {
	mode := "level"
	if !physicsEditorExportSelect.IsUndefined() && !physicsEditorExportSelect.IsNull() {
		mode = physicsEditorExportSelect.Get("value").String()
	}
	switch mode {
	case "config":
		return physicsEditorConfigText()
	case "both":
		return "# LEVEL FILE\n" + physicsEditorLevelText() +
			"\n\n// CONFIG.GO\n" + physicsEditorConfigText()
	default:
		return physicsEditorLevelText()
	}
}

func resetPhysicsEditorTo(settings physicsSettings, debris debrisEditorSnapshot, autoHitVariation float64) {
	physicsConfig = settings
	applyDebrisEditorSnapshot(debris)
	autoPaddleHitVariation = clampFloat(autoHitVariation, 0, 0.90)
	resetAutoPaddleHitPlan()
	refreshCurrentGravity()
	forEachActiveBall(func(b *Ball, _ int) {
		b.omega = clampFloat(b.omega, -physicsConfig.maxSpin, physicsConfig.maxSpin)
	})
	recomputeBrickTilts()
	refreshPhysicsEditorControls()
}

func createPhysicsEditorButton(text string, handler func()) js.Value {
	button := doc.Call("createElement", "button")
	button.Set("textContent", text)
	setStyle(button, "background", "#252525")
	setStyle(button, "color", "#ffffff")
	setStyle(button, "border", "1px solid rgba(255,255,255,0.35)")
	setStyle(button, "borderRadius", "5px")
	setStyle(button, "padding", "7px 11px")
	setStyle(button, "cursor", "pointer")
	callback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			args[0].Call("preventDefault")
			args[0].Call("stopPropagation")
		}
		handler()
		return nil
	})
	keepPhysicsEditorCallback(callback)
	button.Call("addEventListener", "click", callback)
	return button
}

func createPhysicsEditorCheckbox(labelText string, initial bool, handler func(bool)) js.Value {
	label := doc.Call("createElement", "label")
	setStyle(label, "display", "inline-flex")
	setStyle(label, "alignItems", "center")
	setStyle(label, "gap", "7px")
	setStyle(label, "fontSize", "13px")
	setStyle(label, "cursor", "pointer")
	input := doc.Call("createElement", "input")
	input.Set("type", "checkbox")
	input.Set("checked", initial)
	label.Call("appendChild", input)
	text := doc.Call("createElement", "span")
	text.Set("textContent", labelText)
	label.Call("appendChild", text)
	callback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler(input.Get("checked").Bool())
		return nil
	})
	keepPhysicsEditorCallback(callback)
	input.Call("addEventListener", "change", callback)
	return label
}

func appendPhysicsEditorGroup(panel js.Value, titleText string) {
	group := doc.Call("createElement", "div")
	group.Set("textContent", titleText)
	setStyle(group, "margin", "15px 0 7px")
	setStyle(group, "paddingBottom", "4px")
	setStyle(group, "borderBottom", "1px solid rgba(255,255,255,0.18)")
	setStyle(group, "fontSize", "14px")
	setStyle(group, "fontWeight", "700")
	setStyle(group, "letterSpacing", "0.04em")
	panel.Call("appendChild", group)
}

func ensurePhysicsEditorPanel() {

}

func openPhysicsEditor() {
	if physicsEditorVisible {
		return
	}
	markLevelRunAssisted()
	ensurePhysicsEditorPanel()
	physicsEditorPreviousPaused = paused
	physicsEditorLiveSimulation = !paused
	physicsEditorOpeningConfig = physicsConfig
	physicsEditorOpeningDebris = captureDebrisEditorSnapshot()
	physicsEditorOpeningAutoHitVariation = autoPaddleHitVariation
	physicsEditorVisible = true
	leftPressed = false
	rightPressed = false
	mobileLeftHeld = false
	mobileRightHeld = false
	touchControlActive = false
	mouseControlActive = false
	paddle.vx = 0
	resetPaddleSpinHistory()
	debugOverlayVisible = false
	physicsOverlayVisible = false
	statsOverlayVisible = false
	refreshPhysicsEditorControls()

}

func closePhysicsEditor() {
	if !physicsEditorVisible {
		return
	}
	physicsEditorVisible = false
	paused = physicsEditorPreviousPaused
	nativeClipboard = physicsEditorClipboardText()
	leftPressed = false
	rightPressed = false
	syncRenderInterpolation()
}

func togglePhysicsEditor() {
	if physicsEditorVisible {
		closePhysicsEditor()
	} else {
		openPhysicsEditor()
	}
}

func drawOverlayLines(lines []string) {
	panelX, panelY, _, _, maxRows := debugOverlayGeometry(len(lines))

	const lineHeight = 18.0
	const padding = 12.0
	const columnWidth = 420.0

	ctx.Call("save")
	ctx.Set("fillStyle", palette[4])
	ctx.Set("font", "14px GameFont, monospace")
	ctx.Set("textAlign", "left")

	for i, line := range lines {
		column := i / maxRows
		row := i % maxRows
		x := panelX + padding + float64(column)*columnWidth
		y := panelY + padding + lineHeight*float64(row+1) - 3
		ctx.Call("fillText", line, x, y)
	}
	ctx.Call("restore")
}

func drawDebugOverlay() {
	if debugOverlayVisible {
		drawOverlayLines(debugOverlayLines())
	}
}

func drawPhysicsOverlayLines(lines []string) {
	panelX, panelY, _, _, maxRows := physicsOverlayGeometry(len(lines))

	const lineHeight = 18.0
	const padding = 12.0
	const columnWidth = 420.0

	ctx.Call("save")
	ctx.Set("fillStyle", palette[4])
	ctx.Set("font", "14px GameFont, monospace")
	ctx.Set("textAlign", "left")

	for i, line := range lines {
		column := i / maxRows
		row := i % maxRows
		x := panelX + padding + float64(column)*columnWidth
		y := panelY + padding + lineHeight*float64(row+1) - 3
		ctx.Call("fillText", line, x, y)
	}
	ctx.Call("restore")
}

func drawPhysicsOverlay() {
	if physicsOverlayVisible {
		drawPhysicsOverlayLines(physicsOverlayLines())
	}
}

func drawStatsOverlay() {
	if statsOverlayVisible {
		drawPhysicsOverlayLines(statsOverlayLines())
	}
}

func drawFPSMiniOverlay() {
	if !fpsMiniOverlayVisible || physicsOverlayVisible || debugOverlayVisible || statsOverlayVisible {
		return
	}

	const margin = 10.0
	const lineHeight = 20.0
	x := canvasWidth - margin
	y := canvasHeight - margin - lineHeight

	ctx.Call("save")
	ctx.Set("fillStyle", palette[4])
	ctx.Set("font", "16px GameFont, monospace")
	ctx.Set("textAlign", "right")
	ctx.Call("fillText", "FPS "+fmt.Sprintf("%.1f", fpsCurrent), x, y)
	ctx.Call("fillText", "LOWEST "+formatLowestRenderFPS(), x, y+lineHeight)
	ctx.Call("restore")
}

func drawCenteredOverlay() {
	ctx.Call("save")
	ctx.Set("fillStyle", "rgba(0, 0, 0, 0.5)")
	ctx.Call("fillRect", 0, 0, canvasWidth, canvasHeight)
	ctx.Call("restore")
}

func draw(alpha float64) {
	renderState := interpolatedRenderSnapshot(alpha)
	ctx.Set("fillStyle", palette[0])
	ctx.Call("fillRect", 0, 0, canvasWidth, canvasHeight)

	debrisRenderedLastFrame = 0
	debrisSkippedLastFrame = 0

	// Slow debris stays behind living bricks. Fast debris gets a second depth pass
	// after the brick cache, so energetic shards visibly fly over intact bricks.
	// Each fragment is still drawn exactly once; the extra work is only a cheap
	// velocity-squared classification during the second scan.
	drawBrickDebris(alpha, false)

	if bricksDirty {
		rebuildBrickCanvas()
	}
	ctx.Call("drawImage", brickCanvas, 0, 0)
	drawBrickDebris(alpha, true)
	drawZapperBolts()

	if renderState.blackHoleActive && showBlackHole {
		ctx.Set("fillStyle", "#000000")
		ctx.Set("strokeStyle", "#9d4edd")
		ctx.Set("lineWidth", 5)
		ctx.Call("beginPath")
		ctx.Call("arc", renderState.blackHoleX, renderState.blackHoleY, 28, 0, 2*math.Pi)
		ctx.Call("fill")
		ctx.Call("stroke")

		ctx.Set("strokeStyle", "#c77dff")
		ctx.Set("lineWidth", 2)
		ctx.Call("beginPath")
		ctx.Call("arc", renderState.blackHoleX, renderState.blackHoleY, 42, 0, 2*math.Pi)
		ctx.Call("stroke")
	}

	for i := 0; i < renderState.ballCount; i++ {
		fillColor, spinMarkerColor := palette[3], palette[5]
		if i%2 == 1 {
			fillColor, spinMarkerColor = palette[7], palette[8]
		}
		renderBall := renderState.balls[i]
		drawBall(renderBall.x, renderBall.y, ballRadius, renderBall.angle, fillColor, spinMarkerColor)
	}

	ctx.Set("fillStyle", palette[1])
	ctx.Call("beginPath")
	ctx.Call("roundRect", renderState.paddleX, paddle.y, paddle.w, paddle.h, paddleRadius)
	ctx.Call("fill")

	ctx.Set("fillStyle", palette[4])
	ctx.Set(
		"font",
		"18px GameFont, monospace",
	)
	// Match the P display: 10 units from the canvas edge plus 12 units of
	// internal text padding. Its first-line baseline would be 37 at the top.
	const (
		hudTextX      = 22.0
		hudFirstLineY = 37.0
		hudLineStep   = 30.0
	)
	updateHUDCache()
	ctx.Call("fillText", hudLivesText, hudTextX, hudFirstLineY)
	ctx.Call("fillText", hudLevelText, hudTextX, hudFirstLineY+hudLineStep)
	ctx.Call("fillText", hudScoreText, hudTextX, hudFirstLineY+2*hudLineStep)

	currentSpeed := math.Hypot(ball.vx, ball.vy)
	ctx.Call("fillText", "Speed: "+fmt.Sprintf("%.0f", currentSpeed), hudTextX, hudFirstLineY+3*hudLineStep)

	if math.Abs(ball.omega) > 100 {
		ctx.Set("fillStyle", "#ff0000")
	}
	ctx.Call("fillText", "Spin:  "+fmt.Sprintf("%+.0f", ball.omega), hudTextX, hudFirstLineY+4*hudLineStep)
	ctx.Set("fillStyle", palette[4])

	for i, message := range statusMessages {
		y := hudFirstLineY + 160.0 + float64(i)*hudLineStep
		ctx.Call("fillText", message.text, hudTextX, y)
	}

	if waitingForStart && !gameOver {
		drawCenteredOverlay()
		ctx.Set("fillStyle", palette[4])
		ctx.Set("textAlign", "center")
		ctx.Set("font", "72px GameFont, monospace")
		ctx.Call("fillText", levelStartTitle, canvasWidth/2, canvasHeight/2)
		ctx.Set("font", "28px GameFont, monospace")
		//ctx.Call("fillText", "Press Space, Enter, click, or touch", canvasWidth/2, canvasHeight/2+70)
		ctx.Set("textAlign", "start")
	}

	if levelAdvancePending && !gameOver {
		drawCenteredOverlay()
		run := levelRunStats
		if levelCompletedStatsValid {
			run = levelCompletedStats
		}
		centerX := canvasWidth / 2
		centerY := canvasHeight / 2
		assisted := !levelRunRecordEligible

		// Center the complete title/table/prompt composition as one vertical block.
		// The title keeps its original large 64px size; only the result table is smaller.
		blockHeight := 418.0
		if assisted {
			blockHeight += 42.0
		}
		blockTop := centerY - blockHeight/2
		titleY := blockTop + 58.0
		headerY := titleY + 56.0
		rowY := headerY + 38.0

		ctx.Set("fillStyle", palette[4])
		ctx.Set("textAlign", "center")
		ctx.Set("font", "64px GameFont, monospace")
		ctx.Call("fillText", "LEVEL COMPLETE "+gameSpeedResultLabel(run.speedPercent, run.mixedSpeed), centerX, titleY)

		// Results are a centered table, while every cell is left-aligned.
		const (
			tableWidth = 1080.0
			labelWidth = 235.0
			runWidth   = 195.0
			wasWidth   = 195.0
			bestWidth  = 195.0
			rowStep    = 36.0
		)
		tableLeft := centerX - tableWidth/2
		labelX := tableLeft
		runX := tableLeft + labelWidth
		wasX := runX + runWidth
		bestX := wasX + wasWidth
		newX := bestX + bestWidth

		ctx.Set("textAlign", "left")
		ctx.Set("font", "18px GameFont, monospace")
		ctx.Call("fillText", "THIS RUN", runX, headerY)
		ctx.Call("fillText", "WAS", wasX, headerY)
		ctx.Call("fillText", "BEST", bestX, headerY)

		ctx.Set("font", "24px GameFont, monospace")
		drawResultRow := func(row int, label, current, was, best string, isNew bool) {
			y := rowY + float64(row)*rowStep
			ctx.Call("fillText", label, labelX, y)
			ctx.Call("fillText", current, runX, y)
			ctx.Call("fillText", was, wasX, y)
			ctx.Call("fillText", best, bestX, y)
			if isNew {
				ctx.Call("fillText", "NEW BEST!", newX, y)
			}
		}

		wasChain := "--"
		if levelPreviousBests.chainFound {
			wasChain = strconv.Itoa(levelPreviousBests.bestChain) + " bricks"
		}
		bestChain := "--"
		if levelCurrentBests.chainFound {
			bestChain = strconv.Itoa(levelCurrentBests.bestChain) + " bricks"
		}
		drawResultRow(0, "TIME", formatLevelTimeSeconds(run.elapsedSeconds), formatBestTime(levelPreviousBests), formatBestTime(levelCurrentBests), levelCompletionNewRecords.time)
		drawResultRow(1, "FASTEST BALL", fmt.Sprintf("%.0f", run.fastestSpeed), formatBestInt(levelPreviousBests.fastestSpeed, levelPreviousBests.speedFound), formatBestInt(levelCurrentBests.fastestSpeed, levelCurrentBests.speedFound), levelCompletionNewRecords.speed)
		drawResultRow(2, "FASTEST SPIN", fmt.Sprintf("%.0f", run.fastestSpin), formatBestInt(levelPreviousBests.fastestSpin, levelPreviousBests.spinFound), formatBestInt(levelCurrentBests.fastestSpin, levelCurrentBests.spinFound), levelCompletionNewRecords.spin)
		drawResultRow(3, "BEST CHAIN", strconv.Itoa(run.bestChain)+" bricks", wasChain, bestChain, levelCompletionNewRecords.chain)
		drawResultRow(4, "BALLS LOST", strconv.Itoa(run.ballsLost), formatBestInt(levelPreviousBests.ballsLost, levelPreviousBests.ballsLostFound), formatBestInt(levelCurrentBests.ballsLost, levelCurrentBests.ballsLostFound), levelCompletionNewRecords.ballsLost)

		lastRowY := rowY + 4*rowStep
		nextY := lastRowY + 112.0
		ctx.Set("textAlign", "center")
		if assisted {
			ctx.Set("font", "20px GameFont, monospace")
			message := "ASSISTED RUN - PERSONAL BESTS NOT UPDATED"
			if run.mixedSpeed {
				message = "MIXED SPEED - PERSONAL BESTS NOT UPDATED"
			}
			ctx.Call("fillText", message, centerX, nextY)
			nextY += 42.0
		}

		// Keep the continue prompt large and centered, with about twice the title-to-stats gap below the table.
		ctx.Set("font", "28px GameFont, monospace")
		ctx.Call("fillText", "SPACE / ENTER / CLICK / TAP TO CONTINUE", centerX, nextY)
		ctx.Set("textAlign", "start")
	}

	if paused && !physicsEditorVisible && !gameOver && !waitingForStart && !levelAdvancePending {
		drawCenteredOverlay()
		ctx.Set("fillStyle", palette[4])
		ctx.Set("textAlign", "center")
		ctx.Set("font", "72px GameFont, monospace")
		ctx.Call("fillText", "PAUSED", canvasWidth/2, canvasHeight/2)
		ctx.Set("font", "28px GameFont, monospace")
		//ctx.Call("fillText", "Press Space, Escape, click, or touch", canvasWidth/2, canvasHeight/2+70)
		ctx.Set("textAlign", "start")
	}

	if gameOver {
		drawCenteredOverlay()
		ctx.Set("fillStyle", palette[4])
		ctx.Set("textAlign", "center")
		centerX := canvasWidth / 2
		centerY := canvasHeight / 2

		if win {
			if fullGameCompletion.valid && fullGameCompletion.levelCount == len(levels) {
				assisted := !fullGameCompletion.recordEligible
				blockHeight := 330.0
				if assisted {
					blockHeight += 40.0
				}
				blockTop := centerY - blockHeight/2
				titleY := blockTop + 100.0
				descriptionY := titleY + 58.0
				fullHeaderY := descriptionY + 52.0
				fullRowY := fullHeaderY + 36.0

				ctx.Set("font", "112px GameFont, monospace")
				ctx.Call("fillText", "YOU WIN!", centerX, titleY)
				ctx.Set("font", "22px GameFont, monospace")
				ctx.Call("fillText", "You beat the full game in one go. "+gameSpeedResultLabel(fullGameCompletion.speedPercent, fullGameCompletion.mixedSpeed), centerX, descriptionY)

				wasTime := "--"
				if fullGameCompletion.previousBestFound {
					wasTime = formatFullGameDurationMs(fullGameCompletion.previousBestTimeMs)
				}
				bestTime := "--"
				if fullGameCompletion.bestFound {
					bestTime = formatFullGameDurationMs(fullGameCompletion.bestTimeMs)
				}

				// Center the full table as a block; individual cells remain left-aligned.
				const (
					fullTableWidth = 1280.0
					fullLabelWidth = 150.0
					fullRunWidth   = 330.0
					fullWasWidth   = 330.0
					fullBestWidth  = 330.0
				)
				fullLeft := centerX - fullTableWidth/2
				fullLabelX := fullLeft
				fullRunX := fullLabelX + fullLabelWidth
				fullWasX := fullRunX + fullRunWidth
				fullBestX := fullWasX + fullWasWidth
				fullNewX := fullBestX + fullBestWidth

				ctx.Set("textAlign", "left")
				ctx.Set("font", "16px GameFont, monospace")
				ctx.Call("fillText", "THIS RUN", fullRunX, fullHeaderY)
				ctx.Call("fillText", "WAS", fullWasX, fullHeaderY)
				ctx.Call("fillText", "BEST", fullBestX, fullHeaderY)
				ctx.Set("font", "20px GameFont, monospace")
				ctx.Call("fillText", "TIME", fullLabelX, fullRowY)
				ctx.Call("fillText", formatFullGameDurationMs(fullGameCompletion.timeMs), fullRunX, fullRowY)
				ctx.Call("fillText", wasTime, fullWasX, fullRowY)
				ctx.Call("fillText", bestTime, fullBestX, fullRowY)
				if fullGameCompletion.recordEligible && fullGameCompletion.newRecord {
					ctx.Call("fillText", "NEW RECORD!", fullNewX, fullRowY)
				}

				nextY := fullRowY + 40.0
				ctx.Set("textAlign", "center")
				if assisted {
					ctx.Set("font", "18px GameFont, monospace")
					message := "ASSISTED RUN - FULL-GAME RECORD NOT UPDATED"
					if fullGameCompletion.mixedSpeed {
						message = "MIXED SPEED - FULL-GAME RECORD NOT UPDATED"
					}
					ctx.Call("fillText", message, centerX, nextY)
					nextY += 40.0
				}
				ctx.Set("font", "28px GameFont, monospace")
				ctx.Call("fillText", "SPACE / ENTER / CLICK / TAP TO CONTINUE", centerX, nextY+22.0)
			} else {
				// Fallback win screen, also centered as one composition.
				ctx.Set("font", "112px GameFont, monospace")
				ctx.Call("fillText", "YOU WIN!", centerX, centerY-34.0)
				ctx.Set("font", "28px GameFont, monospace")
				ctx.Call("fillText", "SPACE / ENTER / CLICK / TAP TO CONTINUE", centerX, centerY+72.0)
			}
		} else {
			// Keep the ordinary game-over composition centered as well.
			ctx.Set("font", "88px GameFont, monospace")
			ctx.Call("fillText", "GAME OVER", centerX, centerY-32.0)
			ctx.Set("font", "28px GameFont, monospace")
			ctx.Call("fillText", "SPACE / ENTER / CLICK / TAP TO RETRY", centerX, centerY+70.0)
		}

		ctx.Set("textAlign", "start")
	}

	drawDebugOverlay()
	drawPhysicsOverlay()
	drawStatsOverlay()
	drawFPSMiniOverlay()
}

func drawBall(x, y, radius, angle float64, fillColor, strokeColor string) {
	ctx.Call("save")
	ctx.Call("translate", x, y)
	ctx.Call("rotate", angle)
	ctx.Set("fillStyle", fillColor)
	ctx.Call("beginPath")
	ctx.Call("arc", 0, 0, radius, 0, 2*math.Pi)
	ctx.Call("fill")
	ctx.Set("strokeStyle", strokeColor)
	ctx.Set("lineWidth", 2)
	ctx.Call("beginPath")
	ctx.Call("moveTo", 0, -radius)
	ctx.Call("lineTo", 0, radius)
	ctx.Call("stroke")
	ctx.Set("fillStyle", strokeColor)
	ctx.Call("beginPath")
	ctx.Call("arc", 0, 0, 2, 0, 2*math.Pi)
	ctx.Call("fill")
	ctx.Call("restore")
}

// ---- Game loop ----
func gameLoop(this js.Value, args []js.Value) interface{} {
	// This budget spans all fixed physics steps executed by one visual frame.
	brickHitStartsThisFrame = 0

	defer func() {
		if r := recover(); r != nil {
			js.Global().Get("console").Call("error", "Panic in gameLoop:", r)
		}
	}()

	// Gamepads are polled once per visual frame, as recommended by the browser
	// Gamepad API. Replay owns the transport while active, so live gamepad actions
	// are suppressed until replay exits.
	if !replayModeActive {
		pollGamepadInput()
	}

	now := js.Global().Get("performance").Call("now").Float()
	rawDt := 0.0
	if lastTime != 0 {
		rawDt = (now - lastTime) / 1000.0
	}
	lastTime = now

	if replayModeActive {
		sampleLowestRenderFPS(0, false)
		updateReplayPlayback(rawDt)
		drawReplayCurrentFrame()
		js.Global().Call("requestAnimationFrame", loopFunc)
		return nil
	}

	visibleForFPS := pageIsVisibleForFPS()
	sampleLowestRenderFPS(rawDt, visibleForFPS && !waitingForStart && !paused && !gameOver && !levelAdvancePending && (!physicsEditorVisible || physicsEditorLiveSimulation))
	if visibleForFPS && rawDt > 0 && rawDt < 1.0 {
		fpsSampleElapsed += rawDt
		fpsSampleFrames++
		if fpsSampleElapsed >= renderFPSSampleWindowSeconds {
			sample := float64(fpsSampleFrames) / fpsSampleElapsed
			if fpsCurrent == 0 {
				fpsCurrent = sample
			} else {
				fpsCurrent = fpsCurrent*0.65 + sample*0.35
			}

			updateDebrisAdaptiveRenderStride()
			fpsSampleElapsed = 0
			fpsSampleFrames = 0
		}
	} else if !visibleForFPS || rawDt >= 1.0 {
		// Do not let a hidden/background tab or a long resume gap become the
		// permanent session minimum. Start a fresh visible-page sample instead.
		fpsSampleElapsed = 0
		fpsSampleFrames = 0
	}

	frameDt := rawDt * gameSpeedScale()
	if frameDt < 0 {
		frameDt = 0
	}
	if frameDt > physicsMaxFrameDelta {
		physicsSampleDroppedTime += frameDt - physicsMaxFrameDelta
		physicsDroppedTimeTotal += frameDt - physicsMaxFrameDelta
		physicsWarningTimer = physicsWarningHoldSeconds
		frameDt = physicsMaxFrameDelta
	}
	advanceBrickFlipAnimation(frameDt)
	physicsAccumulator += frameDt

	computeStart := js.Now()
	steps := 0
	for physicsAccumulator+1e-12 >= physicsStepSeconds && steps < physicsMaxCatchUpSteps {
		beginPhysicsStepForRendering()
		update(physicsStepSeconds)
		applyPendingMagicBrickFlip()
		physicsAccumulator -= physicsStepSeconds
		steps++
	}
	if physicsAccumulator >= physicsStepSeconds {
		dropped := physicsAccumulator - math.Mod(physicsAccumulator, physicsStepSeconds)
		physicsAccumulator = math.Mod(physicsAccumulator, physicsStepSeconds)
		physicsSampleDroppedTime += dropped
		physicsDroppedTimeTotal += dropped
		physicsWarningTimer = physicsWarningHoldSeconds
	}
	computeSeconds := (js.Now() - computeStart) / 1000.0

	// Console I/O is intentionally outside the physics-compute timer. The hit
	// detection itself is still measured; only browser DevTools logging is excluded.
	flushCornerPhysicsDebugLogs()

	physicsLastFrameSteps = steps
	if steps > physicsPeakFrameSteps {
		physicsPeakFrameSteps = steps
	}
	if rawDt > 0 && rawDt < 1.0 {
		physicsSampleElapsed += rawDt
		physicsSampleSteps += steps
		physicsSampleComputeTime += computeSeconds
		if physicsSampleElapsed >= 0.5 {
			physicsStepRateCurrent = float64(physicsSampleSteps) / physicsSampleElapsed
			physicsRealtimePercent = float64(physicsSampleSteps) * physicsStepSeconds / (physicsSampleElapsed * gameSpeedScale()) * 100
			physicsComputeLoad = physicsSampleComputeTime / physicsSampleElapsed * 100
			if physicsSampleDroppedTime > 0 || physicsComputeLoad >= 90 {
				physicsWarningTimer = physicsWarningHoldSeconds
			}
			physicsSampleElapsed = 0
			physicsSampleSteps = 0
			physicsSampleComputeTime = 0
			physicsSampleDroppedTime = 0
		}
	}
	if physicsWarningTimer > 0 {
		physicsWarningTimer = math.Max(0, physicsWarningTimer-rawDt)
	}

	renderInterpolationAlpha = clampFloat(physicsAccumulator/physicsStepSeconds, 0, 1)
	draw(renderInterpolationAlpha)
	js.Global().Call("requestAnimationFrame", loopFunc)
	return nil
}

func clampFloat(x, min, max float64) float64 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

func captureRenderSnapshot() renderSnapshot {
	snapshot := renderSnapshot{
		ballCount:       activeBallCount(),
		paddleX:         paddle.x,
		blackHoleX:      blackHoleX,
		blackHoleY:      blackHoleY,
		blackHoleActive: blackHoleActive,
	}
	for i := 0; i < snapshot.ballCount && i < len(snapshot.balls); i++ {
		b := activeBallAt(i)
		if b == nil {
			continue
		}
		snapshot.balls[i] = ballRenderSnapshot{x: b.x, y: b.y, angle: b.angle}
	}
	return snapshot
}

func syncRenderInterpolation() {
	previousRenderSnapshot = captureRenderSnapshot()
	renderSnapshotReady = true
	renderInterpolationAlpha = 0
}

func beginPhysicsStepForRendering() {
	previousRenderSnapshot = captureRenderSnapshot()
	renderSnapshotReady = true
}

func lerpFloat(a, b, alpha float64) float64 {
	return a + (b-a)*alpha
}

// TODO(render): If uneven browser/compositor pacing remains visible, consider
// bounded visual-only extrapolation of moving objects by no more than one physics
// step, suppressed around collisions. Keep it disabled unless interpolation alone
// proves insufficient; it must never modify the authoritative physics state.
func interpolatedRenderSnapshot(alpha float64) renderSnapshot {
	current := captureRenderSnapshot()
	if !renderSnapshotReady {
		previousRenderSnapshot = current
		renderSnapshotReady = true
		return current
	}

	alpha = clampFloat(alpha, 0, 1)
	result := current
	result.paddleX = lerpFloat(previousRenderSnapshot.paddleX, current.paddleX, alpha)
	if mouseControlActive {
		result.paddleX = current.paddleX
	}
	result.blackHoleX = lerpFloat(previousRenderSnapshot.blackHoleX, current.blackHoleX, alpha)
	result.blackHoleY = lerpFloat(previousRenderSnapshot.blackHoleY, current.blackHoleY, alpha)

	// A spawn/removal/promote changes slot identity, so draw the current positions
	// immediately for that frame rather than interpolating from a stale ball slot.
	if previousRenderSnapshot.ballCount == current.ballCount {
		for i := 0; i < current.ballCount && i < len(current.balls); i++ {
			result.balls[i].x = lerpFloat(previousRenderSnapshot.balls[i].x, current.balls[i].x, alpha)
			result.balls[i].y = lerpFloat(previousRenderSnapshot.balls[i].y, current.balls[i].y, alpha)
			result.balls[i].angle = lerpFloat(previousRenderSnapshot.balls[i].angle, current.balls[i].angle, alpha)
		}
	}
	if previousRenderSnapshot.blackHoleActive != current.blackHoleActive {
		result.blackHoleX = current.blackHoleX
		result.blackHoleY = current.blackHoleY
	}

	return result
}

func applyOverspeedDrag(b *Ball, dt float64, settings *physicsSettings) {
	if b == nil || settings == nil || dt <= 0 || settings.maxSpeed <= 0 || settings.overspeedHalfLife <= 0 {
		return
	}

	speed := math.Hypot(b.vx, b.vy)
	if speed <= settings.maxSpeed || speed <= 0 {
		return
	}

	excess := speed - settings.maxSpeed
	excess *= math.Exp(-math.Ln2 * dt / settings.overspeedHalfLife)
	targetSpeed := settings.maxSpeed + excess
	scale := targetSpeed / speed
	b.vx *= scale
	b.vy *= scale
}

func clampBallToPhysicsSettings(b *Ball, settings *physicsSettings) {
	// Linear overspeed is intentionally not hard-clamped. It is allowed as a
	// temporary result of spin-to-speed transfer and decays in flight.
	b.omega = clampFloat(b.omega, -settings.maxSpin, settings.maxSpin)
}

func verticalDragToHorizontalDelta(deltaY float64) float64 {
	// Finger down moves paddle right; finger up moves paddle left.
	rect := canvas.Call("getBoundingClientRect")
	height := rect.Get("height").Float()
	if height <= 0 {
		return 0
	}

	scaleY := canvasHeight / height
	return deltaY * scaleY * 2.2
}

// ---- Desktop pointer lock and iOS standalone helpers ----
func pointerLockElement() js.Value {
	element := doc.Get("pointerLockElement")
	if !element.IsUndefined() && !element.IsNull() {
		return element
	}
	element = doc.Get("mozPointerLockElement")
	if !element.IsUndefined() && !element.IsNull() {
		return element
	}
	element = doc.Get("webkitPointerLockElement")
	if !element.IsUndefined() && !element.IsNull() {
		return element
	}
	return js.Null()
}

func pointerLockActive() bool {
	element := pointerLockElement()
	return !element.IsNull() && element.Equal(canvas)
}

func pointerLockMethod(target js.Value, names ...string) string {
	if target.IsUndefined() || target.IsNull() {
		return ""
	}
	for _, name := range names {
		fn := target.Get(name)
		if !fn.IsUndefined() && !fn.IsNull() && fn.Type() == js.TypeFunction {
			return name
		}
	}
	return ""
}

func pointerLockSupported() bool {
	return pointerLockMethod(canvas, "requestPointerLock", "mozRequestPointerLock", "webkitRequestPointerLock") != ""
}

func requestMousePointerLock() bool {
	method := pointerLockMethod(canvas, "requestPointerLock", "mozRequestPointerLock", "webkitRequestPointerLock")
	if method == "" {
		return false
	}

	succeeded := true
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				succeeded = false
				log("Pointer lock request failed: " + fmt.Sprint(recovered))
				showStatus("Mouse capture unavailable", 2.0)
			}
		}()
		promise := canvas.Call(method)
		if !promise.IsUndefined() && !promise.IsNull() && promise.Type() == js.TypeObject {
			catchMethod := promise.Get("catch")
			if catchMethod.Type() == js.TypeFunction {
				failure := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					reason := "request rejected"
					if len(args) > 0 {
						reason = fmt.Sprint(args[0])
					}
					log("Pointer lock rejected: " + reason)
					showStatus("Mouse capture unavailable", 2.0)
					return nil
				})
				browserUICallbacks = append(browserUICallbacks, failure)
				promise.Call("catch", failure)
			}
		}
	}()
	return succeeded
}

func releaseMousePointerLock() bool {
	method := pointerLockMethod(doc, "exitPointerLock", "mozExitPointerLock", "webkitExitPointerLock")
	if method == "" {
		return false
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			log("Pointer lock release failed: " + fmt.Sprint(recovered))
		}
	}()
	doc.Call(method)
	return true
}

func movePaddleByLockedMouse(e js.Value) {
	movement := e.Get("movementX")
	if movement.Type() != js.TypeNumber {
		movement = e.Get("mozMovementX")
	}
	if movement.Type() != js.TypeNumber {
		movement = e.Get("webkitMovementX")
	}
	if movement.Type() != js.TypeNumber {
		return
	}

	rect := canvas.Call("getBoundingClientRect")
	displayWidth := rect.Get("width").Float()
	if displayWidth <= 0 {
		return
	}

	deltaX := movement.Float() * canvasWidth / displayWidth * defaultMousePointerLockSensitivity
	maxX := math.Max(0, canvasWidth-paddle.w)
	mousePaddleTargetX = clampFloat(mousePaddleTargetX+deltaX, 0, maxX)
	mouseControlActive = true
	leftPressed = false
	rightPressed = false

	if !paused && !gameOver && !waitingForStart && !levelAdvancePending {
		paddle.x = mousePaddleTargetX
	}
}

func setupPointerLock() {
	pointerLockChange = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		locked := pointerLockActive()
		if locked == mousePointerLocked {
			return nil
		}
		mousePointerLocked = locked
		if locked {
			mousePaddleTargetX = paddle.x
			mouseControlActive = true
			leftPressed = false
			rightPressed = false
			showStatus("Mouse captured - right-click to release", 2.0)
		} else {
			mouseControlActive = false
			paddle.vx = 0
			resetPaddleSpinHistory()
			showStatus("Mouse released", 1.5)
		}
		return nil
	})
	pointerLockError = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		mousePointerLocked = false
		showStatus("Mouse capture unavailable", 2.0)
		return nil
	})
	lockedMouseMove = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if !mousePointerLocked || paused || len(args) == 0 {
			return nil
		}
		movePaddleByLockedMouse(args[0])
		return nil
	})

	doc.Call("addEventListener", "pointerlockchange", pointerLockChange)
	doc.Call("addEventListener", "mozpointerlockchange", pointerLockChange)
	doc.Call("addEventListener", "webkitpointerlockchange", pointerLockChange)
	doc.Call("addEventListener", "pointerlockerror", pointerLockError)
	doc.Call("addEventListener", "mozpointerlockerror", pointerLockError)
	doc.Call("addEventListener", "webkitpointerlockerror", pointerLockError)
	doc.Call("addEventListener", "mousemove", lockedMouseMove)
}

func appleMobileBrowser() bool {
	navigator := js.Global().Get("navigator")
	if navigator.IsUndefined() || navigator.IsNull() {
		return false
	}
	userAgent := strings.ToLower(navigator.Get("userAgent").String())
	if strings.Contains(userAgent, "iphone") || strings.Contains(userAgent, "ipad") || strings.Contains(userAgent, "ipod") {
		return true
	}
	platform := navigator.Get("platform").String()
	maxTouchPoints := navigator.Get("maxTouchPoints")
	return platform == "MacIntel" && maxTouchPoints.Type() == js.TypeNumber && maxTouchPoints.Int() > 1
}

func standaloneDisplayMode() bool {
	navigator := js.Global().Get("navigator")
	if !navigator.IsUndefined() && !navigator.IsNull() {
		standalone := navigator.Get("standalone")
		if standalone.Type() == js.TypeBoolean && standalone.Bool() {
			return true
		}
	}
	window := js.Global().Get("window")
	matchMedia := window.Get("matchMedia")
	if matchMedia.Type() == js.TypeFunction {
		return window.Call("matchMedia", "(display-mode: standalone)").Get("matches").Bool()
	}
	return false
}

func ensureMeta(name, content string) {
	head := doc.Get("head")
	if head.IsUndefined() || head.IsNull() {
		return
	}
	selector := `meta[name="` + name + `"]`
	meta := head.Call("querySelector", selector)
	if meta.IsUndefined() || meta.IsNull() {
		meta = doc.Call("createElement", "meta")
		meta.Set("name", name)
		head.Call("appendChild", meta)
	}
	meta.Set("content", content)
}

func ensureStandaloneMetadata() {
	appTitle := strings.TrimSpace(doc.Get("title").String())
	if appTitle == "" {
		appTitle = "Postgravity"
	}
	ensureMeta("apple-mobile-web-app-capable", "yes")
	ensureMeta("apple-mobile-web-app-title", appTitle)
	ensureMeta("apple-mobile-web-app-status-bar-style", "black-translucent")
	ensureMeta("theme-color", "#000000")

	head := doc.Get("head")
	if head.IsUndefined() || head.IsNull() {
		return
	}
	viewport := head.Call("querySelector", `meta[name="viewport"]`)
	if viewport.IsUndefined() || viewport.IsNull() {
		viewport = doc.Call("createElement", "meta")
		viewport.Set("name", "viewport")
		viewport.Set("content", "width=device-width,initial-scale=1,maximum-scale=1,user-scalable=no,viewport-fit=cover")
		head.Call("appendChild", viewport)
	} else {
		content := viewport.Get("content").String()
		if !strings.Contains(content, "viewport-fit=cover") {
			if strings.TrimSpace(content) != "" {
				content += ","
			}
			viewport.Set("content", content+"viewport-fit=cover")
		}
	}

	manifest := head.Call("querySelector", `link[rel="manifest"]`)
	if manifest.IsUndefined() || manifest.IsNull() {
		manifest = doc.Call("createElement", "link")
		manifest.Set("rel", "manifest")
		manifest.Set("href", "manifest.webmanifest")
		head.Call("appendChild", manifest)
	}
}

func showIOSStandaloneHint() {
	if standaloneDisplayMode() {
		showStatus("Already running from Home Screen", 2.0)
		return
	}

	existing := doc.Call("getElementById", "iosStandaloneHint")
	if !existing.IsUndefined() && !existing.IsNull() {
		existing.Get("style").Set("display", "flex")
		return
	}

	overlay := doc.Call("createElement", "div")
	overlay.Set("id", "iosStandaloneHint")
	overlay.Set("innerHTML", `<div style="box-sizing:border-box;width:min(560px,92vw);padding:22px;border:1px solid rgba(255,255,255,.25);border-radius:12px;background:#16213e;color:#fff;font:700 15px/1.45 GameFont,monospace;text-align:center;box-shadow:0 16px 50px rgba(0,0,0,.55)"><div style="font-size:22px;margin-bottom:10px">Full-screen on iPhone</div><div style="color:rgba(255,255,255,.78);margin-bottom:16px">In Safari, tap <b>Share</b>, choose <b>Add to Home Screen</b>, then launch the game from its icon.</div><button id="iosStandaloneHintClose" style="padding:10px 18px;border:1px solid rgba(255,255,255,.3);border-radius:8px;background:rgba(255,255,255,.12);color:#fff;font:700 15px GameFont,monospace">Close</button></div>`)
	style := overlay.Get("style")
	style.Set("position", "fixed")
	style.Set("inset", "0")
	style.Set("zIndex", "100001")
	style.Set("display", "flex")
	style.Set("alignItems", "center")
	style.Set("justifyContent", "center")
	style.Set("padding", "max(18px, env(safe-area-inset-top)) max(18px, env(safe-area-inset-right)) max(18px, env(safe-area-inset-bottom)) max(18px, env(safe-area-inset-left))")
	style.Set("background", "rgba(8,10,24,.94)")
	doc.Get("body").Call("appendChild", overlay)

	closeButton := doc.Call("getElementById", "iosStandaloneHintClose")
	closeCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			args[0].Call("preventDefault")
			args[0].Call("stopPropagation")
		}
		overlay.Get("style").Set("display", "none")
		return nil
	})
	browserUICallbacks = append(browserUICallbacks, closeCallback)
	closeButton.Call("addEventListener", "click", closeCallback)
}

// ---- Input ----
func pointerPaddleX(e js.Value) (float64, bool) {
	rect := canvas.Call("getBoundingClientRect")
	displayWidth := rect.Get("width").Float()
	if displayWidth <= 0 {
		return 0, false
	}

	scaleX := canvasWidth / displayWidth
	pointerX := (e.Get("clientX").Float() - rect.Get("left").Float()) * scaleX
	x := pointerX - paddle.w/2
	x = clampFloat(x, 0, math.Max(0, canvasWidth-paddle.w))
	return x, true
}

func setMousePaddleTarget(e js.Value) {
	x, ok := pointerPaddleX(e)
	if !ok {
		return
	}
	mousePaddleTargetX = x
	mouseControlActive = true

	// Position immediately for mouse responsiveness. Do not update
	// paddlePreviousX here: the fixed-step update needs the displacement to
	// calculate a physically useful paddle velocity for spin transfer.
	if !paused && !gameOver && !waitingForStart && !levelAdvancePending {
		paddle.x = x
	}
}

// Touch-follow mode intentionally remains direct: its physics update branch does
// not integrate paddle.vx again, unlike the old desktop mouse path.
func movePaddleToPointer(e js.Value) {
	x, ok := pointerPaddleX(e)
	if !ok {
		return
	}

	paddle.vx = (x - paddle.x) * 12
	paddle.x = x
}

func bindMobileButton(id string, handler func()) {
	button := doc.Call("getElementById", id)
	if button.IsUndefined() || button.IsNull() {
		return
	}

	fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			args[0].Call("preventDefault")
		}
		unlockAudioFromGesture()
		handler()
		return nil
	})

	js.Global().Set("__breakout_"+id, fn)
	button.Call("addEventListener", "pointerdown", fn)
}

func setupInput() {
	setupPointerLock()

	keyDown = js.FuncOf(func(this js.Value, args []js.Value) (ret interface{}) {
		defer func() {
			if r := recover(); r != nil {
				js.Global().Get("console").Call("error", "Panic in keydown handler:", fmt.Sprint(r))
				ret = nil
			}
		}()

		if len(args) == 0 {
			return nil
		}
		e := args[0]
		key := e.Get("key").String()
		code := e.Get("code").String()
		targetTag := ""
		target := e.Get("target")
		if !target.IsUndefined() && !target.IsNull() {
			tagName := target.Get("tagName")
			if !tagName.IsUndefined() && !tagName.IsNull() {
				targetTag = strings.ToLower(tagName.String())
			}
		}
		editorControlFocused := physicsEditorVisible &&
			(targetTag == "input" || targetTag == "select" || targetTag == "button")
		if !editorControlFocused {
			e.Call("preventDefault")
			if handleReplayShortcut(key, code, e.Get("shiftKey").Bool(), e.Get("repeat").Bool()) {
				return nil
			}
		}

		// Unlock/resume audio on the first keyboard gesture, including the key
		// that leaves the waiting screen. Embedded sample decoding is already
		// asynchronous, so this path never waits for a sample.
		unlockAudioFromGesture()

		// E opens the live physics/debris tuner. It preserves the current pause state;
		// the panel itself can run/freeze the simulation.
		if (key == "e" || key == "E") && !e.Get("repeat").Bool() {
			togglePhysicsEditor()
			return nil
		}
		if (key == "o" || key == "O") && !e.Get("repeat").Bool() && targetTag != "select" && targetTag != "button" {
			toggleAutoPaddle()
			return nil
		}
		if physicsEditorVisible {
			return nil
		}

		if !replayModeActive && !e.Get("repeat").Bool() && gameSpeedShortcut(key) {
			return nil
		}

		if (key == "y" || key == "Y") && !e.Get("repeat").Bool() {
			flipBricksVertically()
			return nil
		}

		// Temporary development unlock. It is deliberately not stored.
		if (key == "u" || key == "U") && !e.Get("repeat").Bool() {
			devAllLevelsUnlocked = !devAllLevelsUnlocked
			if devAllLevelsUnlocked {
				showStatus("Developer level unlock ON", 2.0)
			} else {
				permanentLimit := permanentUnlockedLimit()
				if currentLevelIndex > permanentLimit {
					jumpToLevel(permanentLimit)
				}
				showStatus("Developer level unlock OFF", 2.0)
			}
			return nil
		}

		// P cycles physics diagnostics -> level/config diagnostics -> level records -> off.
		// I is retained as an alias. F independently toggles the compact FPS readout;
		// Shift+F resets its visible-page LOWEST measurement without hiding it.
		if (key == "p" || key == "P" || key == "i" || key == "I") && !e.Get("repeat").Bool() {
			cycleDiagnosticsOverlay()
			return nil
		}
		if (key == "f" || key == "F") && !e.Get("repeat").Bool() {
			if e.Get("shiftKey").Bool() {
				resetLowestRenderFPS()
			} else {
				fpsMiniOverlayVisible = !fpsMiniOverlayVisible
			}
			return nil
		}

		// R toggles the persistent debris master gate. It is deliberately handled
		// before game-state checks so it works on READY, pause, win, and game-over.
		if (key == "r" || key == "R") && !e.Get("repeat").Bool() {
			toggleDebrisMaster()
			return nil
		}

		// Page Up / Page Down navigate levels in every game state, including the
		// win and game-over screens. Check both key and code for browser/keyboard
		// compatibility.
		pageDownPressed := key == "PageDown" || code == "PageDown"
		pageUpPressed := key == "PageUp" || code == "PageUp"
		if pageDownPressed && !e.Get("repeat").Bool() {
			limit := activeUnlockedLimit()
			if currentLevelIndex < limit {
				jumpToLevel(currentLevelIndex + 1)
			} else if limit == len(levels)-1 {
				jumpToLevel(0)
			} else {
				showStatus("Finish this level to unlock the next", 2.0)
			}
			return nil
		}
		if pageUpPressed && !e.Get("repeat").Bool() {
			limit := activeUnlockedLimit()
			if currentLevelIndex > 0 {
				previous := currentLevelIndex - 1
				if autoPaddleEnabled && previous == 13 {
					previous--
				}
				jumpToLevel(previous)
			} else if limit > 0 {
				jumpToLevel(limit)
			}
			return nil
		}

		// Start a new game after winning.
		if gameOver && win {
			if (key == " " || key == "Enter") && !e.Get("repeat").Bool() {
				jumpToLevel(0)
			}
			return nil
		}

		// Retry the same level after losing all lives. This must come
		// before pause handling so Space retries instead of toggling pause.
		if gameOver && !win {
			if (key == " " || key == "Enter") && !e.Get("repeat").Bool() {
				retryCurrentLevel()
			}
			return nil
		}

		if waitingForStart {
			if (key == " " || key == "Enter" ||
				key == "ArrowLeft" || key == "ArrowRight" ||
				key == "a" || key == "A" || key == "d" || key == "D" ||
				code == "AltLeft" || code == "AltRight") &&
				!e.Get("repeat").Bool() {
				waitingForStart = false
				paused = false
				leftPressed = false
				rightPressed = false
				paddle.vx = 0
				showStatus("Go!", 1.0)
			}
			return nil
		}

		if levelAdvancePending {
			if (key == " " || key == "Enter") && !e.Get("repeat").Bool() {
				advanceFromLevelComplete()
			}
			return nil
		}

		// Manual last-resort rescue. It uses the same safe-position search as the
		// automatic fourth-failure fallback, but ignores the performance gate
		// because this is an explicit player action.
		if (key == "t" || key == "T") && !e.Get("repeat").Bool() {
			if !paused {
				markLevelRunAssisted()
				teleportBallToSafeArea(&ball, true)
			}
			return nil
		}

		// Sound toggle.
		if (key == "s" || key == "S") && !e.Get("repeat").Bool() {
			toggleSound()
			return nil
		}

		// Toggle magnetism relative to the level default.
		if (key == "m" || key == "M") && !e.Get("repeat").Bool() {
			markLevelRunAssisted()
			magnetCheat = !magnetCheat
			if magnetIsActive() {
				showStatus("Magnets on", 2.0)
			} else {
				showStatus("Magnets off", 2.0)
			}
			return nil
		}

		// Toggle Zapper relative to the level default.
		if (key == "z" || key == "Z") && !e.Get("repeat").Bool() {
			markLevelRunAssisted()
			zapperCheat = !zapperCheat
			resetAllBallZapperStates()
			if zapperIsActive() {
				showStatus("Zapper on", 2.0)
			} else {
				showStatus("Zapper off", 2.0)
			}
			return nil
		}

		if key == "2" && !e.Get("repeat").Bool() {
			markLevelRunAssisted()
			if spawnAdditionalBall() {
				showStatus("Dev multi ball "+strconv.Itoa(activeBallCount())+"/"+strconv.Itoa(maxConfiguredBalls()), 1.5)
			} else {
				boostAllActiveBalls(defaultMultiBallSpeedBoost)
				showStatus("Dev speed boost", 1.5)
			}
			return nil
		}

		// Pause/resume. Ignore key-repeat so holding the key toggles only once.
		if (key == " " || key == "Escape") && !e.Get("repeat").Bool() {
			paused = !paused
			leftPressed = false
			rightPressed = false
			paddle.vx = 0
			return nil
		}

		// Paddle controls: arrows, A/D, and separate left/right Alt keys.
		if key == "ArrowLeft" || key == "a" || key == "A" || code == "AltLeft" {
			mouseControlActive = false
			leftPressed = true
		} else if key == "ArrowRight" || key == "d" || key == "D" || code == "AltRight" {
			mouseControlActive = false
			rightPressed = true
		}
		return nil
	})
	js.Global().Get("document").Call("addEventListener", "keydown", keyDown)

	keyUp = js.FuncOf(func(this js.Value, args []js.Value) (ret interface{}) {
		defer func() {
			if r := recover(); r != nil {
				js.Global().Get("console").Call("error", "Panic in keyup handler:", fmt.Sprint(r))
				ret = nil
			}
		}()

		if len(args) == 0 {
			return nil
		}
		e := args[0]
		if physicsEditorVisible {
			return nil
		}
		e.Call("preventDefault")
		key := e.Get("key").String()
		code := e.Get("code").String()
		if key == "ArrowLeft" || key == "a" || key == "A" || code == "AltLeft" {
			leftPressed = false
		} else if key == "ArrowRight" || key == "d" || key == "D" || code == "AltRight" {
			rightPressed = false
		}
		return nil
	})
	js.Global().Get("document").Call("addEventListener", "keyup", keyUp)

	orientationChange := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		recalibratePhoneTilt()
		return nil
	})
	js.Global().Set("__breakoutOrientationChange", orientationChange)
	js.Global().Call("addEventListener", "orientationchange", orientationChange)

	// Pointer controls work with mouse, touch, and pen.
	pointerDown = js.FuncOf(func(this js.Value, args []js.Value) (ret interface{}) {
		defer func() {
			if r := recover(); r != nil {
				js.Global().Get("console").Call("error", "Panic in pointerdown handler:", fmt.Sprint(r))
				ret = nil
			}
		}()

		if len(args) == 0 {
			return nil
		}

		e := args[0]
		e.Call("preventDefault")
		if physicsEditorVisible || replayModeActive {
			return nil
		}

		pointerType := e.Get("pointerType").String()
		if pointerType == "mouse" {
			button := e.Get("button")
			if button.Type() == js.TypeNumber && button.Int() == 2 {
				leftPressed = false
				rightPressed = false
				if pointerLockActive() {
					releaseMousePointerLock()
				} else {
					setMousePaddleTarget(e)
					requestMousePointerLock()
				}
				return nil
			}
		}

		// Clicking or tapping the visible debug panel copies its complete text.
		// Consume the event so it does not also launch, unpause, or move the paddle.
		if report, inside := overlayReportAtPointer(e); inside {
			if copyTextToClipboard(report) {
				showStatus("Debug info copied", 1.5)
			} else {
				showStatus("Clipboard unavailable", 1.5)
			}
			return nil
		}

		unlockAudioFromGesture()

		if pointerType == "mouse" {
			button := e.Get("button")
			if button.Type() == js.TypeNumber && button.Int() != 0 {
				return nil
			}
			leftPressed = false
			rightPressed = false
		}

		if gameOver && win {
			jumpToLevel(0)
			return nil
		}

		if levelAdvancePending {
			advanceFromLevelComplete()
			return nil
		}

		if waitingForStart {
			waitingForStart = false
			leftPressed = false
			rightPressed = false
			paddle.vx = 0
			showStatus("Go!", 1.0)
			return nil
		}

		if paused {
			paused = false
			leftPressed = false
			rightPressed = false
			touchControlActive = false
			paddle.vx = 0
			showStatus("Go!", 1.0)
			return nil
		}

		if levelAdvancePending {
			return nil
		}

		if pointerType == "touch" || pointerType == "pen" {
			pointerID := e.Get("pointerId").Int()

			if gameOver && !win {
				retryCurrentLevel()
			}

			if mobileControlsEnabled {
				switch mobileControlMode {
				case "vertical":
					if !touchControlActive {
						touchControlActive = true
						touchPointerID = pointerID
						touchLastY = e.Get("clientY").Float()
						paddle.vx = 0
						canvas.Call("setPointerCapture", e.Get("pointerId"))
					}
				case "follow":
					if !touchControlActive {
						touchControlActive = true
						touchPointerID = pointerID
						canvas.Call("setPointerCapture", e.Get("pointerId"))
					}
					movePaddleToPointer(e)
				case "two-thumb":
					rect := canvas.Call("getBoundingClientRect")
					midX := rect.Get("left").Float() + rect.Get("width").Float()/2
					if e.Get("clientX").Float() < midX {
						if mobileLeftPointerID < 0 {
							mobileLeftPointerID = pointerID
							mobileLeftHeld = true
						}
					} else if mobileRightPointerID < 0 {
						mobileRightPointerID = pointerID
						mobileRightHeld = true
					}
					touchControlActive = mobileLeftHeld || mobileRightHeld
					canvas.Call("setPointerCapture", e.Get("pointerId"))
				case "tilt":
					if phoneTiltListenerSet {
						recalibratePhoneTilt()
					}
				}
			}
			return nil
		}

		if pointerType == "mouse" && gameOver && !win {
			retryCurrentLevel()
		}

		return nil
	})
	canvas.Call("addEventListener", "pointerdown", pointerDown)

	// Suppress the browser menu and use right-click as the deliberate Pointer Lock
	// toggle gesture. Escape remains the browser-provided emergency release.
	contextMenu := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			args[0].Call("preventDefault")
		}
		return nil
	})
	browserUICallbacks = append(browserUICallbacks, contextMenu)
	canvas.Call("addEventListener", "contextmenu", contextMenu)

	pointerMove = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if paused || len(args) == 0 {
			return nil
		}

		e := args[0]
		pointerType := e.Get("pointerType").String()
		pointerID := e.Get("pointerId").Int()

		if (pointerType == "touch" || pointerType == "pen") &&
			mobileControlsEnabled {
			switch mobileControlMode {
			case "vertical":
				if touchControlActive && pointerID == touchPointerID {
					y := e.Get("clientY").Float()
					deltaY := y - touchLastY
					touchLastY = y

					deltaX := verticalDragToHorizontalDelta(deltaY)
					paddle.x += deltaX
					paddle.vx = deltaX * 60

					if paddle.x < 0 {
						paddle.x = 0
					}
					if paddle.x+paddle.w > canvasWidth {
						paddle.x = canvasWidth - paddle.w
					}
				}
			case "follow":
				if touchControlActive && pointerID == touchPointerID {
					movePaddleToPointer(e)
				}
			}
			e.Call("preventDefault")
			return nil
		}

		if pointerType == "mouse" {
			e.Call("preventDefault")
			if !mousePointerLocked {
				leftPressed = false
				rightPressed = false
				setMousePaddleTarget(e)
			}
		}

		return nil
	})
	canvas.Call("addEventListener", "pointermove", pointerMove)

	pointerUp = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			e := args[0]
			pointerID := e.Get("pointerId").Int()
			pointerType := e.Get("pointerType").String()
			if (pointerType == "touch" || pointerType == "pen") &&
				mobileControlsEnabled && mobileControlMode == "tilt" {
				requestPhoneTiltPermission()
			}

			if mobileControlMode == "two-thumb" {
				if pointerID == mobileLeftPointerID {
					mobileLeftPointerID = -1
					mobileLeftHeld = false
				}
				if pointerID == mobileRightPointerID {
					mobileRightPointerID = -1
					mobileRightHeld = false
				}
				touchControlActive = mobileLeftHeld || mobileRightHeld
				if !touchControlActive {
					paddle.vx = 0
				}
			} else if touchControlActive && pointerID == touchPointerID {
				touchControlActive = false
				touchPointerID = -1
				paddle.vx = 0
			}

			pointerValue := e.Get("pointerId")
			if !pointerValue.IsUndefined() &&
				canvas.Call("hasPointerCapture", pointerValue).Bool() {
				canvas.Call("releasePointerCapture", pointerValue)
			}
		}
		return nil
	})
	canvas.Call("addEventListener", "pointerup", pointerUp)

	pointerCancel = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			pointerID := args[0].Get("pointerId").Int()
			if mobileControlMode == "two-thumb" {
				if pointerID == mobileLeftPointerID {
					mobileLeftPointerID = -1
					mobileLeftHeld = false
				}
				if pointerID == mobileRightPointerID {
					mobileRightPointerID = -1
					mobileRightHeld = false
				}
				touchControlActive = mobileLeftHeld || mobileRightHeld
			} else if touchControlActive && pointerID == touchPointerID {
				touchControlActive = false
				touchPointerID = -1
			}
			if !touchControlActive {
				paddle.vx = 0
			}
		}
		return nil
	})
	canvas.Call("addEventListener", "pointercancel", pointerCancel)

	mouseLeave = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if !mousePointerLocked {
			mouseControlActive = false
		}
		return nil
	})
	canvas.Call("addEventListener", "mouseleave", mouseLeave)

	fullscreenToggle = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handleGamepadFullscreenPress()
		return nil
	})
	js.Global().Set("breakoutToggleFullscreen", fullscreenToggle)

	bindMobileButton("fullscreenButton", func() {
		handleGamepadFullscreenPress()
	})

	bindMobileButton("pauseButton", func() {
		if physicsEditorVisible || gameOver {
			return
		}
		paused = !paused
		leftPressed = false
		rightPressed = false
		paddle.vx = 0
	})

	bindMobileButton("soundButton", func() {
		toggleSound()
	})

	bindMobileButton("previousLevelButton", func() {
		limit := activeUnlockedLimit()
		if currentLevelIndex > 0 {
			jumpToLevel(currentLevelIndex - 1)
		} else if limit > 0 {
			jumpToLevel(limit)
		}
	})

	bindMobileButton("nextLevelButton", func() {
		limit := activeUnlockedLimit()
		if currentLevelIndex < limit {
			jumpToLevel(currentLevelIndex + 1)
		} else if limit == len(levels)-1 {
			jumpToLevel(0)
		} else {
			showStatus("Finish this level to unlock the next", 2.0)
		}
	})

	bindMobileButton("magnetsButton", func() {
		markLevelRunAssisted()
		magnetCheat = !magnetCheat
		if magnetIsActive() {
			showStatus("Magnets on", 2.0)
		} else {
			showStatus("Magnets off", 2.0)
		}
	})

	bindMobileButton("zapperButton", func() {
		markLevelRunAssisted()
		zapperCheat = !zapperCheat
		resetAllBallZapperStates()
		if zapperIsActive() {
			showStatus("Zapper on", 2.0)
		} else {
			showStatus("Zapper off", 2.0)
		}
	})
}

// ---- main ----
func main() {
	nativeMain()
}
