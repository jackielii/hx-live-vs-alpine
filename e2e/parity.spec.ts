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

    test("bind sets the placeholder", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/bind`);
      await expect(page.locator("input")).toHaveAttribute("placeholder", "Type here...");
      expect(errors).toEqual([]);
    });

    test("on reacts to the Enter key only", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/on`);
      const input = page.locator("input");
      await input.pressSequentially("abc");
      await expect(page.locator("span")).toHaveText("");
      await input.press("Enter");
      await expect(page.locator("span")).toHaveText("Enter pressed");
      expect(errors).toEqual([]);
    });

    test("init runs once at load", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/init`);
      await expect(page.locator("span")).toHaveText("Initialised!");
      expect(errors).toEqual([]);
    });

    test("html renders markup", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/html`);
      await expect(page.locator("span strong")).toHaveText("calebporzio");
      expect(errors).toEqual([]);
    });

    test("effect derives a value", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/effect`);
      await expect(page.locator("span")).toHaveText("5");
      await page.getByRole("button", { name: "Change Message" }).click();
      await expect(page.locator("span")).toHaveText("12");
      expect(errors).toEqual([]);
    });

    test("ignore leaves the subtree alone", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/ignore`);
      const spans = page.locator("span");
      await expect(spans.nth(0)).toHaveText("processed");
      await expect(spans.nth(1)).toHaveText("untouched");
      expect(errors).toEqual([]);
    });

    test("ref removes the referenced element", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/ref`);
      await expect(page.getByText("Hello")).toBeVisible();
      await page.getByRole("button", { name: "Remove Text" }).click();
      await expect(page.getByText("Hello")).toHaveCount(0);
      expect(errors).toEqual([]);
    });

    test("model mirrors the input", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/model`);
      await page.locator("input").fill("hi");
      await expect(page.locator("span")).toHaveText("hi");
      expect(errors).toEqual([]);
    });

    test("for appends to the list", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/for`);
      await expect(page.locator("li")).toHaveText(["Red", "Orange", "Yellow"]);
      await page.getByRole("button", { name: "Add Green" }).click();
      await expect(page.locator("li")).toHaveText(["Red", "Orange", "Yellow", "Green"]);
      expect(errors).toEqual([]);
    });

    test("if toggles the contents", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/if`);
      await expect(page.getByText("Contents...")).toBeHidden();
      await page.getByRole("button", { name: "Toggle" }).click();
      await expect(page.getByText("Contents...")).toBeVisible();
      await page.getByRole("button", { name: "Toggle" }).click();
      await expect(page.getByText("Contents...")).toBeHidden();
      expect(errors).toEqual([]);
    });

    test("cloak hides the element and removes its marker", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/cloak`);
      await expect(page.getByText("This will not")).toBeHidden();
      await expect(page.locator("[x-cloak], [hx-cloak]")).toHaveCount(0);
      expect(errors).toEqual([]);
    });

    test("el is the current element", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/el`);
      await page.getByRole("button").click();
      await expect(page.getByRole("button")).toHaveText("Hello World!");
      expect(errors).toEqual([]);
    });

    test("dispatch reaches an ancestor listener", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/dispatch`);
      await page.getByRole("button", { name: "Notify" }).click();
      await expect(page.locator("span")).toHaveText("Notified!");
      expect(errors).toEqual([]);
    });

    test("root reads the component root", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/root`);
      await page.getByRole("button", { name: "Say Hi" }).click();
      await expect(page.getByRole("button")).toHaveText("Hello World!");
      expect(errors).toEqual([]);
    });

    test("nexttick reads after the re-render", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/nexttick`);
      await page.getByRole("button").click();
      await expect(page.getByRole("button")).toHaveText("Hello World!");
      await expect(page.locator("span")).toHaveText("Hello World!");
      expect(errors).toEqual([]);
    });

    test("store is shared across components", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/store`);
      const content = page.getByText("Content");
      await expect(content).not.toHaveClass(/\bdark\b/);
      await page.getByRole("button", { name: "Toggle Dark Mode" }).click();
      await expect(content).toHaveClass(/\bdark\b/);
      await page.getByRole("button", { name: "Toggle Dark Mode" }).click();
      await expect(content).not.toHaveClass(/\bdark\b/);
      expect(errors).toEqual([]);
    });

    test("watch reports the new value", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/watch`);
      await page.getByRole("button", { name: "Toggle Open" }).click();
      await expect(page.locator("span")).toHaveText("open is now true");
      await page.getByRole("button", { name: "Toggle Open" }).click();
      await expect(page.locator("span")).toHaveText("open is now false");
      expect(errors).toEqual([]);
    });
  });
}
