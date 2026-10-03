import { describe, expect, it } from 'vitest';
import { catalogFeedback } from './catalogFeedback.js';

describe('catalog results belong to their requested filters', () => {
  const catalog = { episodes: [], total: 24, page: 2 };
  it('hides a previous collection, query, year or page while the new result loads', () => {
    for (const key of ['other-collection', 'other-query', 'other-year', 'other-page']) {
      expect(catalogFeedback({ key: 'previous', state: 'ready', catalog }, key)).toEqual({ catalog: null, error: '' });
    }
  });
  it('does not present an obsolete request error as the current search error', () => {
    expect(catalogFeedback({ key: 'old', state: 'error', error: 'Failed old search' }, 'new')).toEqual({ catalog: null, error: '' });
  });
  it('shows the matching result or its explicit failure without previous cards', () => {
    expect(catalogFeedback({ key: 'current', state: 'ready', catalog }, 'current')).toEqual({ catalog, error: '' });
    expect(catalogFeedback({ key: 'current', state: 'error', error: 'Catalog unavailable' }, 'current')).toEqual({ catalog: null, error: 'Catalog unavailable' });
  });
});
