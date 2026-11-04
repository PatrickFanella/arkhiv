import { useEffect, useState, type ReactNode } from 'react';
import { request } from './model.js';

export type EngineVersion = { api_version: string; collection?: string; collections?: string[] | null };
// An engine bound to one collection must match it; an engine scoped by a list must include it.
// An unscoped engine serves every collection, so a brand may still open its own.
export function engineServesCollection(engine: EngineVersion, collection: string): boolean {
  if (engine.api_version !== '1') return false;
  if (engine.collection) return engine.collection === collection;
  if (Array.isArray(engine.collections)) return engine.collections.includes(collection);
  return true;
}

export function Compatibility({ collection, children }: { collection: string; children: ReactNode }) {
  const [ready, setReady] = useState(false);
  const [error, setError] = useState(false);
  useEffect(() => {
    const controller = new AbortController();
    void request<EngineVersion>('/api/version', controller.signal).then(value => {
      if (!engineServesCollection(value, collection)) throw new Error('Configuration mismatch');
      setReady(true);
    }).catch(() => { if (!controller.signal.aborted) setError(true); });
    return () => controller.abort();
  }, [collection]);
  if (error) return <main><h1>Archive unavailable</h1><p role="alert">This archive is temporarily unavailable. Please try again shortly.</p><button onClick={() => location.reload()}>Try again</button></main>;
  if (!ready) return <main><p role="status">Opening the archive…</p></main>;
  return children;
}
