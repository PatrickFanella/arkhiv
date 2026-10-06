import { useEffect, useRef, useState, type RefObject } from 'react';

export type VideoPreferences = { volume: number; muted: boolean };
export const videoPreferencesKey = 'arkhiv.video.preferences.v1';
const defaults: VideoPreferences = { volume: 1, muted: false };
let memoryPreferences = defaults;

export function readVideoPreferences(storage: Pick<Storage, 'getItem'>, fallback: VideoPreferences = defaults): VideoPreferences {
  try {
    const value: unknown = JSON.parse(storage.getItem(videoPreferencesKey) || 'null');
    if (typeof value !== 'object' || value === null) return fallback;
    const { volume, muted } = value as Partial<VideoPreferences>;
    if (typeof volume !== 'number' || !Number.isFinite(volume) || volume < 0 || volume > 1 || typeof muted !== 'boolean') return fallback;
    return { volume, muted };
  } catch { return fallback; }
}

export function saveVideoPreferences(storage: Pick<Storage, 'setItem'>, value: VideoPreferences): void {
  if (!Number.isFinite(value.volume) || value.volume < 0 || value.volume > 1 || typeof value.muted !== 'boolean') return;
  try { storage.setItem(videoPreferencesKey, JSON.stringify(value)); } catch { /* Playback works without storage. */ }
}

export function useVideoPreferences(video: RefObject<HTMLVideoElement | null>) {
  const [preferences, setPreferences] = useState(defaults);
  const fallback = useRef(memoryPreferences);
  useEffect(() => {
    const element = video.current;
    if (!element) return;
    let stored = fallback.current;
    try { stored = readVideoPreferences(window.localStorage, memoryPreferences); } catch { /* Storage can be disabled. */ }
    element.volume = stored.volume;
    element.muted = stored.muted;
    setPreferences(stored);
    const changed = () => {
      const next = { volume: element.volume, muted: element.muted };
      fallback.current = next;
      memoryPreferences = next;
      setPreferences(next);
      try { saveVideoPreferences(window.localStorage, next); } catch { /* Storage can be disabled. */ }
    };
    element.addEventListener('volumechange', changed);
    return () => element.removeEventListener('volumechange', changed);
  }, [video]);
  return preferences;
}
