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
config_protect_confdir:
  - "$tmpdir/etc/luet/config.protect.d"
subsets_defdir:
  - "$tmpdir/etc/luet/subsets.conf.d"
EOF
}

oneTimeTearDown() {
    rm -rf "$tmpdir"
}

testBuild() {
    mkdir $tmpdir/testbuild
    $LUET_BUILD build --config $tmpdir/luet-build.yaml \
      --tree "$ROOT_DIR/tests/fixtures/buildableseed" \
      --destination $tmpdir/testbuild \
      --compression gzip test/c > /dev/null
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create package dep B' "[ -e '$tmpdir/testbuild/b-test-1.0.package.tar.gz' ]"
    assertTrue 'create package' "[ -e '$tmpdir/testbuild/c-test-1.0.package.tar.gz' ]"
}

testRepo() {
    assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
    $LUET_BUILD create-repo --config $tmpdir/luet-build.yaml \
      --tree "$ROOT_DIR/tests/fixtures/buildableseed" \
      --output $tmpdir/testbuild \
      --packages $tmpdir/testbuild \
      --name "test" \
      --descr "Test Repo" \
      --urls $tmpdir/testrootfs \
      --type disk > /dev/null

    createst=$?
    assertEquals 'create repo successfully' "$createst" "0"
    assertTrue 'create repository' "[ -e '$tmpdir/testbuild/repository.yaml' ]"
}

testConfig() {
    mkdir $tmpdir/testrootfs
    cat <<EOF > $tmpdir/luet.yaml
general:
  debug: true
system:
  rootfs: $tmpdir/testrootfs
  database_engine: "memory"
config_from_host: true
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
    $LUET install -y --config $tmpdir/luet.yaml test/c
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed' "[ -e '$tmpdir/testrootfs/c' ]"
}

testCleanup() {
    $LUET cleanup --config $tmpdir/luet.yaml
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package cleaned' "[ ! -e '$tmpdir/testrootfs/packages/c-test-1.0.package.tar.gz' ]"
}

testInstall2() {
    $LUET install --sync-repos -y --config $tmpdir/luet.yaml --system-target $tmpdir/foo test/c
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'db not created' "[ ! -e '$tmpdir/foo/var/cache/luet/luet.db' ]"
    assertTrue 'package installed' "[ -e '$tmpdir/foo/c' ]"
}

testCleanup2() {
    $LUET cleanup --config $tmpdir/luet.yaml --system-target $tmpdir/foo
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package cleaned' "[ ! -e '$tmpdir/foo/packages/c-test-1.0.package.tar.gz' ]"
}

testInstall3() {
        cat <<EOF > $tmpdir/luet2.yaml
general:
  debug: true
config_from_host: true
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
    $LUET install --sync-repos -y --config $tmpdir/luet2.yaml --system-target $tmpdir/baz test/c
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed' "[ -e '$tmpdir/baz/c' ]"
}

testCleanup3() {
    $LUET cleanup --config $tmpdir/luet2.yaml --system-target $tmpdir/baz
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package cleaned' "[ ! -e '$tmpdir/baz/packages/c-test-1.0.package.tar.gz' ]"
}

testInstall4() {
    $LUET install --sync-repos -y --config $tmpdir/luet2.yaml --system-target $tmpdir/bad --system-engine boltdb test/c
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed' "[ -e '$tmpdir/bad/c' ]"
    assertTrue 'db created' "[ -d '$tmpdir/bad/var/cache/luet' ]"
}

testCleanup4() {
    $LUET cleanup --config $tmpdir/luet2.yaml --system-target $tmpdir/bad --system-engine boltdb
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package cleaned' "[ ! -e '$tmpdir/bad/packages/c-test-1.0.package.tar.gz' ]"
}
# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

