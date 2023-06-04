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
  $LUET_BUILD tree genidx --only-upper-level -t "$ROOT_DIR/tests/fixtures/upgrade_integration"
  genidx=$?
  assertEquals 'genidx successfully' "$genidx" "0"

  mkdir $tmpdir/testbuild
  $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/upgrade_integration" --destination $tmpdir/testbuild --compression gzip test/b@1.0 > ${OUTPUT}
  buildst=$?
  assertTrue 'create package B 1.0' "[ -e '$tmpdir/testbuild/b-test-1.0.package.tar.gz' ]"
  assertEquals 'builds successfully' "$buildst" "0"

  $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/upgrade_integration" --destination $tmpdir/testbuild --compression gzip test/b@1.1 > ${OUTPUT}
  buildst=$?
  assertEquals 'builds successfully' "$buildst" "0"
  assertTrue 'create package B 1.1' "[ -e '$tmpdir/testbuild/b-test-1.1.package.tar.gz' ]"

  $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/upgrade_integration" --destination $tmpdir/testbuild --compression gzip test/a@1.0 > ${OUTPUT}
  buildst=$?
  assertEquals 'builds successfully' "$buildst" "0"
  assertTrue 'create package A 1.0' "[ -e '$tmpdir/testbuild/a-test-1.0.package.tar.gz' ]"

  $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/upgrade_integration" --destination $tmpdir/testbuild --compression gzip test/a@1.1 > ${OUTPUT}
  buildst=$?
  assertEquals 'builds successfully' "$buildst" "0"

  assertTrue 'create package A 1.1' "[ -e '$tmpdir/testbuild/a-test-1.1.package.tar.gz' ]"

  $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/upgrade_integration" --destination $tmpdir/testbuild --compression gzip test/a@1.2 > ${OUTPUT}
  buildst=$?
  assertEquals 'builds successfully' "$buildst" "0"

  assertTrue 'create package A 1.2' "[ -e '$tmpdir/testbuild/a-test-1.2.package.tar.gz' ]"


  $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/upgrade_integration" --destination $tmpdir/testbuild --compression gzip test/c@1.0 > ${OUTPUT}
  buildst=$?
  assertEquals 'builds successfully' "$buildst" "0"
  assertTrue 'create package C 1.0' "[ -e '$tmpdir/testbuild/c-test-1.0.package.tar.gz' ]"

}

testRepo() {
  assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
  $LUET_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/upgrade_integration" \
  --output $tmpdir/testbuild \
  --packages $tmpdir/testbuild \
  --name "test" \
  --descr "Test Repo" \
  --urls $tmpdir/testrootfs \
  --type http > ${OUTPUT}

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
    $LUET install --sync-repos -y --config $tmpdir/luet.yaml test/b@1.0
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed B' "[ -e '$tmpdir/testrootfs/test5' ]"

    $LUET install --sync-repos -y --config $tmpdir/luet.yaml test/a@1.0
    assertTrue 'package installed A' "[ -e '$tmpdir/testrootfs/testaa' ]"
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    $LUET install --sync-repos -y --config $tmpdir/luet.yaml test/c@1.0
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed C' "[ -e '$tmpdir/testrootfs/c' ]"
}

testUpgrade() {
    upgrade=$($LUET --config $tmpdir/luet.yaml upgrade -y)
    installst=$?
    echo "$upgrade"
    assertEquals 'upgrade test successfully' "$installst" "0"
    assertTrue 'package uninstalled B' "[ ! -e '$tmpdir/testrootfs/test5' ]"
    assertTrue 'package installed B' "[ -e '$tmpdir/testrootfs/newc' ]"
    assertTrue 'package uninstalled A' "[ ! -e '$tmpdir/testrootfs/testaa' ]"
    assertTrue 'package installed new A' "[ -e '$tmpdir/testrootfs/testlatest' ]"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2
