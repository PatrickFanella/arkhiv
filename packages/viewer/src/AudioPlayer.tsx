// Continuous recording controls adapted from the pinned GPL Dedhed player.
import { useRef } from 'react';
import { useAudioControls } from './useAudioControls.js';
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
 let start = 0, end = 0, url = '';
 try {
  url = audioURL(asset);
  [start,end] = setBounds(asset.recording,set);
  if (station) { start = 0; end = Math.min(end,station.duration); radioOffset(station.offset,station.receivedAt,station.latency,end,Date.now()); }
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) throw new Error('Invalid audio bounds');
 } catch { url = ''; }
 const {position,playing,pending,error,volume,seek,toggle,changeVolume,events,setPending,setError}=useAudioControls({audio,start,end,onEnded});
 function metadataReady() {
  const offset = station ? radioOffset(station.offset,station.receivedAt,station.latency,end,Date.now()) : initialOffset ?? start;
  seek(offset); setPending(false);
  if (autoPlay && audio.current) void audio.current.play().catch(() => setError('Playback did not start. Press Play to try again.'));
 }
 if (!url) return <p role="alert">This recording is unavailable.</p>;
 const track = currentTrack(asset.recording,position);
 return <section className="audio-player" aria-label="Audio player" aria-busy={pending}>
  <audio ref={audio} src={url} preload="metadata" onLoadedMetadata={metadataReady} {...events} />
  <p>{station ? 'Shared station' : set?.title || 'Full recording'} · {asset.recording.title}</p>
  <button onClick={() => {void toggle();}}>{playing ? 'Pause' : 'Play'}</button>
  {station && onRejoin && <button onClick={onRejoin}>Rejoin current program</button>}
  <label>Recording position <input type="range" min={start} max={end} step="0.1" value={Math.max(start,Math.min(end,position))} onChange={event => seek(Number(event.target.value))} /></label>
  <p><time>{clock(position)}</time> / <time>{clock(end)}</time> · {track?.title || 'Track information unavailable'}</p>
  <label>Volume <input type="range" min={0} max={1} step="0.05" value={volume} onChange={event => changeVolume(Number(event.target.value))} /></label>
  {pending && <p role="status">Loading audio…</p>}{error && <p role="alert">{error}</p>}
  <ol className="tracks">{asset.recording.tracks.map(t => <li key={t.position}><button disabled={t.start < start || t.start >= end} aria-current={track?.position === t.position ? 'true' : undefined} onClick={() => seek(t.start)}>{t.title || `Track ${t.position+1}`}</button> <time>{clock(t.duration)}</time></li>)}</ol>
  <dl>{[['Source',asset.recording.source],['Taper',asset.recording.taper],['Transfer',asset.recording.transferer]].filter(([,value])=>value).map(([label,value])=><div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl>
  {(asset.recording.warnings || []).map((warning,index)=><p key={index}>{warning}</p>)}
  <a href={asset.source_url} target="_blank" rel="noreferrer">Source recording at Internet Archive</a>
 </section>;
}
