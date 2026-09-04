'use strict';

const fs = require('node:fs');
const http = require('node:http');
const path = require('node:path');
const {chromium} = require('playwright-core');

function assertEqual(actual, expected, description) {
  if (actual !== expected) {
    throw new Error(`${description}: got ${JSON.stringify(actual)}, want ${JSON.stringify(expected)}`);
  }
}

function staticServer(root) {
  return http.createServer((request, response) => {
    const requestPath = decodeURIComponent(new URL(request.url, 'http://localhost').pathname);
    const relative = requestPath.endsWith('/') ? `${requestPath}index.html` : requestPath;
    const filename = path.resolve(root, `.${relative}`);
    if (filename !== root && !filename.startsWith(`${root}${path.sep}`)) {
      response.writeHead(403).end('forbidden');
      return;
    }
    try {
      const data = fs.readFileSync(filename);
      response.writeHead(200, {'Content-Type': 'text/html; charset=utf-8'}).end(data);
    } catch (error) {
      response.writeHead(error.code === 'ENOENT' ? 404 : 500).end('not found');
    }
  });
}

async function main() {
  const root = path.resolve(process.argv[2] || process.env.SITE_OUTPUT_DIR || 'docs');
  const server = staticServer(root);
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  const address = server.address();
  const baseURL = `http://127.0.0.1:${address.port}`;
  const browser = await chromium.launch({channel: 'chrome', headless: true});

  async function destination(route, languages, storedLanguage) {
    const context = await browser.newContext({locale: languages[0]});
	await context.route('**/*', (route) => {
	  const target = route.request().url();
	  return target.startsWith(`${baseURL}/`) ? route.continue() : route.abort();
	});
    await context.addInitScript(({configuredLanguages, stored}) => {
      Object.defineProperty(navigator, 'languages', {get: () => configuredLanguages});
      Object.defineProperty(navigator, 'language', {get: () => configuredLanguages[0]});
      if (stored === null) {
        localStorage.removeItem('picfetch-language');
      } else {
        localStorage.setItem('picfetch-language', stored);
      }
    }, {configuredLanguages: languages, stored: storedLanguage});
    const page = await context.newPage();
	await page.goto(`${baseURL}${route}`, {waitUntil: 'domcontentloaded'});
    const result = new URL(page.url()).pathname;
    await context.close();
    return result;
  }

  try {
    for (const locale of ['de', 'de-DE', 'de-AT', 'de-CH']) {
      assertEqual(await destination('/', [locale], null), '/de/', `${locale} first visit`);
    }
    assertEqual(await destination('/', ['fr-FR'], null), '/', 'non-German first visit');
    assertEqual(await destination('/', ['en-GB', 'de-DE'], null), '/', 'ordered preference uses the first language');
    assertEqual(await destination('/', ['de-DE'], 'en'), '/', 'stored English overrides browser German');
    assertEqual(await destination('/', ['en-GB'], 'de'), '/de/', 'stored German overrides browser English');
    assertEqual(await destination('/de/', ['en-GB'], null), '/de/', 'explicit German route stays open');
    assertEqual(await destination('/amp/', ['de-DE'], null), '/amp/', 'English AMP route never redirects');
    assertEqual(await destination('/de/amp/', ['en-GB'], null), '/de/amp/', 'German AMP route never redirects');

    const context = await browser.newContext({locale: 'en-GB'});
	await context.route('**/*', (route) => {
	  const target = route.request().url();
	  return target.startsWith(`${baseURL}/`) ? route.continue() : route.abort();
	});
    const page = await context.newPage();
	await page.goto(`${baseURL}/`, {waitUntil: 'domcontentloaded'});
    const stored = await page.evaluate(() => {
      const germanLink = document.querySelector('[data-language="de"]');
      germanLink.addEventListener('click', (event) => event.preventDefault());
      germanLink.click();
      return localStorage.getItem('picfetch-language');
    });
    assertEqual(stored, 'de', 'manual German selection is stored before navigation');
    await context.close();

    process.stdout.write('browser language behavior: PASS\n');
  } finally {
    await browser.close();
    await new Promise((resolve) => server.close(resolve));
  }
}

main().catch((error) => {
  process.stderr.write(`browser language behavior: FAIL: ${error.message}\n`);
  process.exitCode = 1;
});
