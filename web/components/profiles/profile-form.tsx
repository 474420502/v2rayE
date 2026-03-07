'use client';

import { useState } from 'react';
import { Hysteria2Form } from '@/components/profiles/hysteria2-form';
import { ProtocolSelector } from '@/components/profiles/protocol-selector';
import { ShadowsocksForm } from '@/components/profiles/shadowsocks-form';
import { TrojanForm } from '@/components/profiles/trojan-form';
import { blankProfile, type Protocol } from '@/components/profiles/protocols';
import { TransportForm } from '@/components/profiles/transport-form';
import { TUICForm } from '@/components/profiles/tuic-form';
import { VLESSForm } from '@/components/profiles/vless-form';
import { VMessForm } from '@/components/profiles/vmess-form';
import type { Hysteria2Config, ProfileItem, ShadowsocksConfig, TransportConfig, TrojanConfig, TUICConfig, VLESSConfig, VMessConfig } from '@/lib/types';

export type ProfileFormState = Omit<ProfileItem, 'id'>;

export function ProfileForm({
    initial,
    onSave,
    onCancel,
    isSaving,
}: {
    initial: ProfileFormState;
    onSave: (profile: ProfileFormState) => void;
    onCancel: () => void;
    isSaving: boolean;
}) {
    const [form, setForm] = useState<ProfileFormState>(initial);
    const proto = form.protocol as Protocol;

    const set = (patch: Partial<ProfileFormState>) => setForm((current) => ({ ...current, ...patch }));
    const setVmess = (patch: Partial<VMessConfig>) =>
        setForm((current) => ({ ...current, vmess: { ...current.vmess!, ...patch } }));
    const setVless = (patch: Partial<VLESSConfig>) =>
        setForm((current) => ({ ...current, vless: { ...current.vless!, ...patch } }));
    const setSS = (patch: Partial<ShadowsocksConfig>) =>
        setForm((current) => ({ ...current, shadowsocks: { ...current.shadowsocks!, ...patch } }));
    const setTrojan = (patch: Partial<TrojanConfig>) =>
        setForm((current) => ({ ...current, trojan: { ...current.trojan!, ...patch } }));
    const setHy2 = (patch: Partial<Hysteria2Config>) =>
        setForm((current) => ({ ...current, hysteria2: { ...current.hysteria2!, ...patch } }));
    const setTUIC = (patch: Partial<TUICConfig>) =>
        setForm((current) => ({ ...current, tuic: { ...current.tuic!, ...patch } }));
    const setTransport = (patch: Partial<TransportConfig>) =>
        setForm((current) => ({
            ...current,
            transport: { ...current.transport, network: current.transport?.network ?? 'tcp', ...patch },
        }));

    return (
        <div className="modal-overlay">
            <div className="modal-box" style={{ maxWidth: 600, width: '95%' }}>
                <h3 style={{ marginBottom: 16 }}>{(initial as ProfileItem).id ? '编辑节点' : '新增节点'}</h3>

                <div className="form-grid">
                    <label>别名</label>
                    <input value={form.name} onChange={(e) => set({ name: e.target.value })} placeholder="节点别名" />

                    <label>协议</label>
                    <ProtocolSelector
                        value={form.protocol as Protocol}
                        onChange={(nextProtocol) => {
                            setForm(blankProfile(nextProtocol));
                        }}
                    />

                    <label>地址</label>
                    <input value={form.address} onChange={(e) => set({ address: e.target.value })} placeholder="域名或IP" />

                    <label>端口</label>
                    <input type="number" value={form.port} onChange={(e) => set({ port: Number(e.target.value) })} placeholder="443" />

                    {proto === 'vmess' && form.vmess && <VMessForm value={form.vmess} onChange={setVmess} />}

                    {proto === 'vless' && form.vless && <VLESSForm value={form.vless} onChange={setVless} />}

                    {proto === 'shadowsocks' && form.shadowsocks && <ShadowsocksForm value={form.shadowsocks} onChange={setSS} />}

                    {proto === 'trojan' && form.trojan && <TrojanForm value={form.trojan} onChange={setTrojan} />}

                    {proto === 'hysteria2' && form.hysteria2 && <Hysteria2Form value={form.hysteria2} onChange={setHy2} />}

                    {proto === 'tuic' && form.tuic && <TUICForm value={form.tuic} onChange={setTUIC} />}

                    {['vmess', 'vless', 'trojan'].includes(proto) && (
                        <TransportForm transport={form.transport} onChange={setTransport} />
                    )}
                </div>

                <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 16 }}>
                    <button onClick={onCancel} disabled={isSaving}>
                        取消
                    </button>
                    <button className="primary" onClick={() => onSave(form)} disabled={isSaving}>
                        {isSaving ? '保存中...' : '保存'}
                    </button>
                </div>
            </div>
        </div>
    );
}