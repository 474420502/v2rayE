'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { api } from '@/lib/api/client';
import type { SubscriptionItem, SubscriptionUpsertInput } from '@/lib/types';

const EMPTY_FORM: SubscriptionUpsertInput = {
    remarks: '',
    url: '',
    enabled: true,
    userAgent: 'v2rayN/7.x',
    filter: '',
    convertTarget: '',
    autoUpdateMinutes: 120
};

export default function SubscriptionsPage() {
    const [items, setItems] = useState<SubscriptionItem[]>([]);
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const [keyword, setKeyword] = useState('');
    const [editingId, setEditingId] = useState('');
    const [form, setForm] = useState<SubscriptionUpsertInput>(EMPTY_FORM);
    const fallbackTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

    const load = useCallback(async () => {
        try {
            const list = await api.getSubscriptions();
            setItems(list);
            setError('');
        } catch (e) {
            setError(e instanceof Error ? e.message : '加载失败');
        }
    }, []);

    useEffect(() => {
        void load();

        const startFallbackPolling = () => {
            if (fallbackTimerRef.current !== null) {
                return;
            }
            fallbackTimerRef.current = setInterval(() => {
                void load();
            }, 3000);
        };

        const stopFallbackPolling = () => {
            if (fallbackTimerRef.current === null) {
                return;
            }
            clearInterval(fallbackTimerRef.current);
            fallbackTimerRef.current = null;
        };

        const endpoint = `${process.env.NEXT_PUBLIC_API_BASE ?? '/api'}/events/stream`;
        const source = new EventSource(endpoint, { withCredentials: true });

        source.onopen = () => {
            stopFallbackPolling();
        };

        source.onmessage = (event) => {
            try {
                const payload = JSON.parse(event.data) as { event?: string };
                if (payload.event === 'subscription.updated') {
                    void load();
                }
            } catch {
                void load();
            }
        };

        source.onerror = () => {
            startFallbackPolling();
        };

        return () => {
            source.close();
            stopFallbackPolling();
        };
    }, [load]);

    const runUpdateAll = async () => {
        setLoading(true);
        try {
            await api.updateSubscriptions();
            await load();
            setError('');
        } catch (e) {
            setError(e instanceof Error ? e.message : '更新失败');
        } finally {
            setLoading(false);
        }
    };

    const runUpdateOne = async (id: string) => {
        setLoading(true);
        try {
            await api.updateSubscriptionById(id);
            await load();
            setError('');
        } catch (e) {
            setError(e instanceof Error ? e.message : '更新失败');
        } finally {
            setLoading(false);
        }
    };

    const filtered = useMemo(() => {
        const text = keyword.trim().toLowerCase();
        if (!text) {
            return items;
        }
        return items.filter((item) =>
            `${item.remarks} ${item.url} ${item.userAgent ?? ''} ${item.filter ?? ''}`.toLowerCase().includes(text)
        );
    }, [items, keyword]);

    const enabledCount = useMemo(() => items.filter((item) => item.enabled).length, [items]);

    const submitForm = async () => {
        setLoading(true);
        try {
            if (editingId) {
                await api.updateSubscription(editingId, form);
            } else {
                await api.createSubscription(form);
            }
            setEditingId('');
            setForm(EMPTY_FORM);
            await load();
            setError('');
        } catch (e) {
            setError(e instanceof Error ? e.message : '保存失败');
        } finally {
            setLoading(false);
        }
    };

    const editItem = (item: SubscriptionItem) => {
        setEditingId(item.id);
        setForm({
            remarks: item.remarks,
            url: item.url,
            enabled: item.enabled ?? true,
            userAgent: item.userAgent ?? 'v2rayN/7.x',
            filter: item.filter ?? '',
            convertTarget: item.convertTarget ?? '',
            autoUpdateMinutes: item.autoUpdateMinutes ?? 120
        });
    };

    const removeItem = async (id: string) => {
        setLoading(true);
        try {
            await api.deleteSubscription(id);
            if (editingId === id) {
                setEditingId('');
                setForm(EMPTY_FORM);
            }
            await load();
            setError('');
        } catch (e) {
            setError(e instanceof Error ? e.message : '删除失败');
        } finally {
            setLoading(false);
        }
    };

    const resetForm = () => {
        setEditingId('');
        setForm(EMPTY_FORM);
    };

    return (
        <section className="page">
            <div className="page-header">
                <div>
                    <h2>订阅</h2>
                    <p className="muted">补齐新增、编辑、删除与高级字段，缩小和 v2rayN 订阅管理的差距。</p>
                </div>
                <div className="stats-inline">
                    <span>总数 {items.length}</span>
                    <span>启用 {enabledCount}</span>
                    <span>{editingId ? '编辑模式' : '新增模式'}</span>
                </div>
            </div>
            <div className="toolbar">
                <button className="primary" disabled={loading} onClick={() => void runUpdateAll()}>
                    全量更新
                </button>
                <button onClick={() => void load()}>刷新列表</button>
                <input
                    placeholder="搜索名称、URL、UA、过滤规则"
                    value={keyword}
                    onChange={(event) => setKeyword(event.target.value)}
                />
            </div>
            {error ? <p className="status-error">{error}</p> : null}
            <div className="panel-grid two-up">
                <section className="panel">
                    <h3>{editingId ? '编辑订阅源' : '新增订阅源'}</h3>
                    <div className="form-grid two-up">
                        <div className="field">
                            <label htmlFor="sub-remarks">名称</label>
                            <input
                                id="sub-remarks"
                                value={form.remarks}
                                onChange={(event) => setForm((prev) => ({ ...prev, remarks: event.target.value }))}
                            />
                        </div>
                        <div className="field">
                            <label htmlFor="sub-interval">自动更新（分钟）</label>
                            <input
                                id="sub-interval"
                                type="number"
                                min={0}
                                value={form.autoUpdateMinutes ?? 0}
                                onChange={(event) =>
                                    setForm((prev) => ({ ...prev, autoUpdateMinutes: Number(event.target.value) || 0 }))
                                }
                            />
                        </div>
                    </div>
                    <div className="field">
                        <label htmlFor="sub-url">URL</label>
                        <input
                            id="sub-url"
                            value={form.url}
                            onChange={(event) => setForm((prev) => ({ ...prev, url: event.target.value }))}
                        />
                    </div>
                    <div className="form-grid two-up">
                        <div className="field">
                            <label htmlFor="sub-ua">User-Agent</label>
                            <input
                                id="sub-ua"
                                value={form.userAgent ?? ''}
                                onChange={(event) => setForm((prev) => ({ ...prev, userAgent: event.target.value }))}
                            />
                        </div>
                        <div className="field">
                            <label htmlFor="sub-target">转换目标</label>
                            <input
                                id="sub-target"
                                value={form.convertTarget ?? ''}
                                onChange={(event) => setForm((prev) => ({ ...prev, convertTarget: event.target.value }))}
                                placeholder="clash / sing-box / 原始"
                            />
                        </div>
                    </div>
                    <div className="field">
                        <label htmlFor="sub-filter">过滤规则</label>
                        <input
                            id="sub-filter"
                            value={form.filter ?? ''}
                            onChange={(event) => setForm((prev) => ({ ...prev, filter: event.target.value }))}
                            placeholder="例如 HK|JP|Premium"
                        />
                    </div>
                    <label className="toggle-row">
                        <input
                            type="checkbox"
                            checked={form.enabled}
                            onChange={(event) => setForm((prev) => ({ ...prev, enabled: event.target.checked }))}
                        />
                        启用此订阅
                    </label>
                    <div className="toolbar compact">
                        <button className="primary" disabled={loading} onClick={() => void submitForm()}>
                            {editingId ? '保存修改' : '新增订阅'}
                        </button>
                        <button disabled={loading} onClick={resetForm}>重置</button>
                    </div>
                </section>
                <section className="panel">
                    <h3>订阅清单</h3>
                    <div className="table-wrap compact-table">
                        <table>
                            <thead>
                                <tr>
                                    <th>名称</th>
                                    <th>状态</th>
                                    <th>最近更新</th>
                                    <th>高级参数</th>
                                    <th>操作</th>
                                </tr>
                            </thead>
                            <tbody>
                                {filtered.map((item) => (
                                    <tr key={item.id} className={editingId === item.id ? 'row-active' : ''}>
                                        <td>
                                            <strong>{item.remarks}</strong>
                                            <div className="cell-note">{item.url}</div>
                                        </td>
                                        <td>
                                            <span className={item.enabled ? 'status-pill ok' : 'status-pill muted'}>
                                                {item.enabled ? '启用' : '停用'}
                                            </span>
                                        </td>
                                        <td>{item.updatedAt ?? '-'}</td>
                                        <td>
                                            <div className="cell-note">UA: {item.userAgent ?? '-'}</div>
                                            <div className="cell-note">过滤: {item.filter ?? '-'}</div>
                                            <div className="cell-note">转换: {item.convertTarget ?? '-'}</div>
                                        </td>
                                        <td>
                                            <div className="actions-inline">
                                                <button disabled={loading} onClick={() => void runUpdateOne(item.id)}>
                                                    更新
                                                </button>
                                                <button disabled={loading} onClick={() => editItem(item)}>
                                                    编辑
                                                </button>
                                                <button className="danger" disabled={loading} onClick={() => void removeItem(item.id)}>
                                                    删除
                                                </button>
                                            </div>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                </section>
            </div>
        </section>
    );
}