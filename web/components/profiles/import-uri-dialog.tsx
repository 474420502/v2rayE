'use client';

type ImportUriDialogProps = {
    open: boolean;
    value: string;
    isImporting: boolean;
    onChange: (value: string) => void;
    onClose: () => void;
    onImport: () => void;
};

export function ImportUriDialog({
    open,
    value,
    isImporting,
    onChange,
    onClose,
    onImport,
}: ImportUriDialogProps) {
    if (!open) {
        return null;
    }

    return (
        <div className="modal-overlay">
            <div className="modal-box" style={{ maxWidth: 480, width: '95%' }}>
                <h3>导入节点链接</h3>
                <p className="muted" style={{ marginBottom: 12 }}>
                    支持 vmess://、vless://、ss://、trojan://、hy2://、tuic:// 链接
                </p>
                <input
                    value={value}
                    onChange={(e) => onChange(e.target.value)}
                    placeholder="粘贴分享链接..."
                    style={{ width: '100%', marginBottom: 12 }}
                />
                <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
                    <button onClick={onClose} disabled={isImporting}>
                        取消
                    </button>
                    <button className="primary" onClick={onImport} disabled={isImporting || !value.trim()}>
                        {isImporting ? '导入中...' : '导入'}
                    </button>
                </div>
            </div>
        </div>
    );
}