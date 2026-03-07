#!/usr/bin/env node

import { spawn } from 'node:child_process';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';

const API_PORT = process.env.V2RAYN_STATE_PORT || String(18300 + Math.floor(Math.random() * 200));
const API_ORIGIN = `http://127.0.0.1:${API_PORT}`;
const WORK_DIR = mkdtempSync(join(tmpdir(), 'v2raye-state-check-'));
const STATE_PATH = join(WORK_DIR, 'native-state.json');
const STALE_CONFIG = join(WORK_DIR, 'stale-runtime.json');

function sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}

async function waitForHealth(maxAttempts = 100, intervalMs = 100) {
    for (let i = 0; i < maxAttempts; i += 1) {
        try {
            const response = await fetch(`${API_ORIGIN}/api/health`);
            if (response.ok) return true;
        } catch {
            // retry
        }
        await sleep(intervalMs);
    }
    return false;
}

async function api(path, method = 'GET', body = undefined) {
    const response = await fetch(`${API_ORIGIN}${path}`, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body
    });
    const text = await response.text();
    const parsed = text ? JSON.parse(text) : null;
    if (!response.ok || !parsed || parsed.code !== 0) {
        throw new Error(`${method} ${path} failed: HTTP=${response.status} body=${text}`);
    }
    return parsed.data;
}

function startBackend() {
    return spawn('go', ['run', './cmd/server'], {
        cwd: '../backend-go',
        stdio: 'inherit',
        env: {
            ...process.env,
            V2RAYN_BACKEND_MODE: 'native',
            V2RAYN_API_ADDR: `127.0.0.1:${API_PORT}`,
            V2RAYN_NATIVE_STATE_PATH: STATE_PATH
        }
    });
}

async function stopBackend(proc) {
    if (!proc || proc.killed) return;
    proc.kill('SIGTERM');
    await new Promise((resolve) => {
        const timer = setTimeout(() => {
            try {
                proc.kill('SIGKILL');
            } catch {
                // ignore
            }
            resolve();
        }, 1500);
        proc.once('exit', () => {
            clearTimeout(timer);
            resolve();
        });
    });
}

function prepareDirtyState() {
    writeFileSync(STALE_CONFIG, '{"stale":true}\n', 'utf-8');
    const dirty = {
        running: true,
        coreType: 'xray',
        currentProfileId: 'p1',
        pid: 999999,
        lastConfigPath: STALE_CONFIG,
        updatedAt: new Date().toISOString()
    };
    writeFileSync(STATE_PATH, JSON.stringify(dirty, null, 2), 'utf-8');
}

function readState() {
    return JSON.parse(readFileSync(STATE_PATH, 'utf-8'));
}

async function main() {
    process.stdout.write(`[INFO] API_ORIGIN=${API_ORIGIN}\n`);
    process.stdout.write(`[INFO] STATE_PATH=${STATE_PATH}\n`);

    prepareDirtyState();

    let backend = null;
    try {
        backend = startBackend();
        if (!(await waitForHealth())) {
            throw new Error('backend not ready');
        }

        const status = await api('/api/core/status');
        if (status?.running) {
            throw new Error('expected running=false after dirty-state recovery');
        }

        try {
            readFileSync(STALE_CONFIG, 'utf-8');
            throw new Error('stale runtime config file should be removed');
        } catch (err) {
            if (!(err && typeof err === 'object' && 'code' in err && err.code === 'ENOENT')) {
                throw err;
            }
        }

        const normalized = readState();
        if (normalized.running !== false) {
            throw new Error('normalized state should set running=false');
        }
        if (normalized.pid && normalized.pid !== 0) {
            throw new Error(`normalized state pid should be 0/empty, got ${normalized.pid}`);
        }

        process.stdout.write('[SUMMARY] PASS\n');
    } finally {
        await stopBackend(backend);
        try {
            rmSync(WORK_DIR, { recursive: true, force: true });
        } catch {
            // ignore
        }
    }
}

main().catch((err) => {
    process.stderr.write(`${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
});
