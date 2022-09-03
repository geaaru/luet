#!/bin/bash

testsourcedir=$(dirname "${BASH_SOURCE[0]}")
source ${testsourcedir}/_common.sh

oneTimeSetUp() {
  export tmpdir="${TEST_TMPDIR:-$(mktemp -d)}"
}

oneTimeTearDown() {
  if [ -z "${SKIP_CLEAN}" ] ; then
    rm -rf "$tmpdir"
  fi
}

testBuild() {
    mkdir $tmpdir/testbuild
    $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/finalizers_uninstall" --destination $tmpdir/testbuild --compression gzip --all
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create package' "[ -e '$tmpdir/testbuild/alpine-seed-1.0.package.tar.gz' ]"
    assertTrue 'create package' "[ -e '$tmpdir/testbuild/pkg1-app-1.0.package.tar.gz' ]"
}

testRepo() {
    assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
    $LUET_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/finalizers_uninstall" \
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
  database_path: "/var/cache/luet"
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
     type: "disk"
     enable: true
     urls:
       - "$tmpdir/testbuild"
EOF
    $LUET --version
    $LUET config --config $tmpdir/luet.yaml
    res=$?
    assertEquals 'config test successfully' "$res" "0"
}

testInstall() {
    $LUET repo update --config $tmpdir/luet.yaml
    $LUET miner d --config $tmpdir/luet.yaml main seed/alpine-1.0 app/pkg1
    $LUET miner i --config $tmpdir/luet.yaml main seed/alpine-1.0 app/pkg1
    #$LUET install -y --config $tmpdir/luet.yaml test/c-1.0 > /dev/null
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed' "[ -e '$tmpdir/testrootfs/bin/busybox' ]"
    assertTrue 'finalizer does not run' "[ -e '$tmpdir/testrootfs/tmp/foo' ]"
}


testUninstall() {
    $LUET miner rm app/pkg1 --config $tmpdir/luet.yaml
    installst=$?
    assertEquals 'uninstall test successfully' "$installst" "0"
    assertTrue 'finalizer uninstall not runs' "[ ! -e '$tmpdir/testrootfs/tmp/foo' ]"
}

testInstall2() {
  echo "Running luet miner i again..."
    $LUET miner i --config $tmpdir/luet.yaml main app/pkg1
    #$LUET install -y --config $tmpdir/luet.yaml test/c-1.0 > /dev/null
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed' "[ -e '$tmpdir/testrootfs/bin/busybox' ]"
    assertTrue 'finalizer does not run' "[ -e '$tmpdir/testrootfs/tmp/foo' ]"
}

testUninstall2() {
  echo "Running luet uninstall..."
  $LUET s --installed . --config $tmpdir/luet.yaml
    $LUET uninstall app/pkg1 --config $tmpdir/luet.yaml
    installst=$?
    assertEquals 'uninstall test successfully' "$installst" "0"
    assertTrue 'finalizer uninstall not runs' "[ ! -e '$tmpdir/testrootfs/tmp/foo' ]"
}

testCleanup() {
    $LUET cleanup --config $tmpdir/luet.yaml
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

