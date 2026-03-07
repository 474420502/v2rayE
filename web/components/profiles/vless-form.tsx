'use client';

import type { VLESSConfig } from '@/lib/types';

type VLESSFormProps = {
    value: VLESSConfig;
    onChange: (patch: Partial<VLESSConfig>) => void;
};

export function VLESSForm({ value, onChange }: VLESSFormProps) {
    return (
        <>
            <label>UUID</label>
            <input value={value.uuid} onChange={(e) => onChange({ uuid: e.target.value })} placeholder="UUID" />

            <label>Flow</label>
            <select value={value.flow ?? ''} onChange={(e) => onChange({ flow: e.target.value })}>
                {['', 'xtls-rprx-vision', 'xtls-rprx-vision-udp443'].map((item) => (
                    <option key={item} value={item}>
                        {item || '(none)'}
                    </option>
                ))}
            </select>
        </>
    );
}