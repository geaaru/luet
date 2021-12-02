#!/bin/bash
if [ $(id -u) -ne 0 ]
  then echo "Please run the installer with sudo/as root"
  exit
fi

set -ex
export LUET_NOLOCK=true

LUET_VERSION="v0.20.0-geaaru"
LUET_ROOTFS=${LUET_ROOTFS:-/}
LUET_DATABASE_PATH=${LUET_DATABASE_PATH:-/var/luet/db}
LUET_DATABASE_ENGINE=${LUET_DATABASE_ENGINE:-boltdb}
LUET_CONFIG_PROTECT=${LUET_CONFIG_PROTECT:-1}

GITHUB_USER="${GITHUB_USER:-geaaru}"
GITHUB_BRANCH="${GITHUB_BRANCH:-master}"

curl -L https://github.com/${GITHUB_USER}/luet/releases/download/${LUET_VERSION}/luet-${LUET_VERSION}-linux-amd64 --output /usr/bin/luet
chmod +x /usr/bin/luet

mkdir -p /etc/luet/repos.conf.d || true
mkdir -p $LUET_DATABASE_PATH || true
mkdir -p /var/tmp/luet || true

if [ "${LUET_CONFIG_PROTECT}" = "1" ] ; then
  mkdir -p /etc/luet/config.protect.d || true
  curl -L https://raw.githubusercontent.com/${GITHUB_USER}/luet/${GITHUB_BRANCH}/contrib/config/config.protect.d/01_etc.yml.example --output /etc/luet/config.protect.d/01_etc.yml
fi
curl -L https://raw.githubusercontent.com/geaaru/repo-index/master/packages/geaaru-repo-index.yml --output /etc/luet/repos.conf.d/geaaru-repo-index.yml

cat > /etc/luet/luet.yaml <<EOF
general:
  debug: false
system:
  rootfs: ${LUET_ROOTFS}
  database_path: "${LUET_DATABASE_PATH}"
  database_engine: "${LUET_DATABASE_ENGINE}"
  tmpdir_base: "/var/tmp/luet"
EOF

luet repo update
luet install -y repository/mottainai-stable repository/geaaru-repo-index --force
luet install --sync-repos -y system/luet-extensions system/luet-${GITHUB_USER} --force

#rm -rf luet
