#!/usr/bin/env node

import { readdirSync, readFileSync, statSync, mkdirSync, writeFileSync } from 'node:fs';
import { join, relative } from 'node:path';

const ROOT = process.cwd();
const TARGETS = [
    '../v2rayN/v2rayN/ServiceLib',
    '../v2rayN/v2rayN.Desktop',
    '../v2rayN/v2rayN/GlobalHotKeys'
];

const OUTPUT_DIR = process.env.V2RAYN_COUPLING_OUTPUT_DIR || './.artifacts/coupling-scan';

const RULES = [
    { key: 'updateViewInvoke', label: '_updateView?.Invoke(...)', regex: /_updateView\?\.Invoke\s*\(/g },
    { key: 'eViewAction', label: 'EViewAction references', regex: /\bEViewAction\b/g },
    { key: 'globalHotkey', label: 'GlobalHotkey* references', regex: /\bGlobalHotkey\w*\b/g },
    { key: 'hide2TrayWhenClose', label: 'Hide2TrayWhenClose', regex: /\bHide2TrayWhenClose\b/g },
    { key: 'trayMenuServersLimit', label: 'TrayMenuServersLimit', regex: /\bTrayMenuServersLimit\b/g },
    { key: 'windowSizeItem', label: 'WindowSizeItem', regex: /\bWindowSizeItem\b/g },
    { key: 'globalHotkeySettingVm', label: 'GlobalHotkeySettingViewModel', regex: /\bGlobalHotkeySettingViewModel\b/g }
];

function walkFiles(dir, output = []) {
    const entries = readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
        const absPath = join(dir, entry.name);
        if (entry.isDirectory()) {
            walkFiles(absPath, output);
            continue;
        }
        if (entry.isFile() && (absPath.endsWith('.cs') || absPath.endsWith('.axaml') || absPath.endsWith('.csproj'))) {
            output.push(absPath);
        }
    }
    return output;
}

function countMatches(text, regex) {
    const matches = text.match(regex);
    return matches ? matches.length : 0;
}

function scanFile(filePath) {
    const text = readFileSync(filePath, 'utf-8');
    const hits = [];
    for (const rule of RULES) {
        const count = countMatches(text, rule.regex);
        if (count > 0) {
            hits.push({
                key: rule.key,
                label: rule.label,
                count
            });
        }
    }
    return hits;
}

function main() {
    const startedAt = new Date().toISOString();
    const files = [];
    for (const target of TARGETS) {
        const absTarget = join(ROOT, target);
        try {
            const st = statSync(absTarget);
            if (!st.isDirectory()) {
                continue;
            }
            files.push(...walkFiles(absTarget));
        } catch {
        }
    }

    const summary = {};
    for (const rule of RULES) {
        summary[rule.key] = {
            label: rule.label,
            total: 0,
            files: []
        };
    }

    for (const filePath of files) {
        const hits = scanFile(filePath);
        if (hits.length === 0) {
            continue;
        }
        for (const hit of hits) {
            summary[hit.key].total += hit.count;
            summary[hit.key].files.push({
                path: relative(ROOT, filePath),
                count: hit.count
            });
        }
    }

    for (const rule of RULES) {
        summary[rule.key].files.sort((a, b) => b.count - a.count || a.path.localeCompare(b.path));
    }

    const report = {
        startedAt,
        finishedAt: new Date().toISOString(),
        workspaceRoot: ROOT,
        scannedFiles: files.length,
        targets: TARGETS,
        summary
    };

    mkdirSync(OUTPUT_DIR, { recursive: true });
    const stamp = startedAt.replace(/[:.]/g, '-');
    const outPath = join(OUTPUT_DIR, `coupling-scan-${stamp}.json`);
    writeFileSync(outPath, `${JSON.stringify(report, null, 2)}\n`, 'utf-8');

    process.stdout.write(`[SUMMARY] scannedFiles=${files.length}\n`);
    for (const rule of RULES) {
        process.stdout.write(`[HIT] ${rule.key}=${summary[rule.key].total}\n`);
    }
    process.stdout.write(`[ARTIFACT] ${outPath}\n`);
}

main();
