#!/usr/bin/env node

import { mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';

const OUTPUT_DIR = process.env.V2RAYN_DRILL_OUTPUT_DIR || './.artifacts/bridge-drill';
const TREND_LIMIT = Math.max(1, Number(process.env.V2RAYN_BRIDGE_WARNING_TREND_LIMIT || 5));
const BASELINE_LIMIT = Math.max(1, Number(process.env.V2RAYN_BRIDGE_ADVICE_BASELINE_LIMIT || 10));
const WARN_P95_MS = Number(process.env.V2RAYN_BRIDGE_ADVICE_P95_WARN_MS || 1200);
const CRIT_P95_MS = Number(process.env.V2RAYN_BRIDGE_ADVICE_P95_CRIT_MS || 2000);
const SLOW_BUCKET_RATIO_WARN = Number(process.env.V2RAYN_BRIDGE_ADVICE_SLOW_BUCKET_RATIO_WARN || 0.3);
const DEGRADE_P95_RATIO_WARN = Number(process.env.V2RAYN_BRIDGE_ADVICE_DEGRADE_P95_RATIO_WARN || 1.3);
const DEGRADE_P95_RATIO_CRIT = Number(process.env.V2RAYN_BRIDGE_ADVICE_DEGRADE_P95_RATIO_CRIT || 1.8);

const BLOCK_CONSEC_CRIT = Math.max(1, Number(process.env.V2RAYN_BRIDGE_RELEASE_CONSEC_CRIT_BLOCK || 2));
const PROMOTE_MAX_LATEST_WARN = Math.max(0, Number(process.env.V2RAYN_BRIDGE_RELEASE_PROMOTE_MAX_WARN || 1));
const BLOCK_TOTAL_WARNING = Math.max(0, Number(process.env.V2RAYN_BRIDGE_RELEASE_BLOCK_TOTAL_WARNING || 0));
const BLOCK_LATEST_WARNING = Math.max(0, Number(process.env.V2RAYN_BRIDGE_RELEASE_BLOCK_LATEST_WARNING || 0));
const DECISION_JSON_PATH = process.env.V2RAYN_BRIDGE_RELEASE_DECISION_JSON || '';

function nowStamp() {
    return new Date().toISOString().replace(/[:.]/g, '-');
}

function defaultDecisionJsonPath() {
    return join('./.artifacts/bridge-release-decision', `bridge-release-decision-${nowStamp()}.json`);
}

function writeDecisionArtifact(path, payload) {
    const outputPath = path && path.trim().length > 0 ? path : defaultDecisionJsonPath();
    mkdirSync(dirname(outputPath), { recursive: true });
    writeFileSync(outputPath, `${JSON.stringify(payload, null, 2)}\n`, 'utf-8');
    process.stdout.write(`[JSON_ARTIFACT] ${outputPath}\n`);
}

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

function evaluateReport(report, baselineReports) {
    const minimal = sampleByScenario(report, 'minimal-allowlist');
    const all = sampleByScenario(report, 'all-actions');

    const minimalBaseline = summarizeBaseline(baselineReports, 'minimal-allowlist');
    const allBaseline = summarizeBaseline(baselineReports, 'all-actions');

    const minimalResult = evaluateSample(minimal, minimalBaseline);
    const allResult = evaluateSample(all, allBaseline);

    return {
        critical: minimalResult.critical + allResult.critical,
        warning: minimalResult.warning + allResult.warning,
        minimalP95: toNumber(minimal?.bridgeLatencyP95Ms, 0),
        allP95: toNumber(all?.bridgeLatencyP95Ms, 0)
    };
}

function decisionFromTrend(rows) {
    const latest = rows[rows.length - 1];
    let consecutiveCritical = 0;
    for (let index = rows.length - 1; index >= 0; index -= 1) {
        if (rows[index].critical > 0) {
            consecutiveCritical += 1;
            continue;
        }
        break;
    }

    const totalCritical = rows.reduce((sum, row) => sum + row.critical, 0);
    const totalWarning = rows.reduce((sum, row) => sum + row.warning, 0);

    if (consecutiveCritical >= BLOCK_CONSEC_CRIT) {
        return {
            state: 'BLOCK',
            reason: `consecutiveCritical=${consecutiveCritical} (>=${BLOCK_CONSEC_CRIT})`,
            totalCritical,
            totalWarning,
            latest,
            consecutiveCritical
        };
    }

    if (BLOCK_TOTAL_WARNING > 0 && totalWarning >= BLOCK_TOTAL_WARNING) {
        return {
            state: 'BLOCK',
            reason: `totalWarning=${totalWarning} (>=${BLOCK_TOTAL_WARNING})`,
            totalCritical,
            totalWarning,
            latest,
            consecutiveCritical
        };
    }

    if (BLOCK_LATEST_WARNING > 0 && latest.warning >= BLOCK_LATEST_WARNING) {
        return {
            state: 'BLOCK',
            reason: `latestWarning=${latest.warning} (>=${BLOCK_LATEST_WARNING})`,
            totalCritical,
            totalWarning,
            latest,
            consecutiveCritical
        };
    }

    if (latest.critical > 0 || latest.warning > PROMOTE_MAX_LATEST_WARN || totalCritical > 0 || totalWarning > 0) {
        return {
            state: 'OBSERVE',
            reason: `latest critical=${latest.critical}, latest warning=${latest.warning}, total critical=${totalCritical}, total warning=${totalWarning}`,
            totalCritical,
            totalWarning,
            latest,
            consecutiveCritical
        };
    }

    return {
        state: 'PROMOTE',
        reason: `latest warning<=${PROMOTE_MAX_LATEST_WARN} and no critical/warning in trend window`,
        totalCritical,
        totalWarning,
        latest,
        consecutiveCritical
    };
}

function fileTag(filePath) {
    const chunks = filePath.split('/');
    return chunks[chunks.length - 1];
}

function main() {
    const startedAt = new Date().toISOString();
    const artifacts = pickArtifacts(OUTPUT_DIR);
    if (artifacts.length === 0) {
        process.stderr.write(`[bridge-release-decision] no artifacts found in ${OUTPUT_DIR}\n`);
        process.stderr.write('[bridge-release-decision] run npm run drill:bridge first\n');
        process.exit(1);
        return;
    }

    const reports = artifacts.map(readReport);
    const start = Math.max(0, reports.length - TREND_LIMIT);
    const rows = [];

    process.stdout.write(`[DECISION_INPUT] limit=${TREND_LIMIT} baselineLimit=${BASELINE_LIMIT} blockConsecCritical=${BLOCK_CONSEC_CRIT}\n`);
    process.stdout.write('[TABLE] run|min_p95|all_p95|critical|warning|file\n');

    for (let idx = start; idx < reports.length; idx += 1) {
        const baselineStart = Math.max(0, idx - BASELINE_LIMIT);
        const baselineReports = reports.slice(baselineStart, idx);
        const score = evaluateReport(reports[idx], baselineReports);
        rows.push({ ...score, file: artifacts[idx] });

        process.stdout.write(
            `${idx - start + 1}|${score.minimalP95}|${score.allP95}|${score.critical}|${score.warning}|${fileTag(artifacts[idx])}\n`
        );
    }

    const decision = decisionFromTrend(rows);
    process.stdout.write(`[SUMMARY] runs=${rows.length} critical=${decision.totalCritical} warning=${decision.totalWarning} latestCritical=${decision.latest.critical} latestWarning=${decision.latest.warning}\n`);
    process.stdout.write(`[DECISION] ${decision.state} reason=${decision.reason}\n`);

    const finishedAt = new Date().toISOString();
    writeDecisionArtifact(DECISION_JSON_PATH, {
        ok: decision.state !== 'BLOCK',
        startedAt,
        finishedAt,
        input: {
            outputDir: OUTPUT_DIR,
            trendLimit: TREND_LIMIT,
            baselineLimit: BASELINE_LIMIT,
            thresholds: {
                warnP95Ms: WARN_P95_MS,
                critP95Ms: CRIT_P95_MS,
                slowBucketRatioWarn: SLOW_BUCKET_RATIO_WARN,
                degradeP95RatioWarn: DEGRADE_P95_RATIO_WARN,
                degradeP95RatioCrit: DEGRADE_P95_RATIO_CRIT,
                blockConsecutiveCritical: BLOCK_CONSEC_CRIT,
                promoteMaxLatestWarning: PROMOTE_MAX_LATEST_WARN,
                blockTotalWarning: BLOCK_TOTAL_WARNING,
                blockLatestWarning: BLOCK_LATEST_WARNING
            }
        },
        summary: {
            runs: rows.length,
            totalCritical: decision.totalCritical,
            totalWarning: decision.totalWarning,
            latestCritical: decision.latest.critical,
            latestWarning: decision.latest.warning,
            consecutiveCritical: decision.consecutiveCritical
        },
        decision: {
            state: decision.state,
            reason: decision.reason
        },
        rows
    });

    if (decision.state === 'BLOCK') {
        process.exit(1);
    }
}

try {
    main();
} catch (err) {
    process.stderr.write(`${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
}
