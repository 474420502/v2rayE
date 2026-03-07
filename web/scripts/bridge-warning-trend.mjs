#!/usr/bin/env node

import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

const OUTPUT_DIR = process.env.V2RAYN_DRILL_OUTPUT_DIR || './.artifacts/bridge-drill';
const TREND_LIMIT = Math.max(1, Number(process.env.V2RAYN_BRIDGE_WARNING_TREND_LIMIT || 5));
const BASELINE_LIMIT = Math.max(1, Number(process.env.V2RAYN_BRIDGE_ADVICE_BASELINE_LIMIT || 10));
const WARN_P95_MS = Number(process.env.V2RAYN_BRIDGE_ADVICE_P95_WARN_MS || 1200);
const CRIT_P95_MS = Number(process.env.V2RAYN_BRIDGE_ADVICE_P95_CRIT_MS || 2000);
const SLOW_BUCKET_RATIO_WARN = Number(process.env.V2RAYN_BRIDGE_ADVICE_SLOW_BUCKET_RATIO_WARN || 0.3);
const DEGRADE_P95_RATIO_WARN = Number(process.env.V2RAYN_BRIDGE_ADVICE_DEGRADE_P95_RATIO_WARN || 1.3);
const DEGRADE_P95_RATIO_CRIT = Number(process.env.V2RAYN_BRIDGE_ADVICE_DEGRADE_P95_RATIO_CRIT || 1.8);

function toNumber(value, fallback = 0) {
    return typeof value === 'number' && Number.isFinite(value) ? value : fallback;
}

function pickArtifacts(dir) {
    const files = readdirSync(dir, { withFileTypes: true })
        .filter((entry) => entry.isFile() && entry.name.startsWith('bridge-drill-') && entry.name.endsWith('.json'))
        .map((entry) => entry.name)
        .sort();
    return files.map((name) => join(dir, name));
}

function readReport(filePath) {
    return JSON.parse(readFileSync(filePath, 'utf-8'));
}

function sampleByScenario(report, scenario) {
    const samples = Array.isArray(report?.samples) ? report.samples : [];
    return samples.find((sample) => sample?.scenario === scenario) || null;
}

function summarizeBaseline(reports, scenarioName) {
    const values = reports
        .map((report) => sampleByScenario(report, scenarioName))
        .filter(Boolean)
        .map((sample) => toNumber(sample.bridgeLatencyP95Ms, 0))
        .filter((value) => value > 0);

    if (!values.length) {
        return { count: 0, avgP95: 0 };
    }

    return {
        count: values.length,
        avgP95: values.reduce((sum, value) => sum + value, 0) / values.length
    };
}

function evaluateSample(sample, baseline) {
    if (!sample) {
        return { critical: 1, warning: 0 };
    }

    let critical = 0;
    let warning = 0;

    const p95 = toNumber(sample.bridgeLatencyP95Ms, 0);
    const fallbackLogCount = toNumber(sample.fallbackLogCount, 0);
    const reasons = sample?.fallbackReasons && typeof sample.fallbackReasons === 'object' ? sample.fallbackReasons : {};
    const buckets = sample?.bridgeLatencyBuckets && typeof sample.bridgeLatencyBuckets === 'object' ? sample.bridgeLatencyBuckets : {};
    const latencyCount = Math.max(1, toNumber(sample.bridgeLatencyCount, 0));
    const slowBucketRatio = toNumber(buckets.ge1000ms, 0) / latencyCount;

    if (fallbackLogCount > 0) {
        critical += 1;
    }
    if (toNumber(reasons.timeout, 0) > 0) {
        critical += 1;
    }
    if (toNumber(reasons.bridge_command_error, 0) > 0) {
        critical += 1;
    }

    if (p95 >= CRIT_P95_MS) {
        critical += 1;
    } else if (p95 >= WARN_P95_MS) {
        warning += 1;
    }

    if (slowBucketRatio >= SLOW_BUCKET_RATIO_WARN) {
        warning += 1;
    }

    if (baseline.count > 0 && baseline.avgP95 > 0 && p95 > 0) {
        const ratio = p95 / baseline.avgP95;
        if (ratio >= DEGRADE_P95_RATIO_CRIT) {
            critical += 1;
        } else if (ratio >= DEGRADE_P95_RATIO_WARN) {
            warning += 1;
        }
    }

    return { critical, warning };
}

function fileTag(filePath) {
    const chunks = filePath.split('/');
    return chunks[chunks.length - 1];
}

function main() {
    const artifacts = pickArtifacts(OUTPUT_DIR);
    if (artifacts.length === 0) {
        process.stderr.write(`[bridge-warning-trend] no artifacts found in ${OUTPUT_DIR}\n`);
        process.stderr.write('[bridge-warning-trend] run `npm run drill:bridge` first\n');
        process.exit(1);
        return;
    }

    const reports = artifacts.map(readReport);
    const start = Math.max(0, reports.length - TREND_LIMIT);

    process.stdout.write(`[TREND] limit=${TREND_LIMIT} baselineLimit=${BASELINE_LIMIT} dir=${OUTPUT_DIR}\n`);
    process.stdout.write('[TABLE] run|min_p95|all_p95|critical|warning|file\n');

    let totalCritical = 0;
    let totalWarning = 0;

    for (let idx = start; idx < reports.length; idx += 1) {
        const report = reports[idx];
        const filePath = artifacts[idx];

        const baselineStart = Math.max(0, idx - BASELINE_LIMIT);
        const baselineReports = reports.slice(baselineStart, idx);

        const minimal = sampleByScenario(report, 'minimal-allowlist');
        const all = sampleByScenario(report, 'all-actions');

        const minimalBaseline = summarizeBaseline(baselineReports, 'minimal-allowlist');
        const allBaseline = summarizeBaseline(baselineReports, 'all-actions');

        const minimalResult = evaluateSample(minimal, minimalBaseline);
        const allResult = evaluateSample(all, allBaseline);

        const critical = minimalResult.critical + allResult.critical;
        const warning = minimalResult.warning + allResult.warning;

        totalCritical += critical;
        totalWarning += warning;

        const minimalP95 = toNumber(minimal?.bridgeLatencyP95Ms, 0);
        const allP95 = toNumber(all?.bridgeLatencyP95Ms, 0);

        process.stdout.write(`${idx - start + 1}|${minimalP95}|${allP95}|${critical}|${warning}|${fileTag(filePath)}\n`);
    }

    process.stdout.write(`[SUMMARY] runs=${reports.length - start} critical=${totalCritical} warning=${totalWarning}\n`);
    process.stdout.write(`[THRESHOLDS] warnP95=${WARN_P95_MS} critP95=${CRIT_P95_MS} degradeWarn=${DEGRADE_P95_RATIO_WARN}x degradeCrit=${DEGRADE_P95_RATIO_CRIT}x\n`);
}

try {
    main();
} catch (err) {
    process.stderr.write(`${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
}
