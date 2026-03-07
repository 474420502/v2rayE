#!/usr/bin/env node

import { existsSync, rmSync } from 'node:fs';
import { spawn } from 'node:child_process';

const API_PORT = process.env.V2RAYN_CORECMD_PORT || '18005';
const API_ORIGIN = `http://127.0.0.1:${API_PORT}`;
const MARK_FILE = '/tmp/v2raye-corecmd-injected';
const UNSAFE_CMD = `echo safe; touch ${MARK_FILE}`;

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

async function api(path, method = 'GET', body = '{}') {
    const response = await fetch(`${API_ORIGIN}${path}`, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: method === 'GET' ? undefined : body
    });
    const text = await response.text();
    const payload = text ? JSON.parse(text) : null;
    if (!response.ok || !payload || payload.code !== 0) {
        throw new Error(`${method} ${path} failed: HTTP=${response.status} body=${text}`);
    }
    return payload.data;
}

async function main() {
    if (existsSync(MARK_FILE)) {
        rmSync(MARK_FILE, { force: true });
    }

    const backend = spawn('go', ['run', './cmd/server'], {
        cwd: '../backend-go',
        stdio: 'inherit',
        env: {
            ...process.env,
            V2RAYN_API_ADDR: `127.0.0.1:${API_PORT}`,
            V2RAYN_DATA_DIR: '/tmp/v2raye-corecmd-check',
            V2RAYN_XRAY_CMD: UNSAFE_CMD
        }
    });

    try {
        const ready = await waitForHealth();
        if (!ready) {
            throw new Error('corecmd security backend not ready');
        }

        const profile = await api('/api/profiles', 'POST', JSON.stringify({
            name: 'Security-Test',
            protocol: 'vmess',
            address: 'example.com',
            port: 443,
            vmess: {
                uuid: '44444444-4444-4444-4444-444444444444',
                alterId: 0,
                security: 'auto'
            }
        }));
        await api(`/api/profiles/${profile.id}/select`, 'POST', '{}');

        await api('/api/core/start', 'POST', '{}');
        const status = await api('/api/core/status', 'GET');
        if (status?.running) {
            throw new Error('unsafe core command should not start a running process');
        }

        if (existsSync(MARK_FILE)) {
            throw new Error('unsafe core command was executed unexpectedly');
        }

        await api('/api/core/stop', 'POST', '{}');
        process.stdout.write('[SUMMARY] PASS\n');
    } finally {
        backend.kill('SIGTERM');
        if (existsSync(MARK_FILE)) {
            rmSync(MARK_FILE, { force: true });
        }
    }
}

main().catch((err) => {
    process.stderr.write(`${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
});
