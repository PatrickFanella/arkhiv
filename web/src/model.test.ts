import { describe, expect, it } from 'vitest';
import { assetKey, captionURL, playbackURL, targetOffset, type Session } from './model';

const session: Session = { mode: 'channel', episode: { id: 'item', title: 'Example', date: '2026-10-02', description: '', synthetic: false }, playback: { id: 'item', policy: 'source_stream', url: 'https://archive.org/download/item/a.mp4', source_url: 'https://archive.org/details/item', file: { name: 'a.mp4', duration: 60, bytes: 1, width: 854, height: 480 }, snapshot_sha256: 'example' }, offset: 20, receivedAt: 1000, latency: 100, generation: 1 };
describe('shared clock playback', () => {
  it('accounts for response and media-loading time without advancing on-demand playback', () => {
    expect(targetOffset(session, 6000)).toBeCloseTo(25.1);
    expect(targetOffset({ ...session, mode: 'ondemand' }, 6000)).toBe(20);
  });
  it('clamps a delayed join to the asset end and negative elapsed time', () => {
    expect(targetOffset(session, 100000)).toBe(59.75);
    expect(targetOffset(session, 0)).toBe(20);
  });
});
it('pins source captions to the playback item and exact WebVTT filename', () => {
  const caption = { file: 'English captions.vtt', label: 'English', language: 'en', url: 'https://archive.org/download/item/English%20captions.vtt' };
  expect(captionURL(caption, session.playback)).toBe(caption.url);
  for (const url of ['https://evil.test/captions.vtt', 'https://archive.org/download/other/English%20captions.vtt', 'https://archive.org/download/item/English%20captions.vtt?download=1', 'https://user:secret@archive.org/download/item/English%20captions.vtt']) expect(() => captionURL({ ...caption, url }, session.playback)).toThrow();
  expect(() => captionURL({ ...caption, file: '../captions.vtt' }, session.playback)).toThrow();
});
it('reloads a repeated episode when the scheduled slot changes', () => {
 expect(assetKey({ ...session, programKey: 'revision:first' })).not.toBe(assetKey({ ...session, programKey: 'revision:second' }));
});
it('replaces the player when reviewed caption permissions change', () => {
  expect(assetKey(session)).not.toBe(assetKey({ ...session, playback: { ...session.playback, captions: [{ file: 'a.vtt', label: 'English', language: 'en', url: 'https://archive.org/download/item/a.vtt' }] } }));
});
describe('playback URL boundary', () => {
  it('rejects arbitrary or credential-bearing remote URLs and local path escape', () => {
    for (const url of ['https://evil.test/a.mp4', 'https://archive.org.evil.test/download/a.mp4', 'https://user:pass@archive.org/download/a.mp4', 'http://archive.org/download/a.mp4']) expect(() => playbackURL({ ...session.playback, url })).toThrow();
    expect(() => playbackURL({ ...session.playback, policy: 'local_preserve', url: '/api/media/../../private' })).toThrow();
  });
  it('accepts source-hosted download URLs and the local serving contract', () => {
    expect(playbackURL(session.playback)).toBe(session.playback.url);
    expect(playbackURL({ ...session.playback, policy: 'local_preserve', url: '/api/media/item' })).toBe('/api/media/item');
  });
});
