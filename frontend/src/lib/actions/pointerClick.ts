import type { Action } from 'svelte/action';

/**
 * Svelte action for a click handler on a non-interactive container — a backdrop,
 * a drag wrapper, an empty-space target.
 *
 * Only use it where the same effect is reachable without a mouse (an Escape
 * handler, a focusable child element). It carries no ARIA role and adds no
 * keyboard handler, precisely because the element is not a control: giving it
 * `role="button"` would announce a control that a keyboard user cannot reach in
 * a meaningful order.
 *
 * Usage:
 *   <div use:pointerClick={handleBackdropClick}>…</div>
 */
export const pointerClick: Action<HTMLElement, (e: MouseEvent) => void> = (node, handler) => {
  let current = handler;

  function onClick(e: MouseEvent) {
    current(e);
  }

  node.addEventListener('click', onClick);

  return {
    update(next: (e: MouseEvent) => void) {
      current = next;
    },
    destroy() {
      node.removeEventListener('click', onClick);
    }
  };
};
