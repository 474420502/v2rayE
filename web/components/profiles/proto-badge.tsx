'use client';

import { PROTOCOL_COLORS, PROTOCOL_LABELS, type Protocol } from '@/components/profiles/protocols';

export function ProtoBadge({ protocol }: { protocol: string }) {
    const color = PROTOCOL_COLORS[protocol as Protocol] ?? 'var(--slate)';

    return (
        <span
            style={{
                fontSize: 11,
                padding: '2px 6px',
                borderRadius: 4,
                background: color + '22',
                color,
                fontWeight: 600,
                whiteSpace: 'nowrap',
            }}
        >
            {PROTOCOL_LABELS[protocol as Protocol] ?? protocol}
        </span>
    );
}