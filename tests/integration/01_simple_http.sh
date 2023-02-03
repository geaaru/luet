#!/bin/bash

testsourcedir=$(dirname "${BASH_SOURCE[0]}")
source ${testsourcedir}/_common.sh

TEST_PORT="${TEST_PORT:-9090}"

oneTimeSetUp() {
  export tmpdir="${TEST_TMPDIR:-$(mktemp -d)}"
}

oneTimeTearDown() {
    rm -rf "$tmpdir"
    kill '%1' || true
}

testBuild() {
    mkdir $tmpdir/testbuild
    $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/buildableseed" --destination $tmpdir/testbuild --compression zstd test/c@1.0 > /dev/null
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create package dep B' "[ -e '$tmpdir/testbuild/b-test-1.0.package.tar.zst' ]"
    assertTrue 'create package' "[ -e '$tmpdir/testbuild/c-test-1.0.package.tar.zst' ]"
}

testRepo() {
    $LUET_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/buildableseed" \
    --output $tmpdir/testbuild \
    --packages $tmpdir/testbuild \
    --name "test" \
    --descr "Test Repo" \
    --urls $tmpdir/testrootfs \
    --tree-compression zstd \
    --tree-filename foo.tar \
    --meta-filename repository.meta.tar \
    --meta-compression zstd \
    --type disk

    createst=$?
    assertEquals 'create repo successfully' "$createst" "0"

    $LUET_BUILD serve-repo --dir $tmpdir/testbuild --port $TEST_PORT -d &
}

testConfig() {
    mkdir $tmpdir/testrootfs
    cat <<EOF > $tmpdir/luet.yaml
general:
  debug: true
system:
  rootfs: $tmpdir/testrootfs
  database_path: "/"
  database_engine: "boltdb"
config_from_host: true
repos_confdir:
  - "$tmpdir/etc/luet/repos.conf.d"
config_protect_confdir:
  - "$tmpdir/etc/luet/config.protect.d"
subsets_defdir:
  - "$tmpdir/etc/luet/subsets.conf.d"
repositories:
   - name: "main"
     type: "http"
     enable: true
     cached: true
     urls:
       - "http://127.0.0.1:$TEST_PORT"
EOF
    $LUET config --config $tmpdir/luet.yaml
    res=$?
    assertEquals 'config test successfully' "$res" "0"
}

testInstall() {
    $LUET install --sync-repos -y --config $tmpdir/luet.yaml test/c@1.0
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed' "[ -e '$tmpdir/testrootfs/c' ]"
}

testReInstall() {
    output=$($LUET install --sync-repos -y --config $tmpdir/luet.yaml  test/c@1.0)
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertContains 'contains warning' "$output" 'No packages to install'
}

testUnInstall() {
    $LUET uninstall -y --config $tmpdir/luet.yaml test/c@1.0
    installst=$?
    assertEquals 'uninstall test successfully' "$installst" "0"
    assertTrue 'package uninstalled' "[ ! -e '$tmpdir/testrootfs/c' ]"
}

testInstallAgain() {
    assertTrue 'package uninstalled' "[ ! -e '$tmpdir/testrootfs/c' ]"
    output=$($LUET install --sync-repos -y --config $tmpdir/luet.yaml test/c@1.0)
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertNotContains 'contains warning' "$output" 'No packages to install'
    assertTrue 'package installed' "[ -e '$tmpdir/testrootfs/c' ]"
    assertTrue 'package in cache' "[ -e '$tmpdir/testrootfs/packages/c-test-1.0.package.tar.zst' ]"
}

testCleanup() {
    $LUET cleanup --config $tmpdir/luet.yaml
    installst=$?
    assertEquals 'cleanup test successfully' "$installst" "0"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

