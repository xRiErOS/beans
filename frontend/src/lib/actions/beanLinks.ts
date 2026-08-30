import type { Action } from 'svelte/action';
import { beansStore } from '$lib/beans.svelte';
import { ui } from '$lib/uiState.svelte';

/**
 * Svelte action that turns bean references inside rendered content into in-app
 * navigation. The references are real links (`<a href="?bean=…">`), so keyboard
 * users reach and activate them without help; this action only intercepts the
 * resulting activation to select the bean in place instead of reloading.
 *
 * Modified clicks (new tab, new window, download) are left to the browser.
 *
 * Usage:
 *   <div use:beanLinks>{@html renderedHtml}</div>
 */
export const beanLinks: Action<HTMLElement> = (node) => {
  function handleClick(e: MouseEvent) {
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) {
      return;
    }
    const target = (e.target as HTMLElement).closest<HTMLElement>('[data-bean-id]');
    if (!target || !node.contains(target)) return;

    // Prevent the default before the store lookup, not after: a reference whose
    // bean is not in the store (archived, deleted, another store, a false
    // positive of the beans-xxxx matcher) must be inert. Letting the browser
    // follow `?bean=<id>` there pushes a history entry for a bean that is
    // neither selected nor existent.
    e.preventDefault();

    const linkedBean = beansStore.get(target.dataset.beanId!);
    if (!linkedBean) return;

    ui.selectBean(linkedBean);
  }

  node.addEventListener('click', handleClick);

  return {
    destroy() {
      node.removeEventListener('click', handleClick);
    }
  };
};
