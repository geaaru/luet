#!/bin/bash

testsourcedir=$(dirname "${BASH_SOURCE[0]}")
source ${testsourcedir}/_common.sh
TEST_PORT="${TEST_PORT:-9090}"

oneTimeSetUp() {
  export tmpdir="$(mktemp -d)"
}

oneTimeTearDown() {
  rm -rf "$tmpdir"
  kill '%1' || true
}

testBuild() {
  $ANISE_BUILD tree genidx --only-upper-level -t "$ROOT_DIR/tests/fixtures/buildableseed"
  genidx=$?
  assertEquals 'genidx successfully' "$genidx" "0"

  mkdir $tmpdir/testbuild
  $ANISE_BUILD build --tree "$ROOT_DIR/tests/fixtures/buildableseed" --destination $tmpdir/testbuild --compression gzip test/c
  buildst=$?
  assertEquals 'builds successfully' "$buildst" "0"
  assertTrue 'create package dep B' "[ -e '$tmpdir/testbuild/b-test-1.0.package.tar.gz' ]"
  assertTrue 'create package' "[ -e '$tmpdir/testbuild/c-test-1.0.package.tar.gz' ]"
}

testRepo() {
  assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
  $ANISE_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/buildableseed" \
  --output $tmpdir/testbuild \
  --packages $tmpdir/testbuild \
  --name "test" \
  --descr "Test Repo" \
  --urls $tmpdir/testrootfs \
  --type disk > ${OUTPUT}

  createst=$?
  assertEquals 'create repo successfully' "$createst" "0"
  assertTrue 'create repository' "[ -e '$tmpdir/testbuild/repository.yaml' ]"
  $ANISE_BUILD serve-repo --dir $tmpdir/testbuild --port $TEST_PORT &
}

testConfig() {
    mkdir $tmpdir/testrootfs
    cat <<EOF > $tmpdir/anise.yaml
general:
  debug: true
system:
  rootfs: $tmpdir/testrootfs
  database_engine: "memory"
config_from_host: true
repositories:
   - name: "main"
     type: "http"
     enable: true
     cached: true
     urls:
       - "http://127.0.0.1:$TEST_PORT"
EOF
    $ANISE config --config $tmpdir/anise.yaml
    res=$?
    assertEquals 'config test successfully' "$res" "0"
}

testBuildWithNoTree() {
    mkdir $tmpdir/testbuild2
    mkdir $tmpdir/emptytree
    $ANISE_BUILD build --from-repositories --tree $tmpdir/emptytree --config $tmpdir/anise.yaml test/c --destination $tmpdir/testbuild2
    buildst=$?
    assertEquals 'build test successfully' "$buildst" "0"
    assertTrue 'create package' "[ -e '$tmpdir/testbuild2/c-test-1.0.package.tar' ]"
}

testRepo2() {
    assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild2/repository.yaml' ]"
    $ANISE_BUILD create-repo --config $tmpdir/anise.yaml --from-repositories --tree $tmpdir/emptytree \
    --output $tmpdir/testbuild2 \
    --packages $tmpdir/testbuild2 \
    --name "test" \
    --descr "Test Repo" \
    --urls $tmpdir/testrootfs \
    --type disk

    createst=$?
    assertEquals 'create repo successfully' "$createst" "0"
    assertTrue 'create repository' "[ -e '$tmpdir/testbuild2/repository.yaml' ]"
}

testCleanup() {
    $ANISE cleanup --config $tmpdir/anise.yaml
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package cleaned' "[ ! -e '$tmpdir/testrootfs/packages/c-test-1.0.package.tar.gz' ]"
}

testInstall2() {

    cat <<EOF > $tmpdir/anise2.yaml
general:
  debug: true
system:
  rootfs: $tmpdir/testrootfs
  database_engine: "memory"
config_from_host: true
repositories:
   - name: "main"
     type: "disk"
     enable: true
     cached: true
     urls:
       - "$tmpdir/testbuild2"
EOF
    $ANISE install --sync-repos -y --config $tmpdir/anise2.yaml --system-target $tmpdir/foo test/c
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'db not created' "[ ! -e '$tmpdir/foo/var/cache/anise/anise.db' ]"
    assertTrue 'package installed' "[ -e '$tmpdir/foo/c' ]"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

