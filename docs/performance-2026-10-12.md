# Level 26 performance and stability — 2026-10-12

Historical measurements of the implementation at that time. Later audio,
control and watchdog changes are documented in [the current audit](audit-2026-10-12.md);
the values and raw logs below remain unchanged.

Measured on Intel HD Graphics 520, a 1920×1080 display at scale 1.6,
with Quickshell 0.3.1 and Qt 6.11.2 and the OpenGL renderer. The game window was
1080×542 logical pixels (a 2:1 playfield inside the four-pixel frame).
Tests used isolated progress and the automatic paddle. No other benchmark
or build ran alongside the final measurements.

| Mode | Duration | Overall painted FPS | Lowest busy gameplay sample | Samples below 30 FPS |
| --- | ---: | ---: | ---: | ---: |
| Normal gameplay | 180.66 s | 53.5 | 19.2 | 6 / 170 |
| Extra balls, magnets and zapper | 180.54 s | 42.7 | 13.7 | 33 / 152 |

Overall FPS is total painted frames divided by actual elapsed time, including
loading and transitions. Gameplay sample statistics exclude the first two
samples and samples marked as ready, completed or game over. Sampling is
scheduled every second but can take longer when the UI is busy; each sample
uses its actual elapsed time. These are sampled painted FPS, not per-frame
latency percentiles. Both modes reached the 150-piece debris limit.

The stress test restarts level 26 after completion or game over and restores
the additional effects on each restart. Natural powerups and ball losses make
runs variable; these measurements do not constitute a controlled comparison
of individual effects. Busy brick destruction still produces sustained drops
below 30 FPS. Quieter gameplay approaches 60 FPS. A dedicated profile is needed
to separate brick rasterization, physics and frame transport costs.

An earlier test using many Qt SoundEffect streams stopped producing frames
and flooded the log with QSocketNotifier warnings. There was no recorded core
dump or OOM kill. Muted gameplay completed, while merely avoiding forced sound
restarts did not solve the failure. Replacing those streams with one persistent
native mixer allowed both final audio-enabled runs to finish with no socket
warnings. This supports an audio-path cause without identifying the exact
Qt backend defect. The benchmark now fails and stops on repeated socket errors.

Validation: `./check` passed native rendering, Go tests/vet, all 33 engine levels,
replay, tuner, volume and QML checks. The Wayland smoke test passed wall/paddle
playback and fullscreen restoration. Virtual high-DPI 4K and 8K layout checks
passed; performance at those resolutions on real hardware remains unmeasured.

Reproduce:

```sh
PHYSICAL_TEST_WAYLAND=1 PHYSICAL_PERF_STRESS=0 python3 tests/performance.py
PHYSICAL_TEST_WAYLAND=1 python3 tests/performance.py
```

Raw logs: [normal](performance/2026-10-12-level26-normal.log),
[stress](performance/2026-10-12-level26-stress.log).
