'use client';

import { useCallback, useEffect, useState } from 'react';
import { api } from '@/lib/api/client';
import type { RoutingConfig, RoutingDiagnostics, RoutingRule } from '@/lib/types';

const MODES = [
  { value: 'global', label: '全局', desc: '所有流量走代理' },
  { value: 'bypass_cn', label: '绕过大陆', desc: '大陆域名/IP 直连，其余走代理' },
  { value: 'direct', label: '直连', desc: '所有流量直连，不走代理' },
  { value: 'custom', label: '自定义', desc: '仅应用下方自定义规则' },
] as const;

const PRESET_TEMPLATES = [
  {
    key: 'global',
    label: '应用全局模板',
    description: '切换到全局代理，并清空自定义规则。',
    config: {
      mode: 'global',
      domainStrategy: 'AsIs',
      rules: [],
    },
  },
  {
    key: 'bypass_cn',
    label: '应用绕过大陆模板',
    description: '切换到大陆直连模板，并清空自定义规则。',
    config: {
      mode: 'bypass_cn',
      domainStrategy: 'IPIfNonMatch',
      rules: [],
    },
  },
  {
    key: 'direct',
    label: '应用直连模板',
    description: '切换到全直连模板，并清空自定义规则。',
    config: {
      mode: 'direct',
      domainStrategy: 'AsIs',
      rules: [],
    },
  },
] as const;

const OUTBOUND_LABELS: Record<string, string> = {
  proxy: '代理',
  direct: '直连',
  block: '屏蔽',
};

const RULE_TYPES = ['domain', 'ip', 'geoip', 'geosite', 'port', 'protocol'] as const;

function blankRule(): Omit<RoutingRule, 'id'> {
  return { type: 'domain', values: [], outbound: 'direct' };
}

export default function RoutingPage() {
  const [config, setConfig] = useState<RoutingConfig | null>(null);
  const [diagnostics, setDiagnostics] = useState<RoutingDiagnostics | null>(null);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [geoUpdating, setGeoUpdating] = useState(false);
  const [geoMessage, setGeoMessage] = useState('');

  // Rule editing state
  const [editIdx, setEditIdx] = useState<number | null>(null);
  const [editRule, setEditRule] = useState<Omit<RoutingRule, 'id'> | null>(null);
  const [valuesText, setValuesText] = useState('');

  const load = useCallback(async () => {
    try {
      const [rc, diag] = await Promise.all([
        api.getRouting(),
        api.getRoutingDiagnostics(),
      ]);
      setConfig(rc);
      setDiagnostics(diag);
      setError('');
    } catch (e) {
      setError(e instanceof Error ? e.message : '加载路由配置失败');
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  const save = async () => {
    if (!config) return;
    setSaving(true);
    try {
      const updated = await api.updateRouting(config);
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

  const setMode = (mode: string) => {
    if (!config) return;
    setConfig({ ...config, mode });
  };

  const setDomainStrategy = (domainStrategy: string) => {
    if (!config) return;
    setConfig({ ...config, domainStrategy });
  };

  const applyPresetTemplate = (templateKey: (typeof PRESET_TEMPLATES)[number]['key']) => {
    if (!config) return;
    const template = PRESET_TEMPLATES.find((item) => item.key === templateKey);
    if (!template) return;
    const hasExistingRules = (config.rules?.length ?? 0) > 0;
    if (hasExistingRules && !confirm(`应用“${template.label.replace('应用', '').replace('模板', '')}”会清空当前自定义规则，是否继续？`)) {
      return;
    }
    setConfig({
      ...config,
      mode: template.config.mode,
      domainStrategy: template.config.domainStrategy,
      rules: [...template.config.rules],
    });
    setSaved(false);
  };

  const updateGeoData = async () => {
    setGeoUpdating(true);
    try {
      const result = await api.updateRoutingGeoData();
      const parts = [
        `时间 ${result.updatedAt}`,
        `geosite ${result.geositeUpdated ? 'updated' : 'kept'} (${result.geositeBytes} bytes)`,
        `geoip ${result.geoipUpdated ? 'updated' : 'kept'} (${result.geoipBytes} bytes)`,
        `状态 geosite=${result.hasGeoSite ? 'ok' : 'missing'}, geoip=${result.hasGeoIP ? 'ok' : 'missing'}`,
      ];
      setGeoMessage(`数据处理完成: ${parts.join(' | ')}`);
      setError('');
    } catch (e) {
      setError(e instanceof Error ? e.message : '更新 geodata 失败');
    } finally {
      setGeoUpdating(false);
    }
  };

  // Rule CRUD
  const openNewRule = () => {
    setEditIdx(-1);
    setEditRule(blankRule());
    setValuesText('');
  };

  const openEditRule = (idx: number) => {
    if (!config) return;
    const rule = config.rules?.[idx];
    if (!rule) return;
    setEditIdx(idx);
    setEditRule({ type: rule.type, values: rule.values, outbound: rule.outbound });
    setValuesText(rule.values.join('\n'));
  };

  const saveRule = () => {
    if (!config || !editRule) return;
    const values = valuesText.split('\n').map((s) => s.trim()).filter(Boolean);
    const newRule: RoutingRule = {
      id: `rule-${Date.now()}`,
      ...editRule,
      values,
    };
    const rules = [...(config.rules ?? [])];
    if (editIdx === -1) {
      rules.push(newRule);
    } else if (editIdx !== null) {
      rules[editIdx] = { ...rules[editIdx], ...newRule };
    }
    setConfig({ ...config, rules });
    setEditIdx(null);
    setEditRule(null);
  };

  const deleteRule = (idx: number) => {
    if (!config) return;
    const rules = (config.rules ?? []).filter((_, i) => i !== idx);
    setConfig({ ...config, rules });
  };

  const moveRule = (idx: number, dir: -1 | 1) => {
    if (!config) return;
    const rules = [...(config.rules ?? [])];
    const next = idx + dir;
    if (next < 0 || next >= rules.length) return;
    [rules[idx], rules[next]] = [rules[next], rules[idx]];
    setConfig({ ...config, rules });
  };

  if (!config) {
    return <section className="page"><p className="muted">{error || '加载中...'}</p></section>;
  }

  return (
    <section className="page">
      <div className="page-header">
        <div>
          <h2>路由规则</h2>
          <p className="muted">对齐 v2rayN 的路由模式选择 + 自定义规则配置。</p>
        </div>
        <div className="toolbar compact">
          <button onClick={() => void load()}>重置</button>
          <button
            className="primary"
            onClick={() => void save()}
            disabled={saving}
          >
            {saved ? '已保存 ✓' : saving ? '保存中...' : '保存并重启'}
          </button>
        </div>
      </div>

      {error ? <p className="status-error">{error}</p> : null}

      <section className="panel" style={{ marginBottom: 16 }}>
        <h3>预设模板</h3>
        <p className="muted" style={{ marginBottom: 12 }}>
          快速套用常用路由模板。模板会覆盖当前模式、域名策略和自定义规则，点击保存后才会真正生效。
        </p>
        <div style={{ marginBottom: 12, display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
          <button onClick={() => void updateGeoData()} disabled={geoUpdating}>
            {geoUpdating ? '更新数据中...' : '更新绕过大陆数据'}
          </button>
          <span className="muted">下载并校验最新 `dlc.dat`，写入后端 geosite 数据文件。</span>
        </div>
        {geoMessage ? <p className="muted">{geoMessage}</p> : null}
        <div style={{ display: 'grid', gap: 12, gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))' }}>
          {PRESET_TEMPLATES.map((template) => {
            const active = config.mode === template.config.mode && (config.rules?.length ?? 0) === 0;
            return (
              <button
                key={template.key}
                type="button"
                onClick={() => applyPresetTemplate(template.key)}
                style={{
                  textAlign: 'left',
                  padding: '14px 16px',
                  borderRadius: 10,
                  border: '1.5px solid',
                  borderColor: active ? 'var(--blue)' : 'color-mix(in srgb, var(--text) 15%, transparent)',
                  background: active ? 'color-mix(in srgb, var(--blue) 10%, transparent)' : 'transparent',
                }}
              >
                <div style={{ fontWeight: 600, marginBottom: 4 }}>{template.label}</div>
                <div className="muted" style={{ fontSize: 12 }}>{template.description}</div>
              </button>
            );
          })}
        </div>
      </section>

      <section className="panel" style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
          <h3 style={{ margin: 0 }}>运行时路由诊断</h3>
          <button onClick={() => void load()}>刷新诊断</button>
        </div>
        {!diagnostics ? (
          <p className="muted" style={{ marginTop: 12 }}>暂无诊断数据</p>
        ) : (
          <>
            <div className="stats-grid" style={{ marginTop: 12 }}>
              <article className="stat-card accent-blue">
                <span className="stat-label">模式</span>
                <strong>{diagnostics.mode}</strong>
                <span>{diagnostics.domainStrategy}</span>
              </article>
              <article className="stat-card accent-green">
                <span className="stat-label">TUN</span>
                <strong>{diagnostics.tunEnabled ? 'on' : 'off'}</strong>
                <span>mode={diagnostics.tunMode}</span>
              </article>
              <article className="stat-card accent-amber">
                <span className="stat-label">GeoData</span>
                <strong>{diagnostics.geoDataAvailable ? 'ready' : 'partial'}</strong>
                <span>geosite={diagnostics.hasGeoSite ? 'ok' : 'missing'}, geoip={diagnostics.hasGeoIP ? 'ok' : 'missing'}</span>
              </article>
              <article className="stat-card accent-slate">
                <span className="stat-label">规则数</span>
                <strong>{diagnostics.ruleCount}</strong>
                <span>{diagnostics.currentProfileName || diagnostics.currentProfileId || '无当前节点'}</span>
              </article>
            </div>
            {diagnostics.warning ? (
              <p className="status-error" style={{ marginTop: 10 }}>{diagnostics.warning}</p>
            ) : null}
            <p className="muted" style={{ marginTop: 8 }}>
              生成时间: {diagnostics.generatedAt}
            </p>
            <div className="table-wrap" style={{ marginTop: 10 }}>
              <table>
                <thead>
                  <tr>
                    <th>#</th>
                    <th>条件</th>
                    <th>出站</th>
                  </tr>
                </thead>
                <tbody>
                  {diagnostics.rules.map((rule, idx) => {
                    const inbound = Array.isArray(rule.inboundTag) ? rule.inboundTag.join(', ') : '';
                    const domain = Array.isArray(rule.domain) ? rule.domain.join(', ') : '';
                    const ip = Array.isArray(rule.ip) ? rule.ip.join(', ') : '';
                    const protocol = Array.isArray(rule.protocol) ? rule.protocol.join(', ') : '';
                    const port = typeof rule.port === 'string' ? rule.port : '';
                    const cond = [
                      inbound ? `inbound=${inbound}` : '',
                      domain ? `domain=${domain}` : '',
                      ip ? `ip=${ip}` : '',
                      protocol ? `protocol=${protocol}` : '',
                      port ? `port=${port}` : '',
                    ].filter(Boolean).join(' | ');

                    return (
                      <tr key={idx}>
                        <td>{idx + 1}</td>
                        <td style={{ fontFamily: 'monospace', fontSize: 12 }}>{cond || '-'}</td>
                        <td>{String(rule.outboundTag ?? '-')}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </>
        )}
      </section>

      {/* Mode selector */}
      <section className="panel" style={{ marginBottom: 16 }}>
        <h3>路由模式</h3>
        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', marginTop: 12 }}>
          {MODES.map((m) => (
            <label
              key={m.value}
              style={{
                display: 'flex', alignItems: 'flex-start', gap: 8, cursor: 'pointer',
                padding: '10px 14px', borderRadius: 8, border: '1.5px solid',
                borderColor: config.mode === m.value ? 'var(--blue)' : 'color-mix(in srgb, var(--text) 15%, transparent)',
                background: config.mode === m.value ? 'color-mix(in srgb, var(--blue) 10%, transparent)' : 'transparent',
              }}
            >
              <input
                type="radio"
                name="mode"
                value={m.value}
                checked={config.mode === m.value}
                onChange={() => setMode(m.value)}
                style={{ marginTop: 2 }}
              />
              <div>
                <div style={{ fontWeight: 600 }}>{m.label}</div>
                <div className="muted" style={{ fontSize: 12 }}>{m.desc}</div>
              </div>
            </label>
          ))}
        </div>
      </section>

      {/* Domain strategy */}
      <section className="panel" style={{ marginBottom: 16 }}>
        <h3>域名解析策略</h3>
        <select
          value={config.domainStrategy}
          onChange={(e) => setDomainStrategy(e.target.value)}
          style={{ marginTop: 8, width: 240 }}
        >
          {['IPIfNonMatch', 'IPOnDemand', 'AsIs'].map((v) => (
            <option key={v} value={v}>{v}</option>
          ))}
        </select>
      </section>

      {/* Custom rules */}
      <section className="panel">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
          <h3 style={{ margin: 0 }}>自定义规则</h3>
          <button onClick={openNewRule}>+ 添加规则</button>
        </div>
        {(!config.rules || config.rules.length === 0) ? (
          <p className="muted">暂无自定义规则</p>
        ) : (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>类型</th>
                  <th>值（数量）</th>
                  <th>出站</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {config.rules.map((rule, idx) => (
                  <tr key={rule.id}>
                    <td><span style={{ fontFamily: 'monospace' }}>{rule.type}</span></td>
                    <td>
                      <span title={rule.values.join(', ')}>
                        {rule.values.slice(0, 2).join(', ')}
                        {rule.values.length > 2 ? ` (+${rule.values.length - 2})` : ''}
                      </span>
                    </td>
                    <td>
                      <span className={
                        rule.outbound === 'proxy' ? 'status-pill ok' :
                          rule.outbound === 'block' ? 'status-pill warn' : 'status-pill muted'
                      }>
                        {OUTBOUND_LABELS[rule.outbound] ?? rule.outbound}
                      </span>
                    </td>
                    <td>
                      <div className="actions-inline">
                        <button onClick={() => moveRule(idx, -1)} disabled={idx === 0}>↑</button>
                        <button onClick={() => moveRule(idx, 1)} disabled={idx === (config.rules?.length ?? 0) - 1}>↓</button>
                        <button onClick={() => openEditRule(idx)}>编辑</button>
                        <button onClick={() => deleteRule(idx)}>删除</button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {/* Rule editor modal */}
      {editRule !== null && (
        <div className="modal-overlay">
          <div className="modal-box" style={{ maxWidth: 480, width: '95%' }}>
            <h3 style={{ marginBottom: 16 }}>{editIdx === -1 ? '添加规则' : '编辑规则'}</h3>
            <div className="form-grid">
              <label>规则类型</label>
              <select
                value={editRule.type}
                onChange={(e) => setEditRule({ ...editRule, type: e.target.value })}
              >
                {RULE_TYPES.map((t) => <option key={t}>{t}</option>)}
              </select>

              <label>目标出站</label>
              <select
                value={editRule.outbound}
                onChange={(e) => setEditRule({ ...editRule, outbound: e.target.value })}
              >
                {['proxy', 'direct', 'block'].map((v) => (
                  <option key={v} value={v}>{OUTBOUND_LABELS[v]}</option>
                ))}
              </select>

              <label>值（每行一条）</label>
              <textarea
                value={valuesText}
                onChange={(e) => setValuesText(e.target.value)}
                rows={6}
                placeholder={
                  editRule.type === 'geoip' ? 'CN\nPrivate' :
                    editRule.type === 'geosite' ? 'category-ads-all\ncn' :
                      editRule.type === 'domain' ? 'example.com\ndomain:google.com' :
                        'one value per line'
                }
              />
            </div>
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 16 }}>
              <button onClick={() => { setEditIdx(null); setEditRule(null); }}>取消</button>
              <button className="primary" onClick={saveRule}>确定</button>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}
