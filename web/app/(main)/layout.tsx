import { Sidebar } from '@/components/sidebar';

export default function MainLayout({ children }: { children: React.ReactNode }) {
    return (
        <div className="app-shell">
            <Sidebar />
            <main className="content">{children}</main>
        </div>
    );
}