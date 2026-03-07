#!/usr/bin/env node

import { readFileSync } from 'node:fs';

const DECISION_PATH = process.env.V2RAYN_BRIDGE_RELEASE_DECISION_JSON || './.artifacts/bridge-release-decision/latest.prod.json';
const BLOCK_STATES = new Set((process.env.V2RAYN_BRIDGE_RELEASE_BLOCK_STATES || 'BLOCK').split(',').map((item) => item.trim().toUpperCase()).filter(Boolean));
const BLOCK_TOTAL_WARNING = Math.max(0, Number(process.env.V2RAYN_BRIDGE_RELEASE_BLOCK_TOTAL_WARNING || 0));
const BLOCK_LATEST_WARNING = Math.max(0, Number(process.env.V2RAYN_BRIDGE_RELEASE_BLOCK_LATEST_WARNING || 0));

function main() {
    const raw = readFileSync(DECISION_PATH, 'utf-8');
    const payload = JSON.parse(raw);

    const state = String(payload?.decision?.state || '').toUpperCase();
    const reason = String(payload?.decision?.reason || '');
    const summary = payload?.summary && typeof payload.summary === 'object' ? payload.summary : {};

    process.stdout.write(`[CI_GATE] file=${DECISION_PATH}\n`);
    process.stdout.write(`[CI_GATE] state=${state || 'UNKNOWN'} blockStates=${Array.from(BLOCK_STATES).join(',')}\n`);
    process.stdout.write(`[CI_GATE] warningPolicy total>=${BLOCK_TOTAL_WARNING || 'off'} latest>=${BLOCK_LATEST_WARNING || 'off'}\n`);
    process.stdout.write(
        `[CI_GATE] summary runs=${Number(summary.runs || 0)} totalCritical=${Number(summary.totalCritical || 0)} totalWarning=${Number(summary.totalWarning || 0)} latestCritical=${Number(summary.latestCritical || 0)} latestWarning=${Number(summary.latestWarning || 0)}\n`
    );
    if (reason) {
        process.stdout.write(`[CI_GATE] reason=${reason}\n`);
    }

    if (!state) {
        process.stderr.write('[bridge-release-ci-gate] missing decision.state in JSON artifact\n');
        process.exit(1);
        return;
    }

    if (BLOCK_STATES.has(state)) {
        process.stderr.write(`[bridge-release-ci-gate] blocked by state=${state}\n`);
        process.exit(1);
        return;
    }

    const totalWarning = Number(summary.totalWarning || 0);
    const latestWarning = Number(summary.latestWarning || 0);
    if (BLOCK_TOTAL_WARNING > 0 && totalWarning >= BLOCK_TOTAL_WARNING) {
        process.stderr.write(`[bridge-release-ci-gate] blocked by totalWarning=${totalWarning} (>=${BLOCK_TOTAL_WARNING})\n`);
        process.exit(1);
        return;
    }
    if (BLOCK_LATEST_WARNING > 0 && latestWarning >= BLOCK_LATEST_WARNING) {
        process.stderr.write(`[bridge-release-ci-gate] blocked by latestWarning=${latestWarning} (>=${BLOCK_LATEST_WARNING})\n`);
        process.exit(1);
        return;
    }

    process.stdout.write('[bridge-release-ci-gate] pass\n');
}

try {
    main();
} catch (err) {
    process.stderr.write(`${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
}
