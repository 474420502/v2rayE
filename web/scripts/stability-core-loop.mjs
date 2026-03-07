#!/usr/bin/env node

const API_ORIGIN = process.env.V2RAYN_API_ORIGIN || 'http://127.0.0.1:18000';
const TOKEN = process.env.V2RAYN_API_TOKEN || '';
const LOOP = Number(process.env.V2RAYN_LOOP_COUNT || 100);

function url(path) {
    return `${API_ORIGIN}${path}`;
}

function authHeaders() {
    return TOKEN ? { Authorization: `Bearer ${TOKEN}` } : {};
}

async function api(path, method = 'GET', body = undefined) {
    const response = await fetch(url(path), {
        method,
        headers: {
            'Content-Type': 'application/json',
            ...authHeaders()
        },
        body
    });
    const text = await response.text();
    const parsed = text ? JSON.parse(text) : null;
    if (!response.ok || !parsed || parsed.code !== 0) {
        throw new Error(`${method} ${path} failed: HTTP=${response.status} body=${text}`);
    }
    return parsed.data;
}

async function main() {
    process.stdout.write(`[INFO] API_ORIGIN=${API_ORIGIN}\n`);
    process.stdout.write(`[INFO] LOOP=${LOOP}\n`);

    for (let i = 1; i <= LOOP; i += 1) {
        await api('/api/core/start', 'POST', '{}');
        const running = await api('/api/core/status', 'GET');
        if (!running?.running) {
            throw new Error(`loop ${i}: expected running=true after start`);
        }

        await api('/api/core/stop', 'POST', '{}');
        const stopped = await api('/api/core/status', 'GET');
        if (stopped?.running) {
            throw new Error(`loop ${i}: expected running=false after stop`);
        }

        if (i % 10 === 0 || i === LOOP) {
            process.stdout.write(`[OK] ${i}/${LOOP}\n`);
        }
    }

    process.stdout.write('[SUMMARY] PASS\n');
}

main().catch((err) => {
    process.stderr.write(`${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
});
