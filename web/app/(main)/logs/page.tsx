'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import type { LogLine } from '@/lib/types';

const MAX_LINES = 2000;

const LEVEL_COLORS: Record<string, string> = {
  error: 'var(--red, #f87171)',
  warning: '#fbbf24',
  warn: '#fbbf24',
  info: 'var(--green, #4ade80)',
  debug: 'var(--blue, #60a5fa)',
};

function formatTimestamp(ts: string): string {
  // Normalize both ISO (2006-01-02T15:04:05Z) and xray (2006/01/02 15:04:05) formats
  if (ts.length >= 19) {
    return ts.slice(0, 19).replace('T', ' ');
  }
  return ts;
}

function downloadLogs(lines: LogLine[]) {
  const text = lines
    .map((l) => `${l.timestamp} [${l.level}] ${l.message}`)
    .join('\n');
  const blob = new Blob([text], { type: 'text/plain' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `v2raye-logs-${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.txt`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

export default function LogsPage() {
  const [lines, setLines] = useState<LogLine[]>([]);
  const [filter, setFilter] = useState('');
  const [levelFilter, setLevelFilter] = useState('all');
  const [sourceFilter, setSourceFilter] = useState('all'); // all | app | core
  const [autoScroll, setAutoScroll] = useState(true);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);
  const esRef = useRef<EventSource | null>(null);

  const connect = useCallback(() => {
    if (esRef.current) {
      esRef.current.close();
    }
    const endpoint = `${process.env.NEXT_PUBLIC_API_BASE ?? '/api'}/logs/stream`;
    const es = new EventSource(endpoint, { withCredentials: true });
    esRef.current = es;

    es.addEventListener('ready', () => {
      setConnected(true);
      setError('');
    });

    es.addEventListener('log', (e: Event) => {
      const msg = e as MessageEvent;
      try {
        const line = JSON.parse(msg.data) as LogLine;
        setLines((prev) => {
          const next = [...prev, line];
          return next.length > MAX_LINES ? next.slice(next.length - MAX_LINES) : next;
        });
      } catch {
        // ignore parse error
      }
    });

    es.onerror = () => {
      setConnected(false);
      setError('日志流连接断开，正在重连...');
      es.close();
      esRef.current = null;
      setTimeout(connect, 3000);
    };
  }, []);

  useEffect(() => {
    connect();
    return () => {
      esRef.current?.close();
    };
  }, [connect]);

  // Auto-scroll to bottom
  useEffect(() => {
    if (autoScroll && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [lines, autoScroll]);

  const filtered = lines.filter((l) => {
    const matchLevel = levelFilter === 'all' || l.level === levelFilter || (levelFilter === 'warning' && l.level === 'warn');
    const isAppLog = l.message.startsWith('[app]');
    const matchSource =
      sourceFilter === 'all' ||
      (sourceFilter === 'app' && isAppLog) ||
      (sourceFilter === 'core' && !isAppLog);
    const matchText = !filter.trim() || l.message.toLowerCase().includes(filter.trim().toLowerCase());
    return matchLevel && matchSource && matchText;
  });

  const errorCount = lines.filter((l) => l.level === 'error').length;
  const warnCount = lines.filter((l) => l.level === 'warning' || l.level === 'warn').length;

  const clearLines = () => setLines([]);

  return (
    <section className="page">
      <div className="page-header">
        <div>
          <h2>日志</h2>
          <p className="muted">
            实时日志流（嵌入式 xray-core + 应用事件），最多保留 {MAX_LINES} 条。
            {errorCount > 0 ? <span style={{ color: '#f87171', marginLeft: 8 }}>⚠ {errorCount} 个错误</span> : null}
            {warnCount > 0 ? <span style={{ color: '#fbbf24', marginLeft: 8 }}>{warnCount} 个警告</span> : null}
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span className={connected ? 'status-pill ok' : 'status-pill warn'}>
            {connected ? '实时' : '断开'}
          </span>
          <span className="muted" style={{ fontSize: 13 }}>{filtered.length} / {lines.length} 条</span>
        </div>
      </div>

      <div className="toolbar">
        <input
          placeholder="搜索日志内容..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          style={{ minWidth: 180 }}
        />
        <select value={levelFilter} onChange={(e) => setLevelFilter(e.target.value)}>
          {[
            { value: 'all', label: '所有级别' },
            { value: 'error', label: '错误' },
            { value: 'warning', label: '警告' },
            { value: 'info', label: '信息' },
            { value: 'debug', label: '调试' },
          ].map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>
        <select value={sourceFilter} onChange={(e) => setSourceFilter(e.target.value)}>
          <option value="all">全部来源</option>
          <option value="core">xray-core</option>
          <option value="app">应用事件</option>
        </select>
        <label style={{ display: 'flex', alignItems: 'center', gap: 4, cursor: 'pointer' }}>
          <input type="checkbox" checked={autoScroll} onChange={(e) => setAutoScroll(e.target.checked)} />
          自动滚动
        </label>
        <button onClick={clearLines}>清空</button>
        <button onClick={() => { clearLines(); connect(); }}>重连</button>
        <button
          disabled={lines.length === 0}
          onClick={() => downloadLogs(lines)}
        >
          下载日志
        </button>
      </div>

      {error ? <p className="status-error">{error}</p> : null}

      <div
        style={{
          fontFamily: 'monospace',
          fontSize: 13,
          lineHeight: 1.6,
          padding: 12,
          background: 'color-mix(in srgb, var(--panel) 60%, black)',
          borderRadius: 8,
          overflowY: 'auto',
          maxHeight: 'calc(100vh - 300px)',
          minHeight: 200,
        }}
      >
        {filtered.length === 0 && (
          <span style={{ color: 'var(--muted, #888)' }}>
            {connected ? (lines.length > 0 ? '无符合条件的日志' : '等待日志输出...') : '连接中...'}
          </span>
        )}
        {filtered.map((line, idx) => {
          const isApp = line.message.startsWith('[app]');
          return (
            <div key={idx} style={{ display: 'flex', gap: 12, marginBottom: 2 }}>
              <span style={{ color: 'var(--muted, #888)', whiteSpace: 'nowrap', flexShrink: 0, fontSize: 12 }}>
                {formatTimestamp(line.timestamp)}
              </span>
              <span
                style={{
                  color: LEVEL_COLORS[line.level] ?? 'inherit',
                  whiteSpace: 'nowrap',
                  flexShrink: 0,
                  minWidth: 60,
                  fontWeight: 600,
                  fontSize: 12,
                }}
              >
                [{line.level}]
              </span>
              {isApp ? (
                <span style={{ color: '#c084fc', flexShrink: 0, fontSize: 11, alignSelf: 'center' }}>app</span>
              ) : null}
              <span style={{ wordBreak: 'break-all' }}>{isApp ? line.message.slice(6) : line.message}</span>
            </div>
          );
        })}
        <div ref={bottomRef} />
      </div>
    </section>
  );
}
