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

  # Ensure thet repos_confdir is empty to avoid reading
  # repositories availables on host.

    mkdir $tmpdir/repos
    cat <<EOF > $tmpdir/anise-build.yaml
general:
  debug: true
  database_path: "/"
  database_engine: "boltdb"
config_from_host: true
finalizer_envs:
  BUILD_ISO: "1"
repos_confdir:
  - "$tmpdir/repos"
EOF

  $ANISE_BUILD tree genidx --only-upper-level -t "$ROOT_DIR/tests/fixtures/finalizers_envs/"
  genidx=$?
  assertEquals 'genidx successfully' "$genidx" "0"

  mkdir $tmpdir/testbuild
  ${ANISE_BUILD} build --config $tmpdir/anise-build.yaml --tree "$ROOT_DIR/tests/fixtures/finalizers_envs" --destination $tmpdir/testbuild --compression gzip --all > ${OUTPUT}
  buildst=$?
  assertEquals 'builds successfully' "$buildst" "0"
  assertTrue 'create package' "[ -e '$tmpdir/testbuild/alpine-finalizer-envs-seed-1.0.package.tar.gz' ]"
}

testRepo() {
  assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
  ${ANISE_BUILD} create-repo --tree "$ROOT_DIR/tests/fixtures/finalizers_envs" \
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
finalizer_envs:
  - key: "BUILD_ISO"
    value: "1"

repos_confdir:
  - "$tmpdir/repos"

repositories:
   - name: "main"
     type: "disk"
     enable: true
     cached: true
     urls:
       - "$tmpdir/testbuild"
EOF
    ${ANISE} config --config $tmpdir/anise.yaml
    res=$?
    assertEquals 'config test successfully' "$res" "0"
}

testInstall() {
  $ANISE install --sync-repos -y --finalizer-env "CLI_ENV=1" --config $tmpdir/anise.yaml seed/alpine-finalizer-envs@1.0
  installst=$?
  assertEquals 'install test successfully' "$installst" "0"
  assertTrue 'package installed' "[ -e '$tmpdir/testrootfs/bin/busybox' ]"
  assertTrue 'finalizer does not run' "[ -e '$tmpdir/testrootfs/tmp/foo' ]"
  assertTrue 'finalizer env var is not present' "[ ! -e '$tmpdir/testrootfs/tmp/foo2' ]"
  assertTrue 'finalizer env var cli is not present' "[ ! -e '$tmpdir/testrootfs/tmp/foo3' ]"
}


testCleanup() {
  ${ANISE} cleanup --config $tmpdir/anise.yaml
  installst=$?
  assertEquals 'install test successfully' "$installst" "0"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2
