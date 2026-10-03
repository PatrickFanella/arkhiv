import { useEffect, useState, type ReactNode } from 'react';
import { request } from './model.js';

export function Compatibility({ collection, children }: { collection: string; children: ReactNode }) {
  const [ready, setReady] = useState(false);
  const [error, setError] = useState(false);
  useEffect(() => {
    const controller = new AbortController();
    void request<{ api_version: string; collection: string }>('/api/version', controller.signal).then(value => {
      if (value.api_version !== '1' || value.collection !== collection) throw new Error('Configuration mismatch');
      setReady(true);
    }).catch(() => { if (!controller.signal.aborted) setError(true); });
    return () => controller.abort();
  }, [collection]);
  if (error) return <main><h1>Archive unavailable</h1><p role="alert">This archive is temporarily unavailable. Please try again shortly.</p><button onClick={() => location.reload()}>Try again</button></main>;
  if (!ready) return <main><p role="status">Opening the archive…</p></main>;
  return children;
}
