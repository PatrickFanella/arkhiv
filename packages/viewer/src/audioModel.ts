// Adapted from Dedhed e5843b17fd33d2e09c88fe3e5022153a0c9e8adc (GPL-3.0-only).
export type Track = { title: string; position: number; start: number; duration: number };
export type SetList = { title: string; first: number; last: number; reviewed: boolean };
export type Recording = {
  id: string; concert_id: string; artist: string; date: string; venue: string;
  location: string; title: string; kind: string; source: string; taper: string;
  transferer: string; tracks: Track[]; sets: SetList[]; duration: number;
  warnings: string[]; synthetic: boolean;
};
export function clock(seconds: number): string {
  const n = Math.max(0, Math.floor(seconds));
  return n >= 3600 ? `${Math.floor(n / 3600)}:${String(Math.floor(n / 60) % 60).padStart(2, '0')}:${String(n % 60).padStart(2, '0')}` : `${Math.floor(n / 60)}:${String(n % 60).padStart(2, '0')}`;
}
export function setBounds(r: Recording, set?: SetList): [number, number] {
  if (!set) return [0, r.duration];
  if (!set.reviewed || !Number.isInteger(set.first) || !Number.isInteger(set.last) || set.first < 0 || set.last >= r.tracks.length || set.first > set.last) throw new Error('Set boundaries are unavailable');
  return [r.tracks[set.first].start, r.tracks[set.last].start + r.tracks[set.last].duration];
}

export type AudioAsset = { recording: Recording; policy: 'local_preserve'; url: string; source_url: string; snapshot_sha256: string; sha256: string; duration: number };
export function audioURL(asset: AudioAsset): string {
  const source = new URL(asset.source_url);
  if (source.protocol !== 'https:' || source.hostname !== 'archive.org' || source.port || source.username || source.password || source.search || source.hash || !/^\/details\/[A-Za-z0-9][A-Za-z0-9._-]{0,199}$/.test(source.pathname)) throw new Error('Invalid recording source URL.');
  const url = new URL(asset.url, 'https://consumer.invalid');
  if (asset.policy !== 'local_preserve' || !asset.url.startsWith('/api/catalog/') || url.origin !== 'https://consumer.invalid' || !/^\/api\/catalog\/v1\/items\/[A-Za-z0-9._-]+\/representations\/[A-Za-z0-9._-]+\/audio\/media$/.test(url.pathname) || /\.\.|\\/.test(asset.url) || url.hash || [...url.searchParams.keys()].some(key => key !== 'collection') || url.searchParams.getAll('collection').length !== 1 || !/^[A-Za-z0-9][A-Za-z0-9._-]{0,199}$/.test(url.searchParams.get('collection') || '')) throw new Error('Invalid continuous audio URL.');
  return asset.url;
}
export function radioOffset(offset: number, receivedAt: number, latency: number, duration: number, now: number): number {
  if (![offset,receivedAt,latency,duration,now].every(Number.isFinite) || duration <= 0 || offset < 0 || latency < 0) throw new Error('Invalid station clock.');
  return Math.max(0,Math.min(duration-0.01, offset+Math.max(0,now-receivedAt+latency)/1000));
}
export function currentTrack(r: Recording, position: number): Track {
  return r.tracks.find(t => position >= t.start && position < t.start + t.duration) ?? r.tracks[r.tracks.length - 1];
}
