#!/usr/bin/env python3
"""Prepare Quickshell's native module path after a plugin folder is moved."""
import os
import pathlib
import sys
import tempfile

folder = pathlib.Path(sys.argv[1]).resolve()
if not (folder / 'libphysicalrenderer.so').is_file():
    raise SystemExit('Physical runtime is missing; build it or install the prebuilt bundle.')
module = folder / 'qmldir'
definition = f'plugin physicalrenderer {folder}\n'
if module.is_file() and module.read_text() == definition:
    raise SystemExit(0)
with tempfile.NamedTemporaryFile(mode='w', dir=folder, prefix='.qmldir.', delete=False) as output:
    output.write(definition)
    temporary = pathlib.Path(output.name)
try:
    os.replace(temporary, module)
finally:
    temporary.unlink(missing_ok=True)
