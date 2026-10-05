import { targetOffset, type Session } from './model.js';

export type VideoState = 'loading' | 'buffering' | 'blocked' | 'paused' | 'playing' | 'joining' | 'synchronized' | 'unavailable' | 'ended';
// Matches the correction tolerance, not a claim of frame-accurate broadcast.
export const channelToleranceSeconds = 3;
export function videoState(media: Pick<HTMLVideoElement, 'paused' | 'ended' | 'seeking' | 'readyState' | 'currentTime'>, session: Session, now: number): VideoState {
  if (media.readyState === 0) return 'loading';
  if (media.ended) return 'ended';
  if (media.paused) return 'paused';
  if (media.readyState < 3) return 'buffering';
  if (session.mode === 'ondemand') return 'playing';
  return !media.seeking && Math.abs(media.currentTime - targetOffset(session, now)) <= channelToleranceSeconds ? 'synchronized' : 'joining';
}
export function videoStateLabel(mode: Session['mode'], state: VideoState): string {
  if (mode === 'ondemand') return `On demand${state === 'playing' ? '' : ` · ${state}`}`;
  if (state === 'synchronized') return 'On the channel · Scheduled replay';
  if (state === 'paused') return 'Paused · Channel continues';
  return `Scheduled replay · ${state}`;
}
