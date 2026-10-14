# Private testing and packaging

Keep this repository private while testing. `./package` only builds files locally;
it does not push, publish, create a release, or change GitHub visibility.

## Build the two bundles

```sh
./package
```

The output is in `dist/`:

- `physical-omarchy-0.2.0-linux-x86_64.tar.gz`: Omarchy plugin, compiled engine
  and Qt module, assets, licenses, and source for rebuilding.
- `physical-0.2.0-linux-x86_64.tar.gz`: standalone Qt application and runtime
  assets. It needs no Omarchy or Quickshell.

Names follow the manifest version and build machine's architecture. Each bundle
includes `BUILD-INFO.json` and file checksums (except the plugin's generated
native-library path definition); `dist/SHA256SUMS` covers the archives.
Build information records Qt, libc, architecture and the source commit, including
whether the working tree had modifications. `./package --no-build` packages an
existing build; use it only after building the current native sources.

## Test the Omarchy plugin on another machine

Transfer the Omarchy archive privately, then extract it and enter its folder:

```sh
sha256sum -c SHA256SUMS
./install --prebuilt
```

The installer checks the bundle, offers any missing runtime packages, copies it
to `~/.config/omarchy/plugins/b.physical`, restarts the shell, waits for its
plugin scan to discover Physical, and then enables it. This also works after
removing the plugin through Omarchy.
Saved progress is kept. Restarting the shell resets an active game and its RAM
replay. Go, GCC, pkgconf and Wayland build tools are unnecessary in prebuilt mode.

The compiled Qt module should use the same Qt major/minor as the receiving
machine. The prebuilt installer checks that match; patch versions may differ.
If Qt differs, or the binary needs newer system libraries, use `./install`
without `--prebuilt` to build against that machine's runtime. These are native
Linux binaries, not a self-contained AppImage or Flatpak.

Check the ball button, start, sound, right-click capture/release, menu controls,
Space+F fullscreen, retaining capture through Super+F and Super+Alt+F (both
directions), release on switching apps, Escape in replay and live play, and
dense levels 23, 26 and 27.
Before calling the release ready, test on at least one additional machine. Only
Intel HD 520 has been tested so far.

## Run or install the standalone app

Extract the standalone archive. Run it directly, or install it for your user:

```sh
./bin/physical
./install
```

Installation places the executable in `~/.local/bin`, assets in
`~/.local/share/physical-app`, and the white-ball icon and desktop entry in the app
launcher. Qt 6 Quick/QML, Qt Multimedia and a Qt display-platform plugin must
already be installed. On Arch/Omarchy:

```sh
omarchy pkg add qt6-base qt6-declarative qt6-multimedia qt6-wayland
```

The installer checks checksums and missing shared libraries. Install runtime
packages first if needed. The standalone bundle is built against the Qt and libc
versions in `BUILD-INFO.json`; it is not guaranteed to run on an older distro.
Other systems can build from source using compatible Qt 6 development packages,
Go, a C++ compiler, pkg-config, Wayland headers and wayland-scanner. The current
build script expects Arch's `/usr/lib/qt6/moc` and Wayland protocol paths.

Run `./bin/physical` in the source checkout after `./build` to test locally.
`--data-dir PATH` selects a custom asset folder. Standalone progress uses
`~/.local/share/physical/progress.json`; the plugin uses
`~/.local/share/physical-omarchy/progress.json`. Both respect `XDG_DATA_HOME`
and `PHYSICAL_STATE_PATH`. SVG exports share `~/Pictures/Physical/` by default.
The standalone UI uses Qt's desktop palette, while the plugin follows Omarchy's
live theme. Both retain the game's own level palettes and share game controls.

Run the standalone bundle's `./uninstall` to remove app files while retaining
progress and exports. Set `PHYSICAL_PREFIX` for a different installation prefix
on both install and uninstall. The installers do not require root. The standalone installer uses Python 3
to write its desktop entry; running the standalone game itself does not need Python.

## Publishing later

Omarchy's [publishing guide](https://plugins.omarchy.org/publish.html) requires a
public repository, a valid root manifest, README and license. It gives no build
recipe. The plugin installer clones a repository and loads its QML; it does not
run this project's build or install script.

For automatic installation, the repository listed in the marketplace must
already contain the runtime files, including `bin/physical-engine`,
`native/libphysicalrenderer.so`, the placeholder `native/qmldir`, QML, fonts and
sounds. The generated Omarchy folder is suitable as the starting point for that
distribution repository: it includes the compiled files and has no development
`.gitignore` that would hide them. `Plugin.qml` prepares the native module's
absolute local path at startup before loading the game. It needs Python (already
present on Omarchy), performs no build/download, and changes only `native/qmldir`.
Keep the shipped placeholder definition unchanged between versions so this
machine-specific edit does not conflict with normal fast-forward updates.
Rebuild and replace binaries when native code
changes or Omarchy moves to a different Qt version, and restart the shell after
updating compiled modules. Putting an archive in GitHub
Releases alone does not make `omarchy plugin add` download it.

A source-only repository needs explicit manual build instructions before
activation. This project's source repository ignores generated binaries and is
not ready for automatic marketplace installation by itself. No marketplace
submission or public repository has been created.

Code is [MIT](LICENSE), sounds are [CC0 1.0](sounds/LICENSE), and the bundled font
retains its [SIL Open Font License](fonts/LICENCE.md).
