#!/usr/bin/env node

import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

const OUTPUT_DIR = process.env.V2RAYN_DRILL_OUTPUT_DIR || './.artifacts/bridge-drill';

function pickLatestArtifact(dir) {
    const files = readdirSync(dir, { withFileTypes: true })
        .filter((entry) => entry.isFile() && entry.name.startsWith('bridge-drill-') && entry.name.endsWith('.json'))
        .map((entry) => entry.name)
        .sort();

    if (files.length === 0) {
        return null;
    }
    return join(dir, files[files.length - 1]);
}

function num(value) {
    return typeof value === 'number' ? value : 0;
}

function findScenario(samples, name) {
    return Array.isArray(samples) ? samples.find((item) => item?.scenario === name) ?? null : null;
}

function printScenario(label, item) {
    if (!item) {
        process.stdout.write(`[${label}] missing\n`);
        return;
    }
    process.stdout.write(
        `[${label}] ok=${item.ok} wall=${num(item.wallDurationMs)}ms stepElapsed=${num(item.totalStepElapsedMs)}ms fallbackLogs=${num(item.fallbackLogCount)} timeout=${Boolean(item.timeoutHit)}\n`
    );

    const actionCounts = item?.fallbackActions && typeof item.fallbackActions === 'object' ? item.fallbackActions : {};
    const entries = Object.entries(actionCounts)
        .filter(([, count]) => Number(count) > 0)
        .sort((a, b) => Number(b[1]) - Number(a[1]));

    if (entries.length > 0) {
        const top = entries.slice(0, 5).map(([name, count]) => `${name}:${count}`).join(', ');
        process.stdout.write(`[${label}:fallback-actions] ${top}\n`);
    }
}

function main() {
    let artifactPath = process.env.V2RAYN_DRILL_ARTIFACT || '';
    if (!artifactPath) {
        artifactPath = pickLatestArtifact(OUTPUT_DIR) || '';
    }

    if (!artifactPath) {
        process.stderr.write(`[bridge-whitelist-latest] no artifact found in ${OUTPUT_DIR}\n`);
        process.stderr.write('[bridge-whitelist-latest] run `npm run drill:bridge` first\n');
        process.exit(1);
        return;
    }

    const raw = readFileSync(artifactPath, 'utf-8');
    const report = JSON.parse(raw);
    const samples = Array.isArray(report?.samples) ? report.samples : [];
    const minimal = findScenario(samples, 'minimal-allowlist');
    const all = findScenario(samples, 'all-actions');

    process.stdout.write(`[ARTIFACT] ${artifactPath}\n`);
    process.stdout.write(`[RUN] ok=${Boolean(report?.ok)} startedAt=${report?.startedAt ?? 'n/a'} finishedAt=${report?.finishedAt ?? 'n/a'}\n`);

    printScenario('minimal-allowlist', minimal);
    printScenario('all-actions', all);

    if (minimal && all) {
        const wallDelta = num(all.wallDurationMs) - num(minimal.wallDurationMs);
        const stepDelta = num(all.totalStepElapsedMs) - num(minimal.totalStepElapsedMs);
        const fallbackDelta = num(all.fallbackLogCount) - num(minimal.fallbackLogCount);
        process.stdout.write(`[DELTA] wall=${wallDelta >= 0 ? '+' : ''}${wallDelta}ms stepElapsed=${stepDelta >= 0 ? '+' : ''}${stepDelta}ms fallbackLogs=${fallbackDelta >= 0 ? '+' : ''}${fallbackDelta}\n`);
    }

    if (!report?.ok) {
        process.exitCode = 1;
    }
}

try {
    main();
} catch (err) {
    process.stderr.write(`${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
}
