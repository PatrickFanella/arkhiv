import { afterEach, expect, test, vi } from 'vitest';
import { request } from './model.js';
import { catalogRequest } from './catalog.js';

afterEach(() => vi.restoreAllMocks());

test('HTML outage responses give viewers retryable errors without parser details', async () => {
  vi.spyOn(globalThis, 'fetch').mockImplementation(async () => new Response('<html>upstream unavailable</html>', { status: 503 }));
  for (const read of [() => request('/api/episodes'), () => catalogRequest('/api/catalog/v1/items', new AbortController().signal)]) {
    await expect(read()).rejects.toThrow('The archive is temporarily unavailable. Please try again.');
  }
});

test('unreadable successful catalog responses do not become empty results', async () => {
  vi.spyOn(globalThis, 'fetch').mockImplementation(async () => new Response('<html>proxy login page</html>'));
  for (const read of [() => request('/api/episodes'), () => catalogRequest('/api/catalog/v1/items', new AbortController().signal)]) {
    await expect(read()).rejects.toThrow('The archive returned an unreadable response. Please try again.');
  }
});

test('missing items and structured source errors retain their useful context', async () => {
  const fetch = vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(new Response('', { status: 404 })).mockResolvedValueOnce(Response.json({ error: 'This source is not approved for playback.' }, { status: 403 }));
  await expect(request('/api/episodes/missing')).rejects.toThrow('This item is unavailable for this collection.');
  await expect(request('/api/episodes/private')).rejects.toThrow('This source is not approved for playback.');
  expect(fetch).toHaveBeenCalledTimes(2);
});
