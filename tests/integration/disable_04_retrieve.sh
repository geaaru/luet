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
  $ANISE_BUILD tree genidx --only-upper-level -t "$ROOT_DIR/tests/fixtures/retrieve-integration"
  genidx=$?
  assertEquals 'genidx successfully' "$genidx" "0"

  mkdir $tmpdir/testbuild
  [ "$ANISE_BACKEND" == "img" ] && startSkipping
  $ANISE_BUILD build --tree "$ROOT_DIR/tests/fixtures/retrieve-integration" --destination $tmpdir/testbuild --compression gzip test/b > ${OUTPUT}
  buildst=$?
  assertEquals 'builds successfully' "$buildst" "0"
  assertTrue 'create package dep B' "[ -e '$tmpdir/testbuild/b-test-1.0.package.tar.gz' ]"
  assertTrue 'create package' "[ -e '$tmpdir/testbuild/a-test-1.0.package.tar.gz' ]"
}

testRepo() {
  [ "$ANISE_BACKEND" == "img" ] && startSkipping
  assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
  $ANISE_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/retrieve-integration" \
  --output $tmpdir/testbuild \
  --packages $tmpdir/testbuild \
  --name "test" \
  --descr "Test Repo" \
  --urls $tmpdir/testrootfs \
  --type disk > ${OUTPUT}

  createst=$?
  assertEquals 'create repo successfully' "$createst" "0"
  assertTrue 'create repository' "[ -e '$tmpdir/testbuild/repository.yaml' ]"
}

testConfig() {
    mkdir $tmpdir/testrootfs
    cat <<EOF > $tmpdir/anise.yaml
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
    $ANISE config --config $tmpdir/anise.yaml
    res=$?
    assertEquals 'config test successfully' "$res" "0"
}



testInstall() {
    [ "$ANISE_BACKEND" == "img" ] && startSkipping
    $ANISE install --sync-repos -y --config $tmpdir/anise.yaml test/b
    #$ANISE install -y --config $tmpdir/anise.yaml test/c-1.0 > /dev/null
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package B installed' "[ -e '$tmpdir/testrootfs/b' ]"
    val=$(cat "$tmpdir/testrootfs/b")
    assertEquals 'package B content comes from a' "$val" "a"
    assertTrue 'package A installed' "[ -e '$tmpdir/testrootfs/a' ]"
}


testUnInstall() {
    [ "$ANISE_BACKEND" == "img" ] && startSkipping
    $ANISE uninstall -y --config $tmpdir/anise.yaml test/a
    installst=$?
    assertEquals 'uninstall test successfully' "$installst" "0"
    assertTrue 'package uninstalled' "[ ! -e '$tmpdir/testrootfs/b' ]"
    assertTrue 'package uninstalled' "[ ! -e '$tmpdir/testrootfs/a' ]"
}


testCleanup() {
    [ "$ANISE_BACKEND" == "img" ] && startSkipping
    $ANISE cleanup --config $tmpdir/anise.yaml
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed' "[ ! -e '$tmpdir/testrootfs/packages/b-test-1.0.package.tar.gz' ]"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

