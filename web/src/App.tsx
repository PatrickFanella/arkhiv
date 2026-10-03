import { useEffect, useState } from 'react';
import './styles.css';

type Episode = { id: string; title: string; date: string; description: string; synthetic: boolean };
type Catalog = { episodes: Episode[]; total: number; page: number };

export default function App() {
  const [catalog, setCatalog] = useState<Catalog | null>(null);
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(1);
  const [error, setError] = useState('');
  useEffect(() => {
    const controller = new AbortController();
    fetch(`/api/episodes?q=${encodeURIComponent(query)}&page=${page}`, { signal: controller.signal })
      .then(async response => { if (!response.ok) throw new Error('The episode catalog is unavailable.'); return await response.json() as Catalog; })
      .then(result => { setCatalog(result); setError(''); })
      .catch((reason: unknown) => { if (!controller.signal.aborted) setError(reason instanceof Error ? reason.message : 'Catalog unavailable.'); });
    return () => controller.abort();
  }, [query, page]);
  return <main>
    <header><a className="wordmark" href="/">VHNESS<span>INDEPENDENT TELEVISION ARCHIVE</span></a><span className="collection-tag">COLLECTION 001 / DEMOCRACY NOW!</span></header>
    <section className="intro"><p className="eyebrow">THE WAR AND PEACE REPORT</p><h1>A record of<br />another perspective.</h1><p className="lede">Independent news, in its own words. Browse the Democracy Now! archive by broadcast date.</p><p className="status">Metadata foundation · Playback and the shared replay channel are being built.</p></section>
    <section className="shelf" aria-label="Episode library"><div className="shelf-heading"><h2>Broadcast archive</h2><label>Search episodes<input type="search" placeholder="Title or description" value={query} onChange={event => { setQuery(event.target.value); setPage(1); }} /></label></div>
      {error && <p role="alert">{error}</p>}
      {!catalog && !error && <p role="status">Loading episodes…</p>}
      {catalog && <><p className="count">{catalog.total} cataloged episodes</p>{catalog.episodes.length === 0 && <p>No episodes match this search.</p>}<div className="episodes">{catalog.episodes.map(episode => <article key={episode.id}><div className="episode-art"><span>DN!</span><time dateTime={episode.date}>{episode.date}</time></div><div className="episode-copy">{episode.synthetic && <p className="fixture">SYNTHETIC DEVELOPMENT FIXTURE</p>}<h3>{episode.title}</h3><p>{episode.description}</p></div></article>)}</div><nav aria-label="Catalog pages"><button disabled={page === 1} onClick={() => setPage(page - 1)}>Previous</button><span>Page {page}</span><button disabled={page * 24 >= catalog.total} onClick={() => setPage(page + 1)}>Next</button></nav></>}
    </section><footer>VHNESS / SOURCE METADATA STORED LOCALLY<span>Democracy Now! video delivery: source streaming only.</span></footer>
  </main>;
}
