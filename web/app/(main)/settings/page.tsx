'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '@/lib/api/client';
import type { ConfigDto, CoreStatus } from '@/lib/types';

export default function SettingsPage() {
    const [config, setConfig] = useState<ConfigDto | null>(null);
    const [status, setStatus] = useState<CoreStatus | null>(null);
    const [error, setError] = useState('');
    const [saving, setSaving] = useState(false);
    const [saved, setSaved] = useState(false);
    const [cleaning, setCleaning] = useState(false);
    const fallbackTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

    const load = useCallback(async () => {
        try {
            const [cfg, st] = await Promise.all([api.getConfig(), api.getCoreStatus()]);
            setConfig(cfg);
            setStatus(st);
            setError('');
        } catch (e) {
            setError(e instanceof Error ? e.message : '加载失败');
        }
    }, []);

    useEffect(() => {
        void load();
        const startFallback = () => {
            if (fallbackTimerRef.current !== null) return;
            fallbackTimerRef.current = setInterval(() => void load(), 5000);
        };
        const stopFallback = () => {
            if (fallbackTimerRef.current === null) return;
            clearInterval(fallbackTimerRef.current);
            fallbackTimerRef.current = null;
        };
        const endpoint = `${process.env.NEXT_PUBLIC_API_BASE ?? '/api'}/events/stream`;
        const source = new EventSource(endpoint, { withCredentials: true });
        source.onopen = () => stopFallback();
        source.onmessage = (event) => {
            try {
                const payload = JSON.parse(event.data) as {
                    event?: string;
                    data?: unknown;
                };
                if (!payload.event) {
                    void load();
                    return;
                }
                if (payload.event === 'config.updated') {
                    const data = payload.data as {
                        config?: ConfigDto;
                        status?: CoreStatus;
                    } | undefined;
                    if (data?.config) {
                        setConfig((prev) => ({ ...(prev ?? {}), ...data.config }));
                    } else {
                        void load();
                    }
                    if (data?.status) {
                        setStatus(data.status);
                    }
                    return;
                }
                if (payload.event.startsWith('core.')) {
                    const core = payload.data as CoreStatus | undefined;
                    if (core) {
                        setStatus(core);
                    } else {
                        void load();
                    }
                }
            } catch { void load(); }
        };
        source.onerror = () => startFallback();
        return () => { source.close(); stopFallback(); };
    }, [load]);

    const set = (patch: Partial<ConfigDto>) => {
        setConfig((prev) => prev ? { ...prev, ...patch } : prev);
    };

    const tunMode = !config
        ? 'off'
        : typeof config.tunMode === 'string'
            ? config.tunMode
            : config.enableTun
                ? (typeof config.tunStack === 'string' && config.tunStack.length > 0 ? config.tunStack : 'mixed')
                : 'off';

    const save = async () => {
        if (!config) return;
        setSaving(true);
        try {
            const updated = await api.updateConfig(config);
            setConfig(updated);
            setSaved(true);
            setTimeout(() => setSaved(false), 2000);
            setError('');
        } catch (e) {
            setError(e instanceof Error ? e.message : '保存失败');
        } finally {
            setSaving(false);
        }
    };

    if (!config) {
        return <section className="page"><p className="muted">{error || '加载中...'}</p></section>;
    }

    return (
        <section className="page">
            <div className="page-header">
                <div>
                    <h2>设置</h2>
                    <p className="muted">修改代理端口、Xray 可执行文件路径、日志级别和 DNS，保存后会自动重启核心。</p>
                </div>
                <div className="toolbar compact">
                    <button onClick={() => void load()}>重置</button>
                    <button
                        disabled={cleaning}
                        onClick={async () => {
                            setCleaning(true);
                            try {
                                const result = await api.exitCleanup(false);
                                setStatus(result.status);
                                setError(result.proxyCleared ? '' : (result.proxyClearError || '系统代理清理失败'));
                            } catch (e) {
                                setError(e instanceof Error ? e.message : '清理失败');
                            } finally {
                                setCleaning(false);
                            }
                        }}
                    >
                        {cleaning ? '清理中...' : '主动清理残留'}
                    </button>
                    <button
                        disabled={cleaning}
                        onClick={async () => {
                            if (!confirm('将执行清理并关闭后端，确定继续？')) {
                                return;
                            }
                            setCleaning(true);
                            try {
                                await api.exitCleanup(true);
                                setStatus((prev) => prev ? { ...prev, running: false, state: 'stopped' } : prev);
                                setError('后端已触发退出清理，页面稍后会断开连接。');
                            } catch (e) {
                                setError(e instanceof Error ? e.message : '退出清理失败');
                            } finally {
                                setCleaning(false);
                            }
                        }}
                    >
                        清理并退出后端
                    </button>
                    <button
                        disabled={saving || !status?.error?.trim()}
                        onClick={async () => {
                            try {
                                const st = await api.clearCoreError();
                                setStatus(st);
                                setError('');
                            } catch (e) {
                                setError(e instanceof Error ? e.message : '清空错误失败');
                            }
                        }}
                    >
                        清空核心错误
                    </button>
                    <button className="primary" onClick={() => void save()} disabled={saving}>
                        {saved ? '已保存 ✓' : saving ? '保存中...' : '保存并重启'}
                    </button>
                </div>
            </div>

            {error ? <p className="status-error">{error}</p> : null}

            <section className="panel" style={{ marginBottom: 16 }}>
                <h3>代理端口</h3>
                <div className="form-grid" style={{ marginTop: 12 }}>
                    <label>SOCKS5 端口</label>
                    <input
                        type="number" min={1024} max={65535}
                        value={config.socksPort ?? 10808}
                        onChange={(e) => set({ socksPort: Number(e.target.value) })}
                    />
                    <label>HTTP 端口</label>
                    <input
                        type="number" min={1024} max={65535}
                        value={config.httpPort ?? 10809}
                        onChange={(e) => set({ httpPort: Number(e.target.value) })}
                    />
                    <label>Stats 端口 (内部)</label>
                    <input
                        type="number" min={1024} max={65535}
                        value={config.statsPort ?? 10085}
                        onChange={(e) => set({ statsPort: Number(e.target.value) })}
                    />
                    <label>允许局域网连接</label>
                    <input
                        type="checkbox"
                        checked={Boolean(config.allowLan)}
                        onChange={(e) => set({ allowLan: e.target.checked })}
                    />
                </div>
            </section>

            <section className="panel" style={{ marginBottom: 16 }}>
                <h3>核心配置</h3>
                <p className="muted" style={{ marginBottom: 8 }}>
                    运行状态：
                    {(status?.engineMode ?? config.coreEngine ?? 'embedded')}
                    {' -> '}
                    {(status?.engineResolved ?? status?.coreType ?? '-')}
                    {status?.error?.trim()
                        ? `，最近错误：${status.error}${status.errorAt ? ` (${status.errorAt})` : ''}`
                        : ''}
                </p>
                <div className="form-grid" style={{ marginTop: 12 }}>
                    <label>核心引擎模式</label>
                    <select
                        value={typeof config.coreEngine === 'string' ? config.coreEngine : 'xray-core'}
                        onChange={(e) => set({ coreEngine: e.target.value })}
                    >
                        <option value="xray-core">xray-core (内嵌全协议内核)</option>
                    </select>
                    <label>Xray 可执行文件</label>
                    <input
                        value={config.xrayCmd ?? 'xray'}
                        onChange={(e) => set({ xrayCmd: e.target.value })}
                        placeholder="xray 或 /usr/local/bin/xray"
                    />
                    <label>日志级别</label>
                    <select
                        value={config.logLevel ?? 'warning'}
                        onChange={(e) => set({ logLevel: e.target.value })}
                    >
                        {['debug', 'info', 'warning', 'error', 'none'].map((l) => (
                            <option key={l}>{l}</option>
                        ))}
                    </select>
                </div>
                <p className="muted" style={{ marginTop: 8 }}>
                    轻量 embedded 引擎已移除，当前统一由内嵌 xray-core 处理所有协议与 TUN 场景。
                </p>
            </section>

            <section className="panel" style={{ marginBottom: 16 }}>
                <h3>系统代理与 TUN</h3>
                <p className="muted" style={{ marginBottom: 8 }}>
                    系统代理会在核心启动时自动应用、停止时自动清理。TUN 会接管更多不读取桌面代理设置的流量；Linux 下启用 TUN 时请使用 sudo 启动，停止时也建议使用 sudo 以确保路由与网卡清理完整。
                </p>
                <div className="form-grid" style={{ marginTop: 12 }}>
                    <label>系统代理策略</label>
                    <select
                        value={typeof config.systemProxyMode === 'string' ? config.systemProxyMode : 'forced_clear'}
                        onChange={(e) => set({ systemProxyMode: e.target.value })}
                    >
                        <option value="forced_clear">关闭</option>
                        <option value="forced_change">随核心自动应用</option>
                    </select>
                    <label>TUN 模式</label>
                    <select
                        value={tunMode}
                        onChange={(e) => {
                            const value = e.target.value;
                            set({
                                tunMode: value,
                                enableTun: value !== 'off',
                                tunStack: value === 'off' ? config.tunStack ?? 'mixed' : value
                            });
                        }}
                    >
                        <option value="off">关闭</option>
                        <option value="system">system</option>
                        <option value="mixed">mixed</option>
                        <option value="gvisor">gvisor</option>
                    </select>
                    <label>TUN 名称</label>
                    <input
                        value={config.tunName ?? 'xraye0'}
                        onChange={(e) => set({ tunName: e.target.value })}
                    />
                    <label>TUN MTU</label>
                    <input
                        type="number" min={1280} max={9000}
                        value={config.tunMtu ?? 1500}
                        onChange={(e) => set({ tunMtu: Number(e.target.value) })}
                    />
                    <label>自动改默认路由</label>
                    <input
                        type="checkbox"
                        checked={Boolean(config.tunAutoRoute ?? true)}
                        onChange={(e) => set({ tunAutoRoute: e.target.checked })}
                    />
                    <label>严格路由</label>
                    <input
                        type="checkbox"
                        checked={Boolean(config.tunStrictRoute)}
                        onChange={(e) => set({ tunStrictRoute: e.target.checked })}
                    />
                </div>
                <div className="field" style={{ marginTop: 12 }}>
                    <label htmlFor="proxy-exceptions">系统代理例外</label>
                    <textarea
                        id="proxy-exceptions"
                        rows={3}
                        value={typeof config.systemProxyExceptions === 'string' ? config.systemProxyExceptions : ''}
                        onChange={(e) => set({ systemProxyExceptions: e.target.value })}
                        placeholder="localhost,127.0.0.1,::1"
                    />
                </div>
            </section>

            <section className="panel">
                <h3>DNS</h3>
                <p className="muted" style={{ marginBottom: 8 }}>每行一条 DNS 服务器地址</p>
                <textarea
                    style={{ width: '100%', minHeight: 100, fontFamily: 'monospace', fontSize: 13 }}
                    value={(config.dnsList ?? []).join('\n')}
                    onChange={(e) => set({ dnsList: e.target.value.split('\n').map((s) => s.trim()).filter(Boolean) })}
                    placeholder={'8.8.8.8\n1.1.1.1\nhttps://dns.google/dns-query'}
                />
            </section>
        </section>
    );
}
