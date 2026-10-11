package main

import (
	"reflect"
	"testing"
)

func TestCandidateBrickIndicesResetsAndGrowsBuffer(t *testing.T) {
	oldBricks, oldGrid := bricks, brickGrid
	oldRows, oldCols := gridRows, gridCols
	oldW, oldH := gridCellWidth, gridCellHeight
	oldX, oldY := gridOffsetLeft, gridOffsetTop
	t.Cleanup(func() {
		bricks, brickGrid = oldBricks, oldGrid
		gridRows, gridCols = oldRows, oldCols
		gridCellWidth, gridCellHeight = oldW, oldH
		gridOffsetLeft, gridOffsetTop = oldX, oldY
	})

	gridRows, gridCols = 1, 2
	gridCellWidth, gridCellHeight = 100, 100
	gridOffsetLeft, gridOffsetTop = 0, 0
	brickGrid = map[int][]int{0: {3, 1}, 1: {2, 0}}
	buffer := make([]int, 1, 2)
	buffer[0] = 999 // A previous query must not leak into the next result.
	got := candidateBrickIndices(&Ball{x: 100, y: 50, r: 5}, buffer)
	if want := []int{3, 1, 2, 0}; !reflect.DeepEqual(got, want) {
		t.Fatalf("grid traversal: got %v, want %v", got, want)
	}
	if got := candidateBrickIndices(&Ball{x: 1000, y: 1000, r: 5}, got); len(got) != 0 {
		t.Fatalf("empty query retained candidates: %v", got)
	}

	gridRows, gridCols = 0, 0
	bricks = make([]brick, 48)
	got = candidateBrickIndices(&Ball{}, buffer)
	if len(got) != len(bricks) {
		t.Fatalf("fallback truncated: got %d candidates", len(got))
	}
	for i, index := range got {
		if index != i {
			t.Fatalf("fallback order at %d: got %d", i, index)
		}
	}
}

func TestFindBrickContactsDoesNotTruncateDenseOverlap(t *testing.T) {
	oldBricks, oldRows, oldPhysics := bricks, gridRows, physicsConfig
	t.Cleanup(func() { bricks, gridRows, physicsConfig = oldBricks, oldRows, oldPhysics })
	gridRows = 0
	physicsConfig.cornerPhysicsAmount = 0
	bricks = make([]brick, 48)
	for i := range bricks {
		bricks[i] = brick{x: 100, y: 100, w: 100, h: 100, alive: true}
	}
	var buffer [8]brickContact
	got := findBrickContacts(&Ball{x: 150, y: 150, r: 10, vx: 20, vy: 30}, 149, 149, buffer[:0])
	if len(got) != len(bricks) {
		t.Fatalf("dense overlap truncated: got %d contacts", len(got))
	}
	for i, contact := range got {
		if contact.index != i {
			t.Fatalf("contact order at %d: got %d", i, contact.index)
		}
	}
}

func TestFindBrickContactsCommonQueryDoesNotAllocate(t *testing.T) {
	oldBricks, oldRows, oldPhysics := bricks, gridRows, physicsConfig
	t.Cleanup(func() { bricks, gridRows, physicsConfig = oldBricks, oldRows, oldPhysics })
	gridRows = 0
	physicsConfig.cornerPhysicsAmount = 0
	bricks = []brick{{x: 100, y: 100, w: 100, h: 100, alive: true}}
	b := Ball{x: 150, y: 150, r: 10, vx: 20, vy: 30}
	allocations := testing.AllocsPerRun(100, func() {
		var buffer [8]brickContact
		if len(findBrickContacts(&b, 149, 149, buffer[:0])) != 1 {
			panic("expected one contact")
		}
	})
	if allocations != 0 {
		t.Fatalf("common collision query allocated %g times", allocations)
	}
}

func TestUpdateBrickDebrisPreservesSurvivorsAndClearsTail(t *testing.T) {
	oldDebris, oldBricks, oldBall, oldExtras := brickDebris, bricks, ball, extraBalls
	oldTick, oldGravity, oldBlackHole := debrisFieldTick, currentGravity, blackHoleActive
	oldAir, oldAngular, oldMagnet, oldMax := debrisAirDrag, debrisAngularDrag, debrisMagnetScale, debrisMaxSpeed
	t.Cleanup(func() {
		brickDebris, bricks, ball, extraBalls = oldDebris, oldBricks, oldBall, oldExtras
		debrisFieldTick, currentGravity, blackHoleActive = oldTick, oldGravity, oldBlackHole
		debrisAirDrag, debrisAngularDrag, debrisMagnetScale, debrisMaxSpeed = oldAir, oldAngular, oldMagnet, oldMax
	})
	bricks, extraBalls = nil, nil
	ball = Ball{}
	currentGravity, blackHoleActive = 0, false
	debrisAirDrag, debrisAngularDrag, debrisMagnetScale, debrisMaxSpeed = 0, 0, 0, 0
	debrisFieldTick = 0
	brickDebris = []debrisFragment{
		{renderID: 1, age: 2, lifetime: 1, fillColor: "expired"},
		{renderID: 2, x: 300, y: 300, vx: 4, lifetime: 10, radius: 1},
		{renderID: 3, x: 300, y: canvasHeight + debrisOffscreenMargin + 100, lifetime: 10, radius: 1},
		{renderID: 4, x: 400, y: 300, vx: 8, lifetime: 10, radius: 1},
		{renderID: 5, age: 2, lifetime: 1, fillColor: "expired"},
	}
	storage := brickDebris
	updateBrickDebris(0.25)
	if len(brickDebris) != 2 || brickDebris[0].renderID != 2 || brickDebris[1].renderID != 4 {
		t.Fatalf("survivors were lost or reordered: %v", brickDebris)
	}
	if brickDebris[0].x != 301 || brickDebris[0].previousX != 300 || brickDebris[1].x != 402 || brickDebris[1].previousX != 400 {
		t.Fatal("survivor integration or interpolation history changed")
	}
	for i := 2; i < len(storage); i++ {
		if !reflect.DeepEqual(storage[i], debrisFragment{}) {
			t.Fatalf("removed fragment retained in backing storage at %d", i)
		}
	}
}
