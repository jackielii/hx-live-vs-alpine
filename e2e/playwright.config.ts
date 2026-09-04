import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  use: { baseURL: "http://localhost:8899" },
  webServer: {
    command: "npm run e2e:serve",
    port: 8899,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
