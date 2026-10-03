import { useEffect, useRef, useState } from 'react';
import { playbackURL, targetOffset, type Session } from './model';

export function Player({ session, paused, onPaused }: { session: Session; paused: boolean; onPaused: (value: boolean) => void }) {
  const video = useRef<HTMLVideoElement>(null);
  const latest = useRef(session);
  const [error, setError] = useState('');
  const [blocked, setBlocked] = useState(false);
  const [loading, setLoading] = useState(true);
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
    <video ref={video} controls playsInline preload="metadata" aria-label={session.episode.title} onPause={() => { if (video.current?.readyState && !video.current.ended) onPaused(true); }} onPlay={() => { setBlocked(false); onPaused(false); }} onWaiting={() => setLoading(true)} onPlaying={() => setLoading(false)} />
    {loading && !error && <p className="player-message" role="status">Loading source video…</p>}
    {blocked && <button className="player-start" onClick={() => { void video.current?.play().then(() => setBlocked(false)).catch(() => setError('Playback could not start. Please open the source page.')); }}>Start playback</button>}
    {error && <p className="player-message" role="alert">{error}</p>}
    <div className="playback-source"><span>{session.mode === 'channel' ? 'SHARED REPLAY' : 'ON DEMAND'} · ORIGINAL BROADCAST {session.episode.date || 'DATE UNKNOWN'}</span><a href={session.playback.source_url} target="_blank" rel="noreferrer">Open source ↗</a></div>
  </div>;
}
