import { useEffect, useRef, useState } from 'react';
import { captionURL, playbackURL, targetOffset, type Session } from './model.js';

export function Player({ session, paused, onPaused }: { session: Session; paused: boolean; onPaused: (value: boolean) => void }) {
  const video = useRef<HTMLVideoElement>(null);
  const latest = useRef(session);
  const [error, setError] = useState('');
  const [blocked, setBlocked] = useState(false);
  const [loading, setLoading] = useState(true);
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
    const seek = () => { if (element.readyState > 0) element.currentTime = targetOffset(latest.current, performance.now()); };
    const loaded = () => { setLoading(false); seek(); if (!paused) void element.play().catch(() => setBlocked(true)); };
    const failure = () => { setLoading(false); setError('This video could not be loaded from its source. Try again or open the source page.'); };
    element.addEventListener('loadedmetadata', loaded);
    element.addEventListener('error', failure);
    try { element.src = playbackURL(session.playback); element.load(); } catch (reason) { setError(reason instanceof Error ? reason.message : 'Invalid playback source.'); }
    return () => { element.removeEventListener('loadedmetadata', loaded); element.removeEventListener('error', failure); element.removeAttribute('src'); element.load(); };
    // A keyed Player owns one asset; clock updates use latest, without reloading media.
  }, []);
  useEffect(() => {
    const element = video.current;
    if (session.mode === 'channel' && element && element.readyState > 0 && !paused) {
      const expected = targetOffset(session, performance.now());
      if (Math.abs(element.currentTime - expected) > 3) element.currentTime = expected;
    }
  }, [session, paused]);
  return <div className="player">
    <video ref={video} controls playsInline preload="metadata" aria-label={session.episode.title} onPause={() => { if (video.current?.readyState && !video.current.ended) onPaused(true); }} onPlay={() => { setBlocked(false); onPaused(false); }} onWaiting={() => setLoading(true)} onPlaying={() => setLoading(false)}>
      {captions.map(track => <track key={track.file} kind="captions" src={captionURLs[track.file]} srcLang={track.language} label={track.label} onError={() => setCaptionError('Captions could not be loaded from their source.')} />)}
    </video>
    {loading && !error && <p className="player-message" role="status">Loading source video…</p>}
    {blocked && <button className="player-start" onClick={() => { void video.current?.play().then(() => setBlocked(false)).catch(() => setError('Playback could not start. Please open the source page.')); }}>Start playback</button>}
    {error && <p className="player-message" role="alert">{error}</p>}
    {captionError && <p role="status">{captionError}</p>}
    {captions.length > 0 && <label className="caption-control">Captions <select value={selectedCaption} onChange={event => {
      const chosen = event.target.value;
      Array.from(video.current?.textTracks || []).forEach((track, index) => { track.mode = captions[index]?.file === chosen ? 'showing' : 'disabled'; });
      setSelectedCaption(chosen); setCaptionError('');
    }}><option value="">Off</option>{captions.map(track => <option key={track.file} value={track.file}>{track.label}</option>)}</select></label>}
    <div className="playback-source"><span>{session.mode === 'channel' ? 'SHARED REPLAY' : 'ON DEMAND'} · ORIGINAL BROADCAST {session.episode.date || 'DATE UNKNOWN'}</span><a href={session.playback.source_url} target="_blank" rel="noreferrer">Open source ↗</a></div>
  </div>;
}
