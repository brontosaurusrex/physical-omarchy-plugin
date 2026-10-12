# Physical for Linux

A native Qt brick breaker with Go physics, GPU rendering, sound, replay, a physics
lab, and the original game palettes. This app needs no Omarchy or Quickshell.

Run from this folder:

```sh
./bin/physical
```

Install for your user, including a launcher entry and white-ball icon:

```sh
./install
```

Qt 6 Quick/QML, Qt Multimedia and a Qt display-platform plugin must be installed.
This bundle needs compatible Qt and Linux system libraries; see `BUILD-INFO.json`.
On Arch/Omarchy: `omarchy pkg add qt6-base qt6-declarative qt6-multimedia qt6-wayland`.

**On Wayland, right click captures the mouse; right click again releases it.** Space starts
or pauses play, Space+F toggles fullscreen, and Escape returns from replay or
closes the live game. The three-dot menu shows the full controls table, settings,
and volume slider. Audio starts at 25% and saves your changes.

Progress is in `~/.local/share/physical/progress.json`; SVG exports go to
`~/Pictures/Physical/`. `./uninstall` removes app files and keeps your progress
and exports. Set `PHYSICAL_PREFIX` on install and uninstall to choose a prefix
other than `~/.local`. Installed assets are in `share/physical-app`, separate
from saved progress. Your desktop window manager controls placement and tiling.

See [PACKAGING.md](PACKAGING.md) for compatibility and second-machine testing,
and [TECH.md](TECH.md) for what runs on the CPU and GPU.

Code: [MIT](LICENSE). Sounds: [CC0 1.0](share/physical-app/sounds/LICENSE).
Font: [SIL Open Font License](share/physical-app/fonts/LICENCE.md).
