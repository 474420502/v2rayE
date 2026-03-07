'use client';

import type { Hysteria2Config } from '@/lib/types';

type Hysteria2FormProps = {
    value: Hysteria2Config;
    onChange: (patch: Partial<Hysteria2Config>) => void;
};

export function Hysteria2Form({ value, onChange }: Hysteria2FormProps) {
    return (
        <>
            <label>密码</label>
            <input value={value.password} onChange={(e) => onChange({ password: e.target.value })} type="password" placeholder="密码" />

            <label>SNI</label>
            <input value={value.sni ?? ''} onChange={(e) => onChange({ sni: e.target.value })} placeholder="example.com" />

            <label>跳过证书验证</label>
            <input type="checkbox" checked={value.insecure ?? false} onChange={(e) => onChange({ insecure: e.target.checked })} />
        </>
    );
}