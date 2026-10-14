import { readFileSync } from 'node:fs';
import { expect, it, vi } from 'vitest';
import { render } from './entry-prerender';

it('renders the public catalog introduction without browser globals or API requests', () => {
  const profile = JSON.parse(readFileSync(new URL('../public/site.json', import.meta.url), 'utf8'));
  const request = vi.spyOn(globalThis, 'fetch');
  try {
    const html = render(profile);
    expect(html).toContain('<h1');
    expect(html).toContain(profile.headline);
    expect(html).toContain(profile.description);
    expect(request).not.toHaveBeenCalled();
  } finally {
    request.mockRestore();
  }
});
