#!/usr/bin/env node

import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

const OUTPUT_DIR = process.env.V2RAYN_DRILL_OUTPUT_DIR || './.artifacts/bridge-drill';
const STRICT = /^(1|true|yes|on)$/i.test(process.env.V2RAYN_BRIDGE_ADVICE_STRICT || '');
const WARN_P95_MS = Number(process.env.V2RAYN_BRIDGE_ADVICE_P95_WARN_MS || 1200);
const CRIT_P95_MS = Number(process.env.V2RAYN_BRIDGE_ADVICE_P95_CRIT_MS || 2000);
const SLOW_BUCKET_RATIO_WARN = Number(process.env.V2RAYN_BRIDGE_ADVICE_SLOW_BUCKET_RATIO_WARN || 0.3);
const BASELINE_LIMIT = Number(process.env.V2RAYN_BRIDGE_ADVICE_BASELINE_LIMIT || 10);
const DEGRADE_P95_RATIO_WARN = Number(process.env.V2RAYN_BRIDGE_ADVICE_DEGRADE_P95_RATIO_WARN || 1.3);
const DEGRADE_P95_RATIO_CRIT = Number(process.env.V2RAYN_BRIDGE_ADVICE_DEGRADE_P95_RATIO_CRIT || 1.8);
const CONSECUTIVE_CRITICAL_REQUIRED = Math.max(1, Number(process.env.V2RAYN_BRIDGE_ADVICE_CONSEC_CRIT || 1));

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

function pickRecentArtifacts(dir, limit) {
    const files = readdirSync(dir, { withFileTypes: true })
        .filter((entry) => entry.isFile() && entry.name.startsWith('bridge-drill-') && entry.name.endsWith('.json'))
        .map((entry) => entry.name)
        .sort();

    if (files.length === 0) {
        return [];
    }
    return files.slice(Math.max(0, files.length - limit)).map((name) => join(dir, name));
}

function toNumber(value, fallback = 0) {
    return typeof value === 'number' && Number.isFinite(value) ? value : fallback;
}

function totalMapValues(map) {
    if (!map || typeof map !== 'object') {
        return 0;
    }
    return Object.values(map).reduce((sum, value) => sum + toNumber(value), 0);
}

function evaluateScenario(sample) {
    const issues = [];
    const scenario = sample?.scenario || 'unknown';
    const p95 = toNumber(sample?.bridgeLatencyP95Ms, 0);
    const p99 = toNumber(sample?.bridgeLatencyP99Ms, 0);
    const fallbackLogs = toNumber(sample?.fallbackLogCount, 0);
    const reasons = sample?.fallbackReasons && typeof sample.fallbackReasons === 'object' ? sample.fallbackReasons : {};
    const buckets = sample?.bridgeLatencyBuckets && typeof sample.bridgeLatencyBuckets === 'object' ? sample.bridgeLatencyBuckets : {};
    const latencyCount = Math.max(1, toNumber(sample?.bridgeLatencyCount, 0));
    const slowBucketCount = toNumber(buckets.ge1000ms, 0);
    const slowBucketRatio = slowBucketCount / latencyCount;

    if (fallbackLogs > 0) {
        issues.push({
            level: 'critical',
            scenario,
            type: 'fallback',
            message: `${scenario}: fallback logs=${fallbackLogs}`
        });
    }

    const timeoutCount = toNumber(reasons.timeout, 0);
    if (timeoutCount > 0) {
        issues.push({
            level: 'critical',
            scenario,
            type: 'timeout',
            message: `${scenario}: timeout reasons=${timeoutCount}; consider increasing V2RAYN_SERVICELIB_BRIDGE_TIMEOUT_MS or reducing bridge action scope`
        });
    }

    const commandErrCount = toNumber(reasons.bridge_command_error, 0);
    if (commandErrCount > 0) {
        issues.push({
            level: 'critical',
            scenario,
            type: 'bridge_command_error',
            message: `${scenario}: bridge command errors=${commandErrCount}; verify bridge command path/permissions`
        });
    }

    if (p95 >= CRIT_P95_MS) {
        issues.push({
            level: 'critical',
            scenario,
            type: 'latency',
            message: `${scenario}: bridgeLatencyP95Ms=${p95}ms (>=${CRIT_P95_MS}ms)`
        });
    } else if (p95 >= WARN_P95_MS) {
        issues.push({
            level: 'warning',
            scenario,
            type: 'latency',
            message: `${scenario}: bridgeLatencyP95Ms=${p95}ms (>=${WARN_P95_MS}ms)`
        });
    }

    if (slowBucketRatio >= SLOW_BUCKET_RATIO_WARN) {
        issues.push({
            level: 'warning',
            scenario,
            type: 'slow_bucket_ratio',
            message: `${scenario}: ge1000ms ratio=${(slowBucketRatio * 100).toFixed(1)}% (>=${(SLOW_BUCKET_RATIO_WARN * 100).toFixed(1)}%)`
        });
    }

    return {
        scenario,
        p95,
        p99,
        fallbackLogs,
        reasons,
        buckets,
        issues
    };
}

function summarizeBaseline(reports, scenarioName) {
    const values = reports
        .map((report) => (Array.isArray(report?.samples) ? report.samples : []).find((sample) => sample?.scenario === scenarioName))
        .filter(Boolean)
        .map((sample) => toNumber(sample.bridgeLatencyP95Ms, 0))
        .filter((value) => value > 0);

    if (!values.length) {
        return {
            count: 0,
            avgP95: 0
        };
    }

    const avgP95 = values.reduce((acc, value) => acc + value, 0) / values.length;
    return {
        count: values.length,
        avgP95
    };
}

function applyDegradationChecks(evaluations, baselineReports, allIssues, emitLogs = true) {
    for (const item of evaluations) {
        const baseline = summarizeBaseline(baselineReports, item.scenario);
        if (baseline.count === 0 || baseline.avgP95 <= 0 || item.p95 <= 0) {
            continue;
        }

        const ratio = item.p95 / baseline.avgP95;
        if (emitLogs) {
            process.stdout.write(`[BASELINE] ${item.scenario} avgP95=${baseline.avgP95.toFixed(1)}ms samples=${baseline.count} latestP95=${item.p95}ms ratio=${ratio.toFixed(2)}x\n`);
        }

        if (ratio >= DEGRADE_P95_RATIO_CRIT) {
            const issue = {
                level: 'critical',
                scenario: item.scenario,
                type: 'degradation',
                message: `${item.scenario}: p95 regression ratio=${ratio.toFixed(2)}x (>=${DEGRADE_P95_RATIO_CRIT}x baseline)`
            };
            allIssues.push(issue);
            if (emitLogs) {
                process.stdout.write(`[${issue.level.toUpperCase()}] ${issue.message}\n`);
            }
            continue;
        }

        if (ratio >= DEGRADE_P95_RATIO_WARN) {
            const issue = {
                level: 'warning',
                scenario: item.scenario,
                type: 'degradation',
                message: `${item.scenario}: p95 regression ratio=${ratio.toFixed(2)}x (>=${DEGRADE_P95_RATIO_WARN}x baseline)`
            };
            allIssues.push(issue);
            if (emitLogs) {
                process.stdout.write(`[${issue.level.toUpperCase()}] ${issue.message}\n`);
            }
        }
    }
}

function evaluateReport(report, baselineReports = [], emitLogs = false) {
    const samples = Array.isArray(report?.samples) ? report.samples : [];
    const evaluations = samples.map(evaluateScenario);
    const allIssues = [];

    for (const item of evaluations) {
        for (const issue of item.issues) {
            allIssues.push(issue);
        }
    }

    if (baselineReports.length > 0) {
        applyDegradationChecks(evaluations, baselineReports, allIssues, emitLogs);
    }

    const criticalCount = allIssues.filter((issue) => issue.level === 'critical').length;
    const warningCount = allIssues.filter((issue) => issue.level === 'warning').length;

    return {
        evaluations,
        allIssues,
        criticalCount,
        warningCount
    };
}

function hasConsecutiveCriticalHistory(artifactPaths, requiredConsecutive) {
    if (requiredConsecutive <= 1) {
        return true;
    }

    if (artifactPaths.length < requiredConsecutive) {
        return false;
    }

    const reports = artifactPaths
        .map((path) => {
            try {
                return JSON.parse(readFileSync(path, 'utf-8'));
            } catch {
                return null;
            }
        })
        .filter(Boolean);

    if (reports.length < requiredConsecutive) {
        return false;
    }

    const start = reports.length - requiredConsecutive;
    for (let idx = start; idx < reports.length; idx += 1) {
        const baselineStart = Math.max(0, idx - BASELINE_LIMIT);
        const baselineReports = reports.slice(baselineStart, idx);
        const result = evaluateReport(reports[idx], baselineReports);
        if (result.criticalCount === 0) {
            return false;
        }
    }

    return true;
}

function main() {
    const artifactPath = process.env.V2RAYN_DRILL_ARTIFACT || pickLatestArtifact(OUTPUT_DIR);
    if (!artifactPath) {
        process.stderr.write(`[bridge-advice] no artifact found in ${OUTPUT_DIR}\n`);
        process.stderr.write('[bridge-advice] run `npm run drill:bridge` first\n');
        process.exit(1);
        return;
    }

    const report = JSON.parse(readFileSync(artifactPath, 'utf-8'));

    const recentArtifacts = pickRecentArtifacts(OUTPUT_DIR, BASELINE_LIMIT + 1);
    const baselineReports = recentArtifacts
        .filter((path) => path !== artifactPath)
        .slice(-BASELINE_LIMIT)
        .map((path) => {
            try {
                return JSON.parse(readFileSync(path, 'utf-8'));
            } catch {
                return null;
            }
        })
        .filter(Boolean);

    const samples = Array.isArray(report?.samples) ? report.samples : [];
    const evaluations = samples.map(evaluateScenario);

    process.stdout.write(`[ARTIFACT] ${artifactPath}\n`);
    process.stdout.write(`[ADVICE] strict=${STRICT} warnP95=${WARN_P95_MS}ms critP95=${CRIT_P95_MS}ms baselineLimit=${BASELINE_LIMIT} consecutiveCriticalRequired=${CONSECUTIVE_CRITICAL_REQUIRED}\n`);

    const allIssues = [];
    for (const item of evaluations) {
        process.stdout.write(`[SCENARIO] ${item.scenario} p95=${item.p95}ms p99=${item.p99}ms fallbackLogs=${item.fallbackLogs}\n`);
        const reasonTotal = totalMapValues(item.reasons);
        if (reasonTotal > 0) {
            process.stdout.write(`[REASONS] ${item.scenario} ${JSON.stringify(item.reasons)}\n`);
        }
        if (Object.keys(item.buckets).length > 0) {
            process.stdout.write(`[BUCKETS] ${item.scenario} ${JSON.stringify(item.buckets)}\n`);
        }
        for (const issue of item.issues) {
            allIssues.push(issue);
            process.stdout.write(`[${issue.level.toUpperCase()}] ${issue.message}\n`);
        }
    }

    if (baselineReports.length > 0) {
        applyDegradationChecks(evaluations, baselineReports, allIssues, true);
    } else {
        process.stdout.write('[BASELINE] insufficient history for degradation check\n');
    }

    const criticalCount = allIssues.filter((issue) => issue.level === 'critical').length;
    const warningCount = allIssues.filter((issue) => issue.level === 'warning').length;

    process.stdout.write(`[SUMMARY] critical=${criticalCount} warning=${warningCount}\n`);

    if (criticalCount === 0 && warningCount === 0) {
        process.stdout.write('[RECOMMEND] current bridge quality is healthy; keep native default and continue gradual bridge validation.\n');
        return;
    }

    process.stdout.write('[RECOMMEND] prioritize critical items first, then tune high-latency scenarios, whitelist scope, and recent regressions.\n');
    if (STRICT && criticalCount > 0) {
        const orderedArtifacts = pickRecentArtifacts(OUTPUT_DIR, BASELINE_LIMIT + CONSECUTIVE_CRITICAL_REQUIRED + 2);
        const shouldBlock = hasConsecutiveCriticalHistory(orderedArtifacts, CONSECUTIVE_CRITICAL_REQUIRED);
        process.stdout.write(`[STRICT] criticalDetected=true consecutiveRequired=${CONSECUTIVE_CRITICAL_REQUIRED} block=${shouldBlock}\n`);
        if (shouldBlock) {
            process.stderr.write('[bridge-advice] strict mode enabled and blocking threshold reached.\n');
            process.exit(1);
        }
    }
}

try {
    main();
} catch (err) {
    process.stderr.write(`${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
}
