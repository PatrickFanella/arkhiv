import { describe, expect, it } from 'vitest';
import { engineServesCollection } from './Compatibility.js';

describe('engineServesCollection', () => {
  it('requires API version 1', () => {
    expect(engineServesCollection({ api_version: '2', collection: 'dn' }, 'dn')).toBe(false);
  });
  it('matches an engine bound to one collection', () => {
    expect(engineServesCollection({ api_version: '1', collection: 'dn' }, 'dn')).toBe(true);
    expect(engineServesCollection({ api_version: '1', collection: 'other' }, 'dn')).toBe(false);
  });
  it('checks a scoped collection list when no single collection is bound', () => {
    expect(engineServesCollection({ api_version: '1', collection: '', collections: ['dn', 'bill'] }, 'dn')).toBe(true);
    expect(engineServesCollection({ api_version: '1', collection: '', collections: ['bill'] }, 'dn')).toBe(false);
  });
  it('accepts an unscoped engine', () => {
    expect(engineServesCollection({ api_version: '1', collection: '', collections: null }, 'dn')).toBe(true);
    expect(engineServesCollection({ api_version: '1' }, 'dn')).toBe(true);
  });
});
