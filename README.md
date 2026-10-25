# Physical for Omarchy

## Install as an Omarchy plugin

Requires Omarchy. Clone the repository and run:

```sh
git clone https://github.com/brontosaurusrex/physical-omarchy-plugin.git
cd physical-omarchy-plugin
./install
```

The installer offers missing dependencies, builds the binaries, installs the
plugin and restarts the shell. Click the paddle-and-ball icon in the bar to open it.

## Prebuilt bundles

Current bundles target **Linux x86_64 (64-bit Intel/AMD)** and were built with
**Qt 6.11.2**. Compatible Qt/system libraries are required; the plugin installer
requires Qt 6.11.x. ARM builds are not included.

Run `./package` to generate these files in **`dist/`**:

- Omarchy plugin: `physical-omarchy-0.2.0-linux-x86_64.tar.gz`
- Standalone app: `physical-0.2.0-linux-x86_64.tar.gz`

Extract the Omarchy bundle, enter its folder and run:

```sh
./install --prebuilt
```

No Go or C++ compiler is needed. For the standalone bundle, run `./install`
after extraction, or `./bin/physical` to run it directly.

## Flip the brick layout

Press **Y** to flip the current level's brick rows top-to-bottom within their
existing area. A screen wipe switches whole bricks along each row from left to
right. Press Y again to restore the layout with a right-to-left wipe. Bricks
keep their shape throughout the transition.
Destroyed bricks stay destroyed; the ball and paddle keep playing during the wipe.
Each new level starts in its original orientation. Manual Y flips are assisted
and do not replace unassisted records. Replay and SVG export retain the layout
shown at each recorded moment.

## Game speed and brick flips

Keys **6 / 7 / 8 / 9 / 0** select **80 / 90 / 100 / 150 / 200%** game speed.
The status display confirms changes as `@80%`, for example. The last selection
is saved and carries through levels, retries, demo restarts and reopening.
The physics step stays fixed; simulation time advances at the selected pace.

Each speed has separate level and full-game records. Existing records remain
the 100% records. Results show the run's speed; completion times account for the
selected pace and exclude READY, pause and result screens.
Changing speed during a level produces a mixed-speed result without updating
fixed-speed records. A full-game run must keep one speed throughout to qualify.

Brick flips are disabled as magic-brick rewards in normal play. Manual **Y**
remains available and marks the run assisted.
The optional power-up code is retained: custom levels can opt in with
`enableBrickFlip=true`. It uses the same wipe as Y and returns after the normal
`powerUpDuration` (10 seconds by default); another pickup refreshes the timer.

## Continuous demo

Press **O** or use **Demo** in the menu to toggle assisted auto paddle. Demo mode
advances after each completed level, starts a fresh game after losing or winning,
and skips level 14. Results remain visible for two seconds before continuing.
Pause, replay and the editor still stop automatic progression. Turning demo off
restores manual progression, including level 14. Demo runs do not qualify for
unassisted records.

## Catching intermittent FPS drops

The FPS overlay's lowest value measures active gameplay, excluding level title
screens, pauses and transitions. It starts after a short gameplay warm-up;
Shift+F resets it. The current-FPS display still shows the current update rate.

The frame recorder is **disabled by default**. Installing or updating the plugin
does not enable it. When explicitly enabled, it samples once per second during play.
To enable it
before opening Physical (reload the plugin or restart the shell if it is already
loaded):

```sh
mkdir -p "${XDG_STATE_HOME:-$HOME/.local/state}/physical/diagnostics"
touch "${XDG_STATE_HOME:-$HOME/.local/state}/physical/diagnostics/enabled"
```

After a three-second startup/resume grace period, two seconds below 40 FPS in
the window clock, engine updates, reported engine FPS or main-item paints save
the preceding 30 samples and the next ten samples to
`~/.local/state/physical/diagnostics/latest.json`. Incident files are retained
separately, up to eight, with a one-minute cooldown. Closing the game flushes any
unfinished capture and writes `last-session.json` with the latest 30 samples.
The directory follows `$XDG_STATE_HOME` when set. Captures contain timing and game
state counters, without screenshots, key contents or mouse positions.

Records include the window frame clock, engine tick round trips and oldest
pending request, frame parsing/scene preparation, main-item paint CPU time,
Qt frame submissions, mouse event counts, capture/focus state and the FPS overlay.
Round trips include receiving and accepting the engine frame. Paint timing
excludes GPU execution and child layers; Qt submissions do not measure physical
monitor presentation. These measurements help locate a stall, rather than
provide a full GPU profile.

Schema 2 also splits tick latency into command waiting, engine work (including
scene JSON encoding), return-pipe/Qt delivery, and Qt receive-to-accept time.
These stages use shared Unix microsecond timestamps; negative intervals are
rejected, and wall-clock adjustments can affect estimates. Queue counters track
Qt's pending write bytes and commands/mouse commands not yet acknowledged by an
engine reply. Unacknowledged counts are upper bounds, not exact engine queue sizes.

Remove the `enabled` file and reload the plugin to disable recording. Tests may
use `PHYSICAL_FRAME_DIAGNOSTICS=0|1` and `PHYSICAL_DIAGNOSTICS_DIR` to isolate it.
Normal gameplay keeps history in memory; disk writes happen only for incidents
and when closing. The recorder does not request extra frames or inject input.

## More information

[Packaging and compatibility](PACKAGING.md) · [Technical details](TECH.md) ·
[Original source](UPSTREAM.md). Run `./check` for project validation.

Code: [MIT](LICENSE). Sounds: [CC0 1.0](sounds/LICENSE).
Font: [SIL Open Font License](fonts/LICENCE.md).
