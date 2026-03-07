'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ImportUriDialog } from '@/components/profiles/import-uri-dialog';
import { ProfileForm, type ProfileFormState } from '@/components/profiles/profile-form';
import { ProtoBadge } from '@/components/profiles/proto-badge';
import { PROTOCOLS, PROTOCOL_LABELS, blankProfile, type Protocol } from '@/components/profiles/protocols';
import { api } from '@/lib/api/client';
import type { CoreStatus, DelayTestResult, ProfileItem } from '@/lib/types';

// ─── Main page ────────────────────────────────────────────────────────────────

export default function ProfilesPage() {
  const [profiles, setProfiles] = useState<ProfileItem[]>([]);
  const [coreStatus, setCoreStatus] = useState<CoreStatus | null>(null);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [keyword, setKeyword] = useState('');
  const [sourceFilter, setSourceFilter] = useState('all');
  const [error, setError] = useState('');
  const [testingId, setTestingId] = useState('');
  const [delayResults, setDelayResults] = useState<Record<string, DelayTestResult>>({});
  const [editTarget, setEditTarget] = useState<ProfileFormState | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [showImport, setShowImport] = useState(false);
  const [importURI, setImportURI] = useState('');
  const [isImporting, setIsImporting] = useState(false);
  const fallbackTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const load = useCallback(async () => {
    try {
      const [items, status] = await Promise.all([api.getProfiles(), api.getCoreStatus()]);
      setProfiles(items);
      setCoreStatus(status);
      setError('');
    } catch (e) {
      setError(e instanceof Error ? e.message : '加载失败');
    }
  }, []);

  useEffect(() => {
    void load();
    const startFallback = () => {
      if (fallbackTimerRef.current !== null) return;
      fallbackTimerRef.current = setInterval(() => void load(), 3000);
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
        const payload = JSON.parse(event.data) as { event?: string };
        if (payload.event?.startsWith('profile.') || payload.event === 'subscription.updated') {
          void load();
        }
      } catch { void load(); }
    };
    source.onerror = () => startFallback();
    return () => { source.close(); stopFallback(); };
  }, [load]);

  const filtered = useMemo(() => {
    const text = keyword.trim().toLowerCase();
    return profiles.filter((item) => {
      const matchesKeyword = !text ||
        `${item.name} ${item.address} ${item.subName ?? ''}`.toLowerCase().includes(text);
      const matchesSource = sourceFilter === 'all' || item.subName === sourceFilter;
      return matchesKeyword && matchesSource;
    });
  }, [keyword, profiles, sourceFilter]);

  const sources = useMemo(
    () => ['all', ...new Set(profiles.map((item) => item.subName).filter(Boolean) as string[])],
    [profiles]
  );
  const filteredIds = useMemo(() => filtered.map((item) => item.id), [filtered]);
  const allFilteredSelected = useMemo(
    () => filteredIds.length > 0 && filteredIds.every((id) => selectedIds.includes(id)),
    [filteredIds, selectedIds]
  );

  const currentId = coreStatus?.currentProfileId ?? '';

  useEffect(() => {
    setSelectedIds((prev) => prev.filter((id) => profiles.some((item) => item.id === id)));
  }, [profiles]);

  const selectProfile = async (id: string) => {
    try {
      await api.selectProfile(id);
      await load();
      setError('');
    } catch (e) { setError(e instanceof Error ? e.message : '切换失败'); }
  };

  const testDelay = async (id: string) => {
    setTestingId(id);
    try {
      const result = await api.testProfileDelay(id);
      setDelayResults((prev) => ({ ...prev, [id]: result }));
      setError('');
    } catch (e) { setError(e instanceof Error ? e.message : '测速失败'); }
    finally { setTestingId(''); }
  };

  const deleteProfiles = async (ids: string[]) => {
    const uniqueIDs = [...new Set(ids.filter(Boolean))];
    if (uniqueIDs.length === 0) return;
    const message = uniqueIDs.length === 1 ? '确认删除该节点？' : `确认删除选中的 ${uniqueIDs.length} 个节点？`;
    if (!confirm(message)) return;
    try {
      await api.deleteProfiles(uniqueIDs);
      setSelectedIds((prev) => prev.filter((id) => !uniqueIDs.includes(id)));
      await load();
    } catch (e) { setError(e instanceof Error ? e.message : '删除失败'); }
  };

  const toggleSelection = (id: string) => {
    setSelectedIds((prev) => (
      prev.includes(id)
        ? prev.filter((item) => item !== id)
        : [...prev, id]
    ));
  };

  const toggleSelectAllFiltered = () => {
    setSelectedIds((prev) => {
      if (allFilteredSelected) {
        return prev.filter((id) => !filteredIds.includes(id));
      }
      const next = new Set(prev);
      filteredIds.forEach((id) => next.add(id));
      return Array.from(next);
    });
  };

  const openCreate = (protocol: Protocol) => {
    setEditTarget(blankProfile(protocol));
  };

  const openEdit = async (id: string) => {
    try {
      const p = await api.getProfile(id);
      setEditTarget(p);
    } catch (e) { setError(e instanceof Error ? e.message : '加载失败'); }
  };

  const saveProfile = async (form: ProfileFormState) => {
    setIsSaving(true);
    try {
      const existingId = (editTarget as ProfileItem)?.id;
      if (existingId) {
        await api.updateProfile(existingId, form);
      } else {
        await api.createProfile(form);
      }
      setEditTarget(null);
      await load();
    } catch (e) { setError(e instanceof Error ? e.message : '保存失败'); }
    finally { setIsSaving(false); }
  };

  const doImport = async () => {
    const uri = importURI.trim();
    if (!uri) return;
    setIsImporting(true);
    try {
      await api.importProfileFromURI(uri);
      setShowImport(false);
      setImportURI('');
      await load();
    } catch (e) { setError(e instanceof Error ? e.message : '导入失败'); }
    finally { setIsImporting(false); }
  };

  return (
    <section className="page">
      <div className="page-header">
        <div>
          <h2>节点</h2>
          <p className="muted">管理所有代理节点：新增、编辑、批量删除、一键测速、切换当前节点。</p>
        </div>
        <div className="stats-inline">
          <span>合计 {profiles.length}</span>
          <span>订阅 {sources.length - 1}</span>
          <span>已选 {selectedIds.length}</span>
        </div>
      </div>

      <div className="toolbar">
        <input
          placeholder="按名称/地址/订阅搜索"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
        />
        <select value={sourceFilter} onChange={(e) => setSourceFilter(e.target.value)}>
          {sources.map((s) => (
            <option key={s} value={s}>{s === 'all' ? '全部来源' : s}</option>
          ))}
        </select>
        <button onClick={() => void load()}>刷新</button>
        <button onClick={() => void deleteProfiles(selectedIds)} disabled={selectedIds.length === 0}>
          删除选中
        </button>
        <button onClick={() => setShowImport(true)}>导入链接</button>
        <div style={{ position: 'relative' }}>
          <select
            value=""
            onChange={(e) => { if (e.target.value) openCreate(e.target.value as Protocol); }}
            style={{ paddingRight: 8 }}
          >
            <option value="">+ 新增节点</option>
            {PROTOCOLS.map((p) => (
              <option key={p} value={p}>{PROTOCOL_LABELS[p]}</option>
            ))}
          </select>
        </div>
      </div>

      {error ? <p className="status-error">{error}</p> : null}

      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>
                <input
                  type="checkbox"
                  checked={allFilteredSelected}
                  aria-label="选择当前筛选结果"
                  onChange={toggleSelectAllFiltered}
                />
              </th>
              <th>当前</th>
              <th>名称</th>
              <th>协议</th>
              <th>地址</th>
              <th>端口</th>
              <th>延迟</th>
              <th>来源</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((item) => {
              const isCurrent = item.id === currentId;
              const isSelected = selectedIds.includes(item.id);
              const tested = delayResults[item.id];
              const displayDelay = tested?.delayMs ?? item.delayMs;
              return (
                <tr key={item.id} className={isCurrent ? 'row-active' : ''}>
                  <td>
                    <input
                      type="checkbox"
                      checked={isSelected}
                      aria-label={`选择 ${item.name}`}
                      onChange={() => toggleSelection(item.id)}
                    />
                  </td>
                  <td>
                    <span className={isCurrent ? 'status-pill ok' : 'status-pill muted'}>
                      {isCurrent ? '当前' : '•'}
                    </span>
                  </td>
                  <td><strong>{item.name}</strong></td>
                  <td><ProtoBadge protocol={item.protocol} /></td>
                  <td>{item.address || '-'}</td>
                  <td>{item.port || '-'}</td>
                  <td>
                    {displayDelay != null ? `${displayDelay} ms` : '-'}
                    {tested?.message && tested.message !== 'ok' ? (
                      <div className="cell-note">{tested.message}</div>
                    ) : null}
                  </td>
                  <td>{item.subName || '-'}</td>
                  <td>
                    <div className="actions-inline">
                      <button onClick={() => void selectProfile(item.id)}>设为当前</button>
                      <button
                        disabled={testingId === item.id}
                        onClick={() => void testDelay(item.id)}
                      >
                        {testingId === item.id ? '测速中...' : '测速'}
                      </button>
                      <button onClick={() => void openEdit(item.id)}>编辑</button>
                      <button onClick={() => void deleteProfiles([item.id])}>删除</button>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
        {filtered.length === 0 && <p className="muted" style={{ padding: 16 }}>暂无节点，请新增或更新订阅</p>}
      </div>

      {/* Profile edit/create form */}
      {editTarget !== null && (
        <ProfileForm
          initial={editTarget}
          onSave={saveProfile}
          onCancel={() => setEditTarget(null)}
          isSaving={isSaving}
        />
      )}

      <ImportUriDialog
        open={showImport}
        value={importURI}
        isImporting={isImporting}
        onChange={setImportURI}
        onClose={() => { setShowImport(false); setImportURI(''); }}
        onImport={() => void doImport()}
      />
    </section>
  );
}
