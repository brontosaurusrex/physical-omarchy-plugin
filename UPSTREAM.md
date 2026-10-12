# Port source

Imported from `/home/b/source/physical/dev` on 2026-10-11, including its current
uncommitted physics-buffer and canvas batching work. Repository HEAD at import:
`f0ec34186ea28bbff636f878995669696a55fbf4`.

The original project remains unchanged. The copied Go source keeps its physics,
collision ordering, fixed-step scheduler, level parser, powerups, scoring,
progression, debris, replay, and vector export. Browser-specific functions remain
in the source to minimize changes to the engine, but are not used by the QML
front end and are removed from the native binary by Go's linker when unreachable.

Native adaptations replace the `syscall/js` import with a local drawing/storage
adapter, change startup to a line-based native protocol, retain level colors, route original sample/oscillator parameters to a persistent Qt audio mixer, and provide clipboard/SVG file
exports. The browser physics editor is replaced with native QML controls backed
by the same slider definitions and setter functions. The accelerated renderer
uses a local C++ Qt module; fragment instances are compact native commands and
their cached outlines are triangulated for batched scene graph drawing.
Magnetic debris uses an additional direct cell array alongside the original
brick map. It preserves cell contents and lookup order; regression tests compare
its targets with the map path before and after brick destruction.

The original checkout did not declare a license. The author has now licensed
this port's code under [MIT](LICENSE) and dedicated the bundled sounds to the
public domain under [CC0 1.0](sounds/LICENSE). The bundled font retains its
[original SIL Open Font License](fonts/LICENCE.md).

The 2026-10-12 audit compares audio against that original checkout's
`dev/main.go`, `config.go` and runtime sound directories. The port now decodes
runtime Opus samples rather than the uncompressed source masters; the
`sounds/samples.json` manifest records source/output hashes. Wall/paddle and
other synthesized cues retain their original tone parameters. The per-level
room graph now retains the original preset parameters and routing. Native DSP,
32-sample filter updates during room transitions, a 12-second silent tail limit
and output scheduling remain fidelity limits, listed in
[the audio/control audit](docs/audit-2026-10-12.md).

Desktop adaptations include Escape leaving replay (closing the plugin during
live play), Space+F fullscreen, Shift+K / a menu button leaving replay, and a
usable sound toggle on Ready and in menus. Replay uses the same antialiased
brick meshes as live play, with a separate cache for recorded visibility.
The first level's Ready title is “P H Y S I C A L” in the port. These
are native UI/content changes; the original source is not edited.
