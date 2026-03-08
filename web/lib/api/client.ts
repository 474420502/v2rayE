import type {
  ApiEnvelope,
  AvailabilityResult,
  ConfigDto,
  CoreStatus,
  DelayTestResult,
  LogLine,
  ProfileItem,
  RoutingConfig,
  RoutingDiagnostics,
  RoutingHitStats,
  RoutingGeoDataUpdateResult,
  StatsResult,
  SubscriptionItem,
  SubscriptionUpsertInput,
  TunRepairResult
} from '@/lib/types';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE ?? '/api';
const REQUEST_TIMEOUT_MS = Number.parseInt(process.env.NEXT_PUBLIC_API_TIMEOUT_MS ?? '12000', 10);

const API_CODE_MESSAGES: Record<number, string> = {
  40101: '未登录或登录已过期，请重新登录',
  40103: '当前令牌无权限执行该操作',
  40301: '访问被拒绝',
  40401: '目标资源不存在',
  40901: '操作冲突，请稍后重试',
  42201: '请求参数不合法',
  50001: '后端服务内部错误'
};

const HTTP_STATUS_MESSAGES: Record<number, string> = {
  400: '请求参数错误',
  401: '未授权，请重新登录',
  403: '无权限访问',
  404: '接口不存在',
  409: '请求冲突，请稍后重试',
  422: '请求参数校验失败',
  429: '请求过于频繁，请稍后再试',
  500: '后端服务异常',
  502: '网关异常，请稍后重试',
  503: '服务暂不可用，请稍后重试',
  504: '后端响应超时'
};

export class ApiClientError extends Error {
  code?: number;
  status?: number;
  details?: unknown;

  constructor(
    message: string,
    options?: { code?: number; status?: number; details?: unknown }
  ) {
    super(message);
    this.name = 'ApiClientError';
    this.code = options?.code;
    this.status = options?.status;
    this.details = options?.details;
  }
}

function getTokenFromCookie(): string | null {
  if (typeof window === 'undefined') return null;
  const pair = document.cookie
    .split(';')
    .map((c) => c.trim())
    .find((c) => c.startsWith('auth_token='));
  if (!pair) return null;
  const [, raw] = pair.split('=');
  return raw ? decodeURIComponent(raw) : null;
}

function safeTimeoutMs(): number {
  if (Number.isFinite(REQUEST_TIMEOUT_MS) && REQUEST_TIMEOUT_MS > 0) {
    return REQUEST_TIMEOUT_MS;
  }
  return 12000;
}

function joinAbortSignals(a: AbortSignal, b?: AbortSignal | null): AbortSignal {
  if (!b) return a;
  if (typeof AbortSignal.any === 'function') {
    return AbortSignal.any([a, b]);
  }

  const relay = new AbortController();
  const abortRelay = () => relay.abort();
  if (a.aborted || b.aborted) {
    relay.abort();
  } else {
    a.addEventListener('abort', abortRelay, { once: true });
    b.addEventListener('abort', abortRelay, { once: true });
  }
  return relay.signal;
}

export function buildEventSourceUrl(path: string): string {
  const normalized = path.startsWith('/') ? path : `/${path}`;
  const baseUrl = `${API_BASE}${normalized}`;
  const token = getTokenFromCookie();
  if (!token) {
    return baseUrl;
  }
  const sep = baseUrl.includes('?') ? '&' : '?';
  return `${baseUrl}${sep}token=${encodeURIComponent(token)}`;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getTokenFromCookie();
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), safeTimeoutMs());

  let response: Response;
  try {
    response = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...(init?.headers ?? {})
      },
      cache: 'no-store',
      signal: joinAbortSignals(controller.signal, init?.signal)
    });
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      throw new ApiClientError(`请求超时（>${safeTimeoutMs()}ms）`, { status: 504 });
    }
    throw error;
  } finally {
    clearTimeout(timeout);
  }

  const contentType = response.headers.get('content-type') ?? '';
  const isJson = contentType.includes('application/json');
  const body = isJson ? ((await response.json()) as ApiEnvelope<T>) : null;

  if (!response.ok) {
    const message =
      (body && body.message) ||
      HTTP_STATUS_MESSAGES[response.status] ||
      `请求失败（HTTP ${response.status}）`;
    throw new ApiClientError(message, { status: response.status, details: body?.details });
  }

  if (!body) {
    throw new ApiClientError('后端返回格式错误（非 JSON）', { status: response.status });
  }

  if (body.code !== 0) {
    const message =
      API_CODE_MESSAGES[body.code] || body.message || `请求失败（错误码 ${body.code}）`;
    throw new ApiClientError(message, {
      code: body.code,
      status: response.status,
      details: body.details
    });
  }

  return body.data;
}

export const api = {
  // ── Core ────────────────────────────────────────────────────────────────────
  getCoreStatus: () => request<CoreStatus>('/core/status'),
  startCore: () => request<CoreStatus>('/core/start', { method: 'POST', body: '{}' }),
  stopCore: () => request<CoreStatus>('/core/stop', { method: 'POST', body: '{}' }),
  restartCore: () => request<CoreStatus>('/core/restart', { method: 'POST', body: '{}' }),
  clearCoreError: () => request<CoreStatus>('/core/error/clear', { method: 'POST', body: '{}' }),

  // ── Profile CRUD + import ───────────────────────────────────────────────────
  getProfiles: () => request<ProfileItem[]>('/profiles'),
  getProfile: (id: string) => request<ProfileItem>(`/profiles/${id}`),
  createProfile: (profile: Omit<ProfileItem, 'id'>) =>
    request<ProfileItem>('/profiles', { method: 'POST', body: JSON.stringify(profile) }),
  updateProfile: (id: string, profile: Partial<ProfileItem>) =>
    request<ProfileItem>(`/profiles/${id}`, { method: 'PUT', body: JSON.stringify(profile) }),
  deleteProfile: (id: string) =>
    request<unknown>(`/profiles/${id}`, { method: 'DELETE' }),
  deleteProfiles: (ids: string[]) =>
    request<{ deleted: number }>('/profiles/delete', {
      method: 'POST',
      body: JSON.stringify({ ids })
    }),
  selectProfile: (id: string) =>
    request<unknown>(`/profiles/${id}/select`, { method: 'POST', body: '{}' }),
  testProfileDelay: (id: string) =>
    request<DelayTestResult>(`/profiles/${id}/delay`),
  importProfileFromURI: (uri: string) =>
    request<ProfileItem>('/profiles/import', { method: 'POST', body: JSON.stringify({ uri }) }),

  // ── Subscriptions ──────────────────────────────────────────────────────────
  getSubscriptions: () => request<SubscriptionItem[]>('/subscriptions'),
  createSubscription: (payload: SubscriptionUpsertInput) =>
    request<SubscriptionItem>('/subscriptions', { method: 'POST', body: JSON.stringify(payload) }),
  updateSubscription: (id: string, payload: SubscriptionUpsertInput) =>
    request<SubscriptionItem>(`/subscriptions/${id}`, { method: 'PUT', body: JSON.stringify(payload) }),
  deleteSubscription: (id: string) =>
    request<unknown>(`/subscriptions/${id}`, { method: 'DELETE' }),
  updateSubscriptions: () =>
    request<{ updated: number }>('/subscriptions/update', { method: 'POST', body: '{}' }),
  updateSubscriptionById: (id: string) =>
    request<{ updated: number }>(`/subscriptions/${id}/update`, { method: 'POST', body: '{}' }),

  // ── Network & proxy ────────────────────────────────────────────────────────
  getAvailability: () => request<AvailabilityResult>('/network/availability'),
  applySystemProxy: (mode: 'forced_change' | 'forced_clear', exceptions = '') =>
    request<unknown>('/system-proxy/apply', {
      method: 'POST',
      body: JSON.stringify({ mode, exceptions })
    }),
  exitCleanup: (shutdownBackend = false) =>
    request<{
      status: CoreStatus;
      proxyCleared: boolean;
      proxyClearError?: string;
      tunCleaned: boolean;
      shutdownBackend: boolean;
    }>('/app/exit-cleanup', {
      method: 'POST',
      body: JSON.stringify({ shutdownBackend })
    }),

  // ── Config ─────────────────────────────────────────────────────────────────
  getConfig: () => request<ConfigDto>('/config'),
  updateConfig: (config: Partial<ConfigDto>) =>
    request<ConfigDto>('/config', { method: 'PUT', body: JSON.stringify(config) }),

  // ── Routing ────────────────────────────────────────────────────────────────
  getRouting: () => request<RoutingConfig>('/routing'),
  getRoutingDiagnostics: () => request<RoutingDiagnostics>('/routing/diagnostics'),
  getRoutingHitStats: () => request<RoutingHitStats>('/routing/hits'),
  repairTunAndRestart: () => request<TunRepairResult>('/routing/tun/repair', { method: 'POST', body: '{}' }),
  updateRouting: (rc: RoutingConfig) =>
    request<RoutingConfig>('/routing', { method: 'PUT', body: JSON.stringify(rc) }),
  updateRoutingGeoData: () =>
    request<RoutingGeoDataUpdateResult>('/routing/geodata/update', { method: 'POST', body: '{}' }),

  // ── Stats ──────────────────────────────────────────────────────────────────
  getStats: () => request<StatsResult>('/stats'),

  // ── Core logs SSE ─────────────────────────────────────────────────────────
  streamCoreLogs: (onLine: (line: LogLine) => void): () => void => {
    const es = new EventSource(buildEventSourceUrl('/logs/stream'));
    es.addEventListener('log', (e: MessageEvent) => {
      try {
        onLine(JSON.parse(e.data) as LogLine);
      } catch {
        // ignore parse errors
      }
    });
    return () => es.close();
  },

  // ── Metadata events SSE ────────────────────────────────────────────────────
  streamEvents: (onEvent: (ev: { event: string; ts: string; data: unknown }) => void): () => void => {
    const es = new EventSource(buildEventSourceUrl('/events/stream'));
    es.addEventListener('message', (e: MessageEvent) => {
      try {
        onEvent(JSON.parse(e.data));
      } catch {
        // ignore
      }
    });
    return () => es.close();
  }
};
