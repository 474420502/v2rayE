#!/usr/bin/env node

import { spawn } from 'node:child_process';

const API_PORT = process.env.V2RAYN_SUBRES_PORT || '18004';
const API_ORIGIN = `http://127.0.0.1:${API_PORT}`;
const INVALID_SUB_ID = process.env.V2RAYN_INVALID_SUB_ID || 'not-exists';

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

async function request(path, method = 'GET', body = undefined) {
    const response = await fetch(`${API_ORIGIN}${path}`, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body
    });
    const text = await response.text();
    let parsed = null;
    try {
        parsed = text ? JSON.parse(text) : null;
    } catch {
        parsed = text;
    }
    return { status: response.status, body: parsed, raw: text };
}

async function api(path, method = 'GET', body = undefined) {
    const res = await request(path, method, body);
    if (res.status !== 200 || !res.body || res.body.code !== 0) {
        throw new Error(`${method} ${path} failed: HTTP=${res.status} body=${res.raw}`);
    }
    return res.body.data;
}

function deepEqual(a, b) {
    return JSON.stringify(a) === JSON.stringify(b);
}

function startBackend() {
    return spawn('go', ['run', './cmd/server'], {
        cwd: '../backend-go',
        stdio: 'inherit',
        env: {
            ...process.env,
            V2RAYN_BACKEND_MODE: 'native',
            V2RAYN_API_ADDR: `127.0.0.1:${API_PORT}`
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
    let backend = null;

    try {
        backend = startBackend();
        if (!(await waitForHealth())) {
            throw new Error('backend not ready');
        }

        const before = await api('/api/subscriptions');
        const failed = await request(`/api/subscriptions/${INVALID_SUB_ID}/update`, 'POST', '{}');
        if (failed.status !== 404 || failed.body?.code !== 40401) {
            throw new Error(`expected 40401 on invalid update, got status=${failed.status} body=${JSON.stringify(failed.body)}`);
        }

        const after = await api('/api/subscriptions');
        if (!deepEqual(before, after)) {
            throw new Error('subscription list changed after failed update');
        }

        process.stdout.write('[SUMMARY] PASS\n');
    } finally {
        await stopBackend(backend);
    }
}

main().catch((err) => {
    process.stderr.write(`${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
});
