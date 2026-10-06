import { expect, it } from 'vitest';
import { PersonalBrowserStorage } from './personal-storage.js';

it('keeps newer favorite and pending edits when reads succeed but writes fail', () => {
  const disk = new Map([['favorites', '["old"]'], ['pending', '[]']]);
  const cache = new PersonalBrowserStorage(() => ({
    getItem: key => disk.get(key) ?? null,
    setItem: () => { throw new Error('quota'); },
  }));
  cache.write('favorites', ['old', 'new']);
  cache.write('pending', [{ item: 'new', remove: false, baseRevision: 7 }]);
  expect(cache.read('favorites', [])).toEqual(['old', 'new']);
  expect(cache.read('pending', [])).toEqual([{ item: 'new', remove: false, baseRevision: 7 }]);
  expect(cache.temporary('favorites')).toBe(true);
  expect(cache.read('different-product', [])).toEqual([]);
  expect(disk.get('favorites')).toBe('["old"]');
});

it('recovers persistence without losing page changes after storage becomes writable', () => {
  const disk = new Map<string, string>();
  let blocked = true;
  const cache = new PersonalBrowserStorage(() => ({
    getItem: key => disk.get(key) ?? null,
    setItem: (key, value) => { if (blocked) throw new Error('blocked'); disk.set(key, value); },
  }));
  cache.write('favorites', ['new']);
  blocked = false;
  cache.write('favorites', cache.read('favorites', []));
  expect(cache.temporary('favorites')).toBe(false);
  expect(disk.get('favorites')).toBe('["new"]');
});

it('accepts an external storage update and tolerates inaccessible storage', () => {
  const disk = new Map([['favorites', '["external"]']]);
  const cache = new PersonalBrowserStorage(() => ({ getItem: key => disk.get(key) ?? null, setItem: () => { throw new Error('quota'); } }));
  cache.write('favorites', ['page']);
  cache.changed('favorites');
  expect(cache.read('favorites', [])).toEqual(['external']);
  const inaccessible = new PersonalBrowserStorage(() => { throw new Error('denied'); });
  expect(inaccessible.read('favorites', [])).toEqual([]);
  inaccessible.write('favorites', ['page']);
  expect(inaccessible.read('favorites', [])).toEqual(['page']);
  expect(inaccessible.temporary('favorites')).toBe(true);
});
