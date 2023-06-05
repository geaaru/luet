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
  $LUET_BUILD tree genidx --only-upper-level -t "$ROOT_DIR/tests/fixtures/fileconflicts"
  genidx=$?
  assertEquals 'genidx successfully' "$genidx" "0"

  mkdir $tmpdir/testbuild
  $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/fileconflicts" --destination $tmpdir/testbuild --compression gzip --all
  buildst=$?
  assertEquals 'builds successfully' "$buildst" "0"
  assertTrue 'create packages' "[ -e '$tmpdir/testbuild/conflict-test1-1.0.package.tar.gz' ]"
  assertTrue 'create packages' "[ -e '$tmpdir/testbuild/conflict-test2-1.0.package.tar.gz' ]"
}

testRepo() {
  assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
  $LUET_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/fileconflicts" \
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

testReInstall() {
    $LUET install --sync-repos -y --config $tmpdir/luet.yaml test1/conflict
    installst=$?
    assertEquals 'install test succeeded' "$installst" "0"
    $LUET miner ri --config $tmpdir/luet.yaml test1/conflict
    installst=$?
    assertEquals 'reinstall test succeeded' "$installst" "0"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

