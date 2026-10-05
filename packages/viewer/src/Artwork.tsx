import { useState } from 'react';
import { artworkURL, type Episode } from './model.js';

export function EpisodeArtwork({ episode, fallback = 'TV' }: { episode: Episode; fallback?: string }) {
  let url: string | undefined;
  try { url = artworkURL(episode); } catch { /* Invalid references use the fallback. */ }
  const [failedURL, setFailedURL] = useState('');
  return <>{url && failedURL !== url ? <img src={url} alt="" loading="lazy" referrerPolicy="no-referrer" onError={() => setFailedURL(url!)} /> : <span>{fallback}</span>}
    {episode.artwork?.scope === 'series' && url && failedURL !== url && <small>Series artwork</small>}</>;
}
