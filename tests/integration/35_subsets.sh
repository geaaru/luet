#!/bin/bash

export LUET_NOLOCK=true

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
    luet build \
      --config $tmpdir/luet-build.yaml \
      --tree "$ROOT_DIR/tests/fixtures/subsets" \
      --destination $tmpdir/testbuild --compression zstd subset/a
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create package' "[ -e '$tmpdir/testbuild/a-subset-1.0.package.tar.zst' ]"
}

testRepo() {
    assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
    luet create-repo \
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

    cat <<EOF > $tmpdir/config.protect.d/conf1.yml
name: "protect1"
dirs:
- /etc/
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
repos_confdir:
  - "$tmpdir/etc/luet/repos.conf.d"
repositories:
   - name: "main"
     type: "disk"
     enable: true
     urls:
       - "$tmpdir/testbuild"
EOF
    luet config --config $tmpdir/luet.yaml
    res=$?
    assertEquals 'config test successfully' "$res" "0"
}

testInstall() {
  luet install --sync-repos -y --config $tmpdir/luet.yaml subset/a
  installst=$?
  assertEquals 'install test successfully' "$installst" "0"

   assertTrue 'package A /c installed' "[ -e '$tmpdir/testrootfs/c' ]"
   assertTrue 'package A /cd installed' "[ -e '$tmpdir/testrootfs/cd' ]"
   assertTrue 'package A /usr/include/file.h not installed' \
     "[ ! -e '$tmpdir/usr/include/file.h' ]"
}


testUnInstall() {
  luet uninstall -y --full --config $tmpdir/luet.yaml subset/a
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
repositories:
   - name: "main"
     type: "disk"
     enable: true
     urls:
       - "$tmpdir/testbuild"
EOF
  luet config --config $tmpdir/luet.yaml

  LUET_LOGGING__PARANOID="true" luet install --sync-repos -y --config $tmpdir/luet.yaml subset/a
  installst=$?
  assertEquals 'install test successfully' "$installst" "0"

  assertTrue 'package A /c installed' "[ -e '$tmpdir/testrootfs/c' ]"
  assertTrue 'package A /cd installed' "[ -e '$tmpdir/testrootfs/cd' ]"
  assertTrue 'package A /usr/include/file.h installed' \
    "[ -e '$tmpdir/testrootfs/usr/include/file.h' ]"

}

testCleanup() {
  luet cleanup --config $tmpdir/luet.yaml
  installst=$?
  assertEquals 'install test successfully' "$installst" "0"
  assertTrue 'package installed' "[ ! -e '$tmpdir/testrootfs/packages/a-subset-1.0.package.tar.gz' ]"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

