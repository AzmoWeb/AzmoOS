#!/bin/bash -e
# Docker (dépôt officiel), puis activation des services Azmo OS.
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
. /etc/os-release
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian ${VERSION_CODENAME} stable" > /etc/apt/sources.list.d/docker.list
apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
usermod -aG docker "${FIRST_USER_NAME:-azmo}" || true
chown 1000:1000 /DATA
systemctl enable docker avahi-daemon smbd azmoos azmoos-hotspot
# Le pare-feu est installé mais pas activé : l'utilisateur l'active depuis l'interface.
systemctl disable ufw || true
