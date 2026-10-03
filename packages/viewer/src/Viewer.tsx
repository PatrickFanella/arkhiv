import { useEffect, useState } from 'react';
import { Player } from './Player.js';
import { request, type Episode } from './model.js';
import { useViewer } from './useViewer.js';

export type Viewer = ReturnType<typeof useViewer>;

export function Screen({ viewer, children }: { viewer: Viewer; children: React.ReactNode }) {
  const v = viewer;
  return <div className="screen-main" id="screen">
    {v.resolving && <p role="status">Preparing video…</p>}
    {v.session ? <><Player key={v.key} session={v.session} paused={v.paused} onPaused={v.onPaused} />
      <div className="episode-detail"><p className="eyebrow">{v.session.episode.synthetic ? 'SYNTHETIC DEVELOPMENT FIXTURE' : 'ARCHIVAL REPLAY'}</p><h1>{v.session.episode.title}</h1><p>{v.session.episode.description}</p>{v.session.episode.creator && <p>Source credit: {v.session.episode.creator}</p>}<button onClick={() => { void v.watch(v.session!.episode); }}>Watch from beginning</button></div></> : children}
    {v.playError && <p role="alert">{v.playError}</p>}
  </div>;
}

export function ChannelPanel({ viewer }: { viewer: Viewer }) {
  const v = viewer;
  const nextID = v.channel?.next?.episode_id;
  const [next, setNext] = useState<Episode | null>(null);
  useEffect(() => {
    const controller = new AbortController();
    if (nextID) void request<{ episode: Episode }>(`/api/episodes/${encodeURIComponent(nextID)}`, controller.signal).then(value => setNext(value.episode)).catch(() => setNext(null));
    return () => controller.abort();
  }, [nextID]);
  return <aside className="channel-panel" aria-label="Replay schedule"><p className="eyebrow">SCHEDULED REPLAY</p><h2>On the channel</h2><p className="now-label">NOW PLAYING</p>
    {v.channel?.episode ? <><h3>{v.channel.episode.title}</h3><p>Original broadcast <time dateTime={v.channel.episode.date}>{v.channel.episode.date || 'Date unknown'}</time></p></> : <p>{v.channel?.reason || 'Checking the channel…'}</p>}
    <button onClick={() => { void v.tune(); }} disabled={v.channel?.state !== 'playing'}>{v.mode === 'channel' && v.paused ? 'Rejoin current program' : 'Tune in ↗'}</button>
    {v.channel?.next && <div className="next"><p className="eyebrow">UP NEXT</p><p>{next && next.id === nextID ? next.title : 'Next scheduled replay'}</p><time dateTime={v.channel.next.start}>{new Date(v.channel.next.start).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</time></div>}
    <p className="channel-note">Archived programs on a shared schedule. Your controls affect only your viewing.</p>
  </aside>;
}

export function Library({ viewer, heading = 'Episode library', mark = 'TV', yearFilter = true }: { viewer: Viewer; heading?: string; mark?: string; yearFilter?: boolean }) {
  const v = viewer;
  return <section className="shelf" id="library" aria-label={heading} aria-busy={!v.catalog && !v.error}>
    <div className="shelf-heading"><h2>{heading}</h2><div className="filters">
      {yearFilter && <label>Year<input inputMode="numeric" placeholder="Any year" maxLength={4} value={v.year} onChange={event => { v.setYear(event.target.value.replace(/\D/g, '')); v.setPage(1); }} /></label>}
      <label>Search episodes<input type="search" placeholder="Title or description" value={v.query} onChange={event => { v.setQuery(event.target.value); v.setPage(1); }} /></label>
    </div></div>
    {v.error && <p role="alert">{v.error}</p>}{!v.catalog && !v.error && <p role="status">Loading episodes…</p>}
    {v.catalog && <><p className="count" role="status">{v.catalog.total} cataloged {v.catalog.total === 1 ? 'episode' : 'episodes'}</p>
      {v.catalog.episodes.length === 0 && <p>No episodes match this search.</p>}
      <div className="episodes">{v.catalog.episodes.map(episode => <article key={episode.id}>
        <button className="episode-art" aria-label={`Watch ${episode.title}`} onClick={() => { void v.watch(episode); }}><span>{mark}</span><time dateTime={episode.date}>{episode.date || 'Date unknown'}</time></button>
        <div className="episode-copy">{episode.synthetic && <p className="fixture">SYNTHETIC DEVELOPMENT FIXTURE</p>}<h3><button className="text-button" onClick={() => { void v.watch(episode); }}>{episode.title}</button></h3><p>{episode.description}</p></div>
      </article>)}</div>
      <nav aria-label="Catalog pages"><button disabled={v.page === 1} onClick={() => v.setPage(v.page - 1)}>Previous</button><span>Page {v.page}</span><button disabled={v.page * 24 >= v.catalog.total} onClick={() => v.setPage(v.page + 1)}>Next</button></nav></>}
  </section>;
}
