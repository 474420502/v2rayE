#!/usr/bin/env node

import http from 'node:http';

const HOST = process.env.V2RAYN_MOCK_HOST || '127.0.0.1';
const PORT = Number(process.env.V2RAYN_MOCK_PORT || 18000);
const REQUIRE_TOKEN = process.env.V2RAYN_MOCK_REQUIRE_TOKEN === '1';
const TOKEN = process.env.V2RAYN_MOCK_TOKEN || 'mock-token';

const state = {
    core: {
        running: false,
        coreType: 'xray',
        currentProfileId: 'p1',
        state: 'stopped'
    },
    profiles: [
        { id: 'p1', name: 'HK-01', address: 'hk.example.com', port: 443, delayMs: 80, subName: 'default' },
        { id: 'p2', name: 'JP-01', address: 'jp.example.com', port: 443, delayMs: 120, subName: 'default' }
    ],
    subscriptions: [
        { id: 's1', remarks: 'default-sub', url: 'https://example.com/sub', enabled: true, updatedAt: new Date().toISOString() }
    ],
    config: {
        inbound: { enable: true },
        tunModeItem: { enableTun: false },
        coreBasicItem: { autoRun: false },
        systemProxyItem: { mode: 'clear' }
    },
    availability: {
        available: true,
        elapsedMs: 23,
        message: 'ok'
    }
};

function envelope(data, message = 'ok') {
    return { code: 0, message, data };
}

function errorEnvelope(code, message, details) {
    return { code, message, details };
}

function json(res, status, body) {
    res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' });
    res.end(JSON.stringify(body));
}

function readBody(req) {
    return new Promise((resolve, reject) => {
        const chunks = [];
        req.on('data', (chunk) => chunks.push(chunk));
        req.on('end', () => {
            const raw = Buffer.concat(chunks).toString('utf8').trim();
            if (!raw) {
                resolve({});
                return;
            }
            try {
                resolve(JSON.parse(raw));
            } catch {
                reject(new Error('invalid json'));
            }
        });
        req.on('error', reject);
    });
}

function isAuthorized(req) {
    if (!REQUIRE_TOKEN) return true;
    const header = req.headers.authorization || '';
    return header === `Bearer ${TOKEN}`;
}

function notFound(res) {
    json(res, 404, errorEnvelope(40401, 'not found'));
}

const server = http.createServer(async (req, res) => {
    const method = req.method || 'GET';
    const requestUrl = new URL(req.url || '/', `http://${HOST}:${PORT}`);
    const path = requestUrl.pathname;

    if (path === '/api/health' && method === 'GET') {
        json(res, 200, envelope({ status: 'healthy', ts: new Date().toISOString() }));
        return;
    }

    if (path.startsWith('/api/') && !isAuthorized(req)) {
        json(res, 401, errorEnvelope(40101, 'unauthorized'));
        return;
    }

    if (path === '/api/core/status' && method === 'GET') {
        json(res, 200, envelope(state.core));
        return;
    }

    if (path === '/api/core/start' && method === 'POST') {
        state.core.running = true;
        state.core.state = 'running';
        json(res, 200, envelope(state.core));
        return;
    }

    if (path === '/api/core/stop' && method === 'POST') {
        state.core.running = false;
        state.core.state = 'stopped';
        json(res, 200, envelope(state.core));
        return;
    }

    if (path === '/api/core/restart' && method === 'POST') {
        state.core.running = true;
        state.core.state = 'running';
        json(res, 200, envelope(state.core));
        return;
    }

    if (path === '/api/profiles' && method === 'GET') {
        json(res, 200, envelope(state.profiles));
        return;
    }

    const selectMatch = path.match(/^\/api\/profiles\/([^/]+)\/select$/);
    if (selectMatch && method === 'POST') {
        const profileId = selectMatch[1];
        const exists = state.profiles.some((item) => item.id === profileId);
        if (!exists) {
            json(res, 404, errorEnvelope(40401, 'profile not found'));
            return;
        }
        state.core.currentProfileId = profileId;
        json(res, 200, envelope({ selected: profileId }));
        return;
    }

    if (path === '/api/subscriptions' && method === 'GET') {
        json(res, 200, envelope(state.subscriptions));
        return;
    }

    if (path === '/api/subscriptions/update' && method === 'POST') {
        state.subscriptions = state.subscriptions.map((item) => ({ ...item, updatedAt: new Date().toISOString() }));
        json(res, 200, envelope({ updated: state.subscriptions.length }));
        return;
    }

    const updateOneMatch = path.match(/^\/api\/subscriptions\/([^/]+)\/update$/);
    if (updateOneMatch && method === 'POST') {
        const subId = updateOneMatch[1];
        const target = state.subscriptions.find((item) => item.id === subId);
        if (!target) {
            json(res, 404, errorEnvelope(40401, 'subscription not found'));
            return;
        }
        target.updatedAt = new Date().toISOString();
        json(res, 200, envelope({ updated: 1 }));
        return;
    }

    if (path === '/api/network/availability' && method === 'GET') {
        json(res, 200, envelope(state.availability));
        return;
    }

    if (path === '/api/system-proxy/apply' && method === 'POST') {
        try {
            const body = await readBody(req);
            const mode = body?.mode;
            if (mode !== 'forced_change' && mode !== 'forced_clear') {
                json(res, 422, errorEnvelope(42201, 'invalid mode'));
                return;
            }
            state.config.systemProxyItem = {
                mode,
                exceptions: body?.exceptions || ''
            };
            json(res, 200, envelope({ applied: true, mode }));
        } catch {
            json(res, 422, errorEnvelope(42201, 'invalid json'));
        }
        return;
    }

    if (path === '/api/config' && method === 'GET') {
        json(res, 200, envelope(state.config));
        return;
    }

    if (path === '/api/config' && method === 'PUT') {
        try {
            const body = await readBody(req);
            state.config = body;
            json(res, 200, envelope(state.config));
        } catch {
            json(res, 422, errorEnvelope(42201, 'invalid json'));
        }
        return;
    }

    notFound(res);
});

server.listen(PORT, HOST, () => {
    process.stdout.write(`[mock-api] listening on http://${HOST}:${PORT}\n`);
    process.stdout.write(`[mock-api] token required: ${REQUIRE_TOKEN ? `yes (${TOKEN})` : 'no'}\n`);
});

process.on('SIGINT', () => server.close(() => process.exit(0)));
process.on('SIGTERM', () => server.close(() => process.exit(0)));
