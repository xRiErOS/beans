import { test, expect } from './fixtures';

/**
 * Bean references in rendered markdown are real anchors (`href="?bean=…"`), so
 * keyboard users reach them. The beanLinks action intercepts the activation and
 * selects the bean in place.
 *
 * A reference whose bean is not in the store — archived, deleted, from another
 * store, or a false positive of the `beans-xxxx` matcher — must do nothing at
 * all. If the action returns without preventDefault, the browser follows the
 * anchor to `?bean=<id>`; the selection does not follow (the layout applies
 * `selectedBeanId` exactly once), so the URL ends up advertising a bean that is
 * neither selected nor existent, and a reload lands on a dangling selection.
 */
test.describe('Bean reference links', () => {
  test('a reference to a bean that is not in the store does not navigate', async ({
    beans,
    backlogPage,
    page
  }) => {
    const hostId = beans.create('Host Bean', { status: 'todo', type: 'task' });
    // Shaped like a bean reference so the markdown matcher linkifies it, but no
    // such bean exists in this store.
    const danglingId = 'beans-zzzz';
    beans.run(['update', hostId, '--body-append', `Superseded by ${danglingId}.`]);

    await backlogPage.goto(1);
    await backlogPage.selectBean('Host Bean');

    const body = page.locator('.bean-body');
    await expect(body).toBeVisible({ timeout: 10_000 });

    const link = body.locator(`a[data-bean-id="${danglingId}"]`);
    await expect(link).toBeVisible();
    await expect(link).toHaveAttribute('href', `?bean=${danglingId}`);

    const urlBefore = page.url();
    const historyBefore = await page.evaluate(() => history.length);
    await link.click();

    // The click is a no-op: the browser must not follow the anchor. The URL
    // alone is not enough evidence — uiState rewrites `?bean=` back to the
    // selected bean via replaceState, which hides the navigation. The pushed
    // history entry does not get rewritten, so that is what pins it.
    await page.waitForTimeout(500);
    expect(await page.evaluate(() => history.length)).toBe(historyBefore);
    expect(page.url()).toBe(urlBefore);
  });
});
