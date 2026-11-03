#!/bin/bash -e
# Copie d'Azmo OS dans l'image (préparé par image/build-image.sh ou la CI).
install -m 0755 files/azmod "${ROOTFS_DIR}/usr/local/bin/azmod"
install -m 0644 files/azmoos.service "${ROOTFS_DIR}/etc/systemd/system/azmoos.service"
install -m 0644 files/azmoos-hotspot.service "${ROOTFS_DIR}/etc/systemd/system/azmoos-hotspot.service"
install -d -m 0755 "${ROOTFS_DIR}/usr/local/lib/azmoos"
install -m 0755 files/azmoos-hotspot.sh "${ROOTFS_DIR}/usr/local/lib/azmoos/azmoos-hotspot.sh"
install -m 0644 files/motd "${ROOTFS_DIR}/etc/motd"
install -d -m 0755 "${ROOTFS_DIR}/usr/share/azmoos"
rm -rf "${ROOTFS_DIR}/usr/share/azmoos/web"
cp -r files/web "${ROOTFS_DIR}/usr/share/azmoos/web"
install -d -m 0755 "${ROOTFS_DIR}/DATA"
install -d -m 0700 "${ROOTFS_DIR}/etc/azmoos"
