#!/bin/bash
if [ $(id -u) -ne 0 ]
  then echo "Please run the installer with sudo/as root"
  exit
fi

set -ex
export ANISE_NOLOCK=true

GITHUB_USER="${GITHUB_USER:-macaroni-os}"
GITHUB_BRANCH="${GITHUB_BRANCH:-geaaru}"

ANISE_VERSION="v0.41.0-${GITHUB_USER}"
ANISE_ROOTFS=${ANISE_ROOTFS:-/}
ANISE_DATABASE_PATH=${ANISE_DATABASE_PATH:-/var/cache/anise}
ANISE_DATABASE_ENGINE=${ANISE_DATABASE_ENGINE:-boltdb}
ANISE_CONFIG_PROTECT=${ANISE_CONFIG_PROTECT:-1}
ANISE_ARCH=${ANISE_ARCH:-x86_64}

curl -L https://github.com/${GITHUB_USER}/anise/releases/download/${ANISE_VERSION}/anise-${ANISE_VERSION}-Linux-${ANISE_ARCH} --output /usr/bin/anise
chmod +x /usr/bin/anise

mkdir -p /etc/anise/repos.conf.d || true
mkdir -p $ANISE_DATABASE_PATH || true
mkdir -p /var/tmp/anise || true

if [ "${ANISE_CONFIG_PROTECT}" = "1" ] ; then
  mkdir -p /etc/anise/config.protect.d || true
  curl -L https://raw.githubusercontent.com/${GITHUB_USER}/anise/${GITHUB_BRANCH}/contrib/config/config.protect.d/01_etc.yml.example --output /etc/anise/config.protect.d/01_etc.yml
fi
curl -L https://raw.githubusercontent.com/geaaru/repo-index/master/packages/geaaru-repo-index.yml --output /etc/anise/repos.conf.d/geaaru-repo-index.yml

if [ ! -e /etc/anise/anise.yaml ] ; then

  cat > /etc/anise/anise.yaml <<EOF
general:
  debug: false
system:
  rootfs: ${ANISE_ROOTFS}
  database_path: "${ANISE_DATABASE_PATH}"
  database_engine: "${ANISE_DATABASE_ENGINE}"
  tmpdir_base: "/var/tmp/anise"
EOF

fi

# Until next release of anise
if [ ! -e /etc/luet ] ; then
  cd /etc
  ln -s anise luet
  cd -
  cd /etc/anise
  ln -s anise.yaml luet.yaml
  cd -
fi

if [ "${ANISE_ARCH}" = "x86_64" ] ; then
  anise repo update
  anise install -y repository/mottainai-stable repository/geaaru-repo-index --force
  anise install --sync-repos -y system/anise-${GITHUB_USER} --force
else
  echo "Luet ARM repositories are not available yet."
fi

#rm -rf lue
