// Continuous recording controls adapted from the pinned GPL Dedhed player.
import { useEffect, useRef, useState } from 'react';
import { useAudioControls } from './useAudioControls.js';
import { channelSeekTarget } from './channelSync.js';
import { audioURL, audioPlaybackState, clock, currentTrack, radioOffset, setBounds, type AudioAsset, type SetList } from './audioModel.js';

export type AudioPlayerProps = {
 asset: AudioAsset;
 set?: SetList;
 initialOffset?: number;
 autoPlay?: boolean;
 station?: { offset: number; receivedAt: number; latency: number; duration: number; programKey: string };
 onRejoin?: () => void;
 onEnded?: () => void;
 onPlaybackState?: (state: string) => void;
};

export function AudioPlayer(props: AudioPlayerProps) {
 const key = `${props.asset.url}:${props.station?.programKey || 'ondemand'}:${props.set?.first ?? ''}:${props.set?.last ?? ''}`;
 return <AudioSession key={key} {...props} />;
}
function AudioSession({asset,set,initialOffset,autoPlay,station,onRejoin,onEnded,onPlaybackState}: AudioPlayerProps) {
 const audio = useRef<HTMLAudioElement>(null);
 let start = 0, end = 0, url = '';
 try {
  url = audioURL(asset);
  [start,end] = setBounds(asset.recording,set);
  if (station) { start = 0; end = Math.min(end,station.duration); radioOffset(station.offset,station.receivedAt,station.latency,end,Date.now()); }
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) throw new Error('Invalid audio bounds');
 } catch { url = ''; }
 const {position,playing,pending,error,volume,muted,seek,toggle,changeVolume,changeMuted,events,setPending,setError}=useAudioControls({audio,start,end,onEnded});
 const [playbackState,setPlaybackState]=useState('loading');
 useEffect(()=>{
  const element=audio.current;if(!element)return;
  const update=()=>{
   if(station && !element.paused && !element.ended){
    const target=channelSeekTarget(element,radioOffset(station.offset,station.receivedAt,station.latency,end,Date.now()));
    if(target!==null)element.currentTime=target;
   }
   const state=error ? 'unavailable' : audioPlaybackState(element,station);setPlaybackState(state);onPlaybackState?.(state);
  };
  const names=['playing','pause','waiting','seeked','loadedmetadata','ended'];names.forEach(name=>element.addEventListener(name,update));
  const timer=setInterval(update,1000);
  return()=>{clearInterval(timer);names.forEach(name=>element.removeEventListener(name,update));};
  onPlaybackState?.('loading');
 },[station,end,onPlaybackState,error]);
 function metadataReady() {
  const offset = station ? radioOffset(station.offset,station.receivedAt,station.latency,end,Date.now()) : initialOffset ?? start;
  seek(offset); setPending(false);
  if (autoPlay && audio.current) void audio.current.play().catch(() => setError('Playback did not start. Press Play to try again.'));
 }
 if (!url) return <p role="alert">This recording is unavailable.</p>;
 const track = currentTrack(asset.recording,position);
 return <section className="audio-player" aria-label="Audio player" aria-busy={pending}>
  <audio ref={audio} crossOrigin="anonymous" src={url} preload="metadata" onLoadedMetadata={metadataReady} {...events} />
  <p role="status">{station ? playbackState==='synchronized' ? 'On the station · Scheduled replay' : playbackState==='paused' ? 'Paused · Station continues' : `Scheduled replay · ${error ? 'blocked or unavailable' : playbackState}` : `On demand · ${set?.title || 'Full recording'}`}</p><p>{asset.recording.title} · Original date: <time>{asset.recording.date || 'Unknown'}</time></p>
  <button disabled={Boolean(station && !playing && !onRejoin)} onClick={() => {if(station && !playing)onRejoin?.();else void toggle();}}>{playing ? 'Pause' : station ? 'Resume at current program' : 'Play'}</button>
  {station && onRejoin && <button onClick={onRejoin}>Rejoin current program</button>}
  {!station && <label>Recording position <input type="range" min={start} max={end} step="0.1" value={Math.max(start,Math.min(end,position))} onChange={event => seek(Number(event.target.value))} /></label>}
  <p><time>{clock(position)}</time> / <time>{clock(end)}</time> · {track?.title || 'Track information unavailable'}</p>
  <label>Volume <input type="range" min={0} max={1} step="0.05" value={volume} onChange={event => changeVolume(Number(event.target.value))} /></label>
  <button aria-pressed={muted} onClick={()=>changeMuted(!muted)}>{muted ? 'Unmute' : 'Mute'}</button>
  {pending && <p role="status">Loading audio…</p>}{error && <p role="alert">{error}</p>}
  <ol className="tracks">{asset.recording.tracks.map(t => <li key={t.position}><button disabled={Boolean(station) || t.start < start || t.start >= end} aria-current={track?.position === t.position ? 'true' : undefined} onClick={() => seek(t.start)}>{t.title || `Track ${t.position+1}`}</button> <time>{clock(t.duration)}</time></li>)}</ol>
  <dl>{[['Artist',asset.recording.artist],['Venue',asset.recording.venue],['Location',asset.recording.location],['Source',asset.recording.source],['Taper',asset.recording.taper],['Transfer',asset.recording.transferer]].filter(([,value])=>value).map(([label,value])=><div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl>
  {(asset.recording.warnings || []).map((warning,index)=><p key={index}>{warning}</p>)}
  <a href={asset.source_url} target="_blank" rel="noreferrer">Source recording at Internet Archive</a>
 </section>;
}
