#!/usr/bin/env python3
"""Verify the bank consumed by Qt; optionally verify the original source hashes."""
import hashlib, json, pathlib, sys, wave
project=pathlib.Path(__file__).resolve().parents[1]
manifest=json.loads((project/'sounds/samples.json').read_text())
assert len(manifest)==33
names={entry['output'] for entry in manifest}
assert {f'brick{i}.wav' for i in range(21)} <= names
assert len([name for name in names if name.startswith('feature-')])==12
source=pathlib.Path(sys.argv[1]) if len(sys.argv)>1 else None
for entry in manifest:
    path=project/'sounds'/entry['output']
    assert hashlib.sha256(path.read_bytes()).hexdigest()==entry['output_sha256'], path
    with wave.open(str(path)) as clip:
        assert clip.getnchannels()==2 and clip.getsampwidth()==2 and clip.getframerate()==48000 and clip.getnframes()>0, path
    if source:
        original=source/entry['source']
        assert hashlib.sha256(original.read_bytes()).hexdigest()==entry['source_sha256'], original
print('PASS: 21 brick / 12 feature runtime samples, stereo PCM and asset hashes'+(' including original source' if source else ''))
