import { test, expect } from "@playwright/test";

test("index sizes every demo frame to its content", async ({ page }) => {
  await page.goto("./");
  const frames = page.locator("iframe[data-frame]");
  const count = await frames.count();
  expect(count).toBeGreaterThan(12);
  const hx = await page.locator('iframe[data-frame][src*="/frame/hxlive/"]').count();
  expect(count).toBe(2 * hx + 5); // 23 demo rows × 2 + 5 none rows
  for (const frame of await frames.all()) {
    // frames.js sets the height from the child's report; the inline default is 120px.
    await expect.poll(async () => frame.evaluate((el) => parseFloat(el.style.height))).not.toBe(120);
    const fits = await frame.evaluate((el) => {
      const doc = (el as HTMLIFrameElement).contentDocument!;
      return doc.documentElement.scrollHeight <= doc.documentElement.clientHeight;
    });
    expect(fits).toBe(true);
  }

  // The main stylesheet references fonts by absolute URL; they must resolve
  // under the current base (this is what a wrong Vite `base` breaks).
  const css = await page.locator('link[rel="stylesheet"]').first().getAttribute("href");
  expect(css).toBeTruthy();
  const cssBody = await (await page.request.get(new URL(css!, page.url()).toString())).text();
  const font = cssBody.match(/url\(([^)]+\.woff2)\)/);
  expect(font, "stylesheet references a font").toBeTruthy();
  const fontRes = await page.request.get(new URL(font![1], page.url()).toString());
  expect(fontRes.status(), font![1]).toBe(200);
});
