# Physical for Omarchy

## Install as an Omarchy plugin

Requires Omarchy. Clone the repository and run:

```sh
git clone https://github.com/brontosaurusrex/physical-omarchy-plugin.git
cd physical-omarchy-plugin
./install
```

The installer offers missing dependencies, builds the binaries, installs the
plugin and restarts the shell. Click the ball icon in the bar to open it.

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

## More information

[Packaging and compatibility](PACKAGING.md) · [Technical details](TECH.md) ·
[Original source](UPSTREAM.md). Run `./check` for project validation.

Code: [MIT](LICENSE). Sounds: [CC0 1.0](sounds/LICENSE).
Font: [SIL Open Font License](fonts/LICENCE.md).
