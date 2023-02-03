#!/bin/bash

testsourcedir=$(dirname "${BASH_SOURCE[0]}")
source ${testsourcedir}/_common.sh

oneTimeSetUp() {
export tmpdir="$(mktemp -d)"
}

oneTimeTearDown() {
  rm -rf "$tmpdir"
}

testBuild() {
    mkdir $tmpdir/testbuild
    $LUET_BUILD build --all --concurrency 1 --tree "$ROOT_DIR/tests/fixtures/qlearning" --destination $tmpdir/testbuild --compression gzip
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create package dep B' "[ -e '$tmpdir/testbuild/b-test-1.0.package.tar.gz' ]"
    assertTrue 'create package' "[ -e '$tmpdir/testbuild/c-test-1.0.package.tar.gz' ]"
}

testRepo() {
    assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
    $LUET_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/qlearning" \
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
  database_path: "/"
  database_engine: "boltdb"
config_from_host: true
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
    $LUET install --sync-repos -y --config $tmpdir/luet.yaml test/c
    #$LUET install -y --config $tmpdir/luet.yaml test/c-1.0 > /dev/null
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package C installed' "[ -e '$tmpdir/testrootfs/c' ]"
}

testFullInstall() {
    # Without --force the solver exists with error without install packages
    output=$($LUET install --sync-repos -y --config $tmpdir/luet.yaml test/d test/f test/e test/a --force)
    installst=$?
    assertEquals 'cannot install' "$installst" "0"
    assertTrue 'package D installed' "[ -e '$tmpdir/testrootfs/d' ]"
    assertTrue 'package F installed' "[ -e '$tmpdir/testrootfs/f' ]"
    assertTrue 'package E not installed' "[ ! -e '$tmpdir/testrootfs/e' ]"
    assertTrue 'package A not installed' "[ ! -e '$tmpdir/testrootfs/a' ]"
}

testInstallAgain() {
    output=$($LUET install --sync-repos -y --config $tmpdir/luet.yaml test/d test/f test/e test/a --force)
    installst=$?
    echo "$output"
    assertEquals 'install test successfully' "0" "$installst"
    assertContains 'contains warning' "$output" 'No packages to install'
    assertTrue 'package D installed' "[ -e '$tmpdir/testrootfs/d' ]"
    assertTrue 'package F installed' "[ -e '$tmpdir/testrootfs/f' ]"
    assertTrue 'package E not installed' "[ ! -e '$tmpdir/testrootfs/e' ]"
    assertTrue 'package A not installed' "[ ! -e '$tmpdir/testrootfs/a' ]"
}

testCleanup() {
    $LUET cleanup --config $tmpdir/luet.yaml
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed' "[ ! -e '$tmpdir/testrootfs/packages/c-test-1.0.package.tar.gz' ]"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

