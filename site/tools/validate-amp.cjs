'use strict';

const crypto = require('node:crypto');
const fs = require('node:fs');
const path = require('node:path');
const zlib = require('node:zlib');
const amphtmlValidator = require('amphtml-validator');

const compressedValidatorPath = path.join(__dirname, 'validator_wasm.js.gz');
const expectedCompressedSHA256 = '5ff8f38054cce8836b94e3a3f5ffa73a73b33c3bfc106e33118c93d527bfef56';
const expectedValidatorSHA256 = 'f1f4629ab36f555f5a9937a5cf00f0ef780bf025883c035d06389b151c896f26';

function sha256(value) {
  return crypto.createHash('sha256').update(value).digest('hex');
}

function fail(message) {
  process.stderr.write(`${message}\n`);
  process.exitCode = 1;
}

function configuredFilenames() {
  const root = process.env.SITE_OUTPUT_DIR || 'docs';
  const locales = (process.env.SITE_LOCALES || 'en,de').split(',').map(value => value.trim()).filter(Boolean);
  return locales.map(locale => {
    if (locale === 'en') return path.join(root, 'amp', 'index.html');
    if (locale === 'de') return path.join(root, 'de', 'amp', 'index.html');
    throw new Error(`unsupported AMP locale ${JSON.stringify(locale)}`);
  });
}

async function main() {
  const filenames = process.argv.length > 2 ? process.argv.slice(2) : configuredFilenames();
  if (filenames.length === 0) {
    throw new Error('no AMP locales or filenames were provided');
  }

  const compressed = fs.readFileSync(compressedValidatorPath);
  const compressedHash = sha256(compressed);
  if (compressedHash !== expectedCompressedSHA256) {
    throw new Error(`pinned AMP validator archive checksum mismatch: got ${compressedHash}`);
  }

  const validatorSource = zlib.gunzipSync(compressed);
  const validatorHash = sha256(validatorSource);
  if (validatorHash !== expectedValidatorSHA256) {
    throw new Error(`pinned AMP validator checksum mismatch: got ${validatorHash}`);
  }

  const validator = amphtmlValidator.newInstance(validatorSource.toString('utf8'));
  await validator.init();
  for (const filename of filenames) {
    let input;
    try {
      input = fs.readFileSync(filename, 'utf8');
    } catch (error) {
      fail(`${filename}: ${error.message}`);
      continue;
    }
    const result = validator.validateString(input);
    process.stdout.write(`${filename}: ${result.status}\n`);
    for (const error of result.errors) {
      const location = `${filename}:${error.line}:${error.col}`;
      const reference = error.specUrl ? ` (see ${error.specUrl})` : '';
      const message = `${location} ${error.severity}: ${error.message}${reference}\n`;
      (error.severity === 'ERROR' ? process.stderr : process.stdout).write(message);
    }
    if (result.status !== 'PASS') {
      process.exitCode = 1;
    }
  }
}

main().catch((error) => fail(`pinned AMP validation failed: ${error.message}`));
