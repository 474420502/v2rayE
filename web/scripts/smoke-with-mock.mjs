#!/usr/bin/env node

import { spawn } from 'node:child_process';

const ORIGIN = process.env.V2RAYN_API_ORIGIN || 'http://127.0.0.1:18000';

function sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}

async function waitForHealth(maxAttempts = 40, intervalMs = 100) {
    for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
        try {
            const response = await fetch(`${ORIGIN}/api/health`);
            if (response.ok) {
                return true;
            }
        } catch {
            // ignore and retry
        }
        await sleep(intervalMs);
    }
    return false;
}

async function run() {
    const mock = spawn('node', ['./scripts/mock-api.mjs'], {
        stdio: 'inherit',
        env: {
            ...process.env,
            V2RAYN_MOCK_HOST: '127.0.0.1',
            V2RAYN_MOCK_PORT: '18000'
        }
    });

    const ready = await waitForHealth();
    if (!ready) {
        process.stderr.write('[smoke-with-mock] mock api not ready\n');
        mock.kill('SIGTERM');
        process.exit(1);
        return;
    }

    const smoke = spawn('node', ['./scripts/api-smoke.mjs'], {
        stdio: 'inherit',
        env: {
            ...process.env,
            V2RAYN_API_ORIGIN: ORIGIN,
            V2RAYN_ENABLE_WRITE: '1'
        }
    });

    smoke.on('exit', (code) => {
        mock.kill('SIGTERM');
        process.exit(code ?? 1);
    });
}

run().catch((error) => {
    process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`);
    process.exit(1);
});
