'use client';

import type { TrojanConfig } from '@/lib/types';

type TrojanFormProps = {
    value: TrojanConfig;
    onChange: (patch: Partial<TrojanConfig>) => void;
};

export function TrojanForm({ value, onChange }: TrojanFormProps) {
    return (
        <>
            <label>密码</label>
            <input value={value.password} onChange={(e) => onChange({ password: e.target.value })} type="password" placeholder="密码" />
        </>
    );
}