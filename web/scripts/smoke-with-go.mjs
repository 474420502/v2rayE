#!/usr/bin/env node

import { spawn } from 'node:child_process';

const ORIGIN = process.env.V2RAYN_API_ORIGIN || 'http://127.0.0.1:18000';
const API_ADDR = process.env.V2RAYN_API_ADDR || '127.0.0.1:18000';

function sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}

async function waitForHealth(maxAttempts = 60, intervalMs = 150) {
    for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
        try {
            const response = await fetch(`${ORIGIN}/api/health`);
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

async function run() {
    let backend = null;
    let startedByScript = false;

    const alreadyReady = await waitForHealth(3, 100);
    if (!alreadyReady) {
        backend = spawn('go', ['run', './cmd/server'], {
            cwd: '../backend-go',
            stdio: 'inherit',
            env: {
                ...process.env,
                V2RAYN_API_ADDR: API_ADDR
            }
        });
        startedByScript = true;
    }

    const ready = await waitForHealth();
    if (!ready) {
        process.stderr.write('[smoke-with-go] go api not ready\n');
        if (startedByScript && backend) {
            backend.kill('SIGTERM');
        }
        process.exit(1);
        return;
    }

    const smoke = spawn('node', ['./scripts/api-smoke.mjs'], {
        stdio: 'inherit',
        env: {
            ...process.env,
            V2RAYN_API_ORIGIN: ORIGIN,
            V2RAYN_ENABLE_WRITE: process.env.V2RAYN_ENABLE_WRITE || '1'
        }
    });

    smoke.on('exit', (code) => {
        if (startedByScript && backend) {
            backend.kill('SIGTERM');
        }
        process.exit(code ?? 1);
    });
}

run().catch((error) => {
    process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`);
    process.exit(1);
});
