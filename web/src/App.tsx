import { useEffect, useRef, useState } from 'react';
import { Player } from './Player';
import { assetKey, request, type Channel, type Episode, type Playback, type Session } from './model';
import './styles.css';

type Catalog = { episodes: Episode[]; total: number; page: number };
export default function App() {
  const [catalog, setCatalog] = useState<Catalog | null>(null);
  const [query, setQuery] = useState('');
  const [year, setYear] = useState('');
  const [page, setPage] = useState(1);
  const [error, setError] = useState('');
  const [playError, setPlayError] = useState('');
  const [channel, setChannel] = useState<Channel | null>(null);
  const [session, setSession] = useState<Session | null>(null);
  const [paused, setPaused] = useState(false);
  const [mode, setMode] = useState<'ondemand' | 'channel'>('ondemand');
  const action = useRef(0);
  const currentKey = useRef('');
  useEffect(() => {
    const controller = new AbortController();
    request<Catalog>(`/api/episodes?q=${encodeURIComponent(query)}&year=${encodeURIComponent(year)}&page=${page}`, controller.signal)
      .then(result => { setCatalog(result); setError(''); })
      .catch((reason: unknown) => { if (!controller.signal.aborted) setError(reason instanceof Error ? reason.message : 'Catalog unavailable.'); });
    return () => controller.abort();
  }, [query, year, page]);
  useEffect(() => {
    const controller = new AbortController();
    let inFlight = false;
    const update = async () => {
      if (inFlight) return;
      inFlight = true;
      const began = performance.now();
      try {
        const next = await request<Channel>('/api/channel', controller.signal);
        if (controller.signal.aborted) return;
        setChannel(next);
        if (mode === 'channel') {
          if (next.state === 'playing' && next.episode && next.playback) {
            const episode = next.episode; const playback = next.playback;
            setSession(previous => {
              const changed = !previous || previous.episode.id !== episode.id || previous.playback.url !== playback.url || previous.mode !== 'channel';
              return { mode: 'channel', episode, playback, offset: next.offset, receivedAt: performance.now(), latency: (performance.now() - began) / 2, generation: changed ? action.current : previous.generation };
            });
          } else { setSession(null); }
        }
      } catch { if (!controller.signal.aborted) { setChannel({ state: 'unavailable', reason: 'The channel is temporarily unavailable.', revision: '', offset: 0, server_time: '' }); if (mode === 'channel') setSession(null); } }
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
    setMode('ondemand'); setPlayError('');
    try {
      const playback = await request<Playback>(`/api/playback/${encodeURIComponent(episode.id)}`);
      if (token !== action.current) return;
      setPaused(false);
      setSession({ mode: 'ondemand', episode, playback, offset: 0, receivedAt: performance.now(), latency: 0, generation: token });
      document.querySelector('#screen')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    } catch (reason) { if (token === action.current) setPlayError(reason instanceof Error ? reason.message : 'Playback unavailable.'); }
  };
  const tune = async () => {
    const token = ++action.current; const began = performance.now(); setPlayError('');
    try {
      const next = await request<Channel>('/api/channel');
      if (token !== action.current) return;
      setChannel(next); setMode('channel'); setPaused(false);
      if (next.state !== 'playing' || !next.episode || !next.playback) { setSession(null); setPlayError(next.reason || 'No current program.'); return; }
      setSession({ mode: 'channel', episode: next.episode, playback: next.playback, offset: next.offset, receivedAt: performance.now(), latency: (performance.now() - began) / 2, generation: token });
    } catch (reason) { setPlayError(reason instanceof Error ? reason.message : 'Channel unavailable.'); }
  };
  const key = session ? assetKey(session) : '';
  return <main>
    <header><a className="wordmark" href="/">VHNESS<span>INDEPENDENT TELEVISION ARCHIVE</span></a><span className="collection-tag">COLLECTION 001 / DEMOCRACY NOW!</span></header>
    <section className="screen-layout" id="screen" aria-label="Viewing screen">
      <div className="screen-main">{session ? <><Player key={key} session={session} paused={paused} onPaused={value => { if (currentKey.current === key) setPaused(value); }} /><div className="episode-detail"><p className="eyebrow">{session.episode.synthetic ? 'SYNTHETIC DEVELOPMENT FIXTURE' : 'ARCHIVAL REPLAY'}</p><h1>{session.episode.title}</h1><p>{session.episode.description}</p>{session.episode.creator && <p>Source credit: {session.episode.creator}</p>}<button onClick={() => { void watch(session.episode); }}>Watch from beginning</button></div></> : <div className="intro"><p className="eyebrow">THE WAR AND PEACE REPORT</p><h1>A record of<br />another perspective.</h1><p className="lede">Independent news, in its own words. Choose a broadcast or join the shared replay channel.</p></div>}{playError && <p role="alert">{playError}</p>}</div>
      <aside className="channel-panel"><p className="eyebrow">SHARED REPLAY CHANNEL</p><h2>One schedule.<br />The same moment.</h2><p className="now-label">NOW PLAYING</p>{channel?.episode ? <><h3>{channel.episode.title}</h3><p>Original broadcast <time dateTime={channel.episode.date}>{channel.episode.date}</time></p></> : <p>{channel?.reason || 'Checking the channel…'}</p>}<button onClick={() => { void tune(); }} disabled={channel?.state !== 'playing'}>{mode === 'channel' && paused ? 'Rejoin current program' : 'Tune in ↗'}</button>{channel?.next && <p className="next">UP NEXT · {channel.next.episode_id}<br />{new Date(channel.next.start).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</p>}<p className="channel-note">Archived broadcasts on a shared timeline. Your player controls affect only your viewing.</p></aside>
    </section>
    <section className="shelf" aria-label="Episode library"><div className="shelf-heading"><h2>Broadcast archive</h2><div className="filters"><label>Year<input inputMode="numeric" placeholder="Any year" maxLength={4} value={year} onChange={event => { setYear(event.target.value.replace(/\D/g, '')); setPage(1); }} /></label><label>Search episodes<input type="search" placeholder="Title or description" value={query} onChange={event => { setQuery(event.target.value); setPage(1); }} /></label></div></div>
      {error && <p role="alert">{error}</p>}{!catalog && !error && <p role="status">Loading episodes…</p>}
      {catalog && <><p className="count">{catalog.total} cataloged {catalog.total === 1 ? 'episode' : 'episodes'}</p>{catalog.episodes.length === 0 && <p>No episodes match this search.</p>}<div className="episodes">{catalog.episodes.map(episode => <article key={episode.id}><button className="episode-art" aria-label={`Watch ${episode.title}`} onClick={() => { void watch(episode); }}><span>DN!</span><time dateTime={episode.date}>{episode.date || 'DATE UNKNOWN'}</time></button><div className="episode-copy">{episode.synthetic && <p className="fixture">SYNTHETIC DEVELOPMENT FIXTURE</p>}<h3><button className="text-button" onClick={() => { void watch(episode); }}>{episode.title}</button></h3><p>{episode.description}</p></div></article>)}</div><nav aria-label="Catalog pages"><button disabled={page === 1} onClick={() => setPage(page - 1)}>Previous</button><span>Page {page}</span><button disabled={page * 24 >= catalog.total} onClick={() => setPage(page + 1)}>Next</button></nav></>}
    </section><footer>VHNESS / INDEPENDENT NEWS, PRESERVED<span>Democracy Now! · Video streamed from Internet Archive.</span></footer>
  </main>;
}
