import { describe, it, expect } from 'vitest';
import { renderMarkdown, linkifyBeanIds } from './markdown';

// Bean references were rendered as <a> without href: not focusable, not
// reachable by keyboard, and no way to open one in a new tab. The click
// handler stays, but it now enhances a real link instead of replacing it.
describe('bean references', () => {
  it('render as focusable links in markdown', async () => {
    const html = await renderMarkdown('see beans-a1b2 for details');

    expect(html).toContain('href="?bean=beans-a1b2"');
  });

  it('render as focusable links in plain text', () => {
    const html = linkifyBeanIds('see beans-a1b2');

    expect(html).toContain('href="?bean=beans-a1b2"');
  });
});
