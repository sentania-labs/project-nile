#!/usr/bin/env python3
"""Capture Sun Temple and Mausoleum built through its physical supply chain in Godot."""
import argparse,os,shutil,signal,subprocess,sys,tempfile,time
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2]
p=argparse.ArgumentParser(description=__doc__);p.add_argument('--godot',default='godot');p.add_argument('--display',action='store_true');p.add_argument('--kind',choices=['suntemple','smallmausoleum'],default='suntemple');p.add_argument('--output',type=Path,default=Path('/mnt/holding/project-nile/temple-review'));a=p.parse_args()
if not a.display:
 subprocess.run(['xvfb-run','-a','-s','-screen 0 1920x1080x24',sys.executable,__file__,*sys.argv[1:],'--display'],check=True);sys.exit()
out=a.output.resolve();out.mkdir(parents=True,exist_ok=True)
subprocess.run(['go','test','./internal/sandbox','-run','TestTempleMonumentsPhysicalSandstoneAndGuilds/'+a.kind+'','-count=1'],cwd=ROOT,env=dict(os.environ,NILE_TEMPLE_MONUMENT_DIR=str(out)),check=True)
binary=out/'nile-sim';subprocess.run(['go','build','-o',str(binary),'./cmd/nile-sim'],cwd=ROOT,env=dict(os.environ,CGO_ENABLED='0'),check=True)
with tempfile.TemporaryDirectory(prefix='nile-monument-families-') as temp:
 save=Path(temp)/'city.nile';shutil.copyfile(out/(a.kind+'-prepared.nile'),save);ready=Path(temp)/'ready'
 env=dict(os.environ,NILE_SIM=str(binary),NILE_SANDBOX_SAVE=str(save),NILE_TEMPLE_MONUMENT_DIR=str(out),NILE_MONUMENT_READY=str(ready),NILE_TEMPLE_KIND=a.kind)
 with (out/'godot.log').open('w') as log:
  game=subprocess.Popen([a.godot,'--audio-driver','Dummy','--path',str(ROOT/'client'),'--script','res://tests/temple_monument_capture.gd'],cwd=ROOT,env=env,stdout=log,stderr=subprocess.STDOUT);video=None
  try:
   deadline=time.monotonic()+30
   while not ready.exists() and game.poll() is None and time.monotonic()<deadline:time.sleep(.1)
   if not ready.exists():raise RuntimeError('Godot failed initial placement; inspect godot.log')
   video=subprocess.Popen(['ffmpeg','-y','-loglevel','error','-f','x11grab','-framerate','20','-video_size','1920x1080','-i',os.environ['DISPLAY'],'-c:v','libx264','-preset','veryfast','-crf','20','-pix_fmt','yuv420p',str(out/(a.kind+'.mp4'))],stdin=subprocess.DEVNULL)
   if game.wait(timeout=180):raise RuntimeError('Godot monument flow failed; inspect godot.log')
  finally:
   if game.poll() is None:game.terminate();game.wait(timeout=10)
   if video and video.poll() is None:video.send_signal(signal.SIGINT);video.wait(timeout=15)
 text=(out/'godot.log').read_text()
 if 'NILE_TEMPLE_MONUMENT_PASS' not in text or 'ERROR:' in text:raise RuntimeError(text)
 print(text)
 print(out/(a.kind+'.mp4'))
