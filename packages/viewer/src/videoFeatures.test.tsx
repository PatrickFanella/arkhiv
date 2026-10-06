import { describe, expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { artworkURL, type Episode, type Session } from './model.js';
import { EpisodeArtwork } from './Artwork.js';
import { Player } from './Player.js';
import { readVideoPreferences, saveVideoPreferences } from './videoPreferences.js';
import { videoCatalogURL } from './videoCatalog.js';
import { videoState, videoStateLabel } from './videoState.js';

const episode: Episode = { id: 'episode', title: 'Episode', description: '', date: '', synthetic: true };
const session: Session = { mode: 'channel', episode, playback: { id: 'episode', policy: 'local_preserve', url: '/api/media/episode', source_url: 'https://archive.org/details/episode', snapshot_sha256: 'a'.repeat(64), file: { name: 'episode.mp4', duration: 60, width: 640, height: 480, bytes: 100 } }, offset: 10, receivedAt: 0, latency: 0, generation: 1 };

describe('video preferences', () => {
  it('preserves mute and volume, and tolerates blocked or corrupt storage', () => {
    const values = new Map<string, string>();
    const storage = { getItem: (key: string) => values.get(key) || null, setItem: (key: string, value: string) => { values.set(key, value); } };
    saveVideoPreferences(storage, { volume: 0.35, muted: true });
    expect(readVideoPreferences(storage)).toEqual({ volume: 0.35, muted: true });
    saveVideoPreferences(storage, { volume: NaN, muted: false });
    expect(readVideoPreferences(storage).muted).toBe(true);
    expect(readVideoPreferences({ getItem: () => '{bad' })).toEqual({ volume: 1, muted: false });
    expect(readVideoPreferences({ getItem: () => '{"volume":2,"muted":true}' }).volume).toBe(1);
    expect(() => saveVideoPreferences({ setItem: () => { throw new Error('blocked'); } }, { volume: 0.5, muted: true })).not.toThrow();
  });
});

it('validates artwork source identity and shows honest fallback/series labels', () => {
  const series: Episode = { ...episode, source_item_id: 'series', artwork: { file: 'cover.jpg', url: 'https://archive.org/download/series/cover.jpg', source_item_id: 'series', scope: 'series', snapshot_sha256: 'a'.repeat(64) } };
  expect(artworkURL(series)).toContain('/series/cover.jpg');
  expect(renderToStaticMarkup(<EpisodeArtwork episode={series} />)).toContain('Series artwork');
  expect(renderToStaticMarkup(<EpisodeArtwork episode={episode} />)).toContain('TV');
  for (const url of ['https://evil.test/download/series/cover.jpg', 'https://archive.org/download/other/cover.jpg', 'https://archive.org/download/series/cover.jpg?x=1']) {
    expect(() => artworkURL({ ...series, artwork: { ...series.artwork!, url } })).toThrow();
  }
});

it('retains scope and encodes combined filters and special characters', () => {
  const url = new URL(videoCatalogURL('collection', 'a&b', '', 2, { subject: 'Science & nature', durationMax: '100' }), 'https://fixture.test');
  expect(url.searchParams.get('collection')).toBe('collection');
  expect(url.searchParams.get('q')).toBe('a&b');
  expect(url.searchParams.get('subject')).toBe('Science & nature');
  expect(url.searchParams.get('duration_max')).toBe('100');
  expect(url.searchParams.get('page')).toBe('2');
});

it('only labels active, ready, non-seeking playback near the schedule as synchronized', () => {
  const media = { paused: false, ended: false, seeking: false, readyState: 4, currentTime: 11 };
  expect(videoState(media, session, 1000)).toBe('synchronized');
  expect(videoState({ ...media, currentTime: 1 }, session, 1000)).toBe('joining');
  expect(videoState({ ...media, seeking: true }, session, 1000)).toBe('joining');
  expect(videoState({ ...media, readyState: 2 }, session, 1000)).toBe('buffering');
  expect(videoStateLabel('channel', videoState({ ...media, paused: true }, session, 1000))).toBe('Paused · Channel continues');
});

it('exposes channel controls without scrubbing and preserves native on-demand controls', () => {
  const channel = renderToStaticMarkup(<Player session={session} paused={false} onPaused={() => {}} />);
  expect(channel).toContain('Rejoin channel');
  expect(channel).toContain('Channel playback controls');
  expect(channel).not.toMatch(/<video[^>]* controls/);
  expect(channel).toContain('Volume');
  const demand = renderToStaticMarkup(<Player session={{ ...session, mode: 'ondemand' }} paused={false} onPaused={() => {}} />);
  expect(demand).toMatch(/<video[^>]* controls/);
});
