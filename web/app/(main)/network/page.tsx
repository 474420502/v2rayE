'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { api, buildEventSourceUrl } from '@/lib/api/client';
import type { AvailabilityResult, ConfigDto, CoreStatus, RoutingConfig } from '@/lib/types';

type NetworkSnapshot = {
    status: CoreStatus;
    availability: AvailabilityResult;
    config: ConfigDto;
    routing: RoutingConfig;
};

export default function NetworkPage() {
    const [snapshot, setSnapshot] = useState<NetworkSnapshot | null>(null);
    const [exceptions, setExceptions] = useState('');
    const [error, setError] = useState('');
    const [busy, setBusy] = useState('');
    const fallbackTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

    const apply = async (mode: 'forced_change' | 'forced_clear') => {
        setBusy(mode);
        try {
            await api.applySystemProxy(mode, exceptions);
            await check();
            setError('');
        } catch (e) {
            setError(e instanceof Error ? e.message : '操作失败');
        } finally {
            setBusy('');
        }
    };

    const check = useCallback(async () => {
        try {
            const [status, availability, config, routing] = await Promise.all([
                api.getCoreStatus(),
                api.getAvailability(),
                api.getConfig(),
                api.getRouting()
            ]);
            setSnapshot({
                status,
                availability,
                config,
                routing
            });
            const persistedExceptions =
                typeof config.systemProxyExceptions === 'string' ? config.systemProxyExceptions : '';
            setExceptions((current) => (current === persistedExceptions ? current : persistedExceptions));
            setError('');
        } catch (e) {
            setError(e instanceof Error ? e.message : '检测失败');
        }
    }, []);

    const proxyMode = useMemo(() => {
        const mode = snapshot?.config.systemProxyMode;
        if (typeof mode !== 'string' || mode.length === 0) {
            return '-';
        }
        return mode;
    }, [snapshot]);

    const routingMode = useMemo(() => snapshot?.routing.mode ?? '-', [snapshot]);

    const tunMode = useMemo(() => {
        const config = snapshot?.config;
        if (!config) {
            return '-';
        }
        if (typeof config.tunMode === 'string' && config.tunMode.length > 0) {
            return config.tunMode;
        }
        if (config.enableTun) {
            return typeof config.tunStack === 'string' && config.tunStack.length > 0 ? config.tunStack : 'mixed';
        }
        return 'off';
    }, [snapshot]);

    const desktopBackend = useMemo(() => {
        const backend = snapshot?.config.systemProxyBackend;
        return typeof backend === 'string' && backend.length > 0 ? backend : '-';
    }, [snapshot]);

    const endpointHost = useMemo(() => {
        const host = snapshot?.config.listenAddr;
        return host && host !== '0.0.0.0' ? host : '127.0.0.1';
    }, [snapshot]);

    const httpEndpoint = useMemo(
        () => `${endpointHost}:${snapshot?.config.httpPort ?? 10809}`,
        [endpointHost, snapshot]
    );

    const socksEndpoint = useMemo(
        () => `${endpointHost}:${snapshot?.config.socksPort ?? 10808}`,
        [endpointHost, snapshot]
    );

    const coreError = useMemo(() => snapshot?.status.error?.trim() ?? '', [snapshot]);

    const tunEnabled = tunMode !== '-' && tunMode !== 'off';

    const tunPermissionFailure = useMemo(() => {
        if (!tunEnabled || coreError.length === 0) {
            return false;
        }
        return /(root|privilege|permission denied|operation not permitted|cap_net_admin|requires root|timed out waiting for tun interface|ip command)/i.test(coreError);
    }, [coreError, tunEnabled]);

    const systemProxyState = useMemo(() => {
        if (proxyMode === 'forced_change') {
            return snapshot?.status.running
                ? {
                    tone: 'ok',
                    title: '系统代理已接管',
                    detail: `桌面代理后端 ${desktopBackend} 已指向 ${httpEndpoint}，浏览器和遵守系统代理设置的应用会自动跟随。`
                }
                : {
                    tone: 'warn',
                    title: '等待核心启动后接管',
                    detail: '系统代理策略已配置为自动应用，但当前核心未运行，桌面代理不会自动出站。'
                };
        }
        if (proxyMode === 'pac') {
            return {
                tone: 'warn',
                title: 'PAC 尚未实现',
                detail: 'Linux 桌面集成目前未实现 PAC，当前不能依赖 PAC 接管系统流量。'
            };
        }
        return {
            tone: 'muted',
            title: '系统代理未接管',
            detail: '当前为 forced_clear。即使本地 10809 可用，桌面应用也不会自动走代理。'
        };
    }, [desktopBackend, httpEndpoint, proxyMode, snapshot?.status.running]);

    const tunState = useMemo(() => {
        if (!tunEnabled) {
            return {
                tone: 'muted',
                title: 'TUN 未启用',
                detail: '当前不会透明接管 CLI、容器、Electron 或不读取桌面代理设置的程序。'
            };
        }
        if (snapshot?.status.running) {
            return {
                tone: 'ok',
                title: 'TUN 已启用',
                detail: `当前模式 ${tunMode}。更多不读取系统代理的流量会尝试进入 xray-core。`
            };
        }
        if (tunPermissionFailure) {
            return {
                tone: 'warn',
                title: 'TUN 因权限失败',
                detail: `检测到核心错误与权限或路由相关：${coreError}`
            };
        }
        return {
            tone: 'warn',
            title: 'TUN 未运行',
            detail: coreError || 'TUN 已配置，但当前核心未运行或 TUN 入口尚未成功建立。'
        };
    }, [coreError, snapshot?.status.running, tunEnabled, tunMode, tunPermissionFailure]);

    const coverageState = useMemo(() => {
        if (tunEnabled && snapshot?.status.running) {
            return {
                tone: 'ok',
                title: '系统级接管优先',
                detail: '优先使用 TUN 覆盖更多程序；系统代理仍可作为桌面应用的补充入口。'
            };
        }
        if (proxyMode === 'forced_change') {
            return {
                tone: 'warn',
                title: '仅桌面代理接管',
                detail: '浏览器等会跟随，但命令行、部分 Electron、容器和自带网络栈程序通常不会自动走代理。'
            };
        }
        return {
            tone: 'muted',
            title: '未自动接管系统流量',
            detail: '目前只有手动设置应用内代理、或显式使用 127.0.0.1:10809 / 10808 的请求才会走代理。'
        };
    }, [proxyMode, snapshot?.status.running, tunEnabled]);

    useEffect(() => {
        void check();

        const startFallbackPolling = () => {
            if (fallbackTimerRef.current !== null) {
                return;
            }
            fallbackTimerRef.current = setInterval(() => {
                void check();
            }, 3000);
        };

        const stopFallbackPolling = () => {
            if (fallbackTimerRef.current === null) {
                return;
            }
            clearInterval(fallbackTimerRef.current);
            fallbackTimerRef.current = null;
        };

        const source = new EventSource(buildEventSourceUrl('/events/stream'), { withCredentials: true });

        source.onopen = () => {
            stopFallbackPolling();
        };

        source.onmessage = (event) => {
            try {
                const payload = JSON.parse(event.data) as { event?: string };
                if (
                    payload.event === 'proxy.changed' ||
                    payload.event === 'config.updated' ||
                    payload.event === 'routing.updated' ||
                    payload.event?.startsWith('core.')
                ) {
                    void check();
                }
            } catch {
                void check();
            }
        };

        source.onerror = () => {
            startFallbackPolling();
        };

        return () => {
            source.close();
            stopFallbackPolling();
        };
    }, [check]);

    return (
        <section className="page">
            <div className="page-header">
                <div>
                    <h2>系统代理与网络</h2>
                    <p className="muted">系统代理只影响遵守桌面代理设置的应用；Xray 路由模式决定进入代理内核后的分流，两者不是同一个开关。</p>
                </div>
                <div className="stats-inline">
                    <span>系统代理 {proxyMode}</span>
                    <span>Xray 路由 {routingMode}</span>
                    <span>TUN {tunMode}</span>
                    <span>
                        引擎 {(snapshot?.status.engineMode ?? snapshot?.config.coreEngine ?? 'embedded')}
                        {' -> '}
                        {(snapshot?.status.engineResolved ?? snapshot?.status.coreType ?? '-')}
                    </span>
                </div>
            </div>
            <section className="panel" style={{ marginBottom: 16 }}>
                <h3>流量接管状态</h3>
                <div className="capture-grid">
                    <article className={`capture-card tone-${systemProxyState.tone}`}>
                        <div className={`status-pill ${systemProxyState.tone}`}>{systemProxyState.title}</div>
                        <strong>系统代理</strong>
                        <p>{systemProxyState.detail}</p>
                    </article>
                    <article className={`capture-card tone-${tunState.tone}`}>
                        <div className={`status-pill ${tunState.tone}`}>{tunState.title}</div>
                        <strong>TUN</strong>
                        <p>{tunState.detail}</p>
                    </article>
                    <article className={`capture-card tone-${coverageState.tone}`}>
                        <div className={`status-pill ${coverageState.tone}`}>{coverageState.title}</div>
                        <strong>覆盖范围</strong>
                        <p>{coverageState.detail}</p>
                    </article>
                </div>
            </section>
            <section className="panel" style={{ marginBottom: 16 }}>
                <h3>系统代理说明</h3>
                <table>
                    <tbody>
                        <tr>
                            <th>桌面代理后端</th>
                            <td>{desktopBackend}</td>
                        </tr>
                        <tr>
                            <th>HTTP / HTTPS</th>
                            <td>{httpEndpoint}</td>
                        </tr>
                        <tr>
                            <th>SOCKS5</th>
                            <td>{socksEndpoint}</td>
                        </tr>
                        <tr>
                            <th>适用范围</th>
                            <td>浏览器和遵守桌面代理设置的应用；命令行和不读取桌面代理的程序不会自动跟随。</td>
                        </tr>
                        <tr>
                            <th>Xray 路由模式</th>
                            <td>{routingMode}</td>
                        </tr>
                        <tr>
                            <th>TUN 模式</th>
                            <td>{tunMode}</td>
                        </tr>
                    </tbody>
                </table>
            </section>
            <div className="field">
                <label htmlFor="exceptions">代理例外（可选）</label>
                <textarea
                    id="exceptions"
                    rows={3}
                    value={exceptions}
                    onChange={(event) => setExceptions(event.target.value)}
                    placeholder="localhost,127.0.0.1"
                />
            </div>
            <div className="toolbar">
                <button className="primary" disabled={busy !== ''} onClick={() => void apply('forced_change')}>
                    应用系统代理
                </button>
                <button disabled={busy !== ''} onClick={() => void apply('forced_clear')}>
                    清理系统代理
                </button>
                <button onClick={() => void check()}>可用性检测</button>
                <button
                    disabled={busy !== '' || !snapshot?.status.error?.trim()}
                    onClick={async () => {
                        setBusy('clear-core-error');
                        try {
                            await api.clearCoreError();
                            await check();
                            setError('');
                        } catch (e) {
                            setError(e instanceof Error ? e.message : '操作失败');
                        } finally {
                            setBusy('');
                        }
                    }}
                >
                    清空核心错误
                </button>
            </div>
            {error ? <p className="status-error">{error}</p> : null}
            <div className="panel-grid">
                <section className="panel">
                    <h3>网络检测</h3>
                    {snapshot ? (
                        <table>
                            <tbody>
                                <tr>
                                    <th>可用性</th>
                                    <td className={snapshot.availability.available ? 'status-ok' : 'status-error'}>
                                        {snapshot.availability.available ? '可用' : '不可用'}
                                    </td>
                                </tr>
                                <tr>
                                    <th>耗时</th>
                                    <td>{snapshot.availability.elapsedMs ?? '-'} ms</td>
                                </tr>
                                <tr>
                                    <th>信息</th>
                                    <td>{snapshot.availability.message ?? '-'}</td>
                                </tr>
                                <tr>
                                    <th>系统代理模式</th>
                                    <td>{proxyMode}</td>
                                </tr>
                                <tr>
                                    <th>桌面代理后端</th>
                                    <td>{desktopBackend}</td>
                                </tr>
                                <tr>
                                    <th>Xray 路由模式</th>
                                    <td>{routingMode}</td>
                                </tr>
                                <tr>
                                    <th>TUN 模式</th>
                                    <td>{tunMode}</td>
                                </tr>
                                <tr>
                                    <th>引擎策略与实际执行</th>
                                    <td>
                                        {(snapshot.status.engineMode ?? snapshot.config.coreEngine ?? 'embedded')}
                                        {' -> '}
                                        {(snapshot.status.engineResolved ?? snapshot.status.coreType ?? '-')}
                                    </td>
                                </tr>
                                <tr>
                                    <th>核心最近错误</th>
                                    <td>
                                        {snapshot.status.error?.trim()
                                            ? `${snapshot.status.error}${snapshot.status.errorAt ? ` (${snapshot.status.errorAt})` : ''}`
                                            : '-'}
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    ) : null}
                </section>
            </div>
        </section>
    );
}