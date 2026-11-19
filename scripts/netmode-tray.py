#!/usr/bin/env python3
"""netmode tray — 雷神加速器 / v2rayE 一键切换托盘程序

依赖: python3-gi, gir1.2-ayatanaappindicator3-0.1, ubuntu-appindicators 扩展
配合 /usr/local/bin/netmode 使用（该脚本负责真正的 systemctl 切换）。
"""
import os
import sys
import shutil
import fcntl
import subprocess
import threading

import gi
gi.require_version('Gtk', '3.0')
gi.require_version('AyatanaAppIndicator3', '0.1')
from gi.repository import Gtk, GLib, AyatanaAppIndicator3 as AppIndicator

HELPER = '/usr/local/bin/netmode'
ICON_DIR = os.path.expanduser('~/.local/share/icons/netmode')

LABEL = {
    'leigod':  '雷神加速（游戏）',
    'v2ray':   'v2rayE 全代理（TUN）',
    'coexist': '共存（雷神 + v2rayE SOCKS）',
    'off':     '全部关闭',
}
COLOR = {
    'leigod':  (1.00, 0.42, 0.00),
    'v2ray':   (0.23, 0.51, 0.96),
    'coexist': (0.66, 0.33, 0.97),
    'off':     (0.42, 0.45, 0.50),
}
ORDER = ['leigod', 'v2ray', 'coexist', 'off']


def ensure_icons():
    """用 cairo 画 4 个模式图标（颜色区分），返回 {mode: path}。"""
    try:
        import cairo
    except Exception:
        return {}
    os.makedirs(ICON_DIR, exist_ok=True)
    out = {}
    for mode, (r, g, b) in COLOR.items():
        path = os.path.join(ICON_DIR, f'{mode}.png')
        surf = cairo.ImageSurface(cairo.FORMAT_ARGB32, 64, 64)
        ctx = cairo.Context(surf)
        ctx.set_source_rgba(0, 0, 0, 0)
        ctx.paint()
        ctx.arc(32, 32, 28, 0, 6.28318)
        ctx.set_source_rgb(r, g, b)
        ctx.fill()
        ctx.set_line_width(5)
        ctx.arc(32, 32, 15, 0, 6.28318)
        ctx.set_source_rgb(1, 1, 1)
        ctx.stroke()
        ctx.arc(32, 32, 4.5, 0, 6.28318)
        ctx.set_source_rgb(1, 1, 1)
        ctx.fill()
        surf.write_to_png(path)
        out[mode] = path
    return out


class Tray:
    def __init__(self):
        self.icons = ensure_icons()
        self.mode = None
        self.busy = False
        self.syncing = False

        self.ind = AppIndicator.Indicator.new(
            'netmode',
            self.icons.get('off', 'network-vpn'),
            AppIndicator.IndicatorCategory.SYSTEM_SERVICES,
        )
        self.ind.set_status(AppIndicator.IndicatorStatus.ACTIVE)
        self.ind.set_title('网络模式')
        self.build_menu()
        self.refresh()

    # ── UI ──────────────────────────────────────────────────────────
    def build_menu(self):
        menu = Gtk.Menu()

        self.item_head = Gtk.MenuItem(label='正在检测…')
        self.item_head.set_sensitive(False)
        menu.append(self.item_head)
        menu.append(Gtk.SeparatorMenuItem())

        self.radios = {}
        first = None
        for mode in ORDER:
            if first is None:
                item = Gtk.RadioMenuItem(label=LABEL[mode])
                first = item
            else:
                item = Gtk.RadioMenuItem.new_with_label_from_widget(first, LABEL[mode])
            item.connect('toggled', self.on_toggle, mode)
            menu.append(item)
            self.radios[mode] = item

        menu.append(Gtk.SeparatorMenuItem())

        self.item_default = Gtk.MenuItem(label='开机默认：—')
        self.item_default.set_sensitive(False)
        menu.append(self.item_default)

        self.item_set_default = Gtk.MenuItem(label='把当前模式设为开机默认')
        self.item_set_default.connect('activate', self.on_set_default)
        menu.append(self.item_set_default)

        menu.append(Gtk.SeparatorMenuItem())

        self.item_ip = Gtk.MenuItem(label='查询出口 IP')
        self.item_ip.connect('activate', self.on_ip)
        menu.append(self.item_ip)

        self.item_term = Gtk.MenuItem(label='打开 v2rayE 终端')
        self.item_term.connect('activate', self.on_term)
        menu.append(self.item_term)

        menu.append(Gtk.SeparatorMenuItem())
        quit_item = Gtk.MenuItem(label='退出')
        quit_item.connect('activate', lambda *_: Gtk.main_quit())
        menu.append(quit_item)

        menu.show_all()
        self.ind.set_menu(menu)

    # ── 切换 ────────────────────────────────────────────────────────
    def on_toggle(self, item, mode):
        if self.syncing or not item.get_active() or self.busy:
            return
        self.apply(mode)

    def apply(self, mode):
        self.busy = True
        self.item_head.set_label(f'切换中 → {LABEL[mode]} …')

        def work():
            try:
                proc = subprocess.run(['sudo', '-n', HELPER, mode],
                                      capture_output=True, text=True, timeout=90)
                rc, err = proc.returncode, proc.stderr.strip()
            except Exception as exc:               # noqa: BLE001
                rc, err = 1, str(exc)
            GLib.idle_add(self.after_apply, mode, rc, err)

        threading.Thread(target=work, daemon=True).start()

    def after_apply(self, mode, rc, err):
        self.busy = False
        if rc == 0:
            self.notify('模式已切换', LABEL[mode])
        else:
            self.notify('切换失败', err or f'exit {rc}')
        self.refresh()
        return False

    # ── 状态刷新 ────────────────────────────────────────────────────
    def refresh(self):
        if self.busy:
            return True          # a switch is in flight; its own refresh follows

        def work():
            try:
                proc = subprocess.run(['sudo', '-n', HELPER, 'status'],
                                      capture_output=True, text=True, timeout=20)
                mode = proc.stdout.strip()
            except Exception:                      # noqa: BLE001
                mode = ''
            try:
                with open('/etc/netmode/default') as fh:
                    default = fh.read().strip()
            except Exception:                      # noqa: BLE001
                default = ''
            GLib.idle_add(self.render, mode, default)

        threading.Thread(target=work, daemon=True).start()
        return True

    def render(self, mode, default=''):
        known = mode in LABEL
        self.mode = mode if known else 'off'
        text = LABEL.get(mode, mode or '未知')

        self.item_head.set_label(f'当前：{text}')
        self.item_default.set_label(
            f'开机默认：{LABEL.get(default, default or "未设置")}')
        self.syncing = True
        for key, item in self.radios.items():
            item.set_active(key == mode)
        self.syncing = False

        icon = self.icons.get(self.mode) if known else None
        if icon:
            self.ind.set_icon_full(icon, text)
        self.ind.set_title(f'网络模式：{text}')
        return False

    def on_set_default(self, *_):
        mode = self.mode
        if mode not in LABEL:
            return

        def work():
            try:
                proc = subprocess.run(['sudo', '-n', HELPER, 'set-default', mode],
                                      capture_output=True, text=True, timeout=15)
                rc, err = proc.returncode, proc.stderr.strip()
            except Exception as exc:               # noqa: BLE001
                rc, err = 1, str(exc)
            GLib.idle_add(self.after_set_default, mode, rc, err)

        threading.Thread(target=work, daemon=True).start()

    def after_set_default(self, mode, rc, err):
        if rc == 0:
            self.notify('开机默认已设置', LABEL[mode])
        else:
            self.notify('设置失败', err or f'exit {rc}')
        self.refresh()
        return False

    # ── 附加动作 ────────────────────────────────────────────────────
    def on_ip(self, *_):
        self.item_ip.set_label('查询中…')

        def work():
            try:
                ip = subprocess.run(
                    ['curl', '-s', '--max-time', '10', 'https://ifconfig.me'],
                    capture_output=True, text=True, timeout=15,
                ).stdout.strip()
            except Exception:                      # noqa: BLE001
                ip = ''
            GLib.idle_add(self.show_ip, ip)

        threading.Thread(target=work, daemon=True).start()

    def show_ip(self, ip):
        ip = ip or '查询失败'
        self.item_ip.set_label(f'出口 IP：{ip}')
        self.notify('当前出口 IP', ip)
        return False

    def on_term(self, *_):
        term = shutil.which('gnome-terminal') or shutil.which('x-terminal-emulator')
        if not term:
            self.notify('打开失败', '找不到终端程序')
            return
        if term.endswith('gnome-terminal'):
            cmd = [term, '--', 'bash', '-lc', 'v2raye; exec bash']
        else:
            cmd = [term, '-e', 'bash -lc "v2raye; exec bash"']
        try:
            subprocess.Popen(cmd)
        except Exception as exc:                   # noqa: BLE001
            self.notify('打开失败', str(exc))

    @staticmethod
    def notify(title, body):
        try:
            subprocess.Popen(['notify-send', '-a', 'netmode',
                              '-i', 'network-vpn', title, body])
        except Exception:                          # noqa: BLE001
            pass


def main():
    # 单实例
    os.makedirs(os.path.expanduser('~/.cache'), exist_ok=True)
    lock = open(os.path.expanduser('~/.cache/netmode-tray.lock'), 'w')
    try:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except OSError:
        print('netmode-tray 已在运行', file=sys.stderr)
        return 0

    tray = Tray()
    GLib.timeout_add_seconds(20, tray.refresh)
    Gtk.main()
    return 0


if __name__ == '__main__':
    sys.exit(main())
