import { useEffect, useState } from 'react';

export default function App() {
  const [status, setStatus] = useState('Connecting to the archive service…');
  useEffect(() => {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 8000);
    let active = true;
    fetch('/api/health', { signal: controller.signal })
      .then(async response => {
        if (!response.ok) throw new Error('Service unavailable');
        const body = await response.json();
        if (body.status !== 'ok' || body.service !== 'arkhiv') throw new Error('Unexpected service');
        if (active) setStatus('Archive service connected');
      })
      .catch(() => { if (active) setStatus('Archive service unavailable. Try refreshing the page.'); })
      .finally(() => clearTimeout(timeout));
    return () => { active = false; controller.abort(); clearTimeout(timeout); };
  }, []);

  return (
    <main>
      <header className="masthead"><a href="/" aria-label="Arkhiv home">ARKHIV</a><span>Independent print. Preserved.</span></header>
      <section className="introduction" aria-labelledby="archive-title">
        <p className="eyebrow">Collection 001 / In preparation</p>
        <h1 id="archive-title">The Anarchist<br />Zine Archive</h1>
        <p className="description">Pamphlets, independent publications, and the ideas people put into print.</p>
        <div className="notice"><span className="notice-label">The first collection is being prepared.</span><p>No documents have been imported yet. Catalog browsing, search, and on-site reading will arrive with the first collection.</p></div>
      </section>
      <footer><span>Arkhiv · A home for independent archives</span><p role="status">{status}</p></footer>
    </main>
  );
}
