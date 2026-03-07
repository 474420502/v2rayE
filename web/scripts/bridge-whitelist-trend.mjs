#!/usr/bin/env node

import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

const OUTPUT_DIR = process.env.V2RAYN_DRILL_OUTPUT_DIR || './.artifacts/bridge-drill';
const LIMIT = Number(process.env.V2RAYN_DRILL_TREND_LIMIT || 20);

function toNumber(value) {
    return typeof value === 'number' && Number.isFinite(value) ? value : 0;
}

function percentile(values, p) {
    if (!values.length) return 0;
    const sorted = [...values].sort((a, b) => a - b);
    const idx = Math.max(0, Math.min(sorted.length - 1, Math.ceil((p / 100) * sorted.length) - 1));
    return sorted[idx];
}

function avg(values) {
    if (!values.length) return 0;
    return values.reduce((a, b) => a + b, 0) / values.length;
}

function pickArtifacts() {
    const files = readdirSync(OUTPUT_DIR, { withFileTypes: true })
        .filter((entry) => entry.isFile() && entry.name.startsWith('bridge-drill-') && entry.name.endsWith('.json'))
        .map((entry) => entry.name)
        .sort();

    const selected = files.slice(Math.max(0, files.length - LIMIT));
    return selected.map((name) => join(OUTPUT_DIR, name));
}

function parseArtifact(filePath) {
    const raw = readFileSync(filePath, 'utf-8');
    const report = JSON.parse(raw);
    const samples = Array.isArray(report?.samples) ? report.samples : [];
    const minimal = samples.find((item) => item?.scenario === 'minimal-allowlist') || null;
    const all = samples.find((item) => item?.scenario === 'all-actions') || null;
    return {
        filePath,
        ok: Boolean(report?.ok),
        startedAt: report?.startedAt || '',
        minimal,
        all
    };
}

function metricSet(items, field) {
    return items.map((item) => toNumber(item?.[field]));
}

function summarizeScenario(artifacts, scenarioKey) {
    const samples = artifacts
        .map((a) => (scenarioKey === 'minimal' ? a.minimal : a.all))
        .filter(Boolean);

    const wall = metricSet(samples, 'wallDurationMs');
    const step = metricSet(samples, 'totalStepElapsedMs');
    const fallback = metricSet(samples, 'fallbackLogCount');
    const timeout = samples.filter((s) => Boolean(s.timeoutHit)).length;
    const okCount = samples.filter((s) => Boolean(s.ok)).length;

    return {
        count: samples.length,
        okRate: samples.length ? (okCount / samples.length) * 100 : 0,
        wallAvg: avg(wall),
        wallP95: percentile(wall, 95),
        stepAvg: avg(step),
        stepP95: percentile(step, 95),
        fallbackAvg: avg(fallback),
        fallbackP95: percentile(fallback, 95),
        timeoutRate: samples.length ? (timeout / samples.length) * 100 : 0
    };
}

function printSummary(label, s) {
    process.stdout.write(
        `[${label}] count=${s.count} okRate=${s.okRate.toFixed(1)}% wallAvg=${s.wallAvg.toFixed(1)}ms wallP95=${s.wallP95.toFixed(1)}ms stepAvg=${s.stepAvg.toFixed(1)}ms stepP95=${s.stepP95.toFixed(1)}ms fallbackAvg=${s.fallbackAvg.toFixed(2)} fallbackP95=${s.fallbackP95.toFixed(2)} timeoutRate=${s.timeoutRate.toFixed(1)}%\n`
    );
}

function main() {
    const files = pickArtifacts();
    if (!files.length) {
        process.stderr.write(`[bridge-whitelist-trend] no artifacts found in ${OUTPUT_DIR}\n`);
        process.stderr.write('[bridge-whitelist-trend] run `npm run drill:bridge` first\n');
        process.exit(1);
        return;
    }

    const artifacts = files.map(parseArtifact);
    const minimal = summarizeScenario(artifacts, 'minimal');
    const all = summarizeScenario(artifacts, 'all');

    process.stdout.write(`[TREND] files=${files.length} limit=${LIMIT} dir=${OUTPUT_DIR}\n`);
    process.stdout.write(`[RANGE] first=${artifacts[0]?.startedAt || 'n/a'} last=${artifacts[artifacts.length - 1]?.startedAt || 'n/a'}\n`);
    printSummary('minimal-allowlist', minimal);
    printSummary('all-actions', all);

    process.stdout.write(
        `[DELTA] wallAvg=${(all.wallAvg - minimal.wallAvg >= 0 ? '+' : '') + (all.wallAvg - minimal.wallAvg).toFixed(1)}ms ` +
        `wallP95=${(all.wallP95 - minimal.wallP95 >= 0 ? '+' : '') + (all.wallP95 - minimal.wallP95).toFixed(1)}ms ` +
        `stepAvg=${(all.stepAvg - minimal.stepAvg >= 0 ? '+' : '') + (all.stepAvg - minimal.stepAvg).toFixed(1)}ms ` +
        `fallbackAvg=${(all.fallbackAvg - minimal.fallbackAvg >= 0 ? '+' : '') + (all.fallbackAvg - minimal.fallbackAvg).toFixed(2)}\n`
    );
}

try {
    main();
} catch (err) {
    process.stderr.write(`${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
}
