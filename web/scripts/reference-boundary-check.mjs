#!/usr/bin/env node

import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { spawnSync } from 'node:child_process';

const WEB_ROOT = process.cwd();
const REFERENCE_REPO = resolve(WEB_ROOT, '../v2rayN');

if (!existsSync(REFERENCE_REPO)) {
    process.stdout.write(`[SKIP] reference repo not found: ${REFERENCE_REPO}\n`);
    process.exit(0);
}

const result = spawnSync('git', ['status', '--porcelain'], {
    cwd: REFERENCE_REPO,
    encoding: 'utf-8'
});

if (result.error) {
    process.stderr.write(`[ERROR] failed to run git status in ${REFERENCE_REPO}: ${result.error.message}\n`);
    process.exit(2);
}

if (result.status !== 0) {
    process.stderr.write(`[ERROR] git status exited with code ${result.status}\n`);
    if (result.stderr) {
        process.stderr.write(result.stderr);
    }
    process.exit(result.status ?? 2);
}

const output = (result.stdout || '').trim();
if (output.length > 0) {
    process.stderr.write('[FAIL] reference boundary violated: v2rayN has uncommitted changes.\n');
    process.stderr.write(`${output}\n`);
    process.stderr.write('Please keep v2rayN as read-only reference and commit/revert those changes first.\n');
    process.exit(1);
}

process.stdout.write('[PASS] reference boundary check: v2rayN is clean.\n');
