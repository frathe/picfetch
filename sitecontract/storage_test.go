package sitecontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestRegularPageKeepsInteractionsWhenStorageIsUnavailable protects the
// ordinary page's language selector and lightbox in browsers that deny Web
// Storage.
func TestRegularPageKeepsInteractionsWhenStorageIsUnavailable(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	output := t.TempDir()

	build := exec.Command("make", "build",
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+output,
		"SITE_LOCALES=en,de",
		"SITE_FORMATS=regular",
	)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build storage-test site: %v\n%s", err, combined)
	}

	script := filepath.Join(t.TempDir(), "storage-lightbox.cjs")
	if err := os.WriteFile(script, []byte(storageLightboxScript), 0o600); err != nil {
		t.Fatalf("write browser test: %v", err)
	}

	test := exec.Command("node", script, output)
	test.Dir = repo
	test.Env = append(os.Environ(), "NODE_PATH="+filepath.Join(repo, "node_modules"))
	if combined, err := test.CombinedOutput(); err != nil {
		t.Fatalf("regular-page interactions with unavailable storage failed: %v\n%s", err, combined)
	}
}

const storageLightboxScript = `'use strict';

const fs = require('node:fs');
const http = require('node:http');
const path = require('node:path');
const {chromium} = require('playwright-core');

const root = path.resolve(process.argv[2]);
const server = http.createServer((request, response) => {
  const requestPath = decodeURIComponent(new URL(request.url, 'http://localhost').pathname);
  const relative = requestPath.endsWith('/') ? requestPath + 'index.html' : requestPath;
  const filename = path.resolve(root, '.' + relative);
  if (filename !== root && !filename.startsWith(root + path.sep)) {
    response.writeHead(403).end('forbidden');
    return;
  }
  try {
    response.writeHead(200, {'Content-Type': 'text/html; charset=utf-8'}).end(fs.readFileSync(filename));
  } catch (error) {
    response.writeHead(error.code === 'ENOENT' ? 404 : 500).end('not found');
  }
});

(async () => {
  let browser;
  try {
    await new Promise((resolve, reject) => {
      server.once('error', reject);
      server.listen(0, '127.0.0.1', resolve);
    });
    const address = server.address();
    browser = await chromium.launch({channel: 'chrome', headless: true});
    const context = await browser.newContext({locale: 'de-DE'});
    await context.route('**/*', route => route.request().url().startsWith('http://127.0.0.1:') ? route.continue() : route.abort());
    await context.addInitScript(() => {
      const unavailable = () => { throw new DOMException('The operation is insecure.', 'SecurityError'); };
      Object.defineProperty(Storage.prototype, 'getItem', {value: unavailable});
      Object.defineProperty(Storage.prototype, 'setItem', {value: unavailable});
      Object.defineProperty(navigator, 'languages', {get: () => ['de-DE']});
      Object.defineProperty(navigator, 'language', {get: () => 'de-DE'});
    });
    const page = await context.newPage();
    const baseURL = 'http://127.0.0.1:' + address.port;
    await page.goto(baseURL + '/de/', {waitUntil: 'domcontentloaded'});
    const englishLink = page.locator('[data-language="en"]');
    const authoredURL = new URL(await englishLink.getAttribute('href'));
    if (authoredURL.searchParams.has('language')) {
      throw new Error('language selector polluted the authored canonical URL with its fallback marker');
    }
    await englishLink.evaluate((link, localURL) => { link.href = localURL; }, baseURL + '/');
    await englishLink.click();
    await page.waitForLoadState('networkidle');
    const selectedURL = new URL(page.url());
    if (selectedURL.pathname !== '/' || await page.locator('html').getAttribute('lang') !== 'en') {
      throw new Error('English selection was redirected back to German when Web Storage was unavailable');
    }
    await page.locator('.shot img').first().click();
    if (!await page.locator('#lightbox.open').isVisible()) {
      throw new Error('lightbox did not open after storage methods threw SecurityError');
    }
    await context.close();
  } finally {
    try {
      if (browser) {
        await browser.close();
      }
    } finally {
      if (server.listening) {
        await new Promise(resolve => server.close(resolve));
      }
    }
  }
})().catch(error => {
  process.stderr.write(error.message + '\n');
  process.exitCode = 1;
});
`
