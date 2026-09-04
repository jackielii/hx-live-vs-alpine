import { test, expect } from "@playwright/test";

test("every demo frame loads under both libraries without errors", async ({ page, context }) => {
  test.setTimeout(120_000);
  await page.goto("/");
  const srcs = await page
    .locator("iframe[data-frame]")
    .evaluateAll((els) => els.map((e) => e.getAttribute("src")!));
  expect(srcs.length).toBeGreaterThan(12);
  const hx = srcs.filter((s) => s.startsWith("/frame/hxlive/"));
  const al = srcs.filter((s) => s.startsWith("/frame/alpine/"));
  expect(hx.length).toBeLessThan(al.length); // none rows have no hx-live frame
  for (const src of srcs) {
    const p = await context.newPage();
    const errors: string[] = [];
    p.on("console", (m) => {
      if (m.type() === "error") errors.push(m.text());
    });
    p.on("pageerror", (e) => errors.push(String(e)));
    const res = await p.goto(src);
    expect(res?.status(), src).toBe(200);
    await expect(p.locator("body")).not.toBeEmpty();
    await p.waitForTimeout(150);
    expect(errors, src).toEqual([]);
    await p.close();
  }
});
