#!/bin/bash

testsourcedir=$(dirname "${BASH_SOURCE[0]}")
source ${testsourcedir}/_common.sh

oneTimeSetUp() {
  export tmpdir="$(mktemp -d)"
}

oneTimeTearDown() {
  if [ -z "${SKIP_CLEAN}" ] ; then
    rm -rf "$tmpdir"
  fi
}

testBuild() {
  $LUET_BUILD tree genidx --only-upper-level -t "$ROOT_DIR/tests/fixtures/upgrade_provides" \
    -t "$ROOT_DIR/tests/fixtures/upgrade_provides_new"
  genidx=$?
  assertEquals 'genidx successfully' "$genidx" "0"

  mkdir $tmpdir/testbuild
  $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/upgrade_provides" --destination $tmpdir/testbuild --compression gzip --full > ${OUTPUT}
  buildst=$?
  assertTrue 'create package C 1.0' "[ -e '$tmpdir/testbuild/c-test-1.0.package.tar.gz' ]"
  assertTrue 'create package A 1.1' "[ -e '$tmpdir/testbuild/a-test-1.1.package.tar.gz' ]"
  assertTrue 'create package B 1.0' "[ -e '$tmpdir/testbuild/b-test-1.0.package.tar.gz' ]"
  assertTrue 'create package E 1.0' "[ -e '$tmpdir/testbuild/e-test-1.0.package.tar.gz' ]"
  assertEquals 'builds successfully' "$buildst" "0"

  mkdir $tmpdir/testbuild_provides
  $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/upgrade_provides_new" --destination $tmpdir/testbuild_provides --compression gzip --full > ${OUTPUT}
  buildst=$?
  assertTrue 'create package D 2.0' "[ -e '$tmpdir/testbuild_provides/d-test-2.0.package.tar.gz' ]"
  assertEquals 'builds successfully' "$buildst" "0"
}

testRepo() {
  assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
  $LUET_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/upgrade_provides" \
  --output $tmpdir/testbuild \
  --packages $tmpdir/testbuild \
  --name "test" \
  --descr "Test Repo" \
  --urls $tmpdir/testrootfs \
  --type http

  createst=$?
  assertEquals 'create repo successfully' "$createst" "0"
  assertTrue 'create repository' "[ -e '$tmpdir/testbuild/repository.yaml' ]"

  assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild_provides/repository.yaml' ]"
  $LUET_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/upgrade_provides_new" \
  --output $tmpdir/testbuild_provides \
  --packages $tmpdir/testbuild_provides \
  --name "test" \
  --descr "Test Repo" \
  --urls $tmpdir/testrootfs \
  --type http

  createst=$?
  assertEquals 'create repo successfully' "$createst" "0"
  assertTrue 'create repository' "[ -e '$tmpdir/testbuild_provides/repository.yaml' ]"
}

testConfig() {
    mkdir $tmpdir/testrootfs
    cat <<EOF > $tmpdir/luet.yaml
general:
  debug: ${DEBUG_ENABLE}
system:
  rootfs: $tmpdir/testrootfs
  database_path: "/"
  database_engine: "boltdb"
config_from_host: false
repositories:
   - name: "main"
     type: "disk"
     enable: true
     cached: true
     urls:
       - "../testbuild"
EOF
    $LUET config --config $tmpdir/luet.yaml
    res=$?
    assertEquals 'config test successfully' "$res" "0"
}

testUpgrade() {
    $LUET install --sync-repos -y --config $tmpdir/luet.yaml test/a@1.1 test/e@1.0
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed B' "[ -e '$tmpdir/testrootfs/test5' ]"

    cat <<EOF > $tmpdir/luet.yaml
general:
  debug: ${DEBUG_ENABLE}
system:
  rootfs: $tmpdir/testrootfs
  database_path: "/"
  database_engine: "boltdb"
config_from_host: false
repositories:
   - name: "main2"
     type: "disk"
     enable: true
     cached: true
     urls:
       - "../testbuild_provides"
EOF

    $LUET cleanup --config $tmpdir/luet.yaml
    $LUET repo update --config $tmpdir/luet.yaml
    $LUET config --config $tmpdir/luet.yaml
    res=$?
    assertEquals 'config test successfully' "$res" "0"

    $LUET upgrade -y --config $tmpdir/luet.yaml
    installst=$?
    assertEquals 'upgrade test successfully' "$installst" "0"
    assertTrue 'package uninstalled B' "[ ! -e '$tmpdir/testrootfs/test5' ]"
    assertTrue 'package installed D' "[ -e '$tmpdir/testrootfs/pkgd1' ]"
    assertTrue 'package installed E 1.1' "[ -e '$tmpdir/testrootfs/e-1.1' ]"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2
