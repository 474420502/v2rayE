#!/usr/bin/env node

import { spawn } from 'node:child_process';
import { mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

const BASE_PORT = Number(process.env.V2RAYN_DRILL_BASE_PORT || 18400 + Math.floor(Math.random() * 200));
const BRIDGE_CMD = process.env.V2RAYN_SERVICELIB_BRIDGE_CMD || 'node ./scripts/servicelib-bridge-mock.mjs';
const TIMEOUT_MS = String(process.env.V2RAYN_SERVICELIB_BRIDGE_TIMEOUT_MS || '3000');
const SMOKE_TIMEOUT_MS = Number(process.env.V2RAYN_DRILL_SMOKE_TIMEOUT_MS || 60000);
const SCENARIO_TIMEOUT_MS = Number(process.env.V2RAYN_DRILL_SCENARIO_TIMEOUT_MS || 90000);
const OUTPUT_DIR = process.env.V2RAYN_DRILL_OUTPUT_DIR || './.artifacts/bridge-drill';

const SCENARIOS = [
    {
        name: 'minimal-allowlist',
        allowActions: 'core.status,core.start,core.stop,core.restart,config.get,config.update'
    },
    {
        name: 'all-actions',
        allowActions: '*'
    }
];

function sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}

async function waitForHealth(origin, maxAttempts = 100, intervalMs = 100) {
    for (let i = 0; i < maxAttempts; i += 1) {
        try {
            const response = await fetch(`${origin}/api/health`);
            if (response.ok) return true;
        } catch {
            // retry
        }
        await sleep(intervalMs);
    }
    return false;
}

function startBackend(port, allowActions) {
    const backend = spawn('go', ['run', './cmd/server'], {
        cwd: '../backend-go',
        detached: true,
        stdio: ['ignore', 'pipe', 'pipe'],
        env: {
            ...process.env,
            V2RAYN_API_ADDR: `127.0.0.1:${port}`,
            V2RAYN_BACKEND_MODE: 'servicelib-proxy',
            V2RAYN_SERVICELIB_BRIDGE_CMD: BRIDGE_CMD,
            V2RAYN_SERVICELIB_BRIDGE_TIMEOUT_MS: TIMEOUT_MS,
            V2RAYN_SERVICELIB_BRIDGE_ALLOW_ACTIONS: allowActions,
            V2RAYN_SERVICELIB_BRIDGE_METRICS_LOG: '1'
        }
    });

    let output = '';
    backend.stdout.on('data', (chunk) => {
        output += String(chunk);
    });
    backend.stderr.on('data', (chunk) => {
        output += String(chunk);
    });

    return { backend, getOutput: () => output };
}

function releaseBackendHandles(backend) {
    if (!backend) {
        return;
    }
    try {
        backend.stdout?.removeAllListeners('data');
        backend.stderr?.removeAllListeners('data');
        backend.stdout?.destroy();
        backend.stderr?.destroy();
        backend.removeAllListeners();
        backend.unref();
    } catch {
        // ignore
    }
}

async function stopBackend(backend) {
    if (!backend || backend.killed || backend.exitCode !== null) return;

    const signalGroup = (signal) => {
        try {
            if (typeof backend.pid === 'number' && backend.pid > 0) {
                process.kill(-backend.pid, signal);
            }
        } catch {
            try {
                backend.kill(signal);
            } catch {
                // ignore
            }
        }
    };

    try {
        signalGroup('SIGTERM');
    } catch {
        return;
    }
    await new Promise((resolve) => {
        const timer = setTimeout(() => {
            try {
                if (backend.exitCode === null) {
                    signalGroup('SIGKILL');
                }
            } catch {
                // ignore
            }
            resolve();
        }, 1500);
        backend.once('exit', () => {
            clearTimeout(timer);
            resolve();
        });
    });

    releaseBackendHandles(backend);
}

async function runSmoke(origin) {
    return new Promise((resolve, reject) => {
        const startedAt = Date.now();
        const proc = spawn('node', ['./scripts/api-smoke.mjs'], {
            stdio: ['ignore', 'pipe', 'pipe'],
            env: {
                ...process.env,
                V2RAYN_API_ORIGIN: origin,
                V2RAYN_ENABLE_WRITE: '1'
            }
        });

        let stdout = '';
        let stderr = '';
        proc.stdout.on('data', (chunk) => {
            stdout += String(chunk);
        });
        proc.stderr.on('data', (chunk) => {
            stderr += String(chunk);
        });

        const timeout = setTimeout(() => {
            try {
                proc.kill('SIGTERM');
            } catch {
                // ignore
            }
            setTimeout(() => {
                try {
                    if (proc.exitCode === null) {
                        proc.kill('SIGKILL');
                    }
                } catch {
                    // ignore
                }
            }, 1200);
        }, SMOKE_TIMEOUT_MS);

        proc.on('exit', (code) => {
            clearTimeout(timeout);
            const durationMs = Date.now() - startedAt;
            const timeoutHit = durationMs >= SMOKE_TIMEOUT_MS && (code === null || code === 143 || code === 137);
            resolve({ code: timeoutHit ? 124 : code ?? 1, stdout, stderr, durationMs, timeoutHit });
        });

        proc.on('error', reject);
    });
}

function extractReport(stdout) {
    const match = stdout.match(/\{\s*"startedAt"[\s\S]*"ok"\s*:\s*(true|false)\s*\}\s*$/m);
    if (!match) return null;
    try {
        return JSON.parse(match[0]);
    } catch {
        return null;
    }
}

function countBridgeFallbackLogs(text) {
    const matches = text.match(/\[servicelib-proxy\] bridge action=.*fallback enabled:/g);
    return matches ? matches.length : 0;
}

function collectFallbackActions(text) {
    const actionCounts = {};
    const regex = /\[servicelib-proxy\] bridge action=([a-zA-Z0-9._-]+) failed, fallback enabled:/g;
    let match = regex.exec(text);
    while (match) {
        const action = match[1];
        actionCounts[action] = (actionCounts[action] || 0) + 1;
        match = regex.exec(text);
    }
    return actionCounts;
}

function collectBridgeFailureReasons(text) {
    const reasonCounts = {};
    const regex = /\[servicelib-proxy\] bridge action=[a-zA-Z0-9._-]+ failed, fallback enabled: reason=([a-z0-9_\-]+)/g;
    let match = regex.exec(text);
    while (match) {
        const reason = match[1];
        reasonCounts[reason] = (reasonCounts[reason] || 0) + 1;
        match = regex.exec(text);
    }
    return reasonCounts;
}

function collectBridgeLatencyBuckets(text) {
    const bucketCounts = {};
    const elapsedMs = [];
    const regex = /\[servicelib-proxy\] bridge action=[a-zA-Z0-9._-]+ (?:ok|failed, fallback enabled: [^\n]*?)elapsedMs=(\d+) bucket=([a-z0-9_\-]+)/g;
    let match = regex.exec(text);
    while (match) {
        const elapsed = Number(match[1]);
        const bucket = match[2];
        bucketCounts[bucket] = (bucketCounts[bucket] || 0) + 1;
        if (Number.isFinite(elapsed)) {
            elapsedMs.push(elapsed);
        }
        match = regex.exec(text);
    }
    return { bucketCounts, elapsedMs };
}

function percentile(values, q) {
    if (!Array.isArray(values) || values.length === 0) {
        return null;
    }
    const sorted = [...values].sort((a, b) => a - b);
    const idx = Math.min(sorted.length - 1, Math.max(0, Math.ceil((q / 100) * sorted.length) - 1));
    return sorted[idx];
}

function latencyBucketFromMs(elapsedMs) {
    if (elapsedMs < 100) {
        return 'lt100ms';
    }
    if (elapsedMs < 300) {
        return '100to299ms';
    }
    if (elapsedMs < 1000) {
        return '300to999ms';
    }
    return 'ge1000ms';
}

function collectStepLatencyFromReport(report) {
    const values = Array.isArray(report?.steps)
        ? report.steps.map((step) => Number(step?.elapsed)).filter((elapsed) => Number.isFinite(elapsed) && elapsed >= 0)
        : [];
    const bucketCounts = {};
    for (const elapsed of values) {
        const bucket = latencyBucketFromMs(elapsed);
        bucketCounts[bucket] = (bucketCounts[bucket] || 0) + 1;
    }
    return { elapsedMs: values, bucketCounts };
}

async function runScenario(index, scenario) {
    const port = BASE_PORT + index;
    const origin = `http://127.0.0.1:${port}`;
    const { backend, getOutput } = startBackend(port, scenario.allowActions);

    try {
        const execute = async () => {
            const ready = await waitForHealth(origin);
            if (!ready) {
                throw new Error(`backend not ready for ${scenario.name}`);
            }

            const smoke = await runSmoke(origin);
            const backendOutput = getOutput();
            const report = extractReport(smoke.stdout);
            const totalStepElapsedMs = report?.steps?.reduce((acc, step) => acc + (step?.elapsed || 0), 0) ?? null;
            const fallbackLogCount = countBridgeFallbackLogs(backendOutput);
            const fallbackActions = collectFallbackActions(backendOutput);
            const fallbackReasons = collectBridgeFailureReasons(backendOutput);
            const latency = collectBridgeLatencyBuckets(backendOutput);
            const latencySource = latency.elapsedMs.length > 0 ? latency : collectStepLatencyFromReport(report);

            return {
                scenario: scenario.name,
                allowActions: scenario.allowActions,
                origin,
                ok: smoke.code === 0,
                smokeExitCode: smoke.code,
                totalStepElapsedMs,
                wallDurationMs: smoke.durationMs,
                fallbackLogCount,
                fallbackActions,
                fallbackUniqueActions: Object.keys(fallbackActions).length,
                fallbackReasons,
                fallbackUniqueReasons: Object.keys(fallbackReasons).length,
                bridgeLatencyBuckets: latencySource.bucketCounts,
                bridgeLatencyCount: latencySource.elapsedMs.length,
                bridgeLatencyP95Ms: percentile(latencySource.elapsedMs, 95),
                bridgeLatencyP99Ms: percentile(latencySource.elapsedMs, 99),
                steps: report?.steps?.length ?? 0,
                timeoutHit: Boolean(smoke.timeoutHit)
            };
        };

        let scenarioTimeoutId;
        try {
            return await Promise.race([
                execute(),
                new Promise((_, reject) => {
                    scenarioTimeoutId = setTimeout(
                        () => reject(new Error(`scenario timeout: ${scenario.name} exceeded ${SCENARIO_TIMEOUT_MS}ms`)),
                        SCENARIO_TIMEOUT_MS
                    );
                })
            ]);
        } finally {
            if (scenarioTimeoutId) {
                clearTimeout(scenarioTimeoutId);
            }
        }
    } finally {
        await stopBackend(backend);
        releaseBackendHandles(backend);
    }
}

async function main() {
    process.stdout.write(`[INFO] BASE_PORT=${BASE_PORT}\n`);
    process.stdout.write(`[INFO] BRIDGE_CMD=${BRIDGE_CMD}\n`);
    process.stdout.write(`[INFO] BRIDGE_TIMEOUT_MS=${TIMEOUT_MS}\n`);
    process.stdout.write(`[INFO] SMOKE_TIMEOUT_MS=${SMOKE_TIMEOUT_MS}\n`);
    process.stdout.write(`[INFO] SCENARIO_TIMEOUT_MS=${SCENARIO_TIMEOUT_MS}\n`);
    process.stdout.write(`[INFO] OUTPUT_DIR=${OUTPUT_DIR}\n`);

    const startedAt = new Date().toISOString();
    const samples = [];
    for (let i = 0; i < SCENARIOS.length; i += 1) {
        const sample = await runScenario(i, SCENARIOS[i]);
        samples.push(sample);
        process.stdout.write(`[SAMPLE] ${sample.scenario} ok=${sample.ok} stepElapsed=${sample.totalStepElapsedMs}ms wall=${sample.wallDurationMs}ms fallbackLogs=${sample.fallbackLogCount} p95=${sample.bridgeLatencyP95Ms ?? 'n/a'}ms\n`);
    }

    const ok = samples.every((item) => item.ok);
    const finishedAt = new Date().toISOString();
    const report = {
        ok,
        startedAt,
        finishedAt,
        basePort: BASE_PORT,
        bridgeCommand: BRIDGE_CMD,
        bridgeTimeoutMs: Number(TIMEOUT_MS),
        smokeTimeoutMs: SMOKE_TIMEOUT_MS,
        scenarioTimeoutMs: SCENARIO_TIMEOUT_MS,
        samples
    };

    mkdirSync(OUTPUT_DIR, { recursive: true });
    const stamp = startedAt.replace(/[:.]/g, '-');
    const outputPath = join(OUTPUT_DIR, `bridge-drill-${stamp}.json`);
    writeFileSync(outputPath, `${JSON.stringify(report, null, 2)}\n`, 'utf-8');

    process.stdout.write(`\n[SUMMARY] ${ok ? 'PASS' : 'FAIL'}\n`);
    process.stdout.write(`${JSON.stringify(report, null, 2)}\n`);
    process.stdout.write(`[ARTIFACT] ${outputPath}\n`);
    if (!ok) {
        process.exitCode = 1;
    }
}

main().catch((err) => {
    process.stderr.write(`${err instanceof Error ? err.stack || err.message : String(err)}\n`);
    process.exit(1);
});
