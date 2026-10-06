import { useEffect, useRef, useState, type RefObject } from 'react';
import { useMediaPreferences } from './videoPreferences.js';

// Brands own markup and access-resolved sources; shared controls own bounds,
// local transport state, teardown and one completion notification per playback.
export function useAudioControls({audio,start,end,onEnded,initialVolume=1}: {
 audio:RefObject<HTMLAudioElement|null>;start:number;end:number;onEnded?:()=>void;initialVolume?:number;
}) {
 const finished=useRef(false);
 const [position,setPosition]=useState(0),[playing,setPlaying]=useState(false);
 const [pending,setPending]=useState(true),[error,setError]=useState('');
 const {volume,muted}=useMediaPreferences(audio,{volume:Number.isFinite(initialVolume) ? Math.max(0,Math.min(1,initialVolume)) : 1,muted:false});
 useEffect(()=>{const element=audio.current;return()=>{element?.pause();};},[audio]);
 function reset(offset=0){finished.current=false;setPosition(offset);setPending(true);setError('');setPlaying(false);}
 function seek(value:number){
  const element=audio.current;
  if(!element||!Number.isFinite(value)||!Number.isFinite(start)||!Number.isFinite(end)||end<=start)return;
  try{element.currentTime=Math.max(start,Math.min(end,value));setPosition(element.currentTime);if(element.currentTime<end)finished.current=false;}
  catch{setError('This audio position is unavailable.');}
 }
 async function toggle(){
  const element=audio.current;if(!element)return;
  if(!element.paused){element.pause();return;}
  if(element.ended||finished.current||element.currentTime>=end){seek(start);finished.current=false;}
  setError('');try{await element.play();}catch{setPending(false);setError('Playback did not start. Press Play to try again.');}
 }
 function finish(){if(!finished.current){finished.current=true;onEnded?.();}}
 function timeUpdate(){
  const element=audio.current;if(!element)return;
  if(element.currentTime>=end){element.pause();if(element.currentTime!==end)element.currentTime=end;setPosition(end);finish();}
  else setPosition(element.currentTime);
 }
 function changeVolume(value:number){if(!Number.isFinite(value))return;if(audio.current)audio.current.volume=Math.max(0,Math.min(1,value));}
 function changeMuted(value:boolean){if(audio.current)audio.current.muted=value;}
 const events={onTimeUpdate:timeUpdate,onEnded:finish,
  onPlaying:()=>{setPlaying(true);setPending(false);setError('');},onPause:()=>setPlaying(false),onWaiting:()=>setPending(true),
  onError:()=>{setPending(false);setPlaying(false);setError('Audio source unavailable. Retry or choose another recording.');}};
 return {position,playing,pending,error,volume,muted,seek,toggle,reset,changeVolume,changeMuted,events,setPending,setError,setPlaying};
}
