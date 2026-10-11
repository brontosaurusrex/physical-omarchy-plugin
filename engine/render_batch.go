package main

import (
	"encoding/binary"
	"math"
	js "physical/internal/qtbridge"
)

const brickBatchStride = 6 * 8 // x, y, width, height, tilt, flags as float64

var (
	brickBatchData  []byte
	brickBatchBytes js.Value
)

// The existing Go renderer remains available to pages that do not load the
// optional JavaScript helper. Buffers are reused across brick-cache rebuilds.
func drawBrickCanvasBatch() bool {
	helper := js.Global().Get("physicalCanvasBatch")
	if helper.IsUndefined() || helper.IsNull() {
		return false
	}
	drawBricks := helper.Get("drawBricks")
	if drawBricks.Type() != js.TypeFunction {
		return false
	}

	size := len(bricks) * brickBatchStride
	if cap(brickBatchData) < size {
		brickBatchData = make([]byte, size)
	} else {
		brickBatchData = brickBatchData[:size]
	}
	count := 0
	for i := range bricks {
		br := &bricks[i]
		if !br.alive {
			continue
		}
		flags := 0
		if br.magic {
			flags |= 1
		}
		if br.unbreakable {
			flags |= 2
		}
		values := [6]float64{br.x, br.y, br.w, br.h, br.tiltRadians, float64(flags)}
		for j, value := range values {
			offset := count*brickBatchStride + j*8
			binary.LittleEndian.PutUint64(brickBatchData[offset:offset+8], math.Float64bits(value))
		}
		count++
	}
	if brickBatchBytes.IsUndefined() || brickBatchBytes.Get("byteLength").Int() < size {
		brickBatchBytes = js.Global().Get("Uint8Array").New(size)
	}
	js.CopyBytesToJS(brickBatchBytes, brickBatchData[:count*brickBatchStride])
	drawBricks.Invoke(brickCtx, brickBatchBytes, count, physicsConfig.drawBrickTilt, brickRadius,
		palette[2], brickStrokeColor, magicColor, magicStrokeColor, palette[6], unbreakableStrokeColor)
	return true
}
