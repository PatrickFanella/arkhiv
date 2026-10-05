import { describe, expect, it } from 'vitest';
import { channelSeekTarget } from './channelSync.js';

function media(ranges: [number, number][], currentTime = 120, readyState = 4, seeking = false) {
  return { currentTime, readyState, seeking, buffered: {
    length: ranges.length, start: (index: number) => ranges[index][0], end: (index: number) => ranges[index][1],
  } };
}

describe('channel correction while a remote source buffers', () => {
  it('lets a slow initial seek finish across repeated clock updates', () => {
    const loading = media([[119, 121]], 120, 1, true);
    for (const expected of [124, 126, 128, 130]) expect(channelSeekTarget(loading, expected)).toBeNull();
  });
  it('does not chase a live target outside the downloaded ranges or inside a gap', () => {
    const playing = media([[119, 125], [132, 138]]);
    expect(channelSeekTarget(playing, 128)).toBeNull();
    expect(channelSeekTarget(playing, 140)).toBeNull();
    expect(channelSeekTarget(playing, 134)).toBeNull();
    expect(channelSeekTarget(playing, 138)).toBeNull();
  });
  it('corrects drift after the media is playable and the target has buffered', () => {
    const recovered = media([[119, 145]]);
    expect(channelSeekTarget(recovered, 132)).toBe(132);
    expect(channelSeekTarget({ ...recovered, currentTime: 130 }, 132)).toBeNull();
  });
  it('leaves a completed seek with insufficient future data alone', () => {
    expect(channelSeekTarget(media([[119, 145]], 120, 2), 132)).toBeNull();
    expect(channelSeekTarget(media([[119, 145]]), NaN)).toBeNull();
  });
});
