'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { api, buildEventSourceUrl } from '@/lib/api/client';
import type { AvailabilityResult, ConfigDto, CoreStatus, DelayTestResult, LogLine, ProfileItem, StatsResult, SubscriptionItem } from '@/lib/types';

type DashboardSnapshot = {
    status: CoreStatus;
    config: ConfigDto;
    profiles: ProfileItem[];
    subscriptions: SubscriptionItem[];
    availability: AvailabilityResult;
};

function formatBytes(n: number): string {
    if (n < 1024) return `${n} B`;
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
    if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(2)} MB`;
    return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

function formatSpeed(n: number): string {
    return formatBytes(n) + '/s';
}

const LOG_MAX = 120;
const LEVEL_COLORS: Record<string, string> = {
    error: '#f87171',
    warning: '#fbbf24',
    warn: '#fbbf24',
    info: '#4ade80',
    debug: '#60a5fa',
};

export default function DashboardPage() {
    const [snapshot, setSnapshot] = useState<DashboardSnapshot | null>(null);
    const [stats, setStats] = useState<StatsResult | null>(null);
    const [error, setError] = useState<string>('');
    const [busyAction, setBusyAction] = useState<string>('');
    const [testingId, setTestingId] = useState('');
    const [testingAll, setTestingAll] = useState(false);
    const [delayResults, setDelayResults] = useState<Record<string, DelayTestResult>>({});
    const [profileKeyword, setProfileKeyword] = useState('');
    const [logLines, setLogLines] = useState<LogLine[]>([]);
    const [logConnected, setLogConnected] = useState(false);
    const logBottomRef = useRef<HTMLDivElement>(null);
    const logEsRef = useRef<EventSource | null>(null);
    const fallbackTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
    const statsTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

    const loadStatus = useCallback(async () => {
        try {
            const [status, config, profiles, subscriptions, availability] = await Promise.all([
                api.getCoreStatus(),
                api.getConfig(),
                api.getProfiles(),
                api.getSubscriptions(),
                api.getAvailability()
            ]);
            setSnapshot({ status, config, profiles, subscriptions, availability });
            setError('');
        } catch (e) {
            setError(e instanceof Error ? e.message : '加载失败');
        }
    }, []);

    const currentProfile = useMemo(() => {
        if (!snapshot?.status.currentProfileId) return null;
        return snapshot.profiles.find((item) => item.id === snapshot.status.currentProfileId) ?? null;
    }, [snapshot]);

    const enabledSubscriptions = useMemo(
        () => snapshot?.subscriptions.filter((item) => item.enabled).length ?? 0,
        [snapshot]
    );

    const filteredProfiles = useMemo(() => {
        const kw = profileKeyword.trim().toLowerCase();
        if (!kw) return snapshot?.profiles ?? [];
        return (snapshot?.profiles ?? []).filter((p) =>
            `${p.name} ${p.address} ${p.subName ?? ''}`.toLowerCase().includes(kw)
        );
    }, [snapshot, profileKeyword]);

    // SSE: events stream for status updates
    useEffect(() => {
        const startFallbackPolling = () => {
            if (fallbackTimerRef.current !== null) return;
            fallbackTimerRef.current = setInterval(() => { void loadStatus(); }, 3000);
        };
        const stopFallbackPolling = () => {
            if (fallbackTimerRef.current === null) return;
            clearInterval(fallbackTimerRef.current);
            fallbackTimerRef.current = null;
        };

        void loadStatus();

        const source = new EventSource(buildEventSourceUrl('/events/stream'), { withCredentials: true });

        source.onopen = () => { stopFallbackPolling(); };
        source.onmessage = (event) => {
            try {
                const payload = JSON.parse(event.data) as { event?: string; data?: unknown; };
                if (!payload.event) { void loadStatus(); return; }

                if (payload.event.startsWith('core.')) {
                    const status = payload.data as CoreStatus | undefined;
                    if (status) {
                        setSnapshot((prev) => prev ? { ...prev, status } : prev);
                    } else {
                        void loadStatus();
                    }
                    return;
                }
                if (payload.event === 'config.updated') {
                    const eventData = payload.data as { status?: CoreStatus; config?: ConfigDto } | undefined;
                    if (eventData?.status || eventData?.config) {
                        setSnapshot((prev) => {
                            if (!prev) return prev;
                            return {
                                ...prev,
                                status: eventData.status ?? prev.status,
                                config: eventData.config ?? prev.config
                            };
                        });
                    } else {
                        void loadStatus();
                    }
                    return;
                }
                if (payload.event === 'subscription.updated' || payload.event === 'profile.updated' || payload.event === 'profile.selected') {
                    void loadStatus();
                }
            } catch { void loadStatus(); }
        };
        source.onerror = () => { startFallbackPolling(); };

        return () => { source.close(); stopFallbackPolling(); };
    }, [loadStatus]);

    // SSE: log stream (mini console)
    const connectLogs = useCallback(() => {
        if (logEsRef.current) logEsRef.current.close();
        const es = new EventSource(buildEventSourceUrl('/logs/stream'), { withCredentials: true });
        logEsRef.current = es;
        es.addEventListener('ready', () => setLogConnected(true));
        es.addEventListener('log', (e: Event) => {
            const msg = e as MessageEvent;
            try {
                const line = JSON.parse(msg.data) as LogLine;
                setLogLines((prev) => {
                    const next = [...prev, line];
                    return next.length > LOG_MAX ? next.slice(next.length - LOG_MAX) : next;
                });
            } catch { /* ignore */ }
        });
        es.onerror = () => {
            setLogConnected(false);
            es.close();
            logEsRef.current = null;
            setTimeout(connectLogs, 4000);
        };
    }, []);

    useEffect(() => {
        connectLogs();
        return () => { logEsRef.current?.close(); };
    }, [connectLogs]);

    // Auto-scroll log tail
    useEffect(() => {
        logBottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [logLines]);

    // Poll stats every 2 seconds
    useEffect(() => {
        const pollStats = async () => {
            try { setStats(await api.getStats()); } catch { /* ignore */ }
        };
        void pollStats();
        statsTimerRef.current = setInterval(() => void pollStats(), 2000);
        return () => { if (statsTimerRef.current !== null) clearInterval(statsTimerRef.current); };
    }, []);

    const action = async (label: string, runner: () => Promise<CoreStatus> | Promise<unknown>) => {
        setBusyAction(label);
        try {
            await runner();
            await loadStatus();
            setError('');
        } catch (e) {
            setError(e instanceof Error ? e.message : '操作失败');
        } finally {
            setBusyAction('');
        }
    };

    const selectProfile = async (id: string) => {
        try {
            await api.selectProfile(id);
            await loadStatus();
        } catch (e) { setError(e instanceof Error ? e.message : '切换失败'); }
    };

    const testDelay = async (id: string) => {
        setTestingId(id);
        try {
            const result = await api.testProfileDelay(id);
            setDelayResults((prev) => ({ ...prev, [id]: result }));
        } catch { /* ignore */ }
        finally { setTestingId(''); }
    };

    const testAllDelays = async () => {
        setTestingAll(true);
        const profiles = snapshot?.profiles ?? [];
        // Run all delay tests concurrently
        await Promise.allSettled(profiles.map(async (p) => {
            try {
                const result = await api.testProfileDelay(p.id);
                setDelayResults((prev) => ({ ...prev, [p.id]: result }));
            } catch { /* ignore */ }
        }));
        setTestingAll(false);
    };

    const status = snapshot?.status;
    const isRunning = status?.running ?? false;
    const coreState = status?.state ?? (isRunning ? 'running' : 'stopped');

    return (
        <section className="page">
            {/* ── Header ── */}
            <div className="page-header">
                <div>
                    <h2>控制台</h2>
                    <p className="muted">VPN 主操作中心 — 核心控制、节点切换、实时日志</p>
                </div>
                <div className="toolbar compact">
                    <button onClick={() => void loadStatus()}>刷新</button>
                    <button
                        className="primary"
                        disabled={busyAction !== ''}
                        onClick={() => void action('update-subscriptions', api.updateSubscriptions)}
                    >
                        更新订阅
                    </button>
                </div>
            </div>

            {error ? <p className="status-error">{error}</p> : null}

            {/* ── Core control strip ── */}
            <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap', marginBottom: 16 }}>
                <span
                    className={isRunning ? 'status-pill ok' : 'status-pill warn'}
                    style={{ fontSize: 14, padding: '4px 12px', fontWeight: 600 }}
                >
                    {coreState}
                </span>
                <button
                    className="primary"
                    disabled={busyAction !== ''}
                    onClick={() => void action('start-core', api.startCore)}
                >
                    {busyAction === 'start-core' ? '启动中...' : '启动'}
                </button>
                <button
                    disabled={busyAction !== ''}
                    onClick={() => void action('stop-core', api.stopCore)}
                >
                    {busyAction === 'stop-core' ? '停止中...' : '停止'}
                </button>
                <button
                    disabled={busyAction !== ''}
                    onClick={() => void action('restart-core', api.restartCore)}
                >
                    {busyAction === 'restart-core' ? '重启中...' : '重启'}
                </button>
                {status?.error?.trim() ? (
                    <button onClick={() => void action('clear-core-error', api.clearCoreError)}>
                        清除错误
                    </button>
                ) : null}
                <span style={{ marginLeft: 'auto', color: 'var(--muted)', fontSize: 13 }}>
                    ↑ {formatSpeed(stats?.upSpeed ?? 0)} / ↓ {formatSpeed(stats?.downSpeed ?? 0)}
                    &nbsp;&nbsp;
                    累计 ↑{formatBytes(stats?.upBytes ?? 0)} ↓{formatBytes(stats?.downBytes ?? 0)}
                </span>
            </div>

            {/* Error banner */}
            {status?.error?.trim() ? (
                <div style={{
                    padding: '8px 12px',
                    background: 'color-mix(in srgb, #f87171 15%, transparent)',
                    border: '1px solid #f87171',
                    borderRadius: 6,
                    marginBottom: 12,
                    fontSize: 13,
                    color: '#f87171'
                }}>
                    <strong>核心错误：</strong> {status.error}
                    {status.errorAt ? <span style={{ marginLeft: 8, opacity: 0.7 }}>({status.errorAt})</span> : null}
                </div>
            ) : null}

            {/* ── Stats row ── */}
            <div className="stats-grid" style={{ marginBottom: 16 }}>
                <article className="stat-card accent-blue">
                    <span className="stat-label">核心引擎</span>
                    <strong style={{ fontSize: 14 }}>
                        {status?.engineMode ?? snapshot?.config.coreEngine ?? 'embedded'}
                        {' → '}
                        {status?.engineResolved ?? status?.coreType ?? '-'}
                    </strong>
                    <span>{status?.coreType ?? '-'}</span>
                </article>
                <article className="stat-card accent-green">
                    <span className="stat-label">当前节点</span>
                    <strong style={{ fontSize: 14, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {currentProfile?.name ?? '未选择'}
                    </strong>
                    <span>{currentProfile ? `${currentProfile.address}:${currentProfile.port ?? '-'}` : '-'}</span>
                </article>
                <article className="stat-card accent-amber">
                    <span className="stat-label">节点 / 订阅</span>
                    <strong>{snapshot?.profiles.length ?? 0}</strong>
                    <span>订阅 {snapshot?.subscriptions.length ?? 0}，启用 {enabledSubscriptions}</span>
                </article>
                <article className="stat-card accent-slate">
                    <span className="stat-label">网络可用性</span>
                    <strong>
                        {snapshot?.availability.available ? '✓ 可用' : '✗ 不可用'}
                    </strong>
                    <span>
                        {snapshot?.availability.elapsedMs != null
                            ? `${snapshot.availability.elapsedMs} ms`
                            : '未检测'}
                    </span>
                </article>
            </div>

            {/* ── Main two-column layout ── */}
            <div className="panel-grid two-up" style={{ alignItems: 'start' }}>

                {/* Profile list (quick switch) */}
                <section className="panel" style={{ overflow: 'hidden' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
                        <h3 style={{ margin: 0 }}>节点列表</h3>
                        <span className="muted" style={{ fontSize: 13 }}>{filteredProfiles.length} 条</span>
                        <button
                            style={{ marginLeft: 'auto', fontSize: 12 }}
                            disabled={testingAll || (snapshot?.profiles.length ?? 0) === 0}
                            onClick={() => void testAllDelays()}
                        >
                            {testingAll ? '测速中...' : '全部测速'}
                        </button>
                    </div>
                    <input
                        placeholder="搜索节点名称 / 地址..."
                        value={profileKeyword}
                        onChange={(e) => setProfileKeyword(e.target.value)}
                        style={{ width: '100%', marginBottom: 8, boxSizing: 'border-box' }}
                    />
                    <div style={{ overflowY: 'auto', maxHeight: 360 }}>
                        <table style={{ width: '100%' }}>
                            <thead>
                                <tr>
                                    <th style={{ width: 28 }}></th>
                                    <th>名称</th>
                                    <th style={{ width: 60 }}>延迟</th>
                                    <th style={{ width: 60 }}>操作</th>
                                </tr>
                            </thead>
                            <tbody>
                                {filteredProfiles.map((p) => {
                                    const isCurrent = p.id === status?.currentProfileId;
                                    const tested = delayResults[p.id];
                                    const delay = tested?.delayMs ?? p.delayMs;
                                    return (
                                        <tr
                                            key={p.id}
                                            className={isCurrent ? 'row-active' : ''}
                                            style={{ cursor: 'pointer' }}
                                            onClick={() => { if (!isCurrent) void selectProfile(p.id); }}
                                        >
                                            <td>
                                                <span className={isCurrent ? 'status-pill ok' : 'status-pill muted'}
                                                    style={{ fontSize: 11, padding: '1px 5px' }}>
                                                    {isCurrent ? '●' : '○'}
                                                </span>
                                            </td>
                                            <td>
                                                <div style={{ fontWeight: isCurrent ? 600 : 400, fontSize: 13, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: 180 }}>
                                                    {p.name}
                                                </div>
                                                {p.subName ? (
                                                    <div style={{ fontSize: 11, color: 'var(--muted)' }}>{p.subName}</div>
                                                ) : null}
                                            </td>
                                            <td style={{ textAlign: 'right', fontSize: 13 }}>
                                                {testingId === p.id ? (
                                                    <span className="muted">...</span>
                                                ) : delay != null ? (
                                                    <span style={{ color: delay < 200 ? '#4ade80' : delay < 600 ? '#fbbf24' : '#f87171' }}>
                                                        {delay} ms
                                                    </span>
                                                ) : (
                                                    <span className="muted">-</span>
                                                )}
                                            </td>
                                            <td>
                                                <button
                                                    style={{ fontSize: 11, padding: '2px 6px' }}
                                                    disabled={testingId === p.id}
                                                    onClick={(e) => { e.stopPropagation(); void testDelay(p.id); }}
                                                >
                                                    测速
                                                </button>
                                            </td>
                                        </tr>
                                    );
                                })}
                                {filteredProfiles.length === 0 && (
                                    <tr><td colSpan={4} style={{ textAlign: 'center', padding: 16, color: 'var(--muted)' }}>
                                        暂无节点
                                    </td></tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                </section>

                {/* Right column: summary + mini log */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
                    <section className="panel">
                        <h3>运行摘要</h3>
                        <table>
                            <tbody>
                                <tr>
                                    <th>代理端口</th>
                                    <td>
                                        HTTP {snapshot?.config.httpPort ?? 10809} / SOCKS5 {snapshot?.config.socksPort ?? 10808}
                                    </td>
                                </tr>
                                <tr>
                                    <th>TUN 模式</th>
                                    <td>
                                        {snapshot?.config.enableTun
                                            ? (snapshot.config.tunStack ?? 'mixed')
                                            : 'off'}
                                    </td>
                                </tr>
                                <tr>
                                    <th>路由模式</th>
                                    <td>
                                        {snapshot?.config.systemProxyMode ?? '-'}
                                    </td>
                                </tr>
                                <tr>
                                    <th>日志级别</th>
                                    <td>{snapshot?.config.logLevel ?? '-'}</td>
                                </tr>
                                <tr>
                                    <th>累计上传</th>
                                    <td>{formatBytes(stats?.upBytes ?? 0)}</td>
                                </tr>
                                <tr>
                                    <th>累计下载</th>
                                    <td>{formatBytes(stats?.downBytes ?? 0)}</td>
                                </tr>
                            </tbody>
                        </table>
                    </section>

                    {/* Mini log console */}
                    <section className="panel" style={{ flex: 1 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                            <h3 style={{ margin: 0 }}>日志</h3>
                            <span className={logConnected ? 'status-pill ok' : 'status-pill warn'} style={{ fontSize: 11 }}>
                                {logConnected ? '实时' : '断开'}
                            </span>
                            <span className="muted" style={{ fontSize: 12 }}>{logLines.length} 条</span>
                            <button
                                style={{ marginLeft: 'auto', fontSize: 11, padding: '2px 6px' }}
                                onClick={() => setLogLines([])}
                            >
                                清空
                            </button>
                        </div>
                        <div style={{
                            fontFamily: 'monospace',
                            fontSize: 12,
                            lineHeight: 1.5,
                            padding: '8px 10px',
                            background: 'color-mix(in srgb, var(--panel) 60%, black)',
                            borderRadius: 6,
                            overflowY: 'auto',
                            maxHeight: 240,
                            minHeight: 100,
                        }}>
                            {logLines.length === 0 ? (
                                <span style={{ color: 'var(--muted)' }}>
                                    {logConnected ? '等待日志...' : '连接中...'}
                                </span>
                            ) : logLines.map((line, idx) => (
                                <div key={idx} style={{ display: 'flex', gap: 8, marginBottom: 1 }}>
                                    <span style={{ color: 'var(--muted)', flexShrink: 0, fontSize: 11 }}>
                                        {line.timestamp.slice(11, 19)}
                                    </span>
                                    <span style={{
                                        color: LEVEL_COLORS[line.level] ?? 'inherit',
                                        flexShrink: 0,
                                        minWidth: 42,
                                        fontWeight: 600,
                                        fontSize: 11
                                    }}>
                                        [{line.level}]
                                    </span>
                                    <span style={{ wordBreak: 'break-all', fontSize: 12 }}>{line.message}</span>
                                </div>
                            ))}
                            <div ref={logBottomRef} />
                        </div>
                    </section>
                </div>
            </div>
        </section>
    );
}

