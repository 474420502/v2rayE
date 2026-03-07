'use client';

import type { ShadowsocksConfig } from '@/lib/types';

type ShadowsocksFormProps = {
    value: ShadowsocksConfig;
    onChange: (patch: Partial<ShadowsocksConfig>) => void;
};

export function ShadowsocksForm({ value, onChange }: ShadowsocksFormProps) {
    return (
        <>
            <label>加密方式</label>
            <select value={value.method} onChange={(e) => onChange({ method: e.target.value })}>
                {[
                    'aes-128-gcm',
                    'aes-256-gcm',
                    'chacha20-ietf-poly1305',
                    '2022-blake3-aes-128-gcm',
                    '2022-blake3-aes-256-gcm',
                    '2022-blake3-chacha20-poly1305',
                ].map((item) => (
                    <option key={item}>{item}</option>
                ))}
            </select>

            <label>密码</label>
            <input value={value.password} onChange={(e) => onChange({ password: e.target.value })} type="password" placeholder="密码" />
        </>
    );
}