#!/usr/bin/env node

const API_ORIGIN = process.env.V2RAYN_API_ORIGIN || 'http://127.0.0.1:18000';
const TOKEN = process.env.V2RAYN_API_TOKEN || '';
const ENABLE_WRITE = process.env.V2RAYN_ENABLE_WRITE === '1';

function url(path) {
    return `${API_ORIGIN}${path}`;
}

function authHeaders() {
    return TOKEN ? { Authorization: `Bearer ${TOKEN}` } : {};
}

async function request(path, init = {}) {
    const response = await fetch(url(path), {
        ...init,
        headers: {
            'Content-Type': 'application/json',
            ...authHeaders(),
            ...(init.headers || {})
        }
    });

    const text = await response.text();
    let body = null;
    try {
        body = text ? JSON.parse(text) : null;
    } catch {
        body = text;
    }

    if (!response.ok) {
        throw new Error(`${init.method || 'GET'} ${path} -> HTTP ${response.status} ${typeof body === 'string' ? body : JSON.stringify(body)}`);
    }

    if (!body || typeof body !== 'object' || body.code !== 0) {
        throw new Error(`${init.method || 'GET'} ${path} -> API_ERROR ${JSON.stringify(body)}`);
    }

    return body.data;
}

async function step(name, action) {
    const startedAt = Date.now();
    process.stdout.write(`\n[STEP] ${name}\n`);
    try {
        const data = await action();
        const elapsed = Date.now() - startedAt;
        process.stdout.write(`[OK] ${name} (${elapsed}ms)\n`);
        return { ok: true, elapsed, data };
    } catch (error) {
        const elapsed = Date.now() - startedAt;
        process.stdout.write(`[FAIL] ${name} (${elapsed}ms)\n${error instanceof Error ? error.message : String(error)}\n`);
        return { ok: false, elapsed, error: error instanceof Error ? error.message : String(error) };
    }
}

async function main() {
    process.stdout.write(`[INFO] API_ORIGIN=${API_ORIGIN}\n`);
    process.stdout.write(`[INFO] TOKEN=${TOKEN ? 'set' : 'not-set'}\n`);
    process.stdout.write(`[INFO] WRITE_OPS=${ENABLE_WRITE ? 'enabled' : 'disabled'}\n`);

    const report = {
        startedAt: new Date().toISOString(),
        apiOrigin: API_ORIGIN,
        tokenProvided: Boolean(TOKEN),
        writeOps: ENABLE_WRITE,
        steps: []
    };

    report.steps.push({
        name: 'health',
        ...(await step('GET /api/health', () => request('/api/health')))
    });

    report.steps.push({
        name: 'core_status_before',
        ...(await step('GET /api/core/status (before)', () => request('/api/core/status')))
    });

    report.steps.push({
        name: 'profiles_list',
        ...(await step('GET /api/profiles', () => request('/api/profiles')))
    });

    const profiles = report.steps.find((item) => item.name === 'profiles_list')?.data;
    if (Array.isArray(profiles) && profiles.length > 0 && profiles[0]?.id) {
        const firstId = profiles[0].id;
        report.steps.push({
            name: 'profile_delay',
            ...(await step(`GET /api/profiles/${firstId}/delay`, () => request(`/api/profiles/${firstId}/delay`)))
        });
    }

    report.steps.push({
        name: 'subscriptions_list',
        ...(await step('GET /api/subscriptions', () => request('/api/subscriptions')))
    });

    report.steps.push({
        name: 'network_availability',
        ...(await step('GET /api/network/availability', () => request('/api/network/availability')))
    });

    if (ENABLE_WRITE) {
        report.steps.push({
            name: 'core_start',
            ...(await step('POST /api/core/start', () => request('/api/core/start', { method: 'POST', body: '{}' })))
        });

        if (Array.isArray(profiles) && profiles.length > 0 && profiles[0]?.id) {
            const firstId = profiles[0].id;
            report.steps.push({
                name: 'profile_select',
                ...(await step(`POST /api/profiles/${firstId}/select`, () =>
                    request(`/api/profiles/${firstId}/select`, { method: 'POST', body: '{}' })
                ))
            });
        } else {
            report.steps.push({
                name: 'profile_select',
                ok: true,
                elapsed: 0,
                skipped: true,
                data: { skipped: true, reason: '无可选节点，跳过 profiles/{id}/select' }
            });
            process.stdout.write('[SKIP] 无可选节点，跳过 profiles/{id}/select\n');
        }

        report.steps.push({
            name: 'subscriptions_update',
            ...(await step('POST /api/subscriptions/update', () => request('/api/subscriptions/update', { method: 'POST', body: '{}' })))
        });

        report.steps.push({
            name: 'system_proxy_clear',
            ...(await step('POST /api/system-proxy/apply (forced_clear)', () =>
                request('/api/system-proxy/apply', {
                    method: 'POST',
                    body: JSON.stringify({ mode: 'forced_clear', exceptions: '' })
                })
            ))
        });

        report.steps.push({
            name: 'core_stop',
            ...(await step('POST /api/core/stop', () => request('/api/core/stop', { method: 'POST', body: '{}' })))
        });
    }

    report.steps.push({
        name: 'core_status_after',
        ...(await step('GET /api/core/status (after)', () => request('/api/core/status')))
    });

    report.finishedAt = new Date().toISOString();
    report.ok = report.steps.every((item) => item.ok);

    process.stdout.write(`\n[SUMMARY] ${report.ok ? 'PASS' : 'FAIL'}\n`);
    process.stdout.write(`${JSON.stringify(report, null, 2)}\n`);

    if (!report.ok) {
        process.exitCode = 1;
    }
}

main().catch((error) => {
    process.stderr.write(`${error instanceof Error ? error.stack || error.message : String(error)}\n`);
    process.exit(1);
});
