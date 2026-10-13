package qtbridge

import (
	"encoding/json"
	"math"
	"path/filepath"
	"testing"
)

func TestDebrisTransportPreservesInstancesAndBoundsCache(t *testing.T) {
	target := Initialize(nil, filepath.Join(t.TempDir(), "progress.json")).Call("getContext", "2d")
	points := []float64{-7, -4, 3, -6, 8, 2, -2, 7}
	draw := func(id uint32, x float64) {
		target.Call("debris", id, x, 22.3456789012345, math.Cos(.7), math.Sin(.7), "rgb(120,80,240)", 8., false, .4, points)
	}
	BeginFrame()
	for i := uint32(1); i <= 150; i++ {
		draw(i, 123.456789012345+float64(i))
	}
	ops, _ := Frame()
	if len(ops) != 1 || len(DebrisShapes()) != 150 {
		t.Fatal("first frame lost particles or definitions")
	}
	batch := ops[0][1].(*debrisBatch)
	if len(batch.Colors) != 150 || len(batch.Instances) != 750 {
		t.Fatal("particle count changed")
	}
	if batch.Instances[1] != 124.456789012345 || batch.Instances[2] != 22.3456789012345 || batch.Instances[3] != math.Cos(.7) || batch.Instances[4] != math.Sin(.7) {
		t.Fatal("transport changed positions or rotations")
	}
	BeginFrame()
	for i := uint32(1); i <= 150; i++ {
		draw(i, 200.1234567890123+float64(i))
	}
	ops, _ = Frame()
	if len(DebrisShapes()) != 0 {
		t.Fatal("moving particles resent immutable contours")
	}
	encoded, err := json.Marshal(ops)
	if err != nil {
		t.Fatal(err)
	}
	var legacy [][]any
	for i := uint32(1); i <= 150; i++ {
		legacy = append(legacy, []any{"debris", i, 200.1234567890123 + float64(i), 22.3456789012345, math.Cos(.7), math.Sin(.7), "rgb(120,80,240)", 8., false, .4, points})
	}
	old, _ := json.Marshal(legacy)
	if len(encoded)*4 >= len(old)*3 {
		t.Fatalf("packet reduction too small: %d versus %d", len(encoded), len(old))
	}
	t.Logf("150-fragment movement packet: %d -> %d bytes", len(old), len(encoded))
	BeginFrame()
	draw(1, 1)
	target.Call("fillRect", 0, 0, 10, 10)
	draw(2, 2)
	ops, _ = Frame()
	if len(ops) != 3 || ops[1][0] != "fillRect" {
		t.Fatal("batch crossed a drawing-order boundary")
	}
	if len(debrisShapes) != 2 {
		t.Fatal("unused particle contours retained")
	}
	BeginFrame()
	draw(150, 3)
	Frame()
	if len(DebrisShapes()) != 1 {
		t.Fatal("returning fragment did not resend its contour")
	}
	points[0] = -9
	BeginFrame()
	draw(150, 4)
	Frame()
	if len(DebrisShapes()) != 1 || DebrisShapes()[150].Points[0] != -9 {
		t.Fatal("changed contour not refreshed")
	}
	BeginFrame()
	Frame()
	if len(debrisShapes) != 0 {
		t.Fatal("empty scene retained contours")
	}
}
