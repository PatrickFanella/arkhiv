#!/usr/bin/env python3
"""Measure two already-tuned private browser sessions against the server clock."""
import argparse
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
import json
from pathlib import Path
import subprocess
import time
from pilot import ROOT, REPO

EXPRESSION = """async () => {const began=performance.now();const n=await fetch('/api/channel').then(r=>r.json());delete n.episode;if(n.playback)delete n.playback.file.raw;const v=document.querySelector('video');return {server:n,latency:(performance.now()-began)/2000,video:v?{src:v.currentSrc,time:v.currentTime,paused:v.paused,ended:v.ended,ready:v.readyState,error:v.error?.message,frames:v.getVideoPlaybackQuality().totalVideoFrames,audio:v.webkitAudioDecodedByteCount}:null};}"""

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--sessions',nargs=2,required=True)
    parser.add_argument('--seconds',type=int,default=75)
    parser.add_argument('--receipt',required=True)
    args=parser.parse_args()
    if not 5<=args.seconds<=300 or Path(args.receipt).name!=args.receipt: raise RuntimeError('Bounded observation and receipt basename required')
    wrapper=Path.home()/'.codex/skills/playwright/scripts/playwright_cli.sh'
    path=ROOT/'evidence'/args.receipt
    samples=[];deadline=time.monotonic()+args.seconds
    def observe(session):
        output=subprocess.check_output([str(wrapper),'-s='+session,'eval',EXPRESSION,'--raw','--json'],cwd=REPO,text=True,timeout=30)
        result=json.loads(json.loads(output)['result']);result['session']=session
        video=result['video'];server=result['server']
        result['drift_seconds']=None if not video else video['time']-server['offset']-result['latency']
        return result
    with ThreadPoolExecutor(max_workers=2) as pool:
        while time.monotonic()<deadline:
            pair=list(pool.map(observe,args.sessions))
            samples.append({'observed_at':datetime.now(timezone.utc).isoformat(),'sessions':pair})
            temporary=path.with_suffix('.tmp');temporary.write_text(json.dumps({'samples':samples,'scope':'two independent browser contexts; per-session server clock, decoded frames/audio, transition identity and drift'},indent=2)+'\n');temporary.chmod(0o600);temporary.replace(path)
            print(json.dumps(samples[-1]),flush=True)
            time.sleep(5)
    for sample in samples:
        for result in sample['sessions']:
            video=result['video']
            if not video or video.get('error'): raise RuntimeError('Missing or failed player; evidence retained')
    print('Observation saved; assess startup and boundary samples separately from steady-state drift')

if __name__=='__main__':main()
