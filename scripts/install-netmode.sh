#!/usr/bin/env bash
# 安装 netmode 网络模式切换（雷神加速器 / v2rayE / 共存 / 全关）：
#   /usr/local/bin/netmode            root 侧的模式切换与守卫
#   netmode-boot.service              开机应用 /etc/netmode/default
#   ~/.local/bin/netmode-tray.py      桌面托盘（可选，--no-tray 跳过）
#   ~/.config/autostart/*.desktop     托盘自启
#   /etc/sudoers.d/netmode-<user>     只放行 `netmode` 一条命令的 NOPASSWD
#
# 用法：sudo ./scripts/install-netmode.sh [--no-tray] [--no-sudoers]
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PREFIX="${PREFIX:-/usr/local}"
CONFIG_DIR="${CONFIG_DIR:-/etc/netmode}"
UNIT_DIR="${UNIT_DIR:-/etc/systemd/system}"

WITH_TRAY=1
WITH_SUDOERS=1
for arg in "$@"; do
    case "$arg" in
        --no-tray) WITH_TRAY=0 ;;
        --no-sudoers) WITH_SUDOERS=0 ;;
        -h|--help) sed -n '2,12p' "$0"; exit 0 ;;
        *) echo "unknown option: $arg" >&2; exit 2 ;;
    esac
done

if [ "$(id -u)" -ne 0 ]; then
    echo "error: run as root (sudo $0)" >&2
    exit 1
fi

desktop_user() {
    local u="${SUDO_USER:-}"
    if [ -n "$u" ] && [ "$u" != "root" ]; then echo "$u"; return; fi
    getent passwd 1000 | cut -d: -f1
}

echo "[1/5] netmode -> $PREFIX/bin/netmode"
install -m 755 "$ROOT_DIR/scripts/netmode" "$PREFIX/bin/netmode"

echo "[2/5] boot unit -> $UNIT_DIR/netmode-boot.service"
install -m 644 "$ROOT_DIR/docs/systemd/netmode-boot.service" "$UNIT_DIR/netmode-boot.service"
systemctl daemon-reload
systemctl enable netmode-boot.service >/dev/null

echo "[3/5] default mode -> $CONFIG_DIR/default"
mkdir -p "$CONFIG_DIR"
if [ ! -f "$CONFIG_DIR/default" ]; then
    echo off >"$CONFIG_DIR/default"
    echo "      created default=off (set with: netmode set-default coexist)"
fi

USER_NAME="$(desktop_user)"
if [ "$WITH_TRAY" = 1 ] && [ -n "$USER_NAME" ]; then
    echo "[4/5] tray -> ~$USER_NAME/.local/bin/netmode-tray.py"
    user_home="$(getent passwd "$USER_NAME" | cut -d: -f6)"
    install -d -o "$USER_NAME" -g "$(id -gn "$USER_NAME")" "$user_home/.local/bin" "$user_home/.config/autostart"
    install -m 755 -o "$USER_NAME" -g "$(id -gn "$USER_NAME")" \
        "$ROOT_DIR/scripts/netmode-tray.py" "$user_home/.local/bin/netmode-tray.py"
    sed "s#@TRAY_PATH@#$user_home/.local/bin/netmode-tray.py#" \
        "$ROOT_DIR/scripts/netmode-tray.desktop" >"$user_home/.config/autostart/netmode-tray.desktop"
    chown "$USER_NAME:$(id -gn "$USER_NAME")" "$user_home/.config/autostart/netmode-tray.desktop"
    echo "      tray starts at next login; now: sudo -u $USER_NAME $user_home/.local/bin/netmode-tray.py &"
else
    echo "[4/5] tray skipped"
fi

if [ "$WITH_SUDOERS" = 1 ] && [ -n "$USER_NAME" ] && [ -d /etc/sudoers.d ] && command -v visudo >/dev/null; then
    echo "[5/5] sudoers -> /etc/sudoers.d/netmode-$USER_NAME"
    tmp="$(mktemp)"
    printf '%s ALL=(root) NOPASSWD: %s/bin/netmode\n' "$USER_NAME" "$PREFIX" >"$tmp"
    if visudo -cf "$tmp" >/dev/null; then
        install -m 440 -o root -g root "$tmp" "/etc/sudoers.d/netmode-$USER_NAME"
    else
        echo "      warning: generated rule failed visudo, skipped" >&2
    fi
    rm -f "$tmp"
else
    echo "[5/5] sudoers skipped"
fi

echo
echo "check: netmode status && sudo netmode verify"
echo "note : 雷神需要 /opt/leigod + leigod_plugin.service，v2rayE 需要 v2raye-server.service"
