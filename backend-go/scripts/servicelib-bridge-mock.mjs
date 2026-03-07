#!/usr/bin/env node

import { readFileSync } from 'node:fs';

const input = readFileSync(0, 'utf8');
const payload = input ? JSON.parse(input) : {};
const action = payload.action || 'core.status';
const version = payload.version || 'v1';
const requestId = payload.requestId || '';

const coreStopped = { running: false, coreType: 'xray', currentProfileId: 'p1', state: 'stopped' };
const coreRunning = { running: true, coreType: 'xray', currentProfileId: 'p1', state: 'running' };

const dataByAction = {
    'core.status': coreStopped,
    'core.start': coreRunning,
    'core.stop': coreStopped,
    'core.restart': coreRunning,
    'profiles.list': [
        { id: 'p1', name: 'HK-01', address: 'hk.example.com', port: 443, delayMs: 80, subName: 'default' },
        { id: 'p2', name: 'JP-01', address: 'jp.example.com', port: 443, delayMs: 120, subName: 'default' }
    ],
    'profiles.select': { selected: payload?.args?.id || 'p1' },
    'profiles.delay': { available: true, delayMs: 80, message: 'ok' },
    'subscriptions.list': [
        { id: 's1', remarks: 'default-sub', url: 'https://example.com/sub', enabled: true, userAgent: 'v2rayN/7.x', autoUpdateMinutes: 120, updatedAt: new Date().toISOString() }
    ],
    'subscriptions.create': {
        id: 's-created',
        remarks: payload?.args?.remarks || 'created-sub',
        url: payload?.args?.url || 'https://example.com/new',
        enabled: payload?.args?.enabled ?? true,
        userAgent: payload?.args?.userAgent || '',
        filter: payload?.args?.filter || '',
        convertTarget: payload?.args?.convertTarget || '',
        autoUpdateMinutes: payload?.args?.autoUpdateMinutes || 0,
        updatedAt: new Date().toISOString()
    },
    'subscriptions.updateItem': {
        id: payload?.args?.id || 's1',
        remarks: payload?.args?.remarks || 'updated-sub',
        url: payload?.args?.url || 'https://example.com/updated',
        enabled: payload?.args?.enabled ?? true,
        userAgent: payload?.args?.userAgent || '',
        filter: payload?.args?.filter || '',
        convertTarget: payload?.args?.convertTarget || '',
        autoUpdateMinutes: payload?.args?.autoUpdateMinutes || 0,
        updatedAt: new Date().toISOString()
    },
    'subscriptions.delete': { deleted: payload?.args?.id || 's1' },
    'subscriptions.update': 1,
    'subscriptions.updateById': { updated: 1 },
    'network.availability': { available: true, elapsedMs: 23, message: 'ok' },
    'systemProxy.apply': { applied: true, mode: payload?.args?.mode || 'forced_clear' },
    'config.get': {
        inbound: { enable: true, listen: '127.0.0.1', port: 10808, allowLan: false },
        tunModeItem: { enableTun: false, stackMixed: false, mtu: 1500 },
        coreBasicItem: { autoRun: false, logLevel: 'warning', concurrency: 0, skipCertVerify: false, defUserAgent: 'v2rayN/7.x' },
        systemProxyItem: { mode: 'forced_clear', exceptions: '' }
    },
    'config.update': payload?.args?.config || {}
};

if (!Object.prototype.hasOwnProperty.call(dataByAction, action)) {
    process.stdout.write(
        JSON.stringify({
            version,
            requestId,
            success: false,
            error: {
                code: 40401,
                message: `unsupported action: ${action}`,
                details: { action }
            }
        })
    );
    process.exit(0);
}

process.stdout.write(
    JSON.stringify({
        version,
        requestId,
        success: true,
        data: dataByAction[action]
    })
);
