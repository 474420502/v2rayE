#!/usr/bin/env node

import { spawn } from 'node:child_process';

const API_PORT = process.env.V2RAYN_AUTH_PORT || '18002';
const API_ORIGIN = `http://127.0.0.1:${API_PORT}`;
const TOKEN = process.env.V2RAYN_AUTH_TOKEN || 'auth-smoke-token';

function sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}

async function waitForHealth(maxAttempts = 80, intervalMs = 100) {
    for (let i = 0; i < maxAttempts; i += 1) {
        try {
            const response = await fetch(`${API_ORIGIN}/api/health`);
            if (response.ok) {
                return true;
            }
        } catch {
            // retry
        }
        await sleep(intervalMs);
    }
    return false;
}

async function request(path, token) {
    const response = await fetch(`${API_ORIGIN}${path}`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {}
    });
    const text = await response.text();
    let body = null;
    try {
        body = text ? JSON.parse(text) : null;
    } catch {
        body = text;
    }
    return { status: response.status, body };
}

async function main() {
    const backend = spawn('go', ['run', './cmd/server'], {
        cwd: '../backend-go',
        stdio: 'inherit',
        env: {
            ...process.env,
            V2RAYN_API_ADDR: `127.0.0.1:${API_PORT}`,
            V2RAYN_API_TOKEN: TOKEN
        }
    });

    const cleanup = () => {
        backend.kill('SIGTERM');
    };

    try {
        const ready = await waitForHealth();
        if (!ready) {
            throw new Error('auth check backend not ready');
        }

        const unauthorized = await request('/api/core/status', '');
        if (unauthorized.status !== 401 || unauthorized.body?.code !== 40101) {
            throw new Error(`expected 40101 unauthorized, got status=${unauthorized.status} body=${JSON.stringify(unauthorized.body)}`);
        }

        const authorized = await request('/api/core/status', TOKEN);
        if (authorized.status !== 200 || authorized.body?.code !== 0) {
            throw new Error(`expected authorized success, got status=${authorized.status} body=${JSON.stringify(authorized.body)}`);
        }

        process.stdout.write('[SUMMARY] PASS\n');
    } finally {
        cleanup();
    }
}

main().catch((err) => {
    process.stderr.write(`${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
});
