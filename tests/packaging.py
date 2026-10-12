#!/usr/bin/env python3
"""Check relocatable archives, compiler-free installs and retained progress."""
import json
import os
import pathlib
import shutil
import subprocess
import tarfile
import tempfile

project = pathlib.Path(__file__).resolve().parents[1]
subprocess.run([str(project / 'package'), '--no-build'], check=True)
subprocess.run(['sha256sum', '--quiet', '-c', 'SHA256SUMS'], cwd=project / 'dist', check=True)
with tempfile.TemporaryDirectory(prefix='physical package test ') as directory:
    root = pathlib.Path(directory)
    bundles = {}
    for archive in (project / 'dist').glob('*.tar.gz'):
        with tarfile.open(archive) as tar:
            tar.extractall(root, filter='data')
        folder = root / archive.name.removesuffix('.tar.gz')
        subprocess.run(['sha256sum', '--quiet', '-c', 'SHA256SUMS'], cwd=folder, check=True)
        info = json.loads((folder / 'BUILD-INFO.json').read_text())
        bundles[info['package']] = folder
        assert all(not p.is_symlink() for p in folder.rglob('*')), 'Package contains symlinks'
        assert str(project) not in (folder / ('native/qmldir' if info['package'] == 'omarchy' else 'share/physical-app/native/qmldir')).read_text()
    assert set(bundles) == {'omarchy', 'standalone'}
    commands = root / 'commands'
    commands.mkdir()
    log = root / 'commands.log'
    for tool in ['g++', 'gcc', 'go', 'pkg-config', 'wayland-scanner']:
        script = commands / tool
        script.write_text('#!/bin/sh\necho "compiler called" >> "$PHYSICAL_COMMAND_LOG"\nexit 99\n')
        script.chmod(0o755)
    (commands / 'omarchy').write_text('''#!/bin/sh
if [ "$1" = plugin ] && [ "$2" = validate ]; then
  exec /usr/bin/omarchy-plugin-validate "$3"
fi
printf '%s\\n' "$*" >> "$PHYSICAL_COMMAND_LOG"
if [ "$1" = restart ] && [ "$2" = shell ]; then
  rm -f "$PHYSICAL_COMMAND_LOG.ready"
  echo 0 > "$PHYSICAL_COMMAND_LOG.polls"
elif [ "$1" = plugin ] && [ "$2" = enable ]; then
  if [ ! -f "$PHYSICAL_COMMAND_LOG.ready" ]; then
    echo "plugin is not known yet" >&2
    exit 1
  fi
fi
''')
    (commands / 'omarchy-shell').write_text('''#!/bin/sh
printf '%s\\n' "$*" >> "$PHYSICAL_COMMAND_LOG"
if [ "$1" = shell ] && [ "$2" = listPlugins ]; then
  polls=$(cat "$PHYSICAL_COMMAND_LOG.polls")
  polls=$((polls + 1))
  echo "$polls" > "$PHYSICAL_COMMAND_LOG.polls"
  if [ "${PHYSICAL_TEST_DISCOVERY:-}" = missing ]; then
    echo '[]'
  elif [ "$polls" -eq 1 ]; then
    exit 1
  elif [ "$polls" -eq 2 ]; then
    echo 'Not ready to accept queries yet'
  elif [ "$polls" -eq 3 ]; then
    echo '[{"id":"other.plugin"}]'
  else
    test -f "$XDG_CONFIG_HOME/omarchy/plugins/b.physical/manifest.json" || exit 1
    touch "$PHYSICAL_COMMAND_LOG.ready"
    echo '[{"id":"b.physical","enabled":false}]'
  fi
fi
''')
    for tool in ['omarchy', 'omarchy-shell']:
        (commands / tool).chmod(0o755)
    env = dict(os.environ, PATH=str(commands) + ':' + os.environ['PATH'], XDG_CONFIG_HOME=str(root / 'config'),
               PHYSICAL_COMMAND_LOG=str(log), PHYSICAL_PREFIX=str(root / 'prefix with spaces'))
    plugin = bundles['omarchy']
    subprocess.run([str(plugin / 'install'), '--prebuilt'], cwd=plugin, env=env, check=True)
    installed = root / 'config/omarchy/plugins/b.physical'
    for name in ['Plugin.qml', 'Physical.qml', 'GameCore.qml', 'theme/Color.qml', 'native/setup-module.py', 'native/libphysicalrenderer.so', 'bin/physical-engine', 'sounds/LICENSE', 'fonts/LICENCE.md']:
        assert (installed / name).is_file(), f'Missing installed plugin file: {name}'
    assert (installed / 'native/qmldir').read_text() == 'plugin physicalrenderer ' + str(installed / 'native') + '\n'
    assert 'compiler called' not in log.read_text(), 'Prebuilt mode invoked a compiler'
    assert 'restart shell' in log.read_text(), 'New native module would not reload'
    sequence = log.read_text().splitlines()
    assert sequence.index('restart shell') < sequence.index('shell listPlugins') < sequence.index('plugin enable b.physical')
    assert int(pathlib.Path(str(log) + '.polls').read_text()) >= 4, 'Discovery was not awaited'
    # Reproduce removing a copy-installed plugin, then installing it again.
    shutil.rmtree(installed)
    subprocess.run([str(plugin / 'install'), '--prebuilt'], cwd=plugin, env=env, check=True)
    assert (installed / 'manifest.json').is_file()
    # A scan that never registers Physical must fail without attempting enable.
    before_timeout = len(log.read_text().splitlines())
    unavailable = subprocess.run([str(plugin / 'install'), '--prebuilt'], cwd=plugin,
                                 env=dict(env, PHYSICAL_TEST_DISCOVERY='missing'),
                                 stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, timeout=25)
    assert unavailable.returncode and 'did not discover the plugin within 15 seconds' in unavailable.stdout, unavailable.stdout
    assert 'plugin enable b.physical' not in log.read_text().splitlines()[before_timeout:]
    standalone = bundles['standalone']
    prefix = pathlib.Path(env['PHYSICAL_PREFIX'])
    progress = prefix / 'share/physical/progress.json'
    progress.parent.mkdir(parents=True)
    progress.write_text('{"saved":"keep"}\n')
    subprocess.run([str(standalone / 'install')], cwd=standalone, env=env, check=True)
    assert (prefix / 'bin/physical').is_file()
    desktop = prefix / 'share/applications/io.github.brontosaurusrex.Physical.desktop'
    assert 'Exec="' + str(prefix / 'bin/physical') + '"' in desktop.read_text(), desktop.read_text()
    # QApplication initializes its platform even for --help. Keep this check
    # independent of desktop socket access, like the standalone test below.
    subprocess.run([str(prefix / 'bin/physical'), '--help'],
                   env=dict(env, QT_QPA_PLATFORM='offscreen', QT_QPA_PLATFORMTHEME=''),
                   check=True, stdout=subprocess.DEVNULL)
    runtime_env = dict(env, PHYSICAL_STANDALONE_BINARY=str(prefix / 'bin/physical'), PHYSICAL_STANDALONE_DATA=str(prefix / 'share/physical-app'))
    # Restore real PATH: Qt and the engine need ordinary runtime helpers, and
    # the separate prebuilt test above has already checked compiler isolation.
    runtime_env['PATH'] = os.environ['PATH']
    subprocess.run(['python3', str(project / 'tests/standalone.py')], env=runtime_env, check=True)
    subprocess.run([str(standalone / 'uninstall')], env=env, check=True)
    assert progress.read_text() == '{"saved":"keep"}\n', 'Removal deleted progress'
    assert not (prefix / 'share/physical-app').exists()
    assert not (prefix / 'bin/physical').exists()
    # A damaged runtime must be rejected before an installer mutates its target.
    with (standalone / 'bin/physical').open('ab') as binary:
        binary.write(b'corruption')
    rejected = subprocess.run([str(standalone / 'install')], env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    assert rejected.returncode and not (prefix / 'bin/physical').exists(), 'Damaged bundle was installed'
print('PASS: two archives, checksums, portable module paths, compiler-free plugin install/reinstall, delayed discovery and timeout, standalone install/launch/removal, retained progress and corruption rejection')
