#!/usr/bin/env node

import { spawn } from 'node:child_process';
import { mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

const BASE_PORT = Number(process.env.V2RAYN_SEMANTIC_BASE_PORT || 18620);
const OUTPUT_DIR = process.env.V2RAYN_SEMANTIC_OUTPUT_DIR || './.artifacts/bridge-semantic';
const BRIDGE_CMD = process.env.V2RAYN_SERVICELIB_BRIDGE_CMD || 'node ./scripts/servicelib-bridge-mock.mjs';
const BRIDGE_TIMEOUT_MS = String(process.env.V2RAYN_SERVICELIB_BRIDGE_TIMEOUT_MS || '3000');
const SEMANTIC_MODE = (process.env.V2RAYN_SEMANTIC_MODE || 'shape').toLowerCase();

function sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}

async function waitForHealth(origin, attempts = 80, intervalMs = 120) {
    for (let i = 0; i < attempts; i += 1) {
        try {
            const response = await fetch(`${origin}/api/health`);
            if (response.ok) {
                return true;
            }
        } catch {
        }
        await sleep(intervalMs);
    }
    return false;
}

function startBackend(mode, port) {
    const env = {
        ...process.env,
        V2RAYN_API_ADDR: `127.0.0.1:${port}`,
        V2RAYN_BACKEND_MODE: mode
    };
    if (mode === 'servicelib-proxy') {
        env.V2RAYN_SERVICELIB_BRIDGE_CMD = BRIDGE_CMD;
        env.V2RAYN_SERVICELIB_BRIDGE_TIMEOUT_MS = BRIDGE_TIMEOUT_MS;
        env.V2RAYN_SERVICELIB_BRIDGE_ALLOW_ACTIONS = '*';
    }

    const proc = spawn('go', ['run', './cmd/server'], {
        cwd: '../backend-go',
        detached: true,
        stdio: ['ignore', 'pipe', 'pipe'],
        env
    });

    let output = '';
    proc.stdout.on('data', (chunk) => {
        output += String(chunk);
    });
    proc.stderr.on('data', (chunk) => {
        output += String(chunk);
    });

    return { proc, getOutput: () => output };
}

async function stopBackend(proc) {
    if (!proc || proc.killed || proc.exitCode !== null) {
        return;
    }

    const signalGroup = (signal) => {
        try {
            if (typeof proc.pid === 'number' && proc.pid > 0) {
                process.kill(-proc.pid, signal);
                return;
            }
        } catch {
        }
        try {
            proc.kill(signal);
        } catch {
        }
    };

    signalGroup('SIGTERM');
    await new Promise((resolve) => {
        const timer = setTimeout(() => {
            signalGroup('SIGKILL');
            resolve();
        }, 1500);
        proc.once('exit', () => {
            clearTimeout(timer);
            resolve();
        });
    });
}

async function request(origin, path, init = {}) {
    const response = await fetch(`${origin}${path}`, {
        ...init,
        headers: {
            'Content-Type': 'application/json',
            ...(init.headers || {})
        }
    });
    const text = await response.text();
    const body = text ? JSON.parse(text) : {};
    if (!response.ok || body?.code !== 0) {
        throw new Error(`${init.method || 'GET'} ${path} failed: ${response.status} ${text}`);
    }
    return body.data;
}

function shapeOf(value) {
    if (value === null) return 'null';
    if (Array.isArray(value)) {
        return {
            type: 'array',
            item: value.length > 0 ? shapeOf(value[0]) : 'empty'
        };
    }
    if (typeof value !== 'object') {
        return typeof value;
    }
    const keys = Object.keys(value).sort();
    const out = { type: 'object', fields: {} };
    for (const key of keys) {
        out.fields[key] = shapeOf(value[key]);
    }
    return out;
}

function isShapeCompatible(expected, actual) {
    if (typeof expected === 'string' || expected === null) {
        return JSON.stringify(expected) === JSON.stringify(actual);
    }

    if (!expected || !actual || expected.type !== actual.type) {
        return false;
    }

    if (expected.type === 'array') {
        return isShapeCompatible(expected.item, actual.item);
    }

    const expectedFields = expected.fields || {};
    const actualFields = actual.fields || {};
    for (const [key, value] of Object.entries(expectedFields)) {
        if (!(key in actualFields)) {
            return false;
        }
        if (!isShapeCompatible(value, actualFields[key])) {
            return false;
        }
    }
    return true;
}

async function collectScenario(mode, port) {
    const origin = `http://127.0.0.1:${port}`;
    const { proc, getOutput } = startBackend(mode, port);
    try {
        const ready = await waitForHealth(origin);
        if (!ready) {
            throw new Error(`${mode} backend not ready`);
        }

        const report = {};
        report.core_status_before = await request(origin, '/api/core/status');
        const profiles = await request(origin, '/api/profiles');
        report.profiles_list = profiles;

        const firstProfileId = Array.isArray(profiles) && profiles[0]?.id ? profiles[0].id : '';
        if (firstProfileId) {
            report.profile_delay = await request(origin, `/api/profiles/${firstProfileId}/delay`);
            report.profile_select = await request(origin, `/api/profiles/${firstProfileId}/select`, { method: 'POST', body: '{}' });
        }

        report.subscriptions_list = await request(origin, '/api/subscriptions');
        report.subscription_create = await request(origin, '/api/subscriptions', {
            method: 'POST',
            body: JSON.stringify({
                remarks: 'semantic-sub',
                url: 'https://example.com/semantic',
                enabled: true,
                userAgent: 'semantic-agent',
                filter: 'HK|JP',
                convertTarget: 'clash',
                autoUpdateMinutes: 30
            })
        });
        report.subscription_update = await request(origin, '/api/subscriptions/s1', {
            method: 'PUT',
            body: JSON.stringify({
                remarks: 'semantic-updated',
                url: 'https://example.com/semantic-updated',
                enabled: false,
                userAgent: 'semantic-updated-agent',
                filter: 'US',
                convertTarget: 'sing-box',
                autoUpdateMinutes: 15
            })
        });
        report.subscription_delete = await request(origin, '/api/subscriptions/s1', { method: 'DELETE' });
        report.network_availability = await request(origin, '/api/network/availability');
        report.config_get = await request(origin, '/api/config');
        report.config_update = await request(origin, '/api/config', {
            method: 'PUT',
            body: JSON.stringify({
                inbound: { enable: true, listen: '127.0.0.1', port: 10809, allowLan: true },
                tunModeItem: { enableTun: true, stackMixed: true, mtu: 1400 },
                coreBasicItem: { autoRun: true, logLevel: 'info', concurrency: 2, skipCertVerify: true, defUserAgent: 'semantic-agent' },
                systemProxyItem: { mode: 'forced_change', exceptions: 'localhost,127.0.0.1' }
            })
        });
        report.core_start = await request(origin, '/api/core/start', { method: 'POST', body: '{}' });

        report.subscriptions_update = await request(origin, '/api/subscriptions/update', { method: 'POST', body: '{}' });
        report.system_proxy_clear = await request(origin, '/api/system-proxy/apply', {
            method: 'POST',
            body: JSON.stringify({ mode: 'forced_clear', exceptions: '' })
        });
        report.core_stop = await request(origin, '/api/core/stop', { method: 'POST', body: '{}' });
        report.core_status_after = await request(origin, '/api/core/status');

        return {
            mode,
            origin,
            report,
            backendOutput: getOutput()
        };
    } finally {
        await stopBackend(proc);
    }
}

function compare(nativeReport, bridgeReport) {
    const allSteps = Array.from(new Set([...Object.keys(nativeReport), ...Object.keys(bridgeReport)])).sort();
    const mismatches = [];
    for (const step of allSteps) {
        const nativeShape = shapeOf(nativeReport[step]);
        const bridgeShape = shapeOf(bridgeReport[step]);
        const compatible = step === 'config_get' || step === 'subscriptions_list'
            ? isShapeCompatible(nativeShape, bridgeShape)
            : JSON.stringify(nativeShape) === JSON.stringify(bridgeShape);
        if (!compatible) {
            mismatches.push({
                step,
                nativeShape,
                bridgeShape
            });
        }
    }
    return {
        totalSteps: allSteps.length,
        mismatchCount: mismatches.length,
        mismatches
    };
}

function readNumber(value) {
    return typeof value === 'number' && Number.isFinite(value) ? value : 0;
}

function readBool(value) {
    return value === true;
}

function compareValue(nativeReport, bridgeReport) {
    const mismatches = [];

    const compareCoreStep = (step) => {
        const nativeCore = nativeReport[step] || {};
        const bridgeCore = bridgeReport[step] || {};
        const keys = ['running', 'state'];
        for (const key of keys) {
            if (nativeCore[key] !== bridgeCore[key]) {
                mismatches.push({
                    step,
                    field: key,
                    nativeValue: nativeCore[key],
                    bridgeValue: bridgeCore[key]
                });
            }
        }
    };

    compareCoreStep('core_status_before');
    compareCoreStep('core_start');
    compareCoreStep('core_stop');
    compareCoreStep('core_status_after');

    const nativeProfiles = Array.isArray(nativeReport.profiles_list) ? nativeReport.profiles_list : [];
    const bridgeProfiles = Array.isArray(bridgeReport.profiles_list) ? bridgeReport.profiles_list : [];
    if (nativeProfiles.length !== bridgeProfiles.length) {
        mismatches.push({
            step: 'profiles_list',
            field: 'length',
            nativeValue: nativeProfiles.length,
            bridgeValue: bridgeProfiles.length
        });
    }

    const nativeAvailability = nativeReport.network_availability || {};
    const bridgeAvailability = bridgeReport.network_availability || {};
    if (readBool(nativeAvailability.available) !== readBool(bridgeAvailability.available)) {
        mismatches.push({
            step: 'network_availability',
            field: 'available',
            nativeValue: nativeAvailability.available,
            bridgeValue: bridgeAvailability.available
        });
    }

    const nativeUpdate = nativeReport.subscriptions_update || {};
    const bridgeUpdate = bridgeReport.subscriptions_update || {};
    if (readNumber(nativeUpdate.updated) !== readNumber(bridgeUpdate.updated)) {
        mismatches.push({
            step: 'subscriptions_update',
            field: 'updated',
            nativeValue: nativeUpdate.updated,
            bridgeValue: bridgeUpdate.updated
        });
    }

    const nativeConfig = nativeReport.config_update || {};
    const bridgeConfig = bridgeReport.config_update || {};
    for (const field of ['inbound', 'tunModeItem', 'coreBasicItem', 'systemProxyItem']) {
        if (JSON.stringify(shapeOf(nativeConfig[field])) !== JSON.stringify(shapeOf(bridgeConfig[field]))) {
            mismatches.push({
                step: 'config_update',
                field,
                nativeValue: nativeConfig[field],
                bridgeValue: bridgeConfig[field]
            });
        }
    }

    return {
        mode: 'value',
        totalChecks: 13,
        mismatchCount: mismatches.length,
        mismatches
    };
}

async function main() {
    const startedAt = new Date().toISOString();
    const nativePort = BASE_PORT;
    const bridgePort = BASE_PORT + 1;

    process.stdout.write(`[INFO] semantic check base port: ${BASE_PORT}\n`);
    process.stdout.write(`[INFO] bridge command: ${BRIDGE_CMD}\n`);

    const nativeResult = await collectScenario('native', nativePort);
    const bridgeResult = await collectScenario('servicelib-proxy', bridgePort);
    const summary = SEMANTIC_MODE === 'value'
        ? compareValue(nativeResult.report, bridgeResult.report)
        : compare(nativeResult.report, bridgeResult.report);

    const report = {
        startedAt,
        finishedAt: new Date().toISOString(),
        basePort: BASE_PORT,
        bridgeCommand: BRIDGE_CMD,
        bridgeTimeoutMs: Number(BRIDGE_TIMEOUT_MS),
        mode: SEMANTIC_MODE,
        ok: summary.mismatchCount === 0,
        summary,
        native: nativeResult.report,
        bridge: bridgeResult.report
    };

    mkdirSync(OUTPUT_DIR, { recursive: true });
    const stamp = startedAt.replace(/[:.]/g, '-');
    const outputPath = join(OUTPUT_DIR, `bridge-semantic-${stamp}.json`);
    writeFileSync(outputPath, `${JSON.stringify(report, null, 2)}\n`, 'utf-8');

    const total = summary.totalSteps ?? summary.totalChecks ?? 0;
    process.stdout.write(`[SUMMARY] ${report.ok ? 'PASS' : 'FAIL'} mode=${SEMANTIC_MODE} mismatches=${summary.mismatchCount}/${total}\n`);
    if (summary.mismatches.length > 0) {
        for (const item of summary.mismatches) {
            process.stdout.write(`[MISMATCH] ${item.step}\n`);
        }
    }
    process.stdout.write(`[ARTIFACT] ${outputPath}\n`);

    if (!report.ok) {
        process.exitCode = 1;
    }
}

main().catch((error) => {
    process.stderr.write(`${error instanceof Error ? error.stack || error.message : String(error)}\n`);
    process.exit(1);
});
