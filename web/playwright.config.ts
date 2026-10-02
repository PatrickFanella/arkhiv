import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./tests",
  timeout: 30000,
  expect: { timeout: 15000 },
  use: { baseURL: "http://web", headless: true },
  projects: [
    { name: "desktop", use: { viewport: { width: 1280, height: 800 } } },
    { name: "narrow", use: { viewport: { width: 390, height: 844 } } },
  ],
});
