import { useEffect, useState } from 'react';
import type { Recording } from './audioModel.js';
import { readJSON } from './http.js';

export type CatalogMedia = 'print' | 'audio' | 'video';
export type CatalogRepresentation = {
  id: string; media: CatalogMedia; source_item_id: string; snapshot_sha256: string;
  file: string; collections: string[]; legacy_episode_id?: string;
  extension: { print?: { document_id: string; variant: string; pages: number }; audio?: { recording_id: string; concert_id?: string; credits: Record<string,string>; recording?:Recording }; video?: { episode_id: string } };
};
export type CatalogItem = { id: string; title: string; description: string; date: string; creator: string; collections: string[]; representations: CatalogRepresentation[] };
export type CatalogPage = { items: CatalogItem[]; total: number; page: number; search_coverage: string[] };
export type CatalogCollection = { id: string; title: string; policy: string; items: number; media: CatalogMedia[] };
export type CatalogFilter = { collection: string; media: CatalogMedia | ''; query: string; page: number; creator?:string; dateFrom?:string; dateTo?:string };
export type RepresentationTransport = {media:CatalogMedia;policy:string;url:string;source_url:string;file:string};

export function representationURL(asset: RepresentationTransport, action: 'read' | 'play' | 'download' = asset.media === 'print' ? 'read' : 'play'): string {
  const source = new URL(asset.source_url);
  if (source.protocol !== 'https:' || source.hostname !== 'archive.org' || source.port || source.username || source.password || source.search || source.hash || !/^\/details\/[A-Za-z0-9][A-Za-z0-9._-]{0,199}$/.test(source.pathname)) throw new Error('Invalid source identity.');
  const url = new URL(asset.url,'https://consumer.invalid');
  if (url.username || url.password || url.hash || /\\/.test(asset.url) || (!(asset.media === 'audio' && asset.policy === 'source_stream') && /\.\./.test(asset.url))) throw new Error('Invalid media URL.');
  const prefix = `/download/${source.pathname.slice(9)}/`;
  const safeFile = asset.media === 'audio'
    ? !asset.file.includes('\\') && [...asset.file].every(char => char.charCodeAt(0) >= 32 && char.charCodeAt(0) !== 127) && asset.file.split('/').every(part => part !== '' && part !== '.' && part !== '..')
    : !/[/\\\0]/.test(asset.file);
  if (asset.policy === 'source_stream' && url.protocol === 'https:' && url.hostname === 'archive.org' && !url.port && !url.search && url.pathname.startsWith(prefix) && decodeURIComponent(url.pathname.slice(prefix.length)) === asset.file && safeFile) return asset.url;
  if (asset.policy === 'local_preserve' && asset.url.startsWith('/api/catalog/') && url.origin === 'https://consumer.invalid' && /^\/api\/catalog\/v1\/items\/[A-Za-z0-9._-]+\/representations\/[A-Za-z0-9._-]+\/media$/.test(url.pathname) && [...url.searchParams.keys()].every(key=>['collection','action','file'].includes(key)) && url.searchParams.getAll('collection').length===1 && /^[A-Za-z0-9][A-Za-z0-9._-]{0,199}$/.test(url.searchParams.get('collection') || '') && url.searchParams.getAll('file').length===1 && url.searchParams.get('file')===asset.file && url.searchParams.getAll('action').length===1 && url.searchParams.get('action')===action) return asset.url;
  throw new Error('This media source is unavailable.');
}

export function catalogURL(filter: CatalogFilter): string {
  const params = new URLSearchParams({page: String(filter.page)});
  if (filter.collection) params.set('collection',filter.collection);
  if (filter.media) params.set('media',filter.media);
  if (filter.query) params.set('q',filter.query);
  if(filter.creator)params.set('creator',filter.creator);
  if(filter.dateFrom)params.set('date_from',filter.dateFrom);
  if(filter.dateTo)params.set('date_to',filter.dateTo);
  return `/api/catalog/v1/items?${params}`;
}
export async function catalogRequest<T>(url: string, signal: AbortSignal, timeoutMs = 10000): Promise<T> {
  if (!Number.isInteger(timeoutMs) || timeoutMs < 1 || timeoutMs > 60000) throw new Error('Invalid request timeout.');
  const deadline = AbortSignal.timeout(timeoutMs);
  try {
    const response = await fetch(url,{signal:AbortSignal.any([signal,deadline]),cache:'no-store'});
    if (!response.ok) throw new Error(response.status === 404 ? 'This item is unavailable for this collection.' : 'The archive is temporarily unavailable. Please try again.');
    return await readJSON<T>(response);
  } catch (reason) {
    if (deadline.aborted && !signal.aborted) throw new Error('The archive took too long to respond. Please try again.');
    throw reason;
  }
}
export function representationPath(item: CatalogItem, representation: CatalogRepresentation): string {
  return `/api/catalog/v1/items/${encodeURIComponent(item.id)}/representations/${encodeURIComponent(representation.id)}`;
}

// Hide an old response immediately, including the render before effect cleanup.
export function useCatalog(filter: CatalogFilter) {
  const url = catalogURL(filter);
  const [result,setResult] = useState<{url: string; page?: CatalogPage; error?: string}>();
  const [attempt,setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    void catalogRequest<CatalogPage>(url,controller.signal).then(page => {
      if (!controller.signal.aborted) setResult({url,page});
    }).catch(reason => { if (!controller.signal.aborted) setResult({url,error: reason instanceof Error ? reason.message : 'The archive is unavailable.'}); });
    return () => controller.abort();
  },[url,attempt]);
  const current = result?.url === url ? result : undefined;
  return {page: current?.page, error: current?.error, pending: !current, retry: () => {setResult(undefined);setAttempt(value=>value+1);} };
}

// Bookmarks name the logical item; access to a representation is resolved later.
export function catalogFavoriteID(item:string):string { return `catalog:${item}`; }
export function catalogSelectionURL(base:string, collection:string, item:string, representation:string):string {
 const url=new URL(base);url.searchParams.set('collection',collection);url.searchParams.set('item',item);url.searchParams.set('representation',representation);url.hash='';return url.toString();
}
