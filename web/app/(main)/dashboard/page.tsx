'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { api } from '@/lib/api/client';
import type { AvailabilityResult, ConfigDto, CoreStatus, ProfileItem, StatsResult, SubscriptionItem } from '@/lib/types';

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

export default function DashboardPage() {
    const [snapshot, setSnapshot] = useState<DashboardSnapshot | null>(null);
    const [stats, setStats] = useState<StatsResult | null>(null);
    const [error, setError] = useState<string>('');
    const [busyAction, setBusyAction] = useState<string>('');
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
        if (!snapshot?.status.currentProfileId) {
            return null;
        }
        return snapshot.profiles.find((item) => item.id === snapshot.status.currentProfileId) ?? null;
    }, [snapshot]);

    const enabledSubscriptions = useMemo(
        () => snapshot?.subscriptions.filter((item) => item.enabled).length ?? 0,
        [snapshot]
    );

    const averageDelay = useMemo(() => {
        const samples = snapshot?.profiles.filter((item) => typeof item.delayMs === 'number') ?? [];
        if (samples.length === 0) {
            return null;
        }
        const total = samples.reduce((sum, item) => sum + (item.delayMs ?? 0), 0);
        return Math.round(total / samples.length);
    }, [snapshot]);

    useEffect(() => {
        const startFallbackPolling = () => {
            if (fallbackTimerRef.current !== null) {
                return;
            }
            fallbackTimerRef.current = setInterval(() => {
                void loadStatus();
            }, 3000);
        };

        const stopFallbackPolling = () => {
            if (fallbackTimerRef.current === null) {
                return;
            }
            clearInterval(fallbackTimerRef.current);
            fallbackTimerRef.current = null;
        };

        void loadStatus();

        const endpoint = `${process.env.NEXT_PUBLIC_API_BASE ?? '/api'}/events/stream`;
        const source = new EventSource(endpoint, { withCredentials: true });

        source.onopen = () => {
            stopFallbackPolling();
        };

        source.onmessage = (event) => {
            try {
                const payload = JSON.parse(event.data) as {
                    event?: string;
                    data?: unknown;
                };
                if (!payload.event) {
                    void loadStatus();
                    return;
                }

                if (payload.event.startsWith('core.')) {
                    const status = payload.data as CoreStatus | undefined;
                    if (status) {
                        setSnapshot((prev) => {
                            if (!prev) return prev;
                            return { ...prev, status };
                        });
                    } else {
                        void loadStatus();
                    }
                    return;
                }

                if (payload.event === 'config.updated') {
                    const eventData = payload.data as {
                        status?: CoreStatus;
                        config?: ConfigDto;
                    } | undefined;
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
            } catch {
                void loadStatus();
            }
        };

        source.onerror = () => {
            startFallbackPolling();
        };

        return () => {
            source.close();
            stopFallbackPolling();
        };
    }, [loadStatus]);

    // Poll stats every 2 seconds when visible
    useEffect(() => {
        const pollStats = async () => {
            try {
                const s = await api.getStats();
                setStats(s);
            } catch { /* ignore */ }
        };
        void pollStats();
        statsTimerRef.current = setInterval(() => void pollStats(), 2000);
        return () => {
            if (statsTimerRef.current !== null) clearInterval(statsTimerRef.current);
        };
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

    const status = snapshot?.status;

    return (
        <section className="page">
            <div className="page-header">
                <div>
                    <h2>概览</h2>
                    <p className="muted">对齐 v2rayN 的主操作路径，集中显示核心状态、当前节点和订阅概况。</p>
                </div>
                <div className="toolbar compact">
                    <button onClick={() => void loadStatus()}>刷新状态</button>
                    <button className="primary" onClick={() => void action('update-subscriptions', api.updateSubscriptions)}>
                        更新全部订阅
                    </button>
                </div>
            </div>
            <div className="toolbar">
                <button
                    className="primary"
                    disabled={busyAction !== ''}
                    onClick={() => void action('start-core', api.startCore)}
                >
                    启动核心
                </button>
                <button disabled={busyAction !== ''} onClick={() => void action('stop-core', api.stopCore)}>
                    停止核心
                </button>
                <button disabled={busyAction !== ''} onClick={() => void action('restart-core', api.restartCore)}>
                    重启核心
                </button>
                <button
                    disabled={busyAction !== '' || !status?.error?.trim()}
                    onClick={() => void action('clear-core-error', api.clearCoreError)}
                >
                    清空核心错误
                </button>
            </div>
            {error ? <p className="status-error">{error}</p> : null}
            <div className="stats-grid">
                <article className="stat-card accent-blue">
                    <span className="stat-label">核心状态</span>
                    <strong>{status?.state ?? (status?.running ? 'running' : 'stopped')}</strong>
                    <span className={status?.running ? 'status-pill ok' : 'status-pill warn'}>
                        {status?.running ? '运行中' : '已停止'}
                    </span>
                </article>
                <article className="stat-card accent-green">
                    <span className="stat-label">当前节点</span>
                    <strong>{currentProfile?.name ?? '-'}</strong>
                    <span>{currentProfile ? `${currentProfile.address}:${currentProfile.port ?? '-'}` : '未选择'}</span>
                </article>
                <article className="stat-card accent-amber">
                    <span className="stat-label">订阅</span>
                    <strong>{snapshot?.subscriptions.length ?? 0}</strong>
                    <span>启用 {enabledSubscriptions} 条</span>
                </article>
                <article className="stat-card accent-slate">
                    <span className="stat-label">网络可用性</span>
                    <strong>{snapshot?.availability.available ? 'Available' : 'Unavailable'}</strong>
                    <span>
                        {snapshot?.availability.elapsedMs != null ? `${snapshot.availability.elapsedMs} ms` : '未检测'}
                    </span>
                </article>
                <article className="stat-card" style={{ borderLeft: '3px solid #a78bfa' }}>
                    <span className="stat-label">带宽（实时）</span>
                    <strong style={{ fontSize: 15 }}>
                        ↑ {formatSpeed(stats?.upSpeed ?? 0)} / ↓ {formatSpeed(stats?.downSpeed ?? 0)}
                    </strong>
                    <span>累计 ↑{formatBytes(stats?.upBytes ?? 0)} ↓{formatBytes(stats?.downBytes ?? 0)}</span>
                </article>
            </div>
            <div className="panel-grid two-up">
                <section className="panel">
                    <h3>运行摘要</h3>
                    <table>
                        <tbody>
                            <tr>
                                <th>核心类型</th>
                                <td>{status?.coreType ?? '-'}</td>
                            </tr>
                            <tr>
                                <th>引擎策略</th>
                                <td>
                                    {(status?.engineMode ?? snapshot?.config.coreEngine ?? 'embedded')}
                                    {' -> '}
                                    {(status?.engineResolved ?? status?.coreType ?? '-')}
                                </td>
                            </tr>
                            <tr>
                                <th>当前节点 ID</th>
                                <td>{status?.currentProfileId ?? '-'}</td>
                            </tr>
                            <tr>
                                <th>平均延迟</th>
                                <td>{averageDelay != null ? `${averageDelay} ms` : '-'}</td>
                            </tr>
                            <tr>
                                <th>最近错误</th>
                                <td>
                                    {status?.error?.trim()
                                        ? `${status.error}${status.errorAt ? ` (${status.errorAt})` : ''}`
                                        : '-'}
                                </td>
                            </tr>
                            <tr>
                                <th>网络检测</th>
                                <td>{snapshot?.availability.message ?? '-'}</td>
                            </tr>
                        </tbody>
                    </table>
                </section>
                <section className="panel">
                    <h3>当前节点详情</h3>
                    <table>
                        <tbody>
                            <tr>
                                <th>名称</th>
                                <td>{currentProfile?.name ?? '-'}</td>
                            </tr>
                            <tr>
                                <th>地址</th>
                                <td>{currentProfile?.address ?? '-'}</td>
                            </tr>
                            <tr>
                                <th>端口</th>
                                <td>{currentProfile?.port ?? '-'}</td>
                            </tr>
                            <tr>
                                <th>来源订阅</th>
                                <td>{currentProfile?.subName ?? '-'}</td>
                            </tr>
                        </tbody>
                    </table>
                </section>
            </div>
        </section>
    );
}