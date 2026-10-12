package main

import js "physical/internal/qtbridge"

func drawBrickBatch(target js.Value, data []float64, tilt bool) {
	target.Call("bricks", data, tilt, brickRadius,
		palette[2], brickStrokeColor, magicColor, magicStrokeColor, palette[6], unbreakableStrokeColor)
}

// One compact brick batch replaces thousands of canvas commands on every
// cache rebuild. The renderer keeps the original palette, radius and tilt.
func drawBrickCanvasBatch() bool {
	data := make([]float64, 0, len(bricks)*6)
	for i := range bricks {
		br := &bricks[i]
		if !br.alive {
			continue
		}
		flags := 0.0
		if br.magic {
			flags = 1
		}
		if br.unbreakable {
			flags = 2
		}
		data = append(data, br.x, br.y, br.w, br.h, br.tiltRadians, flags)
	}
	drawBrickBatch(brickCtx, data, physicsConfig.drawBrickTilt)
	return true
}
