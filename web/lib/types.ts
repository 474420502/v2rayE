export type ApiEnvelope<T> = {
  code: number;
  message: string;
  data: T;
  details?: unknown;
};

export type CoreStatus = {
  running: boolean;
  coreType?: string;
  engineMode?: string;
  engineResolved?: string;
  currentProfileId?: string;
  state?: 'stopped' | 'starting' | 'running' | 'stopping';
  error?: string;
  errorAt?: string;
};

export type DelayTestResult = {
  available: boolean;
  delayMs?: number;
  message?: string;
};

// ─── Protocol sub-configs ────────────────────────────────────────────────────

export type VMessConfig = {
  uuid: string;
  alterId?: number;
  security?: string;
};

export type VLESSConfig = {
  uuid: string;
  flow?: string;
  encryption?: string;
};

export type ShadowsocksConfig = {
  method: string;
  password: string;
  plugin?: string;
  pluginOpts?: string;
};

export type TrojanConfig = {
  password: string;
};

export type Hysteria2Config = {
  password: string;
  sni?: string;
  insecure?: boolean;
  upMbps?: number;
  downMbps?: number;
  obfs?: string;
  obfsPassword?: string;
};

export type TUICConfig = {
  uuid: string;
  password: string;
  congestionControl?: string;
  sni?: string;
  insecure?: boolean;
  alpn?: string[];
};

export type TransportConfig = {
  network: string;
  wsPath?: string;
  wsHeaders?: Record<string, string>;
  grpcServiceName?: string;
  grpcMode?: string;
  h2Path?: string[];
  h2Host?: string[];
  tls?: boolean;
  sni?: string;
  fingerprint?: string;
  alpn?: string[];
  skipCertVerify?: boolean;
  realityPublicKey?: string;
  realityShortId?: string;
};

export type ProfileItem = {
  id: string;
  name: string;
  protocol: string;
  address: string;
  port: number;
  delayMs?: number;
  subId?: string;
  subName?: string;
  sortOrder?: number;
  vmess?: VMessConfig;
  vless?: VLESSConfig;
  shadowsocks?: ShadowsocksConfig;
  trojan?: TrojanConfig;
  hysteria2?: Hysteria2Config;
  tuic?: TUICConfig;
  transport?: TransportConfig;
};

// ─── Subscriptions ──────────────────────────────────────────────────────────

export type SubscriptionItem = {
  id: string;
  remarks: string;
  url: string;
  enabled?: boolean;
  userAgent?: string;
  filter?: string;
  convertTarget?: string;
  autoUpdateMinutes?: number;
  updatedAt?: string;
  profileCount?: number;
};

export type SubscriptionUpsertInput = {
  remarks: string;
  url: string;
  enabled: boolean;
  userAgent?: string;
  filter?: string;
  convertTarget?: string;
  autoUpdateMinutes?: number;
};

// ─── Routing ────────────────────────────────────────────────────────────────

export type RoutingRule = {
  id: string;
  type: string;
  values: string[];
  outbound: string;
};

export type RoutingConfig = {
  mode: string;
  domainStrategy: string;
  rules?: RoutingRule[];
};

// ─── Stats & logs ───────────────────────────────────────────────────────────

export type StatsResult = {
  upBytes: number;
  downBytes: number;
  upSpeed: number;
  downSpeed: number;
};

export type LogLine = {
  timestamp: string;
  level: string;
  message: string;
};

// ─── Network & config ───────────────────────────────────────────────────────

export type AvailabilityResult = {
  available: boolean;
  elapsedMs?: number;
  message?: string;
};

export type ConfigDto = {
  coreEngine?: 'xray-core' | string;
  listenAddr?: string;
  socksPort?: number;
  httpPort?: number;
  statsPort?: number;
  xrayCmd?: string;
  logLevel?: string;
  allowLan?: boolean;
  enableTun?: boolean;
  tunMode?: 'off' | 'system' | 'mixed' | 'gvisor' | string;
  tunName?: string;
  tunStack?: string;
  tunMtu?: number;
  tunAutoRoute?: boolean;
  tunStrictRoute?: boolean;
  systemProxyMode?: string;
  systemProxyExceptions?: string;
  systemProxyBackend?: string;
  dnsList?: string[];
  [key: string]: unknown;
};
