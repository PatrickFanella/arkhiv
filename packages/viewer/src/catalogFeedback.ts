import type { Episode } from './model.js';

export type Catalog = { episodes: Episode[]; total: number; page: number };
export type CatalogResult =
  | { key: string; state: 'pending' }
  | { key: string; state: 'ready'; catalog: Catalog }
  | { key: string; state: 'error'; error: string };

// A filter change must never present the previous query's results as current.
export function catalogFeedback(result: CatalogResult, key: string): { catalog: Catalog | null; error: string } {
  if (result.key !== key || result.state === 'pending') return { catalog: null, error: '' };
  if (result.state === 'error') return { catalog: null, error: result.error };
  return { catalog: result.catalog, error: '' };
}
