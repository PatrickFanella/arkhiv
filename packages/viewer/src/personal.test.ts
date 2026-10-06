import { afterEach, expect, it, vi } from 'vitest';
import { PersonalError, personalRequest, readLocalFavorites } from './personal.js';

afterEach(() => vi.unstubAllGlobals());
it('retains bounded stable IDs without private metadata and tolerates blocked storage', () => {
  const storage = { getItem: () => JSON.stringify(['episode', 'episode', { title: 'private' }, 'foreign/path', '', 'another']) };
  expect(readLocalFavorites(storage, 'brand')).toEqual(['episode', 'another']);
  expect(readLocalFavorites({ getItem: () => { throw Error('blocked'); } }, 'brand', ['page-copy'])).toEqual(['page-copy']);
});
it('sends CSRF and revision checks without browser bearer tokens', async () => {
  const fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ revision: 4 }) });
  vi.stubGlobal('fetch', fetch);
  await personalRequest('/api/personal/v1/favorites/collection/episode', 'csrf', 'PUT', undefined, 3);
  const options = fetch.mock.calls[0][1] as RequestInit;
  expect(options.credentials).toBe('same-origin');
  expect(options.headers).toEqual({ 'Content-Type': 'application/json', 'X-CSRF-Token': 'csrf', 'If-Match': '3' });
});
it('preserves explicit conflicts for caller review', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 409, json: async () => ({ error: 'Favorites changed.' }) }));
  await expect(personalRequest('/api/personal/v1/favorites')).rejects.toEqual(new PersonalError('Favorites changed.', 409));
});
