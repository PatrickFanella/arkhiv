import { describe, expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { PrintReader, initialPrintPage, type PrintBackend } from './print.js';

describe('shared print reader', () => {
  it('keeps invalid and restored page positions within the document', () => {
    for (const value of [undefined, -1, 0, NaN, Infinity, 1.5]) expect(initialPrintPage(value)).toBe(1);
    expect(initialPrintPage(99, 4)).toBe(4);
    expect(initialPrintPage(3, 4)).toBe(3);
  });
  it('exposes semantic reader controls without requiring a PDF runtime on import', () => {
    const backend: PrintBackend = { open: () => { throw new Error('Server rendering must not fetch a PDF'); } };
    const markup = renderToStaticMarkup(<PrintReader url="/api/approved.pdf" backend={backend} title="Fixture edition" />);
    expect(markup).toContain('aria-label="Document reader"');
    expect(markup).toContain('aria-label="Fixture edition, PDF page 1"');
    expect(markup).toContain('Previous page');
    expect(markup).toContain('Readable text for this page');
    expect(markup).not.toContain('Search this document');
    expect(markup).not.toContain('Download');
  });
});
