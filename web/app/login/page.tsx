'use client';

import { FormEvent, useState } from 'react';
import { useRouter } from 'next/navigation';

export default function LoginPage() {
    const router = useRouter();
    const [token, setToken] = useState('');

    const onSubmit = (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        if (!token.trim()) {
            return;
        }

        document.cookie = `auth_token=${encodeURIComponent(token.trim())}; Path=/; Max-Age=604800; SameSite=Lax`;
        router.push('/dashboard');
    };

    return (
        <div className="login-wrap">
            <form className="login-card" onSubmit={onSubmit}>
                <h2>登录</h2>
                <p className="muted">输入后端签发的访问令牌（Token）</p>
                <div className="field">
                    <label htmlFor="token">Access Token</label>
                    <input
                        id="token"
                        value={token}
                        onChange={(event) => setToken(event.target.value)}
                        placeholder="请输入 Token"
                    />
                </div>
                <button className="primary" type="submit">
                    进入控制台
                </button>
            </form>
        </div>
    );
}