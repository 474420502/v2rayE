#!/usr/bin/env node

import { spawn } from 'node:child_process';
import { rmSync } from 'node:fs';

const API_PORT = process.env.V2RAYN_RECOVERY_PORT || '18003';
const API_ORIGIN = `http://127.0.0.1:${API_PORT}`;
const STATE_PATH = process.env.V2RAYN_NATIVE_STATE_PATH || `/tmp/v2raye/native-state-recovery-${Date.now()}.json`;
const CORE_CMD = process.env.V2RAYN_CORE_CMD || 'sleep 60';
const TARGET_PROFILE = process.env.V2RAYN_RECOVERY_PROFILE || 'p2';

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
            V2RAYN_CORE_CMD: CORE_CMD,
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

async function main() {
    process.stdout.write(`[INFO] API_ORIGIN=${API_ORIGIN}\n`);
    process.stdout.write(`[INFO] STATE_PATH=${STATE_PATH}\n`);
    process.stdout.write(`[INFO] CORE_CMD=${CORE_CMD}\n`);

    let backend = null;
    try {
        backend = startBackend();
        if (!(await waitForHealth())) {
            throw new Error('backend not ready (phase 1)');
        }

        await api(`/api/profiles/${TARGET_PROFILE}/select`, 'POST', '{}');
        await api('/api/core/start', 'POST', '{}');

        const before = await api('/api/core/status');
        if (!before?.running) {
            throw new Error('expected running=true before restart');
        }

        await stopBackend(backend);
        backend = startBackend();

        if (!(await waitForHealth())) {
            throw new Error('backend not ready (phase 2)');
        }

        const after = await api('/api/core/status');
        if (!after?.running) {
            throw new Error('expected running=true after backend restart');
        }
        if (after?.currentProfileId !== TARGET_PROFILE) {
            throw new Error(`expected currentProfileId=${TARGET_PROFILE}, got ${after?.currentProfileId}`);
        }

        await api('/api/core/stop', 'POST', '{}');
        const finalStatus = await api('/api/core/status');
        if (finalStatus?.running) {
            throw new Error('expected running=false after stop');
        }

        process.stdout.write('[SUMMARY] PASS\n');
    } finally {
        await stopBackend(backend);
        try {
            rmSync(STATE_PATH, { force: true });
        } catch {
            // ignore
        }
    }
}

main().catch((err) => {
    process.stderr.write(`${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
});
