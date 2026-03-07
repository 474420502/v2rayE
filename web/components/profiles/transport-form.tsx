'use client';

import type { TransportConfig } from '@/lib/types';

type TransportFormProps = {
    transport?: TransportConfig;
    onChange: (patch: Partial<TransportConfig>) => void;
};

export function TransportForm({ transport, onChange }: TransportFormProps) {
    return (
        <>
            <label>传输协议</label>
            <select value={transport?.network ?? 'tcp'} onChange={(e) => onChange({ network: e.target.value })}>
                {['tcp', 'ws', 'grpc', 'h2', 'xhttp'].map((value) => (
                    <option key={value}>{value}</option>
                ))}
            </select>

            {(transport?.network === 'ws' || transport?.network === 'xhttp') && (
                <>
                    <label>{transport?.network === 'xhttp' ? 'Path' : 'WS Path'}</label>
                    <input value={transport?.wsPath ?? ''} onChange={(e) => onChange({ wsPath: e.target.value })} placeholder="/" />
                    {transport?.network === 'xhttp' && (
                        <>
                            <label>Host</label>
                            <input
                                value={transport?.wsHeaders?.['Host'] ?? ''}
                                onChange={(e) => onChange({ wsHeaders: { ...(transport?.wsHeaders ?? {}), Host: e.target.value } })}
                                placeholder="example.com"
                            />
                        </>
                    )}
                </>
            )}

            {transport?.network === 'grpc' && (
                <>
                    <label>gRPC ServiceName</label>
                    <input value={transport?.grpcServiceName ?? ''} onChange={(e) => onChange({ grpcServiceName: e.target.value })} placeholder="" />
                </>
            )}

            <label>TLS</label>
            <input type="checkbox" checked={transport?.tls ?? false} onChange={(e) => onChange({ tls: e.target.checked })} />

            {transport?.tls && (
                <>
                    <label>SNI</label>
                    <input value={transport?.sni ?? ''} onChange={(e) => onChange({ sni: e.target.value })} placeholder="example.com" />

                    <label>跳过证书验证</label>
                    <input
                        type="checkbox"
                        checked={transport?.skipCertVerify ?? false}
                        onChange={(e) => onChange({ skipCertVerify: e.target.checked })}
                    />

                    <label>Reality 公钥</label>
                    <input
                        value={transport?.realityPublicKey ?? ''}
                        onChange={(e) => onChange({ realityPublicKey: e.target.value })}
                        placeholder="(可选)"
                    />

                    <label>Reality ShortID</label>
                    <input
                        value={transport?.realityShortId ?? ''}
                        onChange={(e) => onChange({ realityShortId: e.target.value })}
                        placeholder="(可选)"
                    />
                </>
            )}
        </>
    );
}