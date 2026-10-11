# Full brick rendering, audio and performance — 2026-10-12

Rounded bricks now use cached GPU triangle meshes instead of repeatedly
rasterizing the entire brick texture. Large meshes are split into chunks of
at most 61,440 vertices (complete triangles). The failure matched
a 16-bit vertex-range limit: the old single-node mesh displayed only 195 of
660 synthetic bricks; the chunked version displayed all 660. The GPU regression
checks the expected palette color at every brick center in an actual window
capture. Native geometry tests also check every vertex, chunk shrinking and
empty frames. Debris and zapper geometry share the safe chunking path.

Normal brick hits choose randomly from all 21 original samples, avoiding
immediate repeats. Activated power-ups use their corresponding original cue
for all 12 feature types. Feature events retain their identity and take
priority over generic brick collisions when the queue is full. The single
native audio stream remains; its mixing loop now reads immutable clip data
directly and supports the engine's pitch scale. Sample sources are listed in
`sounds/samples.json`. Browser stereo positioning, filtering and reverb remain
outside this port's audio implementation.

Hardware: Intel HD Graphics 520, 1920×1080 display at scale 1.6, Qt 6.11.2,
Quickshell 0.3.1, OpenGL. The game uses its default 1080×542 logical-pixel
window. Tests use private progress, automatic paddle, extra balls, magnets,
zapper effects, enabled sound and the default 25% volume. They restart the
same level after completion or game over. No build or other benchmark runs
alongside the final measurements. Rendering instrumentation is disabled.

| Level | Duration | Overall painted FPS | Lowest busy sample | Busy samples below 30 FPS |
| --- | ---: | ---: | ---: | ---: |
| 23 | 180.99 s | 58.2 | 51.2 | 0 / 173 |
| 26 | 180.28 s | 55.6 | 40.0 | 0 / 174 |
| 27 | 180.00 s | 59.2 | 57.1 | 0 / 174 |

All three runs completed without engine errors, rendering stalls, crashes or
Qt socket warnings. Each reached the 150-piece debris limit. Initial loading
samples were about 11 FPS and are included in the overall average. Level 27
spent considerable time with one remaining brick; the supplementary test
below repeatedly reloads it to exercise fresh dense destruction.

The supplementary level 27 run lasted 90.99 seconds with a maximum round
length of 30 seconds. It naturally cleared and restarted four times before
that limit. Overall painted FPS was 58.0, the lowest busy sample was 59.0,
and no busy sample fell below 30 FPS (85 samples). It reached 150 debris
pieces and completed without errors. [Repeated-round log](performance/2026-10-12-gpu-level27-repeat.log).

Raw logs: [level 23](performance/2026-10-12-gpu-level23.log),
[level 26](performance/2026-10-12-gpu-level26.log),
[level 27](performance/2026-10-12-gpu-level27.log).

Overall FPS is total painted frames divided by actual elapsed time, including
loading and transitions. Busy sample statistics exclude the first two samples
and samples marked ready, completed or game over. Sampling runs about once a
second and uses actual elapsed time. These are sampled frame rates, not
per-frame latency percentiles. Power-up and ball changes still make gameplay
variable. `GODEBUG=randseednop=0` re-enables the game's existing fixed random
seed under Go 1.27 for repeatable level initialization. Real 4K/8K performance
has not been measured.

Validation: native rendering tests, Go tests and vet, all 33 engine levels,
replay, SVG export, saved volume and tuner passed. The actual Wayland QML smoke
test passed all 33 levels, fullscreen restoration, contact and feature audio,
the random sample bank, hidden pause and reopening. The full GPU field check
passed 660/660 visible bricks. The benchmark fails immediately on engine
errors, a closed window, stalled rendering or repeated Qt socket errors.

Reproduce:

```sh
PHYSICAL_TEST_WAYLAND=1 ./check
for level in 23 26 27; do
  PHYSICAL_TEST_WAYLAND=1 PHYSICAL_PERF_LEVEL="$level" GODEBUG=randseednop=0 python3 tests/performance.py || break
done
PHYSICAL_TEST_WAYLAND=1 PHYSICAL_PERF_LEVEL=27 PHYSICAL_PERF_SECONDS=90 PHYSICAL_PERF_RESTART_SECONDS=30 GODEBUG=randseednop=0 python3 tests/performance.py
```
