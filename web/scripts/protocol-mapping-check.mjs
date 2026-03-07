#!/usr/bin/env node

import { spawn } from 'node:child_process';
import { chmodSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';

const API_PORT = process.env.V2RAYN_PROTO_PORT || String(18100 + Math.floor(Math.random() * 200));
const API_ORIGIN = `http://127.0.0.1:${API_PORT}`;
const WORK_DIR = mkdtempSync(join(tmpdir(), 'v2raye-proto-check-'));
const OUT_DIR = join(WORK_DIR, 'out');
const OUTPUT_PATH = join(OUT_DIR, 'last.json');
const XRAY_WRAPPER = join(WORK_DIR, 'xray-wrapper.sh');

const CASES = [
    {
        protocol: 'vmess',
        expected: 'vmess',
        profile: {
            name: 'VMess-01',
            protocol: 'vmess',
            address: 'vmess.example.com',
            port: 443,
            vmess: {
                uuid: '11111111-1111-1111-1111-111111111111',
                alterId: 0,
                security: 'auto'
            }
        }
    },
    {
        protocol: 'vless',
        expected: 'vless',
        profile: {
            name: 'VLESS-01',
            protocol: 'vless',
            address: 'vless.example.com',
            port: 443,
            vless: {
                uuid: '22222222-2222-2222-2222-222222222222',
                encryption: 'none'
            }
        }
    },
    {
        protocol: 'shadowsocks',
        expected: 'shadowsocks',
        profile: {
            name: 'SS-01',
            protocol: 'shadowsocks',
            address: 'ss.example.com',
            port: 8388,
            shadowsocks: {
                method: 'aes-128-gcm',
                password: 'secret-pass'
            }
        }
    },
    {
        protocol: 'trojan',
        expected: 'trojan',
        profile: {
            name: 'Trojan-01',
            protocol: 'trojan',
            address: 'trojan.example.com',
            port: 443,
            trojan: {
                password: 'secret-pass'
            }
        }
    },
    {
        protocol: 'hysteria2',
        expected: 'hysteria2',
        profile: {
            name: 'Hysteria2-01',
            protocol: 'hysteria2',
            address: 'hy2.example.com',
            port: 443,
            hysteria2: {
                password: 'secret-pass',
                sni: 'hy2.example.com'
            }
        }
    },
    {
        protocol: 'tuic',
        expected: 'tuic',
        profile: {
            name: 'TUIC-01',
            protocol: 'tuic',
            address: 'tuic.example.com',
            port: 443,
            tuic: {
                uuid: '33333333-3333-3333-3333-333333333333',
                password: 'secret-pass'
            }
        }
    }
];

function createXrayWrapper() {
    writeFileSync(
        XRAY_WRAPPER,
        `#!/usr/bin/env bash
set -euo pipefail
cp "$3" "${OUTPUT_PATH}"
trap 'exit 0' TERM INT
while true; do
    sleep 1
done
`
    );
    chmodSync(XRAY_WRAPPER, 0o755);
}

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
            V2RAYN_API_ADDR: `127.0.0.1:${API_PORT}`,
            V2RAYN_DATA_DIR: WORK_DIR,
            V2RAYN_XRAY_CMD: XRAY_WRAPPER
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

function readRuntimeConfig() {
    const content = readFileSync(OUTPUT_PATH, 'utf-8');
    return JSON.parse(content);
}

async function waitForOutput(maxAttempts = 30, intervalMs = 40) {
    for (let i = 0; i < maxAttempts; i += 1) {
        if (existsSync(OUTPUT_PATH)) {
            return true;
        }
        await sleep(intervalMs);
    }
    return false;
}

async function prepareCaseProfile(caseItem) {
    const existing = await api('/api/profiles');
    if (existing.length > 0) {
        await api('/api/profiles/delete', 'POST', JSON.stringify({ ids: existing.map((item) => item.id) }));
    }
    const created = await api('/api/profiles', 'POST', JSON.stringify(caseItem.profile));
    await api(`/api/profiles/${created.id}/select`, 'POST', '{}');
    return created;
}

async function main() {
    process.stdout.write(`[INFO] API_ORIGIN=${API_ORIGIN}\n`);
    process.stdout.write(`[INFO] WORK_DIR=${WORK_DIR}\n`);

    let backend = null;
    try {
        mkdirSync(OUT_DIR, { recursive: true });
        createXrayWrapper();
        backend = startBackend();
        if (!(await waitForHealth())) {
            throw new Error('backend not ready');
        }

        for (const item of CASES) {
            await prepareCaseProfile(item);
            rmSync(OUTPUT_PATH, { force: true });
            await api('/api/core/start', 'POST', '{}');
            const outputReady = await waitForOutput();
            if (!outputReady) {
                throw new Error(`generated config not found for protocol ${item.protocol}`);
            }

            const generated = readRuntimeConfig();
            const protocol = generated?.outbounds?.[0]?.protocol;
            if (protocol !== item.expected) {
                throw new Error(`protocol=${item.protocol} expected=${item.expected} got outbound=${protocol}`);
            }

            await api('/api/core/stop', 'POST', '{}');
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
