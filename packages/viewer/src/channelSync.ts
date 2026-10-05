// Clock correction must not interrupt an in-flight remote seek or request
// another unbuffered position before the existing media request can finish.
export function channelSeekTarget(
  media: Pick<HTMLMediaElement, 'currentTime' | 'readyState' | 'seeking' | 'buffered'>,
  expected: number,
): number | null {
  if (!Number.isFinite(expected) || media.seeking || media.readyState < 3
      || Math.abs(media.currentTime - expected) <= 3) return null;
  for (let index = 0; index < media.buffered.length; index++) {
    if (expected >= media.buffered.start(index) && expected < media.buffered.end(index)) return expected;
  }
  return null;
}
