#!/usr/bin/env python3
"""Exercise real output recovery with silent test sinks; keep the user's route."""
import pathlib
import queue
import shlex
import subprocess
import tempfile
import threading
import time
import uuid

project=pathlib.Path(__file__).resolve().parents[1]
def pactl(*args):
    return subprocess.check_output(['pactl',*args],text=True,timeout=5).strip()
original=pactl('get-default-sink')
modules=[]
removed=set()
run=None
with tempfile.TemporaryDirectory(prefix='physical-audio-output-') as directory:
    binary=pathlib.Path(directory)/'audio-output-test'
    flags=shlex.split(subprocess.check_output(['pkg-config','--cflags','--libs','Qt6Core','Qt6Multimedia'],text=True))
    subprocess.run(['g++','-O2','-std=c++17','-fPIC',str(project/'tests/audio_output_test.cpp'),'-o',str(binary)]+flags,check=True)
    token=uuid.uuid4().hex[:10]
    descriptions=['PhysicalAudioTestA'+token,'PhysicalAudioTestB'+token]
    try:
        for i,description in enumerate(descriptions):
            module=pactl('load-module','module-null-sink','sink_name=physical_audio_test_'+token+str(i),'sink_properties=device.description='+description)
            modules.append(module)
        if pactl('get-default-sink')!=original:
            raise RuntimeError('Adding test outputs changed the selected output')
        run=subprocess.Popen([str(binary),*descriptions,str(project/'sounds')],text=True,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
        messages=queue.Queue()
        def read():
            for line in run.stdout:messages.put(line)
            messages.put(None)
        threading.Thread(target=read,daemon=True).start()
        passed=False
        deadline=time.monotonic()+30
        while time.monotonic()<deadline:
            try:line=messages.get(timeout=.25)
            except queue.Empty:continue
            if line is None:break
            print(line.strip(),flush=True)
            if 'AUDIO OUTPUT REMOVE FIRST' in line:
                pactl('unload-module',modules[0]);removed.add(modules[0])
            if 'AUDIO OUTPUT PASS:' in line:passed=True
        if not passed:raise RuntimeError('Output recovery failed or timed out')
        run.wait(timeout=5)
        if run.returncode:raise RuntimeError('Output test exited with '+str(run.returncode))
    finally:
        if run and run.poll() is None:
            run.terminate()
            try:run.wait(timeout=3)
            except subprocess.TimeoutExpired:run.kill();run.wait()
        for module in reversed(modules):
            if module not in removed:
                subprocess.run(['pactl','unload-module',module],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=5)
        if pactl('get-default-sink')!=original:
            pactl('set-default-sink',original)
            print('Restored the original default output.',flush=True)
