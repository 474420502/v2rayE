import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
    title: 'v2rayN Web',
    description: 'Ubuntu Linux backend web management UI'
};

export default function RootLayout({
    children
}: Readonly<{
    children: React.ReactNode;
}>) {
    return (
        <html lang="zh-CN">
            <body>{children}</body>
        </html>
    );
}