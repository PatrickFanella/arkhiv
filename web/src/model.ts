export type SourceFile = { name: string; duration: number; width: number; height: number; bytes: number };
export type Episode = { id: string; collection_id?: string; title: string; date: string; description: string; creator?: string; synthetic: boolean };
export type Playback = { id: string; policy: 'source_stream' | 'local_preserve'; url: string; source_url: string; file: SourceFile; snapshot_sha256: string };
export type Program = { episode_id: string; file: string; policy: Playback['policy']; start: string; duration: number };
export type Channel = { server_time: string; revision: string; state: string; reason?: string; offset: number; program?: Program; next?: Program; episode?: Episode; playback?: Playback };
export type Session = { mode: 'ondemand' | 'channel'; episode: Episode; playback: Playback; offset: number; receivedAt: number; latency: number; generation: number; programKey?: string };
export function targetOffset(session: Session, now: number): number {
  const elapsed = session.mode === 'channel' ? Math.max(0, (now - session.receivedAt + session.latency) / 1000) : 0;
  return Math.max(0, Math.min(session.playback.file.duration - 0.25, session.offset + elapsed));
}
export function assetKey(session: Session): string { return `${session.mode}:${session.episode.id}:${session.playback.url}:${session.generation}:${session.programKey || ""}`; }
export function playbackURL(value: Playback): string {
  if (value.policy === 'local_preserve') {
    if (!value.url.startsWith('/api/media/') || value.url.includes('..')) throw new Error('Invalid local playback URL.');
    return value.url;
  }
  const url = new URL(value.url);
  if (url.protocol !== 'https:' || url.hostname !== 'archive.org' || url.username || url.password || url.port || !url.pathname.startsWith('/download/')) throw new Error('Invalid source playback URL.');
  return url.href;
}
export async function request<T>(path: string, signal?: AbortSignal): Promise<T> {
  const deadline = AbortSignal.timeout(10_000);
  const response = await fetch(path, { signal: signal ? AbortSignal.any([signal, deadline]) : deadline });
  const data = await response.json() as T & { error?: string };
  if (!response.ok) throw new Error(data.error || 'The request could not be completed.');
  return data;
}
