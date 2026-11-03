#!/bin/bash -e
# Copie du binaire AzmoOS et du service dans l'image.
install -m 0755 files/azmod "${ROOTFS_DIR}/usr/local/bin/azmod"
install -m 0644 files/azmoos.service "${ROOTFS_DIR}/etc/systemd/system/azmoos.service"
install -m 0644 files/motd "${ROOTFS_DIR}/etc/motd"
install -d -m 0755 "${ROOTFS_DIR}/DATA"
install -d -m 0700 "${ROOTFS_DIR}/etc/azmoos"
