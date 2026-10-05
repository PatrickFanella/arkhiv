import { channelSeekTarget } from './channelSync.js';
import { useEffect, useRef, useState } from 'react';
import { artworkURL, captionURL, playbackURL, targetOffset, type Session } from './model.js';
import { useVideoPreferences } from './videoPreferences.js';
import { videoState, videoStateLabel, type VideoState } from './videoState.js';

export function Player({ session, paused, onPaused, onRejoin, onPlaybackState }: { session: Session; paused: boolean; onPaused: (value: boolean) => void; onRejoin?: () => void; onPlaybackState?: (state: VideoState) => void }) {
  const video = useRef<HTMLVideoElement>(null);
  const container = useRef<HTMLDivElement>(null);
  const preferences = useVideoPreferences(video);
  const latest = useRef(session);
  const active = useRef(true);
  const [error, setError] = useState('');
  const [controlError, setControlError] = useState('');
  const [blocked, setBlocked] = useState(false);
  const [loading, setLoading] = useState(true);
  const [observedState, setObservedState] = useState<VideoState>('loading');
  const [position, setPosition] = useState(0);
  const [posterFailed, setPosterFailed] = useState(false);
  const notify = useRef(onPlaybackState);
  useEffect(() => { notify.current = onPlaybackState; }, [onPlaybackState]);
  let poster: string | undefined;
  try { poster = artworkURL(session.episode); } catch { /* No invalid poster request. */ }
  useEffect(() => {
    if (!poster) return;
    const image = new Image();
    image.referrerPolicy = 'no-referrer';
    image.onerror = () => setPosterFailed(true);
    image.src = poster;
    return () => { image.onerror = null; };
  }, [poster]);
  useEffect(() => {
    const element = video.current;
    if (!element) return;
    const sample = () => { setPosition(element.currentTime); setObservedState(videoState(element, latest.current, performance.now())); };
    const timer = window.setInterval(sample, 500);
    return () => clearInterval(timer);
  }, []);
  const state: VideoState = error ? 'unavailable' : blocked ? 'blocked' : paused ? 'paused' : loading ? 'buffering' : observedState;
  useEffect(() => { notify.current?.(state); }, [state]);
  const [captionSetup] = useState(() => {
    try { return { tracks: (session.playback.captions || []).map(track => ({ ...track, url: captionURL(track, session.playback) })), error: '' }; }
    catch { return { tracks: [], error: 'The source caption reference is invalid.' }; }
  });
  const captions = captionSetup.tracks;
  const [captionError, setCaptionError] = useState(captionSetup.error);
  const [selectedCaption, setSelectedCaption] = useState('');
  const [captionURLs, setCaptionURLs] = useState<Record<string, string>>({});
  const captionBlobs = useRef<string[]>([]);
  useEffect(() => () => { captionBlobs.current.forEach(url => URL.revokeObjectURL(url)); }, []);
  useEffect(() => {
    if (!selectedCaption || captionURLs[selectedCaption]) return;
    const chosen = captions.find(track => track.file === selectedCaption);
    if (!chosen) return;
    const controller = new AbortController();
    void (async () => {
      try {
        const response = await fetch(chosen.url, { signal: AbortSignal.any([controller.signal, AbortSignal.timeout(10_000)]), credentials: 'omit', cache: 'no-store' });
        if (!response.ok || !response.body) throw new Error('Caption source unavailable');
        const reader = response.body.getReader();
        const parts: Uint8Array<ArrayBuffer>[] = [];
        let bytes = 0;
        try {
          for (;;) {
            const chunk = await reader.read();
            if (chunk.done) break;
            bytes += chunk.value.byteLength;
            if (bytes > 2 * 1024 * 1024) throw new Error('Caption source exceeds limit');
            parts.push(new Uint8Array(chunk.value));
          }
        } finally { await reader.cancel(); }
        if (controller.signal.aborted) return;
        const blob = new Blob(parts, { type: 'text/vtt' });
        if (!((await blob.text()).replace(/^\uFEFF/, '')).startsWith('WEBVTT')) throw new Error('Invalid WebVTT');
        if (controller.signal.aborted) return;
        const url = URL.createObjectURL(blob); captionBlobs.current.push(url);
        setCaptionURLs(previous => ({ ...previous, [chosen.file]: url }));
      } catch {
        if (!controller.signal.aborted) {
          Array.from(video.current?.textTracks || []).forEach(track => { track.mode = 'disabled'; });
          setCaptionError('Captions could not be loaded from their source.'); setSelectedCaption('');
        }
      }
    })();
    return () => controller.abort();
  }, [captions, captionURLs, selectedCaption]);
  useEffect(() => {
    const element = video.current;
    if (!element) return;
    const changed = () => {
      const index = Array.from(element.textTracks).findIndex(track => track.mode === 'showing');
      setSelectedCaption(captions[index]?.file || '');
    };
    element.textTracks.addEventListener('change', changed);
    return () => element.textTracks.removeEventListener('change', changed);
  }, [captions]);
  useEffect(() => { latest.current = session; }, [session]);
  useEffect(() => {
    const element = video.current;
    if (!element) return;
    active.current = true;
    const seek = () => { if (element.readyState > 0) element.currentTime = targetOffset(latest.current, performance.now()); };
    const loaded = () => { setLoading(false); seek(); if (!paused) void element.play().catch(() => setBlocked(true)); };
    const failure = () => { setLoading(false); setError('This video could not be loaded from its source. Try again or open the source page.'); };
    element.addEventListener('loadedmetadata', loaded);
    element.addEventListener('error', failure);
    try { element.src = playbackURL(session.playback); element.load(); } catch (reason) { setError(reason instanceof Error ? reason.message : 'Invalid playback source.'); }
    return () => { active.current = false; element.removeEventListener('loadedmetadata', loaded); element.removeEventListener('error', failure); element.removeAttribute('src'); element.load(); };
    // A keyed Player owns one asset; clock updates use latest, without reloading media.
  }, []);
  useEffect(() => {
    const element = video.current;
    if (session.mode === 'channel' && element && element.readyState > 0 && !paused) {
      const expected = targetOffset(session, performance.now());
      const correction = channelSeekTarget(element, expected);
      if (correction !== null) element.currentTime = correction;
    }
  }, [session, paused]);
  const rejoin = () => {
    if (onRejoin) { onRejoin(); return; }
    const element = video.current;
    if (!element) return;
    if (element.readyState > 0) element.currentTime = targetOffset(latest.current, performance.now());
    void element.play().then(() => setBlocked(false)).catch(() => setBlocked(true));
  };
  const clock = (seconds: number) => `${Math.floor(Math.max(0, seconds) / 60)}:${String(Math.floor(Math.max(0, seconds) % 60)).padStart(2, '0')}`;
  return <div className="player" ref={container}>
    <p className="viewing-state" role="status">{videoStateLabel(session.mode, state)}</p>
    <video ref={video} controls={session.mode === 'ondemand'} poster={posterFailed ? undefined : poster} playsInline preload="metadata" aria-label={session.episode.title} onPause={() => { if (active.current && video.current?.readyState && !video.current.ended) onPaused(true); }} onPlay={() => { setBlocked(false); onPaused(false); }} onWaiting={() => setLoading(true)} onPlaying={() => setLoading(false)}>
      {captions.map(track => <track key={track.file} kind="captions" src={captionURLs[track.file]} srcLang={track.language} label={track.label} onError={() => setCaptionError('Captions could not be loaded from their source.')} />)}
    </video>
    {session.episode.artwork?.scope === 'series' && poster && !posterFailed && <small>Series artwork</small>}
    {session.mode === 'channel' && <div className="channel-controls" aria-label="Channel playback controls">
      <button onClick={() => { if (paused || blocked) rejoin(); else video.current?.pause(); }}>{paused || blocked ? 'Resume current channel' : 'Pause'}</button>
      <button onClick={rejoin}>Rejoin channel</button>
      <button aria-pressed={preferences.muted} onClick={() => { if (video.current) video.current.muted = !video.current.muted; }}>{preferences.muted ? 'Unmute' : 'Mute'}</button>
      <label>Volume<input type="range" min="0" max="1" step="0.05" value={preferences.volume} onChange={event => { if (video.current) video.current.volume = Number(event.target.value); }} /></label>
      <span aria-label="Program elapsed and remaining time">{clock(position)} elapsed · {clock(session.playback.file.duration - position)} remaining</span>
      <button onClick={() => { setControlError(''); const promise = document.fullscreenElement ? document.exitFullscreen?.() : container.current?.requestFullscreen?.(); if (!promise) setControlError('Fullscreen is unavailable in this browser.'); else void promise.catch(() => setControlError('Fullscreen is unavailable in this browser.')); }}>Fullscreen</button>
    </div>}
    {loading && !error && <p className="player-message" role="status">Loading source video…</p>}
    {blocked && <button className="player-start" onClick={() => { if (session.mode === 'channel') rejoin(); else void video.current?.play().then(() => setBlocked(false)).catch(() => setError('Playback could not start. Please open the source page.')); }}>Start playback</button>}
    {error && <p className="player-message" role="alert">{error}</p>}
    {controlError && <p role="status">{controlError}</p>}
    {captionError && <p role="status">{captionError}</p>}
    {captions.length > 0 && <label className="caption-control">Captions <select value={selectedCaption} onChange={event => {
      const chosen = event.target.value;
      Array.from(video.current?.textTracks || []).forEach((track, index) => { track.mode = captions[index]?.file === chosen ? 'showing' : 'disabled'; });
      setSelectedCaption(chosen); setCaptionError('');
    }}><option value="">Off</option>{captions.map(track => <option key={track.file} value={track.file}>{track.label}</option>)}</select></label>}
    <div className="playback-source"><span>ARCHIVAL REPLAY · ORIGINAL BROADCAST {session.episode.date || 'DATE UNKNOWN'}</span><a href={session.playback.source_url} target="_blank" rel="noreferrer">Open source ↗</a></div>
  </div>;
}
