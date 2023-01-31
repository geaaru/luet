#!/bin/bash

testsourcedir=$(dirname "${BASH_SOURCE[0]}")
source ${testsourcedir}/_common.sh

oneTimeSetUp() {
export tmpdir="$(mktemp -d)"
  cat <<EOF > $tmpdir/luet-build.yaml
general:
  debug: true
logging:
  enable_emoji: false
  color: false
system:
  rootfs: $tmpdir/testrootfs
  database_path: "/"
  database_engine: "memory"
config_from_host: true
repos_confdir:
  - "$tmpdir/etc/luet/repos.conf.d"
EOF

}

oneTimeTearDown() {
  rm -rf "$tmpdir"
}

testBuild() {
    mkdir $tmpdir/testbuild
    $LUET_BUILD build \
      --config $tmpdir/luet-build.yaml \
      --tree "$ROOT_DIR/tests/fixtures/subsets" \
      --destination $tmpdir/testbuild --compression zstd subset/a
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create package' "[ -e '$tmpdir/testbuild/a-subset-1.0.package.tar.zst' ]"
}

testRepo() {
    assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
    $LUET_BUILD create-repo \
      --config $tmpdir/luet-build.yaml \
      --tree "$ROOT_DIR/tests/fixtures/subsets" \
      --output $tmpdir/testbuild \
      --packages $tmpdir/testbuild \
      --name "test" \
      --descr "Test Repo" \
      --urls $tmpdir/testrootfs \
      --type disk
    createst=$?
    assertEquals 'create repo successfully' "$createst" "0"
    assertTrue 'create repository' "[ -e '$tmpdir/testbuild/repository.yaml' ]"
}

testConfig() {
    mkdir $tmpdir/testrootfs

    mkdir $tmpdir/config.protect.d
    mkdir $tmpdir/etc/luet/subsets.conf.d -p
    mkdir $tmpdir/etc/luet/subsets.def.d -p

    cat <<EOF > $tmpdir/config.protect.d/conf1.yml
name: "protect1"
dirs:
- /etc/
EOF

    cat <<EOF > $tmpdir/etc/luet/subsets.def.d/00-testdata.yml
subsets_def:
  test-data:
    description: "Local subset"
    name: "test-data"
    rules:
    - ^/opt/data
    categories:
    - subset
EOF

    cat <<EOF > $tmpdir/luet.yaml
general:
  debug: true
system:
  rootfs: $tmpdir/testrootfs
  database_path: "/"
  database_engine: "boltdb"
config_protect_confdir:
    - $tmpdir/config.protect.d
config_from_host: true
subsets_confdir:
  - "$tmpdir/etc/luet/subsets.conf.d"
subsets_defdir:
  - "$tmpdir/etc/luet/subsets.def.d"
repos_confdir:
  - "$tmpdir/etc/luet/repos.conf.d"
repositories:
   - name: "main"
     type: "disk"
     enable: true
     cached: true
     urls:
       - "$tmpdir/testbuild"
EOF
    $LUET config --config $tmpdir/luet.yaml
    res=$?
    assertEquals 'config test successfully' "$res" "0"
}

testInstall() {
  $LUET install --sync-repos -y --config $tmpdir/luet.yaml subset/a
  installst=$?
  assertEquals 'install test successfully' "$installst" "0"

   assertTrue 'package A /c installed' "[ -e '$tmpdir/testrootfs/c' ]"
   assertTrue 'package A /cd installed' "[ -e '$tmpdir/testrootfs/cd' ]"
   assertTrue 'package A /usr/include/file.h not installed' \
     "[ ! -e '$tmpdir/testrootfs/usr/include/file.h' ]"

   assertTrue 'package A /opt/data/file not installed' \
     "[ ! -e '$tmpdir/testrootfs/opt/data/file' ]"
}


testUnInstall() {
  $LUET uninstall -y --config $tmpdir/luet.yaml subset/a
  installst=$?
  assertEquals 'uninstall test successfully' "$installst" "0"
  assertTrue 'package uninstalled' "[ ! -e '$tmpdir/testrootfs/c' ]"
  assertTrue 'package uninstalled' "[ ! -e '$tmpdir/testrootfs/cd' ]"
}

testInstall2() {
    cat <<EOF > $tmpdir/luet.yaml
general:
  debug: true
system:
  rootfs: $tmpdir/testrootfs
  database_path: "/"
  database_engine: "boltdb"
config_protect_confdir:
    - $tmpdir/config.protect.d
config_from_host: true
repos_confdir:
  - "$tmpdir/etc/luet/repos.conf.d"
subsets:
  enabled:
    - devel
    - test-data
repositories:
   - name: "main"
     type: "disk"
     enable: true
     urls:
       - "$tmpdir/testbuild"
EOF
  $LUET config --config $tmpdir/luet.yaml

  LUET_LOGGING__PARANOID="true" $LUET install --sync-repos -y --config $tmpdir/luet.yaml subset/a
  installst=$?
  assertEquals 'install test successfully' "$installst" "0"

  assertTrue 'package A /c installed' "[ -e '$tmpdir/testrootfs/c' ]"
  assertTrue 'package A /cd installed' "[ -e '$tmpdir/testrootfs/cd' ]"
  assertTrue 'package A /usr/include/file.h installed' \
    "[ -e '$tmpdir/testrootfs/usr/include/file.h' ]"
   assertTrue 'package A /opt/data/file installed' \
     "[ -e '$tmpdir/testrootfs/opt/data/file' ]"

}

testCleanup() {
  $LUET cleanup --config $tmpdir/luet.yaml
  installst=$?
  assertEquals 'install test successfully' "$installst" "0"
  assertTrue 'package installed' "[ ! -e '$tmpdir/testrootfs/packages/a-subset-1.0.package.tar.gz' ]"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

