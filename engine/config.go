package main

// config.go contains compile-time defaults and tuning values. Runtime state and
// game behavior stay in main.go. Build the package (not main.go alone) so this
// file is included:
//
//   go build -o ../bin/physical-engine .

const (
	buildID = "20260820-multiball-01"

	// Audio mixer.
	audioMixerMaster = 1.00
	audioMixerBricks = 0.20 // was 0.30
	audioMixerMagic  = 0.20
	audioMixerSynths = 0.80

	audioRoomTransitionSeconds = 0.16

	brickHitSampleDirectory = "sounds/brickHits/runtime"
	brickHitPlaybackRateMin = 0.94
	brickHitPlaybackRateMax = 1.07
	brickHitFilterMinHz     = 2400.0
	brickHitFilterMaxHz     = 12900.0
	brickHitGainMin         = 0.16
	brickHitGainMax         = 0.40

	magicFeatureSampleDirectory = "sounds/magicBrickHits/runtime"
	magicFeaturePlaybackRateMin = 0.96
	magicFeaturePlaybackRateMax = 1.04
	magicFeatureFilterMinHz     = 3000.0
	magicFeatureFilterMaxHz     = 16000.0
	magicFeatureHardMaxHz       = 500.0
	magicFeatureGainMin         = 0.15
	magicFeatureGainMax         = 0.32
	magicFeatureMaxActiveVoices = 4
	magicFeatureRetriggerFade   = 0.018

	brickHitMaxActiveVoices    = 8
	brickHitMaxStartsPerFrame  = 3
	brickSoundPanLimit         = 0.75
	embeddedAudioDecodeYieldMS = 12

	minimumCollisionSoundSpeed = 35.0
	collisionSoundCooldown     = 0.045

	zapperBrickPitchScale = 1.32
	zapperBrickStrength   = 0.72

	// Fixed-step engine scheduling. These intentionally remain global and are not
	// level-overridable because levels should not change simulation stability.
	physicsStepHz             = 240.0
	physicsStepSeconds        = 1.0 / physicsStepHz
	physicsMaxCatchUpSteps    = 32
	physicsMaxFrameDelta      = physicsStepSeconds * physicsMaxCatchUpSteps
	physicsWarningHoldSeconds = 3.0

	// Render-FPS sampling. The lowest value ignores the first two visible-page
	// samples so startup does not become the permanent session minimum.
	renderFPSSampleWindowSeconds = 0.5
	renderFPSLowestWarmupSamples = 2

	// Leave results visible briefly before the unattended demo continues.
	autoDemoTransitionSeconds = 2.0

	// Default geometry and gameplay.
	defaultCanvasWidth    = 1800.0
	defaultCanvasHeight   = 900.0
	defaultPaddleWidth    = 220.0
	defaultPaddleHeight   = 30.0
	defaultBallRadius     = 8.0
	defaultBrickRows      = 10
	defaultBrickCols      = 18
	defaultBrickWidth     = 60.0
	defaultBrickHeight    = 20.0
	defaultBrickPadding   = 20.0
	defaultBrickOffsetTop = -1.0

	defaultDigitalPaddleMaxSpeed     = 2800.0
	defaultDigitalPaddleAcceleration = 9000.0
	defaultDigitalPaddleBraking      = 40000.0

	// Raw Gamepad API mapping for the Vivanco 0663-9807 USB Game Device.
	// Axis 0 is horizontal; B0 starts/toggles pause; B1 toggles fullscreen.
	defaultGamepadHorizontalAxis   = 0
	defaultGamepadFireButton       = 0
	defaultGamepadFullscreenButton = 1
	defaultGamepadDeadZone         = 0.12

	// Automatic inspection paddle. O toggles it at runtime. The controller predicts
	// the next paddle-line crossing and approaches it with bounded acceleration.
	defaultAutoPaddleMaxSpeed             = 2400.0
	defaultAutoPaddleAcceleration         = 14000.0
	defaultAutoPaddleBraking              = 22000.0
	defaultAutoPaddlePredictionMaxSeconds = 6.0
	defaultAutoPaddleDeadZone             = 3.0
	// Maximum randomized impact offset from paddle center. 0 always centers;
	// 1 may aim at the extreme left/right edge. A new offset is chosen after
	// every paddle contact and whenever the controller switches target balls.
	defaultAutoPaddleHitVariation = 0.65

	defaultEnableSounds      = true
	defaultAudioRoom         = "none"
	defaultAudioRoomDry      = -1.0
	defaultMobileControlMode = "vertical"

	// Dynamic brick debris. Fragments use bounded lightweight rigid-body physics
	// at the normal fixed-step rate. The active-piece cap prevents mass-destruction
	// effects from turning one frame into thousands of collision bodies.
	defaultDebrisEnabled = true
	// mixed preserves the existing varied debris. triangles and circles force every
	// newly created shard into one cheap shape family for browser performance tests.
	defaultDebrisShapeMode = "mixed" //mixed circles triangles
	defaultDebrisPiecesMin = 6
	defaultDebrisPiecesMax = 18
	defaultDebrisLifetime  = 9.0
	// Each new piece receives lifetime x (1 +/- variation/100). At 16.7% and
	// a 9-second base lifetime, the approximate range is 7.5-10.5 seconds.
	defaultDebrisLifetimeVariationPercent = 16.7
	defaultDebrisFadeDuration             = 3.0
	defaultDebrisStartOpacity             = 0.35
	defaultDebrisStartOpacityVariation    = 0.25
	defaultDebrisFlashDuration            = 0.10
	defaultDebrisFlashOpacity             = 1.00
	defaultDebrisImpactSpeedFactor        = 0.35
	defaultDebrisBallPieceChance          = 0.18
	defaultDebrisTrianglePieceChance      = 0.14
	defaultDebrisStarPieceChance          = 0.06
	defaultDebrisStarPointsMin            = 4
	defaultDebrisStarPointsMax            = 7
	defaultDebrisGlassPieceChance         = 0.20
	defaultDebrisGlassCornersMin          = 7
	defaultDebrisGlassCornersMax          = 14
	defaultDebrisSliverPieceChance        = 0.015
	defaultDebrisMaxChunkAspectRatio      = 1.45
	defaultDebrisSizeScale                = 1.35
	defaultDebrisBrickCollisionDelay      = 0.25 //0.30
	defaultDebrisGravityScale             = 0.7  //1.0
	defaultDebrisAirDrag                  = 0.15
	defaultDebrisRestitution              = 0.48
	defaultDebrisFriction                 = 1.00
	defaultDebrisExplosionSpeedMin        = 0.0
	defaultDebrisExplosionSpeedMax        = 520.0
	defaultDebrisAngularSpeedMin          = 2.0
	defaultDebrisAngularSpeedMax          = 6.0
	defaultDebrisAngularDrag              = 1.10
	defaultDebrisAngularStopSpeed         = 0.10
	defaultDebrisBallInfluence            = 0.00 //0.15
	defaultDebrisFieldScale               = 0.80
	defaultDebrisMagnetScale              = 0.65
	defaultDebrisMaxSpeed                 = 1100.0
	// Shards at or above this linear speed are drawn over living bricks. Slower
	// shards remain behind them. Zero puts every shard in the front layer.
	defaultDebrisFrontLayerSpeed = 600.0
	defaultDebrisMaxActivePieces = 150
	defaultDebrisOffscreenMargin = 120.0

	// Rendering optimizations for older machines. Path2D caches each shard outline,
	// avoiding repeated Go/WASM -> JavaScript path commands on every frame.
	defaultDebrisUsePath2DCache = true
	// Adaptive rendering never removes debris physics. When the observed render rate
	// drops relative to the no-debris baseline, it draws a stable subset of older,
	// slow shards while always retaining fresh and fast-moving fragments.
	defaultDebrisAdaptiveRendering       = true // sucks
	defaultDebrisAdaptiveTargetFPS       = 60.0
	defaultDebrisAdaptiveFreshSeconds    = 0.75
	defaultDebrisAdaptiveAlwaysDrawSpeed = 500.0
	defaultDebrisAdaptiveMaxStride       = 3

	// Last-resort ball rescue. A normal TILT! measures total movement; Orbital
	// tilt measures movement along the orbit's minor axis. Only healthy fixed-step
	// performance samples may count as failures or trigger the automatic teleport.
	defaultBallRescueEnabled             = true
	defaultBallRescueFailureLimit        = 4
	defaultBallRescueMinProgress         = 80.0
	defaultBallRescueCheckDuration       = 1.0
	defaultBallRescueUpperScreenFraction = 1.0 / 3.0
	defaultBallRescueClearance           = 12.0
	defaultBallRescueLaunchSpeed         = 520.0
	defaultBallRescueMinRealtimePercent  = 90.0
	defaultBallRescueMaxComputeLoad      = 85.0

	// Keep simulating the ball this many pixels below the visible floor so
	// reverse gravity or the black hole may still pull it back into play.
	defaultBallBelowFloorGracePixels = 120.0

	// Mouse position is direct. This only caps the measured surface velocity used
	// for collision/spin calculations after a large cursor jump. It is part of
	// physicsSettings so levels and the in-game physics editor may override it.
	defaultPhysicsMousePaddleSpinVelocityLimit = 3000.0 //6000.0
	// Relative desktop mouse distance multiplier while Pointer Lock is active.
	defaultMousePointerLockSensitivity = 1.0

	defaultPaddleRadius      = 12.0
	defaultBrickRadius       = 6.0
	defaultUnbreakableChance = 0.15
	defaultMagicChance       = 0.3
	defaultPowerUpDuration   = 10.0

	// Multi-ball. The primary ball (ball 1) is the only ball allowed to activate
	// magic-brick prizes. Additional balls alternate the two configured ball colors.
	// Once this cap is reached, another Dual Balls prize boosts every active ball.
	defaultMaxBalls                  = 5
	defaultMultiBallSpeedBoost       = 1.10
	defaultMultiBallSpawnSpreadDeg   = 12.0
	defaultBallBallCollisions        = true
	defaultBallBallRestitution       = 0.96
	defaultBallBallCollisionGraceSec = 0.08

	// Synthesized ball-to-ball collision tick. It is deliberately subtle so five
	// balls cannot turn a dense collision cluster into continuous audio chatter.
	defaultBallBallSoundEnabled  = true
	defaultBallBallSoundMinSpeed = 45.0
	defaultBallBallSoundCooldown = 0.060
	defaultBallBallSoundGainMin  = 0.025
	defaultBallBallSoundGainMax  = 0.110

	defaultBlackHoleStrength = 800.0
	// Horizontal path radius. 600 px is three times the previous 200 px travel.
	defaultBlackHoleRange = 600.0
	// The black hole follows a different smooth curved path on each activation,
	// centered slightly above the middle of the playfield.
	defaultBlackHolePathVerticalRange       = 120.0
	defaultBlackHolePathCenterYOffset       = -70.0
	defaultBlackHolePathHorizontalCyclesMin = 0.95
	defaultBlackHolePathHorizontalCyclesMax = 1.15
	defaultBlackHolePathVerticalCyclesMin   = 1.65
	defaultBlackHolePathVerticalCyclesMax   = 2.35
	defaultBlackHolePathWobble              = 0.18

	defaultMagnetStrength       = 600.0
	defaultMagnetRange          = 300.0
	defaultInfluencerMultiplier = 5.0
	defaultLives                = 7
	defaultZapperHitTime        = 0.1
	defaultZapperRange          = 320.0

	// Base physics defaults. Every value in this section that changes level
	// behavior is represented in physicsSettings and may be overridden by a level.
	defaultPhysicsGravity             = 300.0
	defaultPhysicsRestitution         = 0.90
	defaultPhysicsFrictionCoeff       = 0.14 // was 0.10
	defaultPhysicsPaddleBoost         = 900.0
	defaultPhysicsBrickBoost          = 100.0
	defaultPhysicsMaxSpeed            = 920.0 // was 1000.0
	defaultPhysicsMaxSpin             = 1530.0
	defaultPhysicsStuckSpeedThreshold = 85.0
	defaultPhysicsStuckDuration       = 1.2 // was 3.0
	defaultPhysicsTiltUpSpeed         = 520.0
	defaultPhysicsTiltSideMin         = 180.0
	defaultPhysicsTiltSideMax         = 340.0

	defaultPhysicsMagnusCoefficient       = 0.0030 // was 0.0015
	defaultPhysicsMagnusAccelerationScale = 0.25
	defaultPhysicsSpinDrag                = 0.04 // was 0.10
	defaultPhysicsAirDrag                 = 0.010

	defaultPhysicsWallFrictionScale        = 2.00 // was 0.35
	defaultPhysicsBrickFrictionScale       = 2.00 // was 1.00
	defaultPhysicsUnbreakableFrictionScale = 1.80 // was 0.35
	defaultPhysicsPaddleFrictionScale      = 3.20 // was 2.60
	defaultPhysicsPaddleSpinTransfer       = 2.80 // was 2.25
	defaultPhysicsCollisionSpinCoupling    = 2.00
	defaultPhysicsMinimumCollisionGrip     = 0.08
	defaultPhysicsMinimumPaddleGrip        = 0.55 // was 0.45
	defaultPhysicsCollisionSlop            = 0.05
	defaultPhysicsPaddleSpinGraceSeconds   = 0.050
	defaultPhysicsOverspeedHalfLife        = 0.35

	defaultPhysicsWallNoiseCellSize      = 20.0 // was 120.0
	defaultPhysicsWallSideTiltDegrees    = 0.15
	defaultPhysicsWallTopTiltDegrees     = 3.45 // was 0.45
	defaultPhysicsWallCornerFadeDistance = 40.0

	// Brick-corner collisions. Physics uses the same brickRadius as the visible
	// roundRect. Amount blends from the stable classic axis normal (0) to the
	// rounded-corner radial normal (1).
	defaultPhysicsCornerPhysicsEnabled = true
	defaultPhysicsCornerPhysicsAmount  = 1.0
	// false = passable-gap rule; true = every rounded brick corner may respond.
	defaultPhysicsCornerPhysicsAllBricks = false

	// Corner-hit diagnostics. Genuine corner responses are written only to the
	// browser console and numbered sequentially for the whole game session.
	enableCornerPhysicsDebug = true

	wallNoiseIDLeft  int = 1
	wallNoiseIDRight int = 2
	wallNoiseIDTop   int = 3

	defaultPhysicsBrickTiltMinDegrees = 0.2
	defaultPhysicsBrickTiltMaxDegrees = 2.0 // was 1.0
	defaultPhysicsDrawBrickTilt       = true

	defaultPhysicsOrbitMinimumSpeed         = 300.0
	defaultPhysicsOrbitMinimumHitSpeed      = 80.0
	defaultPhysicsOrbitMinorSpeedRatio      = 0.08
	defaultPhysicsOrbitMinorSpeedFloor      = 45.0
	defaultPhysicsOrbitRequiredHits         = 3
	defaultPhysicsOrbitDetectionWindow      = 4.0
	defaultPhysicsOrbitMaximumMinorProgress = 40.0
	defaultPhysicsOrbitHitCooldown          = 0.08
	defaultPhysicsOrbitEscapeSpeed          = 110.0
	defaultPhysicsOrbitEscapeDuration       = 0.90
	physicsOrbitMessageDuration             = 1.5

	statusMessageLimit = 10

	enableHighSpinMessage   = true
	highSpinThreshold       = 100.0
	highSpinMessageDuration = 3.5

	defaultPhoneTiltDeadZone  = 1.0
	defaultPhoneTiltMaxAngle  = 10.0
	defaultPhoneTiltSmoothing = 18.0

	defaultStartBallX  = 1000.0
	defaultStartBallY  = 600.0
	defaultStartBallVx = 180.0
	defaultStartBallVy = -350.0

	defaultEnableLowGravity       = true
	defaultEnablePassThrough      = true
	defaultEnableNuke             = true
	defaultEnableReverseGravity   = true
	defaultEnableDualBalls        = true
	defaultEnableBlackHole        = true
	defaultEnableMagnet           = true
	defaultEnableInfluencer       = true
	defaultEnableZapper           = false
	defaultEnableBreakUnbreakable = true
	defaultEnableBigPaddle        = true

	showBlackHole = false // false

	defaultMagicColor             = "#f1faee"
	defaultMagicStrokeColor       = "#ffd700"
	defaultUnbreakableStrokeColor = "#e76f51"
	defaultBrickStrokeColor       = "#27ae60"
)

// autoPaddleEditorSliderSpecs controls the live automatic-paddle section.
var autoPaddleEditorSliderSpecs = []autoPaddleSliderSpec{
	{group: "Aim", key: "autoPaddleHitVariation", label: "Hit-position variation", configName: "defaultAutoPaddleHitVariation", min: 0, max: 0.90, step: 0.01, precision: 2},
}

// physicsEditorSliderSpecs controls which important physics settings appear in
// the E-key tuning panel. Ranges affect only the editor UI; level files may still
// use any value accepted by the level parser.
var physicsEditorSliderSpecs = []physicsSliderSpec{
	{group: "Motion", key: "gravity", label: "Gravity", configName: "defaultPhysicsGravity", min: -1000, max: 1500, step: 10, precision: 0},
	{group: "Motion", key: "restitution", label: "Restitution", configName: "defaultPhysicsRestitution", min: 0, max: 1, step: 0.01, precision: 2},
	{group: "Motion", key: "airDrag", label: "Air drag", configName: "defaultPhysicsAirDrag", min: 0, max: 0.10, step: 0.001, precision: 3},
	{group: "Motion", key: "magnusCoefficient", label: "Magnus coefficient", configName: "defaultPhysicsMagnusCoefficient", min: 0, max: 0.010, step: 0.0001, precision: 4},
	{group: "Motion", key: "magnusAccelerationScale", label: "Magnus acceleration", configName: "defaultPhysicsMagnusAccelerationScale", min: 0, max: 1, step: 0.01, precision: 2},
	{group: "Motion", key: "spinDrag", label: "Spin drag", configName: "defaultPhysicsSpinDrag", min: 0, max: 0.30, step: 0.005, precision: 3},
	{group: "Limits and boosts", key: "maxSpeed", label: "Maximum speed", configName: "defaultPhysicsMaxSpeed", min: 200, max: 2500, step: 25, precision: 0},
	{group: "Limits and boosts", key: "maxSpin", label: "Maximum spin", configName: "defaultPhysicsMaxSpin", min: 100, max: 3000, step: 25, precision: 0},
	{group: "Limits and boosts", key: "paddleBoost", label: "Paddle boost", configName: "defaultPhysicsPaddleBoost", min: 0, max: 2000, step: 25, precision: 0},
	{group: "Limits and boosts", key: "brickBoost", label: "Brick boost", configName: "defaultPhysicsBrickBoost", min: 0, max: 800, step: 10, precision: 0},
	{group: "Paddle and spin", key: "paddleSpinTransfer", label: "Paddle spin transfer", configName: "defaultPhysicsPaddleSpinTransfer", min: 0, max: 5, step: 0.05, precision: 2},
	{group: "Paddle and spin", key: "mousePaddleSpinVelocityLimit", label: "Mouse spin velocity limit", configName: "defaultPhysicsMousePaddleSpinVelocityLimit", min: 500, max: 8000, step: 100, precision: 0},
	{group: "Paddle and spin", key: "paddleFrictionScale", label: "Paddle friction", configName: "defaultPhysicsPaddleFrictionScale", min: 0, max: 6, step: 0.05, precision: 2},
	{group: "Paddle and spin", key: "minimumPaddleGrip", label: "Minimum paddle grip", configName: "defaultPhysicsMinimumPaddleGrip", min: 0, max: 1, step: 0.01, precision: 2},
	{group: "Paddle and spin", key: "collisionSpinCoupling", label: "Collision spin coupling", configName: "defaultPhysicsCollisionSpinCoupling", min: 0, max: 4, step: 0.05, precision: 2},
	{group: "Paddle and spin", key: "paddleSpinGraceSeconds", label: "Paddle spin memory", configName: "defaultPhysicsPaddleSpinGraceSeconds", min: 0, max: 0.20, step: 0.005, precision: 3},
	{group: "Surface contact", key: "frictionCoeff", label: "Base friction", configName: "defaultPhysicsFrictionCoeff", min: 0, max: 1, step: 0.01, precision: 2},
	{group: "Surface contact", key: "wallFrictionScale", label: "Wall friction", configName: "defaultPhysicsWallFrictionScale", min: 0, max: 4, step: 0.05, precision: 2},
	{group: "Surface contact", key: "brickFrictionScale", label: "Brick friction", configName: "defaultPhysicsBrickFrictionScale", min: 0, max: 4, step: 0.05, precision: 2},
	{group: "Surface contact", key: "unbreakableFrictionScale", label: "Unbreakable friction", configName: "defaultPhysicsUnbreakableFrictionScale", min: 0, max: 4, step: 0.05, precision: 2},
	{group: "Surface contact", key: "overspeedHalfLife", label: "Overspeed half-life", configName: "defaultPhysicsOverspeedHalfLife", min: 0.05, max: 2, step: 0.05, precision: 2},
	{group: "Geometry", key: "wallTopTiltDegrees", label: "Top-wall roughness", configName: "defaultPhysicsWallTopTiltDegrees", min: 0, max: 8, step: 0.05, precision: 2},
	{group: "Geometry", key: "wallSideTiltDegrees", label: "Side-wall roughness", configName: "defaultPhysicsWallSideTiltDegrees", min: 0, max: 4, step: 0.05, precision: 2},
	{group: "Geometry", key: "cornerPhysicsAmount", label: "Corner physics amount", configName: "defaultPhysicsCornerPhysicsAmount", min: 0, max: 1, step: 0.05, precision: 2},
	{group: "Geometry", key: "brickTiltMaxDegrees", label: "Maximum brick tilt", configName: "defaultPhysicsBrickTiltMaxDegrees", min: 0, max: 5, step: 0.05, precision: 2},
}

// debrisEditorSliderSpecs controls the live debris section of the E tuner.
// Settings marked "new pieces" affect future fragments; motion/contact settings
// also affect fragments already in flight.
var debrisEditorSliderSpecs = []debrisSliderSpec{
	{group: "Generation", key: "debrisPiecesMin", label: "Pieces minimum (new)", configName: "defaultDebrisPiecesMin", min: 1, max: 32, step: 1, precision: 0, integer: true},
	{group: "Generation", key: "debrisPiecesMax", label: "Pieces maximum (new)", configName: "defaultDebrisPiecesMax", min: 1, max: 32, step: 1, precision: 0, integer: true},
	{group: "Generation", key: "debrisMaxActivePieces", label: "Active-piece cap", configName: "defaultDebrisMaxActivePieces", min: 0, max: 500, step: 1, precision: 0, integer: true},
	{group: "Generation", key: "debrisLifetime", label: "Lifetime", configName: "defaultDebrisLifetime", min: 0.1, max: 30, step: 0.1, precision: 1},
	{group: "Generation", key: "debrisLifetimeVariationPercent", label: "Lifetime variation % (new)", configName: "defaultDebrisLifetimeVariationPercent", min: 0, max: 95, step: 0.5, precision: 1},
	{group: "Generation", key: "debrisFadeDuration", label: "Fade duration", configName: "defaultDebrisFadeDuration", min: 0, max: 15, step: 0.1, precision: 1},

	{group: "Appearance", key: "debrisStartOpacity", label: "Starting opacity", configName: "defaultDebrisStartOpacity", min: 0, max: 1, step: 0.01, precision: 2},
	{group: "Appearance", key: "debrisStartOpacityVariation", label: "Opacity variation (new)", configName: "defaultDebrisStartOpacityVariation", min: 0, max: 1, step: 0.01, precision: 2},
	{group: "Appearance", key: "debrisFlashDuration", label: "Bright flash duration", configName: "defaultDebrisFlashDuration", min: 0, max: 0.75, step: 0.01, precision: 2},
	{group: "Appearance", key: "debrisFlashOpacity", label: "Bright flash opacity", configName: "defaultDebrisFlashOpacity", min: 0, max: 1, step: 0.01, precision: 2},
	{group: "Appearance", key: "debrisSizeScale", label: "Piece size (new + live)", configName: "defaultDebrisSizeScale", min: 0.25, max: 3, step: 0.05, precision: 2},
	{group: "Appearance", key: "debrisBallPieceChance", label: "Round-piece chance (new)", configName: "defaultDebrisBallPieceChance", min: 0, max: 1, step: 0.01, precision: 2},
	{group: "Appearance", key: "debrisTrianglePieceChance", label: "Triangle chance (new)", configName: "defaultDebrisTrianglePieceChance", min: 0, max: 1, step: 0.01, precision: 2},
	{group: "Appearance", key: "debrisStarPieceChance", label: "Star chance (new)", configName: "defaultDebrisStarPieceChance", min: 0, max: 1, step: 0.01, precision: 2},
	{group: "Appearance", key: "debrisStarPointsMin", label: "Star points minimum (new)", configName: "defaultDebrisStarPointsMin", min: 3, max: 8, step: 1, precision: 0, integer: true},
	{group: "Appearance", key: "debrisStarPointsMax", label: "Star points maximum (new)", configName: "defaultDebrisStarPointsMax", min: 3, max: 8, step: 1, precision: 0, integer: true},
	{group: "Appearance", key: "debrisGlassPieceChance", label: "Glass-polygon chance (new)", configName: "defaultDebrisGlassPieceChance", min: 0, max: 1, step: 0.01, precision: 2},
	{group: "Appearance", key: "debrisGlassCornersMin", label: "Glass corners minimum (new)", configName: "defaultDebrisGlassCornersMin", min: 7, max: 16, step: 1, precision: 0, integer: true},
	{group: "Appearance", key: "debrisGlassCornersMax", label: "Glass corners maximum (new)", configName: "defaultDebrisGlassCornersMax", min: 7, max: 16, step: 1, precision: 0, integer: true},
	{group: "Appearance", key: "debrisSliverPieceChance", label: "Sliver chance (new)", configName: "defaultDebrisSliverPieceChance", min: 0, max: 0.30, step: 0.005, precision: 3},
	{group: "Appearance", key: "debrisMaxChunkAspectRatio", label: "Maximum chunk aspect (new)", configName: "defaultDebrisMaxChunkAspectRatio", min: 1, max: 4, step: 0.05, precision: 2},

	{group: "Launch", key: "debrisExplosionSpeedMin", label: "Explosion speed minimum (new)", configName: "defaultDebrisExplosionSpeedMin", min: 0, max: 2000, step: 10, precision: 0},
	{group: "Launch", key: "debrisExplosionSpeedMax", label: "Explosion speed maximum (new)", configName: "defaultDebrisExplosionSpeedMax", min: 0, max: 2500, step: 10, precision: 0},
	{group: "Launch", key: "debrisImpactSpeedFactor", label: "Ball-impact speed factor (new)", configName: "defaultDebrisImpactSpeedFactor", min: 0, max: 2, step: 0.01, precision: 2},
	{group: "Launch", key: "debrisAngularSpeedMin", label: "Angular speed minimum (new)", configName: "defaultDebrisAngularSpeedMin", min: 0, max: 50, step: 0.25, precision: 2},
	{group: "Launch", key: "debrisAngularSpeedMax", label: "Angular speed maximum (new)", configName: "defaultDebrisAngularSpeedMax", min: 0, max: 50, step: 0.25, precision: 2},

	{group: "Motion and settling", key: "debrisGravityScale", label: "Gravity scale", configName: "defaultDebrisGravityScale", min: 0, max: 3, step: 0.05, precision: 2},
	{group: "Motion and settling", key: "debrisAirDrag", label: "Air drag", configName: "defaultDebrisAirDrag", min: 0, max: 5, step: 0.05, precision: 2},
	{group: "Motion and settling", key: "debrisAngularDrag", label: "Angular drag", configName: "defaultDebrisAngularDrag", min: 0, max: 8, step: 0.05, precision: 2},
	{group: "Motion and settling", key: "debrisAngularStopSpeed", label: "Angular stop threshold", configName: "defaultDebrisAngularStopSpeed", min: 0, max: 5, step: 0.05, precision: 2},
	{group: "Motion and settling", key: "debrisMaxSpeed", label: "Maximum shard speed", configName: "defaultDebrisMaxSpeed", min: 0, max: 2500, step: 25, precision: 0},
	{group: "Motion and settling", key: "debrisFrontLayerSpeed", label: "Front-layer speed threshold", configName: "defaultDebrisFrontLayerSpeed", min: 0, max: 2500, step: 25, precision: 0},
	{group: "Motion and settling", key: "debrisOffscreenMargin", label: "Offscreen cleanup margin", configName: "defaultDebrisOffscreenMargin", min: 0, max: 500, step: 5, precision: 0},

	{group: "Contact and fields", key: "debrisBrickCollisionDelay", label: "Brick collision delay", configName: "defaultDebrisBrickCollisionDelay", min: 0, max: 2, step: 0.01, precision: 2},
	{group: "Contact and fields", key: "debrisRestitution", label: "Restitution", configName: "defaultDebrisRestitution", min: 0, max: 1.5, step: 0.01, precision: 2},
	{group: "Contact and fields", key: "debrisFriction", label: "Friction", configName: "defaultDebrisFriction", min: 0, max: 2, step: 0.01, precision: 2},
	{group: "Contact and fields", key: "debrisBallInfluence", label: "Ball influence", configName: "defaultDebrisBallInfluence", min: 0, max: 1, step: 0.01, precision: 2},
	{group: "Contact and fields", key: "debrisFieldScale", label: "Black-hole / gravity field scale", configName: "defaultDebrisFieldScale", min: 0, max: 3, step: 0.05, precision: 2},
	{group: "Contact and fields", key: "debrisMagnetScale", label: "Magnet scale", configName: "defaultDebrisMagnetScale", min: 0, max: 3, step: 0.05, precision: 2},
}

var defaultPalette = []string{
	"#1d3557", // background
	"#f1faee", // paddle and magic bricks
	"#a8dadc", // normal bricks
	"#e63946", // ball
	"#f1faee", // text
	"#e63946", // spin marker
	"#f4a261", // unbreakable bricks
	"#e63946", // second ball
	"#e63946", // second-ball spin marker
}

var defaultAudioRoomPresets = map[string]audioRoomPreset{
	"none": {
		name: "none", defaultDry: 1,
		inputFilterType: "lowpass", inputFrequency: 20000, inputQ: 0.0001,
		outputFilterType: "lowpass", outputFrequency: 20000, outputQ: 0.0001,
		delay1: 0.01, delay2: 0.02, delay3: 0.04,
	},
	"smallroom": {
		name: "smallroom", defaultDry: 0.78,
		inputFilterType: "highpass", inputFrequency: 80, inputQ: 0.35,
		outputFilterType: "lowpass", outputFrequency: 9000, outputQ: 0.45,
		delay1: 0.011, delay2: 0.023, delay3: 0.041,
		tapGain1: 0.40, tapGain2: 0.28, tapGain3: 0.20, feedback: 0.08,
	},
	"bigopenroom": {
		name: "bigopenroom", defaultDry: 0.82,
		inputFilterType: "highpass", inputFrequency: 70, inputQ: 0.30,
		outputFilterType: "lowpass", outputFrequency: 11000, outputQ: 0.35,
		delay1: 0.070, delay2: 0.150, delay3: 0.310,
		tapGain1: 0.28, tapGain2: 0.20, tapGain3: 0.15, feedback: 0.08,
	},
	"smallhall": {
		name: "smallhall", defaultDry: 0.68,
		inputFilterType: "highpass", inputFrequency: 90, inputQ: 0.35,
		outputFilterType: "lowpass", outputFrequency: 7000, outputQ: 0.50,
		delay1: 0.028, delay2: 0.061, delay3: 0.115,
		tapGain1: 0.38, tapGain2: 0.30, tapGain3: 0.24, feedback: 0.22,
	},
	"bighall": {
		name: "bighall", defaultDry: 0.55,
		inputFilterType: "highpass", inputFrequency: 80, inputQ: 0.35,
		outputFilterType: "lowpass", outputFrequency: 5400, outputQ: 0.55,
		delay1: 0.055, delay2: 0.125, delay3: 0.260,
		tapGain1: 0.34, tapGain2: 0.28, tapGain3: 0.24, feedback: 0.36,
	},
	"cave": {
		name: "cave", defaultDry: 0.42,
		inputFilterType: "highpass", inputFrequency: 65, inputQ: 0.30,
		outputFilterType: "lowpass", outputFrequency: 3000, outputQ: 0.65,
		delay1: 0.075, delay2: 0.190, delay3: 0.420,
		tapGain1: 0.36, tapGain2: 0.30, tapGain3: 0.26, feedback: 0.50,
	},
	"space": {
		name: "space", defaultDry: 0.58,
		inputFilterType: "highpass", inputFrequency: 160, inputQ: 0.40,
		outputFilterType: "lowpass", outputFrequency: 7000, outputQ: 0.35,
		delay1: 0.110, delay2: 0.290, delay3: 0.520,
		tapGain1: 0.22, tapGain2: 0.18, tapGain3: 0.14, feedback: 0.26,
	},
	"matrix": {
		name: "matrix", defaultDry: 0.60,
		inputFilterType: "bandpass", inputFrequency: 1800, inputQ: 2.80,
		outputFilterType: "lowpass", outputFrequency: 6500, outputQ: 1.20,
		delay1: 0.011, delay2: 0.023, delay3: 0.047,
		tapGain1: 0.32, tapGain2: 0.27, tapGain3: 0.22, feedback: 0.58,
	},
	"underwater": {
		name: "underwater", defaultDry: 0.10,
		inputFilterType: "lowpass", inputFrequency: 600, inputQ: 0.80,
		outputFilterType: "lowpass", outputFrequency: 750, outputQ: 0.55,
		delay1: 0.008, delay2: 0.022, delay3: 0.041,
		tapGain1: 0.40, tapGain2: 0.30, tapGain3: 0.23, feedback: 0.18,
	},
}
