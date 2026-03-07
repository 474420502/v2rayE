'use client';

import type { TUICConfig } from '@/lib/types';

type TUICFormProps = {
    value: TUICConfig;
    onChange: (patch: Partial<TUICConfig>) => void;
};

export function TUICForm({ value, onChange }: TUICFormProps) {
    return (
        <>
            <label>UUID</label>
            <input value={value.uuid} onChange={(e) => onChange({ uuid: e.target.value })} placeholder="UUID" />

            <label>密码</label>
            <input value={value.password} onChange={(e) => onChange({ password: e.target.value })} type="password" placeholder="密码" />

            <label>拥塞控制</label>
            <select value={value.congestionControl ?? 'bbr'} onChange={(e) => onChange({ congestionControl: e.target.value })}>
                {['bbr', 'cubic', 'new_reno'].map((item) => (
                    <option key={item}>{item}</option>
                ))}
            </select>
        </>
    );
}