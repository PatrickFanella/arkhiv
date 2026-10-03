#!/usr/bin/env python3
"""Observe an owner-authorized browser's already-started real episode without storing video."""
import argparse
from datetime import datetime, timezone
import json
from pathlib import Path
import subprocess
import time
from pilot import ROOT, REPO

EXPRESSION = """() => { const v=document.querySelector('video'); if(!v)return null; const q=v.getVideoPlaybackQuality(); return {src:v.currentSrc,time:v.currentTime,duration:v.duration,paused:v.paused,ended:v.ended,ready:v.readyState,error:v.error?.message,frames:q.totalVideoFrames,dropped:q.droppedVideoFrames,audio:v.webkitAudioDecodedByteCount,width:v.videoWidth,height:v.videoHeight,rate:v.playbackRate,visible:document.visibilityState}; }"""

def main():
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--session',required=True);parser.add_argument('--source',required=True);parser.add_argument('--started-at',required=True);args=parser.parse_args()
    wrapper=Path.home()/'.codex/skills/playwright/scripts/playwright_cli.sh'
    samples=[];deadline=time.monotonic()+75*60
    path=ROOT/'evidence/full-episode-observation.json'
    while time.monotonic()<deadline:
        result=subprocess.check_output([str(wrapper),'-s='+args.session,'eval',EXPRESSION,'--raw','--json'],cwd=REPO,text=True,timeout=30)
        sample=json.loads(json.loads(result)['result'])
        if not sample or sample['src']!=args.source:raise RuntimeError('Observed player changed or disappeared; no continuous qualification claimed')
        sample['observed_at']=datetime.now(timezone.utc).isoformat();samples.append(sample)
        completed=sample['ended'] and sample['time']>=sample['duration']-.25
        receipt={'session':args.session,'source':args.source,'started_at':args.started_at,'samples':samples,'completed':completed,'scope':'actual browser at normal speed; decoded picture/audio and continuity, no human listening claim; no source video stored'}
        temp=path.with_suffix('.tmp');temp.write_text(json.dumps(receipt,indent=2)+'\n');temp.chmod(0o600);temp.replace(path)
        print(json.dumps(sample),flush=True)
        if sample.get('error') or sample['rate']!=1:raise RuntimeError('Playback error or altered speed; qualification failed')
        if completed:
            if sample['frames']<sample['duration']*10 or not sample['audio']:raise RuntimeError('Insufficient decoded audio/video evidence')
            print('Full real source episode reached natural end; perceptual listening remains separate',flush=True);return
        time.sleep(30)
    raise RuntimeError('Bounded full-episode observation expired; partial evidence retained')

if __name__=='__main__':main()
