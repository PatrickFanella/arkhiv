import { useEffect, useRef, useState } from 'react';
import { assetKey, request, type Channel, type Episode, type Playback, type Session } from './model';

export type Collection = { id: string; title: string; policy: 'source_stream' | 'local_preserve'; episodes: number };
export type Catalog = { episodes: Episode[]; total: number; page: number };
export function useViewer(defaultCollection: string) {
  const [catalog, setCatalog] = useState<Catalog | null>(null);
  const [query, setQuery] = useState('');
  const [collection, setCollection] = useState(defaultCollection);
  const [collections, setCollections] = useState<Collection[]>([]);
  useEffect(() => { const controller = new AbortController(); void request<Collection[]>('/api/collections', controller.signal).then(setCollections).catch(() => {}); return () => controller.abort(); }, []);
  const [year, setYear] = useState('');
  const [page, setPage] = useState(1);
  const [error, setError] = useState('');
  const [playError, setPlayError] = useState('');
  const [resolving, setResolving] = useState(false);
  const [channel, setChannel] = useState<Channel | null>(null);
  const [session, setSession] = useState<Session | null>(null);
  const [paused, setPaused] = useState(false);
  const [mode, setMode] = useState<'ondemand' | 'channel'>('ondemand');
  const action = useRef(0);
  const currentKey = useRef('');
  useEffect(() => {
    const controller = new AbortController();
    request<Catalog>(`/api/episodes?collection=${encodeURIComponent(collection)}&q=${encodeURIComponent(query)}&year=${encodeURIComponent(year)}&page=${page}`, controller.signal)
      .then(result => { setCatalog(result); setError(''); })
      .catch((reason: unknown) => { if (!controller.signal.aborted) setError(reason instanceof Error ? reason.message : 'Catalog unavailable.'); });
    return () => controller.abort();
  }, [query, year, page, collection]);
  useEffect(() => {
    const controller = new AbortController();
    let inFlight = false;
    const update = async () => {
      if (inFlight) return;
      inFlight = true;
      const began = performance.now(); const token = action.current;
      try {
        const next = await request<Channel>('/api/channel', controller.signal);
        if (controller.signal.aborted) return;
        setChannel(next);
        if (mode === 'channel' && token === action.current) {
          if (next.state === 'playing' && next.episode && next.playback) {
            const episode = next.episode; const playback = next.playback; const programKey = `${next.revision}:${next.program?.start || ''}`;
            setSession(previous => {
              const changed = !previous || previous.episode.id !== episode.id || previous.playback.url !== playback.url || previous.mode !== 'channel' || previous.programKey !== programKey;
              return { mode: 'channel', episode, playback, programKey, offset: next.offset, receivedAt: performance.now(), latency: (performance.now() - began) / 2, generation: changed ? action.current : previous.generation };
            });
          } else { setSession(null); }
        }
      } catch { if (!controller.signal.aborted) { setChannel({ state: 'unavailable', reason: 'The channel is temporarily unavailable.', revision: '', offset: 0, server_time: '' }); if (mode === 'channel' && token === action.current) setSession(null); } }
      finally { inFlight = false; }
    };
    void update();
    const timer = window.setInterval(() => { void update(); }, 2000);
    const visible = () => { if (!document.hidden) void update(); };
    document.addEventListener('visibilitychange', visible);
    return () => { controller.abort(); clearInterval(timer); document.removeEventListener('visibilitychange', visible); };
  }, [mode]);
  useEffect(() => { currentKey.current = session ? assetKey(session) : ''; }, [session]);
  const watch = async (episode: Episode) => {
    const token = ++action.current;
    setMode('ondemand'); setPlayError(''); setSession(null); setResolving(true);
    try {
      const playback = await request<Playback>(`/api/playback/${encodeURIComponent(episode.id)}`);
      if (token !== action.current) return;
      setPaused(false);
      setSession({ mode: 'ondemand', episode, playback, offset: 0, receivedAt: performance.now(), latency: 0, generation: token });
      document.querySelector('#screen')?.scrollIntoView({ behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth', block: 'start' });
    } catch (reason) { if (token === action.current) setPlayError(reason instanceof Error ? reason.message : 'Playback unavailable.'); }
    finally { if (token === action.current) setResolving(false); }
  };
  const tune = async () => {
    const token = ++action.current; const began = performance.now(); setPlayError(''); setSession(null); setResolving(true);
    try {
      const next = await request<Channel>('/api/channel');
      if (token !== action.current) return;
      setChannel(next); setMode('channel'); setPaused(false);
      if (next.state !== 'playing' || !next.episode || !next.playback) { setSession(null); setPlayError(next.reason || 'No current program.'); return; }
      setSession({ mode: 'channel', episode: next.episode, playback: next.playback, programKey: `${next.revision}:${next.program?.start || ''}`, offset: next.offset, receivedAt: performance.now(), latency: (performance.now() - began) / 2, generation: token });
    } catch (reason) { if (token === action.current) setPlayError(reason instanceof Error ? reason.message : 'Channel unavailable.'); }
    finally { if (token === action.current) setResolving(false); }
  };
  const key = session ? assetKey(session) : '';
  const onPaused = (value: boolean) => { if (currentKey.current === key) setPaused(value); };
  return { catalog, query, setQuery, collection, setCollection, collections, year, setYear, page, setPage, error, playError, resolving, channel, session, paused, mode, watch, tune, key, onPaused };
}
