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
  [ "$LUET_BACKEND" == "img" ] && startSkipping

  $LUET_BUILD tree genidx --only-upper-level -t "$ROOT_DIR/tests/fixtures/owners"
  genidx=$?
  assertEquals 'genidx successfully' "$genidx" "0"

  mkdir $tmpdir/testbuild
  $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/owners" --destination $tmpdir/testbuild --compression gzip test/unpack test/delta
  buildst=$?
  assertEquals 'builds successfully' "$buildst" "0"
  assertTrue 'create package unpack' "[ -e '$tmpdir/testbuild/unpack-test-1.0.package.tar.gz' ]"
  assertTrue 'create package delta' "[ -e '$tmpdir/testbuild/delta-test-1.0.package.tar.gz' ]"
}

testRepo() {
  [ "$LUET_BACKEND" == "img" ] && startSkipping
  assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
  $LUET_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/owners" \
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
    [ "$LUET_BACKEND" == "img" ] && startSkipping
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
    [ "$LUET_BACKEND" == "img" ] && startSkipping
    $LUET install --sync-repos -y --config $tmpdir/luet.yaml test/unpack test/delta
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    fileUID=$(stat -c "%u" $tmpdir/testrootfs/foo)
    fileGID=$(stat -c "%g" $tmpdir/testrootfs/foo)
    filePerms=$(stat -c "%a" $tmpdir/testrootfs/foo)
    assertEquals 'UID on /foo matches' "1000" "$fileUID"
    assertEquals 'GID on /foo matches' "1001" "$fileGID"
    assertEquals 'bits on /foo matches' "500" "$filePerms"

    fileUID=$(stat -c "%u" $tmpdir/testrootfs/bar)
    fileGID=$(stat -c "%g" $tmpdir/testrootfs/bar)
    filePerms=$(stat -c "%a" $tmpdir/testrootfs/bar)
    assertEquals 'UID on /bar matches' "1000" "$fileUID"
    assertEquals 'GID on /bar matches' "1001" "$fileGID"
    assertEquals 'bits on /bar matches' "600" "$filePerms"
}

testCleanup() {
    [ "$LUET_BACKEND" == "img" ] && startSkipping
    $LUET cleanup --config $tmpdir/luet.yaml
    installst=$?
    assertEquals 'cleanup test successfully' "$installst" "0"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2
