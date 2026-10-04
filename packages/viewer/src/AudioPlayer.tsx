// Continuous recording controls adapted from the pinned GPL Dedhed player.
import { useEffect, useRef, useState } from 'react';
import { audioURL, clock, currentTrack, radioOffset, setBounds, type AudioAsset, type SetList } from './audioModel.js';

export type AudioPlayerProps = {
 asset: AudioAsset;
 set?: SetList;
 initialOffset?: number;
 autoPlay?: boolean;
 station?: { offset: number; receivedAt: number; latency: number; duration: number; programKey: string };
 onRejoin?: () => void;
 onEnded?: () => void;
};

export function AudioPlayer(props: AudioPlayerProps) {
 const key = `${props.asset.url}:${props.station?.programKey || 'ondemand'}:${props.set?.first ?? ''}:${props.set?.last ?? ''}`;
 return <AudioSession key={key} {...props} />;
}
function AudioSession({asset,set,initialOffset,autoPlay,station,onRejoin,onEnded}: AudioPlayerProps) {
 const audio = useRef<HTMLAudioElement>(null);
 const ended = useRef(false);
 const [position,setPosition] = useState(0);
 const [playing,setPlaying] = useState(false);
 const [pending,setPending] = useState(true);
 const [error,setError] = useState('');
 const [volume,setVolume] = useState(1);
 let start = 0, end = 0, url = '';
 try {
  url = audioURL(asset);
  [start,end] = setBounds(asset.recording,set);
  if (station) { start = 0; end = Math.min(end,station.duration); radioOffset(station.offset,station.receivedAt,station.latency,end,Date.now()); }
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) throw new Error('Invalid audio bounds');
 } catch { url = ''; }
 useEffect(() => {
  const element = audio.current;
  return () => { element?.pause(); };
 }, []);
 async function toggle() {
  const a = audio.current; if (!a || !url) return;
  if (!a.paused) { a.pause(); return; }
  if (a.ended || ended.current || a.currentTime >= end) { a.currentTime = start; ended.current=false; }
  setError('');
  try { await a.play(); } catch { setPending(false); setError('Playback did not start. Press Play to try again.'); }
 }
 function seek(value: number) {
  const a = audio.current;
  if (!a || !Number.isFinite(value)) return;
  try { a.currentTime = Math.max(start,Math.min(end,value)); setPosition(a.currentTime); if(a.currentTime<end)ended.current=false; }
  catch { setError('This audio position is unavailable.'); }
 }
 function metadataReady() {
  const offset = station ? radioOffset(station.offset,station.receivedAt,station.latency,end,Date.now()) : initialOffset ?? start;
  seek(offset); setPending(false);
  if (autoPlay && audio.current) void audio.current.play().catch(() => setError('Playback did not start. Press Play to try again.'));
 }
 function timeUpdate() {
  const a = audio.current; if (!a) return;
  if (a.currentTime >= end) { a.pause(); if (a.currentTime !== end) a.currentTime = end; setPosition(end); finish(); }
  else setPosition(a.currentTime);
 }
 function finish() { if (!ended.current) { ended.current=true;onEnded?.(); } }
 if (!url) return <p role="alert">This recording is unavailable.</p>;
 const track = currentTrack(asset.recording,position);
 return <section className="audio-player" aria-label="Audio player" aria-busy={pending}>
  <audio ref={audio} src={url} preload="metadata" onLoadedMetadata={metadataReady} onTimeUpdate={timeUpdate}
   onPlaying={() => {setPlaying(true);setPending(false);setError('');}} onPause={() => setPlaying(false)} onWaiting={() => setPending(true)}
   onError={() => {setPending(false);setPlaying(false);setError('Audio source unavailable. Retry or choose another recording.');}}
   onEnded={finish} />
  <p>{station ? 'Shared station' : set?.title || 'Full recording'} · {asset.recording.title}</p>
  <button onClick={() => {void toggle();}}>{playing ? 'Pause' : 'Play'}</button>
  {station && onRejoin && <button onClick={onRejoin}>Rejoin current program</button>}
  <label>Recording position <input type="range" min={start} max={end} step="0.1" value={Math.max(start,Math.min(end,position))} onChange={event => seek(Number(event.target.value))} /></label>
  <p><time>{clock(position)}</time> / <time>{clock(end)}</time> · {track?.title || 'Track information unavailable'}</p>
  <label>Volume <input type="range" min={0} max={1} step="0.05" value={volume} onChange={event => { const value=Number(event.target.value);setVolume(value);if(audio.current)audio.current.volume=value; }} /></label>
  {pending && <p role="status">Loading audio…</p>}{error && <p role="alert">{error}</p>}
  <ol className="tracks">{asset.recording.tracks.map(t => <li key={t.position}><button disabled={t.start < start || t.start >= end} aria-current={track?.position === t.position ? 'true' : undefined} onClick={() => seek(t.start)}>{t.title || `Track ${t.position+1}`}</button> <time>{clock(t.duration)}</time></li>)}</ol>
  <dl>{[['Source',asset.recording.source],['Taper',asset.recording.taper],['Transfer',asset.recording.transferer]].filter(([,value])=>value).map(([label,value])=><div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl>
  {(asset.recording.warnings || []).map((warning,index)=><p key={index}>{warning}</p>)}
  <a href={asset.source_url} target="_blank" rel="noreferrer">Source recording at Internet Archive</a>
 </section>;
}
