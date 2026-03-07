'use client';

import type { VMessConfig } from '@/lib/types';

type VMessFormProps = {
    value: VMessConfig;
    onChange: (patch: Partial<VMessConfig>) => void;
};

export function VMessForm({ value, onChange }: VMessFormProps) {
    return (
        <>
            <label>UUID</label>
            <input value={value.uuid} onChange={(e) => onChange({ uuid: e.target.value })} placeholder="UUID" />

            <label>加密</label>
            <select value={value.security ?? 'auto'} onChange={(e) => onChange({ security: e.target.value })}>
                {['auto', 'none', 'aes-128-gcm', 'chacha20-poly1305'].map((item) => (
                    <option key={item}>{item}</option>
                ))}
            </select>

            <label>AlterID</label>
            <input type="number" value={value.alterId ?? 0} onChange={(e) => onChange({ alterId: Number(e.target.value) })} />
        </>
    );
}