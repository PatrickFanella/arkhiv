// Adapted from Arkhiv Reader.tsx at 99d499b92cb8d848e0fcb37288ece8bce9de943e.
// GPL-3.0-only. PDF runtime/assets and page layout remain consumer-owned.
import { useEffect, useRef, useState, type FormEvent } from 'react';

export type PrintHit = { page: number; snippet: string };
export interface PrintPage {
  width: number;
  height: number;
  render(canvas: HTMLCanvasElement, scale: number): { promise: Promise<void>; cancel(): void };
  text(): Promise<string>;
}
export interface PrintDocument {
  numPages: number;
  page(number: number): Promise<PrintPage>;
}
export interface PrintBackend {
  open(url: string): { promise: Promise<PrintDocument>; destroy(): Promise<void> };
}
export type PrintReaderProps = {
  url: string;
  backend: PrintBackend;
  title: string;
  initialPage?: number;
  onPageChange?: (page: number) => void;
  search?: (query: string, signal: AbortSignal) => Promise<PrintHit[]>;
};

export function initialPrintPage(value: number | undefined, pages?: number): number {
  const page = value !== undefined && Number.isInteger(value) && value > 0 ? value : 1;
  return pages ? Math.min(page, pages) : page;
}

export function PrintReader(props: PrintReaderProps) {
  return <ReaderSession key={props.url} {...props} />;
}

function ReaderSession({ url, backend, title, initialPage, onPageChange, search }: PrintReaderProps) {
  const [document, setDocument] = useState<PrintDocument | null>(null);
  const [page, setPage] = useState(initialPrintPage(initialPage));
  const [scale, setScale] = useState(1);
  const [width, setWidth] = useState(700);
  const [text, setText] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState('');
  const [searchResult, setSearchResult] = useState<{ query: string; hits?: PrintHit[]; error?: string; pending?: boolean } | null>(null);
  const canvas = useRef<HTMLCanvasElement>(null);
  const wrapper = useRef<HTMLDivElement>(null);
  const searchController = useRef<AbortController | null>(null);
  useEffect(() => () => searchController.current?.abort(), []);
  useEffect(() => {
    if (!wrapper.current) return;
    const observer = new ResizeObserver(entries => setWidth(Math.max(1, entries[0].contentRect.width)));
    observer.observe(wrapper.current);
    return () => observer.disconnect();
  }, []);
  useEffect(() => {
    let active = true;
    let task: ReturnType<PrintBackend['open']> | undefined;
    try {
      task = backend.open(url);
      void task.promise.then(pdf => {
        if (!active) return;
        if (!Number.isInteger(pdf.numPages) || pdf.numPages < 1 || pdf.numPages > 10000) throw new Error('Invalid page count');
        setDocument(pdf); setPage(value => initialPrintPage(value, pdf.numPages));
      }).catch(() => {
        if (active) { setError('The PDF could not be opened. Try again or open its source page.'); setLoading(false); }
      });
    } catch {
      setError('The PDF could not be opened. Try again or open its source page.'); setLoading(false);
    }
    return () => { active = false; if (task) void task.destroy().catch(() => undefined); };
  }, [url, backend]);
  useEffect(() => {
    if (!document || !canvas.current) return;
    let cancelled = false;
    let render: ReturnType<PrintPage['render']> | undefined;
    setLoading(true); setText(''); setError('');
    void (async () => {
      const pdfPage = await document.page(page);
      if (cancelled || !canvas.current) return;
      if (!Number.isFinite(pdfPage.width) || !Number.isFinite(pdfPage.height) || pdfPage.width <= 0 || pdfPage.height <= 0) throw new Error('Invalid page dimensions');
      const requestedScale = Math.min(width / pdfPage.width, 1.5) * scale;
      const renderScale = Math.min(requestedScale, Math.sqrt(16000000 / (pdfPage.width * pdfPage.height)));
      render = pdfPage.render(canvas.current, renderScale);
      await render.promise;
      if (cancelled) return;
      const readable = await pdfPage.text();
      if (!cancelled) { setText(readable); setLoading(false); }
    })().catch(() => { if (!cancelled) { setError('This page could not be rendered.'); setLoading(false); } });
    return () => { cancelled = true; render?.cancel(); };
  }, [document, page, width, scale]);
  function go(next: number) {
    if (document && Number.isInteger(next) && next >= 1 && next <= document.numPages) {
      setPage(next); onPageChange?.(next);
    }
  }
  async function find(event: FormEvent) {
    event.preventDefault();
    if (!search || !query.trim()) return;
    searchController.current?.abort();
    const controller = new AbortController(); searchController.current = controller;
    const requested = query;
    setSearchResult({ query: requested, pending: true });
    try {
      const hits = await search(requested, controller.signal);
      if (!controller.signal.aborted) setSearchResult({ query: requested, hits });
    } catch {
      if (!controller.signal.aborted) setSearchResult({ query: requested, error: 'Document search is unavailable. Please try again.' });
    }
  }
  const result = searchResult?.query === query ? searchResult : null;
  return <section className="reader" aria-label="Document reader" aria-busy={loading}>
    <div className="reader-controls">
      <button onClick={() => go(page - 1)} disabled={!document || page <= 1}>Previous page</button>
      <label>Page <input aria-label="Page number" type="number" min={1} max={document?.numPages || 1} value={page} onChange={event => go(Number(event.target.value))} /></label>
      <span>of {document?.numPages || '…'}</span>
      <button onClick={() => go(page + 1)} disabled={!document || page >= document.numPages}>Next page</button>
      <label>Zoom <select aria-label="Zoom" value={scale} onChange={event => setScale(Number(event.target.value))}>
        <option value={1}>Fit width</option><option value={1.25}>125%</option><option value={1.5}>150%</option><option value={2}>200%</option>
      </select></label>
    </div>
    {search && <form onSubmit={event => { void find(event); }} className="search">
      <label>Search this document <input type="search" value={query} onChange={event => setQuery(event.target.value)} maxLength={200} required /></label><button>Find</button>
    </form>}
    {result?.pending && <p role="status">Searching document text…</p>}
    {result?.error && <p role="alert">{result.error}</p>}
    {result?.hits && <div aria-live="polite"><p>{result.hits.length} matching pages or text sections</p>{result.hits.map((hit, index) => <p key={index}>
      {hit.page > 0 ? <button onClick={() => go(hit.page)}>Page {hit.page}</button> : 'Document text; page mapping unavailable'}: {hit.snippet}
    </p>)}</div>}
    {loading && <p role="status">Loading page…</p>}
    {error && <p role="alert">{error}</p>}
    <div className="canvas-wrap" ref={wrapper}><canvas ref={canvas} aria-label={`${title}, PDF page ${page}`} data-testid="pdf-canvas" /></div>
    <details><summary>Readable text for this page</summary><p>{text || 'No text layer is available on this page.'}</p></details>
  </section>;
}
