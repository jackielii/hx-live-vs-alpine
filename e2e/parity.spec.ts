import { test, expect, type Page } from "@playwright/test";

const libs = ["alpine", "hxlive"] as const;

function collectErrors(page: Page): string[] {
  const errors: string[] = [];
  page.on("console", (m) => {
    if (m.type() === "error") errors.push(m.text());
  });
  page.on("pageerror", (e) => errors.push(String(e)));
  return errors;
}

for (const lib of libs) {
  test.describe(lib, () => {
    test("counter increments", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/counter`);
      const count = page.locator("span");
      await expect(count).toHaveText("0");
      await page.getByRole("button", { name: "Increment" }).click();
      await expect(count).toHaveText("1");
      await page.getByRole("button", { name: "Increment" }).click();
      await expect(count).toHaveText("2");
      expect(errors).toEqual([]);
    });

    test("dropdown toggles and closes on outside click", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/dropdown`);
      const contents = page.getByText("Contents...");
      await expect(contents).toBeHidden();
      await page.getByRole("button", { name: "Toggle" }).click();
      await expect(contents).toBeVisible();
      // body has 16px padding; (2,2) is padding, outside every element.
      await page.locator("body").click({ position: { x: 2, y: 2 } });
      await expect(contents).toBeHidden();
      await page.getByRole("button", { name: "Toggle" }).click();
      await expect(contents).toBeVisible();
      await page.getByRole("button", { name: "Toggle" }).click();
      await expect(contents).toBeHidden();
      expect(errors).toEqual([]);
    });

    test("search filters the list", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/search`);
      const visible = page.locator("li:visible");
      await expect(visible).toHaveCount(3);
      await page.getByPlaceholder("Search...").fill("ba");
      await expect(visible).toHaveCount(2);
      await expect(visible).toHaveText(["bar", "baz"]);
      await page.getByPlaceholder("Search...").fill("baz");
      await expect(visible).toHaveCount(1);
      await page.getByPlaceholder("Search...").fill("");
      await expect(visible).toHaveCount(3);
      expect(errors).toEqual([]);
    });

    test("tabs switch panels and active class", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/tabs`);
      const a = page.getByRole("button", { name: "A" });
      const b = page.getByRole("button", { name: "B" });
      await expect(page.getByText("Panel A")).toBeVisible();
      await expect(page.getByText("Panel B")).toBeHidden();
      await expect(a).toHaveClass(/active/);
      await expect(b).not.toHaveClass(/active/);
      await b.click();
      await expect(page.getByText("Panel B")).toBeVisible();
      await expect(page.getByText("Panel A")).toBeHidden();
      await expect(b).toHaveClass(/active/);
      await expect(a).not.toHaveClass(/active/);
      expect(errors).toEqual([]);
    });

    test("transition shows and hides", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/transition`);
      const hello = page.getByText("Hello");
      await expect(hello).toBeHidden();
      await page.getByRole("button", { name: "Toggle" }).click();
      await expect(hello).toBeVisible();
      await page.getByRole("button", { name: "Toggle" }).click();
      await expect(hello).toBeHidden();
      expect(errors).toEqual([]);
    });

    test("class binding toggles .on", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/classbind`);
      const bold = page.getByRole("button", { name: "Bold" });
      await expect(bold).not.toHaveClass(/\bon\b/);
      await bold.click();
      await expect(bold).toHaveClass(/\bon\b/);
      await bold.click();
      await expect(bold).not.toHaveClass(/\bon\b/);
      expect(errors).toEqual([]);
    });
  });
}
