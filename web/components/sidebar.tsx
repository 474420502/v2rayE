'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import type { Route } from 'next';

const items = [
    { href: '/dashboard', label: '概览' },
    { href: '/profiles', label: '节点' },
    { href: '/subscriptions', label: '订阅' },
    { href: '/routing', label: '路由规则' },
    { href: '/network', label: '系统代理与网络' },
    { href: '/settings', label: '设置' },
    { href: '/logs', label: '日志' }
] as const satisfies ReadonlyArray<{ href: Route; label: string }>;

export function Sidebar() {
    const pathname = usePathname();

    return (
        <aside className="sidebar">
            <div className="sidebar-brand">
                <span className="sidebar-kicker">Linux Web Console</span>
                <h1>v2rayN Web</h1>
                <p>Go backend + Next.js frontend</p>
            </div>
            <nav>
                {items.map((item) => (
                    <Link
                        key={item.href}
                        href={item.href}
                        className={pathname === item.href ? 'active' : ''}
                    >
                        {item.label}
                    </Link>
                ))}
            </nav>
        </aside>
    );
}