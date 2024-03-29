#!/bin/bash

testsourcedir=$(dirname "${BASH_SOURCE[0]}")
source ${testsourcedir}/_common.sh

oneTimeSetUp() {
export tmpdir="$(mktemp -d)"
  cat <<EOF > $tmpdir/luet-build.yaml
general:
  debug: true
logging:
  enable_emoji: false
  color: false
system:
  rootfs: $tmpdir/testrootfs
  database_path: "/"
  database_engine: "memory"
config_from_host: true
repos_confdir:
  - "$tmpdir/etc/luet/repos.conf.d"
EOF


}

oneTimeTearDown() {
   rm -rf "$tmpdir"
}

testBuild() {
  $LUET_BUILD tree genidx --only-upper-level -t "$ROOT_DIR/tests/fixtures/versioning"
  genidx=$?
  assertEquals 'genidx successfully' "$genidx" "0"

  mkdir $tmpdir/testbuild
  $LUET_BUILD build --config $tmpdir/luet-build.yaml \
    --tree "$ROOT_DIR/tests/fixtures/versioning" \
    --destination $tmpdir/testbuild \
    --compression gzip \
    --all > ${OUTPUT}
  buildst=$?
  assertEquals 'builds successfully' "0" "$buildst"

  $LUET_BUILD build --config $tmpdir/luet-build.yaml \
    --tree "$ROOT_DIR/tests/fixtures/versioning" \
    --destination $tmpdir/testbuild \
    --compression gzip \
    media-libs/libsndfile
  buildst=$?
  assertEquals 'builds successfully' "0" "$buildst"

  $LUET_BUILD build --config $tmpdir/luet-build.yaml \
    --tree "$ROOT_DIR/tests/fixtures/versioning" \
    --destination $tmpdir/testbuild \
    --compression gzip \
    '=dev-libs/libsigc++-2-2.10.1+1'
  buildst=$?
  assertEquals 'builds successfully' "0" "$buildst"
}

testRepo() {
  assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
  $LUET_BUILD create-repo \
    --config $tmpdir/luet-build.yaml \
    --tree "$ROOT_DIR/tests/fixtures/versioning" \
    --output $tmpdir/testbuild \
    --packages $tmpdir/testbuild \
    --name "test" \
    --descr "Test Repo" \
    --urls $tmpdir/testrootfs \
    --type disk

  createst=$?
  assertEquals 'create repo successfully' "0" "$createst"
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
repos_confdir:
  - "$tmpdir/etc/luet/repos.conf.d"
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
    assertEquals 'config test successfully' "0" "$res"
}

testInstall() {
    $LUET install --sync-repos -y --config $tmpdir/luet.yaml media-libs/libsndfile
    installst=$?
    assertEquals 'install test successfully' "0" "$installst"
}

testInstall2() {
    $LUET install --sync-repos -y --config $tmpdir/luet.yaml '=dev-libs/libsigc++-2-2.10.1+1'
    installst=$?
    assertEquals 'install test successfully' "0" "$installst"
}


testCleanup() {
    $LUET cleanup --config $tmpdir/luet.yaml
    installst=$?
    assertEquals 'install test successfully' "0" "$installst"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2
