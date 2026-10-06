import { readJSON } from './http.js';

export type SourceFile = { name: string; duration: number; width: number; height: number; bytes: number };
export type Artwork = { file: string; episode_file?: string; url: string; snapshot_sha256: string; source_item_id: string; scope: 'episode' | 'series' };
export type VideoDetails = { source_title: string; title_origin: 'source' | 'editorial'; source_url: string; snapshot_sha256: string; file?: string; format?: string; duration?: number; width?: number; height?: number; bytes?: number };
export type Episode = { id: string; collection_id?: string; title: string; date: string; description: string; creator?: string; rights?: string; license_url?: string; subjects?: string[]; languages?: string[]; source_item_id?: string; source_file?: string; artwork?: Artwork; video?: VideoDetails; synthetic: boolean };
export type Caption = { file: string; label: string; language: string; url: string };
export type Playback = { id: string; policy: 'source_stream' | 'local_preserve'; url: string; source_url: string; file: SourceFile; snapshot_sha256: string; captions?: Caption[] };
export type Program = { episode_id: string; file: string; policy: Playback['policy']; start: string; duration: number };
export type Channel = { server_time: string; revision: string; state: string; reason?: string; offset: number; program?: Program; next?: Program; episode?: Episode; playback?: Playback };
export type Session = { mode: 'ondemand' | 'channel'; episode: Episode; playback: Playback; offset: number; receivedAt: number; latency: number; generation: number; programKey?: string; collection?: string };
export function targetOffset(session: Session, now: number): number {
  const elapsed = session.mode === 'channel' ? Math.max(0, (now - session.receivedAt + session.latency) / 1000) : 0;
  return Math.max(0, Math.min(session.playback.file.duration - 0.25, session.offset + elapsed));
}
export function assetKey(session: Session): string { return `${session.mode}:${session.episode.id}:${session.playback.url}:${session.generation}:${session.programKey || ""}:${JSON.stringify(session.playback.captions || [])}`; }
export function artworkURL(episode: Episode): string | undefined {
  const a = episode.artwork;
  if (!a) return undefined;
  const url = new URL(a.url);
  const item = episode.source_item_id || episode.id;
  const expectedFile = a.file.split('/').map(encodeURIComponent).join('/');
  const expected = `/download/${encodeURIComponent(item)}/${expectedFile}`;
  const safeFile = !/[\\\0]/.test(a.file) && a.file.split('/').every(part => part !== '' && part !== '.' && part !== '..');
  if (url.protocol !== 'https:' || url.hostname !== 'archive.org' || url.username || url.password || url.port || decodeURIComponent(url.pathname) !== decodeURIComponent(expected) || url.search || url.hash || a.source_item_id !== item || !/^[a-f0-9]{64}$/.test(a.snapshot_sha256) || !safeFile || !/\.(jpe?g|png|webp)$/i.test(a.file) || (a.scope !== 'episode' && a.scope !== 'series') || (episode.source_item_id && a.scope === 'episode' && (!a.episode_file || a.episode_file !== episode.source_file))) throw new Error('Invalid source artwork reference.');
  return url.href;
}
export function playbackURL(value: Playback): string {
  if (value.policy === 'local_preserve') {
    if (!value.url.startsWith('/api/media/') || value.url.includes('..')) throw new Error('Invalid local playback URL.');
    return value.url;
  }
  const url = new URL(value.url);
  if (url.protocol !== 'https:' || url.hostname !== 'archive.org' || url.username || url.password || url.port || !url.pathname.startsWith('/download/')) throw new Error('Invalid source playback URL.');
  return url.href;
}
export function captionURL(value: Caption, playback: Playback): string {
  const source = new URL(playback.source_url);
  const url = new URL(value.url);
  const item = source.pathname.split('/').at(-1);
  const prefix = `/download/${item}/`;
  if (source.protocol !== 'https:' || source.hostname !== 'archive.org' || source.username || source.password || source.port || !source.pathname.startsWith('/details/') || !item || url.protocol !== 'https:' || url.hostname !== 'archive.org' || url.username || url.password || url.port || !url.pathname.startsWith(prefix) || decodeURIComponent(url.pathname.slice(prefix.length)) !== value.file || !value.file.toLowerCase().endsWith('.vtt') || /[/\\\0]/.test(value.file) || url.search || url.hash) throw new Error('Invalid source caption URL.');
  return url.href;
}
export async function request<T>(path: string, signal?: AbortSignal): Promise<T> {
  const deadline = AbortSignal.timeout(10_000);
  try {
    const response = await fetch(path, { signal: signal ? AbortSignal.any([signal, deadline]) : deadline });
    const data = await readJSON<T & { error?: string }>(response);
    if (!response.ok) throw new Error(typeof data?.error === 'string' && data.error.trim() ? data.error : 'The request could not be completed.');
    return data;
  } catch (reason) {
    if (deadline.aborted && !signal?.aborted) throw new Error('The archive took too long to respond. Please try again.');
    throw reason;
  }
}
