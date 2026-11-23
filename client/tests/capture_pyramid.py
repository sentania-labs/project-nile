#!/usr/bin/env python3
"""Record actual Godot placement and a deterministic pyramid construction time-lapse.

The prepared fixture has an occupied workforce and treasury; all 240 stone
blocks are subsequently purchased, delivered and installed by the simulation.
"""
import argparse
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time

ROOT=Path(__file__).resolve().parents[2]

def record(args):
    output=args.output.resolve();output.mkdir(parents=True,exist_ok=True)
    subprocess.run(['go','test','./internal/sandbox','-run','TestPyramidPhysicalGuildsStoneRampsAndReplay','-count=1'],cwd=ROOT,env=dict(os.environ,NILE_MONUMENT_FIXTURE_DIR=str(output)),check=True)
    binary=output/'nile-sim'
    subprocess.run(['go','build','-o',str(binary),'./cmd/nile-sim'],cwd=ROOT,env=dict(os.environ,CGO_ENABLED='0'),check=True)
    with tempfile.TemporaryDirectory(prefix='nile-pyramid-') as temporary:
        temp=Path(temporary);save=temp/'city.nile';shutil.copyfile(output/'prepared.nile',save)
        ready=temp/'ready'
        env=dict(os.environ,NILE_SIM=str(binary),NILE_SANDBOX_SAVE=str(save),NILE_PYRAMID_DIR=str(output),NILE_PYRAMID_READY=str(ready))
        with (output/'godot.log').open('w') as log:
            game=subprocess.Popen([args.godot,'--audio-driver','Dummy','--path',str(ROOT/'client'),'--script','res://tests/pyramid_capture.gd'],env=env,stdout=log,stderr=subprocess.STDOUT)
            recorder=None
            try:
                deadline=time.monotonic()+30
                while not ready.exists() and game.poll() is None and time.monotonic()<deadline:time.sleep(.1)
                if not ready.exists():raise RuntimeError('Godot pyramid placement failed; see godot.log')
                recorder=subprocess.Popen(['ffmpeg','-y','-loglevel','error','-f','x11grab','-framerate','20','-video_size','1920x1080','-i',os.environ['DISPLAY'],'-c:v','libx264','-preset','veryfast','-crf','20','-pix_fmt','yuv420p',str(output/'pyramid-construction.mp4')],stdin=subprocess.DEVNULL)
                if game.wait(timeout=180):raise RuntimeError('Pyramid playthrough failed; see godot.log')
            finally:
                if game.poll() is None:game.terminate();game.wait(timeout=10)
                if recorder is not None and recorder.poll() is None:recorder.send_signal(signal.SIGINT);recorder.wait(timeout=15)
        text=(output/'godot.log').read_text()
        if 'NILE_PYRAMID_PASS' not in text or 'ERROR:' in text:raise RuntimeError(text)
        print(text);print(output/'pyramid-construction.mp4')

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--godot',default='godot')
    parser.add_argument('--output',type=Path,default=Path('/mnt/holding/project-nile/pyramid-review'))
    parser.add_argument('--virtual-display',action='store_true',help=argparse.SUPPRESS)
    args=parser.parse_args()
    if args.virtual_display:record(args)
    else:subprocess.run(['xvfb-run','-a','-s','-screen 0 1920x1080x24',sys.executable,str(Path(__file__).resolve()),*sys.argv[1:],'--virtual-display'],check=True)
