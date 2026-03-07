'use client';

import { PROTOCOLS, PROTOCOL_LABELS, type Protocol } from '@/components/profiles/protocols';

type ProtocolSelectorProps = {
    value: Protocol;
    onChange: (protocol: Protocol) => void;
};

export function ProtocolSelector({ value, onChange }: ProtocolSelectorProps) {
    return (
        <select value={value} onChange={(e) => onChange(e.target.value as Protocol)}>
            {PROTOCOLS.map((protocol) => (
                <option key={protocol} value={protocol}>
                    {PROTOCOL_LABELS[protocol]}
                </option>
            ))}
        </select>
    );
}