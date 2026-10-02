import { renderToString } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import App from './App';

describe('bootstrap page', () => {
  it('states that the catalog has not been imported', () => {
    const html = renderToString(<App />);
    expect(html).toContain('No documents have been imported yet.');
    expect(html).toContain('role="status"');
  });
});
