import { describe, expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { ChannelPanel, Library, Screen, type Viewer } from './index.js';

function viewer(): Viewer {
  return {
    catalog: { episodes: [], total: 0, page: 1 }, query: '', setQuery: () => {},
    collection: 'fixture', setCollection: () => {}, collections: [], year: '',
    setYear: () => {}, page: 1, setPage: () => {}, error: '', playError: '',
    resolving: false, channel: { state: 'unavailable', reason: 'Schedule exhausted',
      revision: '', offset: 0, server_time: '' }, session: null, paused: false,
    mode: 'ondemand', watch: async () => {}, tune: async () => {}, key: '', onPaused: () => {},
  };
}

describe('published component contract', () => {
  it('allows a consumer-owned screen and library title with explicit empty states', () => {
    const v = viewer();
    const html = renderToStaticMarkup(<main><Screen viewer={v}><h1>Consumer identity</h1></Screen><Library viewer={v} heading="Brand collection" /></main>);
    expect(html).toContain('Consumer identity');
    expect(html).toContain('Brand collection');
    expect(html).toContain('aria-label="Brand collection"');
    expect(html).toContain('aria-busy="false"');
    expect(html).toContain('No episodes match this search.');
    expect(html).toContain('aria-label="Catalog pages"');
  });
  it('marks pending searches busy and distinguishes an explicit failure', () => {
    const v = viewer();
    v.catalog = null;
    const pending = renderToStaticMarkup(<Library viewer={v} />);
    expect(pending).toContain('aria-busy="true"');
    expect(pending).toContain('Loading episodes…');
    v.error = 'Catalog unavailable';
    const failed = renderToStaticMarkup(<Library viewer={v} />);
    expect(failed).toContain('aria-busy="false"');
    expect(failed).toContain('role="alert"');
    expect(failed).not.toContain('Loading episodes…');
  });
  it('keeps exhausted schedule visible and does not offer an active tune action', () => {
    const html = renderToStaticMarkup(<ChannelPanel viewer={viewer()} />);
    expect(html).toContain('Schedule exhausted');
    expect(html).toMatch(/<button disabled=""/);
    expect(html).toContain('Your controls affect only your viewing.');
  });
});
