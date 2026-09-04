import { test, expect } from "@playwright/test";

test("index sizes every demo frame to its content", async ({ page }) => {
  await page.goto("/");
  const frames = page.locator("iframe[data-frame]");
  const count = await frames.count();
  expect(count).toBeGreaterThan(12);
  expect(count % 2).toBe(1); // 23 demo rows × 2 + 5 none rows × 1 = 51
  for (const frame of await frames.all()) {
    // frames.js sets the height from the child's report; the inline default is 120px.
    await expect.poll(async () => frame.evaluate((el) => parseFloat(el.style.height))).not.toBe(120);
    const fits = await frame.evaluate((el) => {
      const doc = (el as HTMLIFrameElement).contentDocument!;
      return doc.documentElement.scrollHeight <= doc.documentElement.clientHeight;
    });
    expect(fits).toBe(true);
  }
});
