package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	js "physical/internal/qtbridge"
	"testing"
)

// Profile the real fixed-step/drawing/encoding path without a window or audio
// device. This diagnoses engine CPU cost; it does not measure display FPS.
func BenchmarkNativeDenseFrames(b *testing.B) {
	canvas = js.Initialize(nativeLevels, filepath.Join(b.TempDir(), "progress.json"))
	ctx = canvas.Call("getContext", "2d")
	ensureBrickCanvas()
	loadLevels()
	setupInput()
	nativeVisible = true
	encoder := json.NewEncoder(io.Discard)
	for _, level := range []int{23, 26, 27} {
		b.Run(fmt.Sprintf("level%d", level), func(b *testing.B) {
			resetRound := func() {
				rand.Seed(1) // Use GODEBUG=randseednop=0 for repeatable comparisons.
				blackHoleRNG = rand.New(rand.NewSource(1))
				lives, score = defaultLives, 0
				gameOver, win, paused = false, false, false
				startLevel(level - 1)
				if !autoPaddleEnabled {
					nativeKey(nativeCommand{Key: "o", Down: true})
				}
				if waitingForStart || paused {
					nativeKey(nativeCommand{Key: " ", Down: true})
				}
				for _, key := range []string{"z", "m", "2", "2"} {
					nativeKey(nativeCommand{Key: key, Down: true})
				}
				if os.Getenv("PHYSICAL_BENCH_HASH_GRID") == "1" {
					brickGridCells = nil // Exercise the original map lookup path.
				}
			}
			resetRound()
			if paused || waitingForStart || activeBallCount() < 3 {
				b.Fatal("benchmark failed to start active multiball gameplay")
			}
			nativeTime = js.Now()
			lastTime = nativeTime
			b.ReportAllocs()
			b.ResetTimer()
			for frame := 0; frame < b.N; frame++ {
				if frame%900 == 899 || levelAdvancePending || gameOver {
					resetRound()
				}
				js.BeginFrame()
				handleNativeCommand(nativeCommand{Action: "tick", DT: 1. / 60})
				if err := emitNativeFrame(encoder, "tick"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
