import { defineConfig } from "@playwright/test";

// Same specs, served from the exported directory under the project-site base,
// the way GitHub Pages will serve them.
export default defineConfig({
  testDir: ".",
  use: { baseURL: "http://localhost:8898/hx-live-vs-alpine/" },
  webServer: {
    command: "npm run e2e:static:serve",
    url: "http://localhost:8898/hx-live-vs-alpine/",
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
  },
});
