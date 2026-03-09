#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PKG_NAME="v2raye"
RAW_VERSION="${1:-${PKG_VERSION:-0.1.0}}"
PKG_VERSION="${RAW_VERSION#v}"
PKG_ARCH="${PKG_ARCH:-$(dpkg --print-architecture)}"
PKG_MAINTAINER="${PKG_MAINTAINER:-474420502 <474420502@users.noreply.github.com>}"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist}"
BUILD_ROOT="$DIST_DIR/deb-build"
PKG_ROOT="$BUILD_ROOT/${PKG_NAME}_${PKG_VERSION}_${PKG_ARCH}"
DEBIAN_DIR="$PKG_ROOT/DEBIAN"
INSTALL_ROOT="$PKG_ROOT/opt/v2rayE"
LIBEXEC_ROOT="$PKG_ROOT/usr/lib/$PKG_NAME"
SYSTEMD_DIR="$PKG_ROOT/usr/lib/systemd/system"
BIN_DIR="$PKG_ROOT/usr/bin"
OUTPUT_DEB="$DIST_DIR/${PKG_NAME}_${PKG_VERSION}_${PKG_ARCH}.deb"

if ! command -v dpkg-deb >/dev/null 2>&1; then
    echo "[error] dpkg-deb not found. Install dpkg first." >&2
    exit 1
fi

echo "=== Build v2rayE binary ==="
"$ROOT_DIR/scripts/build.sh"

echo "=== Prepare deb layout ==="
rm -rf "$PKG_ROOT"
mkdir -p "$DEBIAN_DIR" "$INSTALL_ROOT" "$LIBEXEC_ROOT" "$SYSTEMD_DIR" "$BIN_DIR" "$DIST_DIR"
chmod 750 "$INSTALL_ROOT"
chmod 755 "$LIBEXEC_ROOT" "$SYSTEMD_DIR" "$BIN_DIR"

install -m 755 "$ROOT_DIR/v2raye" "$LIBEXEC_ROOT/v2raye"
install -m 644 "$ROOT_DIR/docs/systemd/v2raye-server.service" "$SYSTEMD_DIR/v2raye-server.service"
ln -s /usr/lib/$PKG_NAME/v2raye "$BIN_DIR/v2raye"

cat >"$DEBIAN_DIR/control" <<EOF
Package: $PKG_NAME
Version: $PKG_VERSION
Section: net
Priority: optional
Architecture: $PKG_ARCH
Maintainer: $PKG_MAINTAINER
Depends: systemd
Description: v2rayE unified executable and backend service
 A local control plane for proxy profile management with built-in TUI
 and backend API service mode.
EOF

cat >"$DEBIAN_DIR/postinst" <<'EOF'
#!/bin/sh
set -e

if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
    if [ "$1" = "configure" ]; then
        rm -f /opt/v2rayE/v2raye || true
        systemctl enable v2raye-server.service >/dev/null 2>&1 || true
        systemctl restart v2raye-server.service >/dev/null 2>&1 || systemctl start v2raye-server.service >/dev/null 2>&1 || true
    fi
fi

exit 0
EOF

cat >"$DEBIAN_DIR/prerm" <<'EOF'
#!/bin/sh
set -e

if command -v systemctl >/dev/null 2>&1; then
    case "$1" in
        remove|deconfigure|upgrade|failed-upgrade)
            systemctl stop v2raye-server.service >/dev/null 2>&1 || true
            if [ "$1" = "remove" ] || [ "$1" = "deconfigure" ]; then
                systemctl disable v2raye-server.service >/dev/null 2>&1 || true
            fi
            ;;
    esac
fi

exit 0
EOF

cat >"$DEBIAN_DIR/postrm" <<'EOF'
#!/bin/sh
set -e

if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
fi

if [ "$1" = "purge" ]; then
    rm -rf /opt/v2rayE
fi

exit 0
EOF

chmod 755 "$DEBIAN_DIR/postinst" "$DEBIAN_DIR/prerm" "$DEBIAN_DIR/postrm"

echo "=== Build deb package ==="
dpkg-deb -Zxz --build --root-owner-group "$PKG_ROOT" "$OUTPUT_DEB"

echo "=== Done ==="
echo "deb package: $OUTPUT_DEB"
echo "install: sudo apt install $OUTPUT_DEB"
echo "remove : sudo apt remove $PKG_NAME"
echo "purge  : sudo apt purge $PKG_NAME"
