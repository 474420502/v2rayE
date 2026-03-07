'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import type { LogLine } from '@/lib/types';

const MAX_LINES = 500;

const LEVEL_COLORS: Record<string, string> = {
  error: 'var(--red, #f87171)',
  warning: 'var(--amber)',
  warn: 'var(--amber)',
  info: 'var(--green)',
  debug: 'var(--blue)',
};

export default function LogsPage() {
  const [lines, setLines] = useState<LogLine[]>([]);
  const [filter, setFilter] = useState('');
  const [levelFilter, setLevelFilter] = useState('all');
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
    const matchLevel = levelFilter === 'all' || l.level === levelFilter;
    const matchText = !filter.trim() || l.message.toLowerCase().includes(filter.trim().toLowerCase());
    return matchLevel && matchText;
  });

  const clearLines = () => setLines([]);

  return (
    <section className="page">
      <div className="page-header">
        <div>
          <h2>日志</h2>
          <p className="muted">实时显示 Xray 核心进程输出，最多保留 {MAX_LINES} 条记录。</p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span className={connected ? 'status-pill ok' : 'status-pill warn'}>
            {connected ? '已连接' : '断开'}
          </span>
          <span className="muted" style={{ fontSize: 13 }}>{filtered.length} 条</span>
        </div>
      </div>

      <div className="toolbar">
        <input
          placeholder="搜索日志内容..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />
        <select value={levelFilter} onChange={(e) => setLevelFilter(e.target.value)}>
          {['all', 'debug', 'info', 'warning', 'error'].map((l) => (
            <option key={l} value={l}>{l === 'all' ? '所有级别' : l}</option>
          ))}
        </select>
        <label style={{ display: 'flex', alignItems: 'center', gap: 4, cursor: 'pointer' }}>
          <input type="checkbox" checked={autoScroll} onChange={(e) => setAutoScroll(e.target.checked)} />
          自动滚动
        </label>
        <button onClick={clearLines}>清空</button>
        <button onClick={() => { clearLines(); connect(); }}>重连</button>
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
          maxHeight: 'calc(100vh - 280px)',
          minHeight: 200,
        }}
      >
        {filtered.length === 0 && (
          <span style={{ color: 'var(--muted, #888)' }}>
            {connected ? '等待日志输出...' : '连接中...'}
          </span>
        )}
        {filtered.map((line, idx) => (
          <div key={idx} style={{ display: 'flex', gap: 12, marginBottom: 2 }}>
            <span style={{ color: 'var(--muted, #888)', whiteSpace: 'nowrap', flexShrink: 0 }}>
              {line.timestamp}
            </span>
            <span
              style={{
                color: LEVEL_COLORS[line.level] ?? 'inherit',
                whiteSpace: 'nowrap',
                flexShrink: 0,
                minWidth: 50,
                fontWeight: 600,
              }}
            >
              [{line.level}]
            </span>
            <span style={{ wordBreak: 'break-all' }}>{line.message}</span>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>
    </section>
  );
}
