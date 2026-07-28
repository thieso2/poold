const { defineConfig, devices } = require("@playwright/test");

module.exports = defineConfig({
  testDir: "tests/browser",
  outputDir: "test-results",
  timeout: 30000,
  use: {
    baseURL: "http://127.0.0.1:18090",
    screenshot: "only-on-failure",
    trace: "retain-on-failure"
  },
  expect: { timeout: 5000 },
  webServer: {
    command: "go run ./cmd/poold",
    env: {
      POOLD_LISTEN_ADDR: "127.0.0.1:18090",
      POOLD_POOL_ADDR: "127.0.0.1:18990",
      POOLD_DB_PATH: "./test-results/poold-browser.db",
      POOLD_TOKEN: "browser-token"
    },
    url: "http://127.0.0.1:18090/",
    reuseExistingServer: false,
    timeout: 120000
  },
  projects: [
    { name: "webkit-320x568", use: { ...devices["iPhone SE"], viewport: { width: 320, height: 568 } } },
    { name: "webkit-390x844", use: { ...devices["iPhone 13"], viewport: { width: 390, height: 844 } } },
    { name: "webkit-430x932", use: { ...devices["iPhone 14 Pro Max"], viewport: { width: 430, height: 932 } } },
    { name: "webkit-844x390", use: { ...devices["iPhone 13 landscape"], viewport: { width: 844, height: 390 } } },
    { name: "chromium-1280x800", use: { ...devices["Desktop Chrome"], viewport: { width: 1280, height: 800 } } }
  ]
});
