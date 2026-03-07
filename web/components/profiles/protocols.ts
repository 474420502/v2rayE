import type { ProfileItem } from '@/lib/types';

export const PROTOCOLS = ['vmess', 'vless', 'shadowsocks', 'trojan', 'hysteria2', 'tuic'] as const;

export type Protocol = (typeof PROTOCOLS)[number];

export const PROTOCOL_LABELS: Record<Protocol, string> = {
  vmess: 'VMess',
  vless: 'VLESS',
  shadowsocks: 'Shadowsocks',
  trojan: 'Trojan',
  hysteria2: 'Hysteria2',
  tuic: 'TUIC',
};

export const PROTOCOL_COLORS: Record<Protocol, string> = {
  vmess: 'var(--blue)',
  vless: 'var(--green)',
  shadowsocks: 'var(--amber)',
  trojan: 'var(--blue)',
  hysteria2: '#a78bfa',
  tuic: '#34d399',
};

export function blankProfile(protocol: Protocol): Omit<ProfileItem, 'id'> {
  const base = { name: '', protocol, address: '', port: 443 } as Omit<ProfileItem, 'id'>;
  switch (protocol) {
    case 'vmess':
      return { ...base, vmess: { uuid: '', security: 'auto' } };
    case 'vless':
      return { ...base, vless: { uuid: '', flow: '', encryption: 'none' } };
    case 'shadowsocks':
      return { ...base, port: 8388, shadowsocks: { method: 'aes-128-gcm', password: '' } };
    case 'trojan':
      return { ...base, trojan: { password: '' } };
    case 'hysteria2':
      return { ...base, hysteria2: { password: '' } };
    case 'tuic':
      return { ...base, tuic: { uuid: '', password: '' } };
  }
}