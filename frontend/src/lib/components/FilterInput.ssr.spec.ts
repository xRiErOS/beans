import { describe, it, expect } from 'vitest';
import { render } from 'svelte/server';
import FilterInput from './FilterInput.svelte';

// FilterInput read navigator.platform unguarded, which throws the moment the
// component is rendered outside a browser — SSR, prerendering with ssr enabled,
// or a node-environment test. The keyboard hint has to degrade, not explode.
describe('FilterInput without a browser', () => {
  it('renders and falls back to the non-Mac shortcut hint', () => {
    const { body } = render(FilterInput);

    expect(body).toContain('Ctrl+/');
  });
});
