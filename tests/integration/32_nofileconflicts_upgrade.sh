#!/bin/bash

export LUET_NOLOCK=true
export LUET_BUILD=luet-build
export LUET=luet

oneTimeSetUp() {
export tmpdir="$(mktemp -d)"
}

oneTimeTearDown() {
    rm -rf "$tmpdir"
}

testBuild() {
    mkdir $tmpdir/testbuild
    $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/nofileconflicts_upgrade" --destination $tmpdir/testbuild --compression gzip --all
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create packages' "[ -e '$tmpdir/testbuild/noconflict-test1-1.0.package.tar.gz' ]"
    assertTrue 'create packages' "[ -e '$tmpdir/testbuild/noconflict-test1-1.1.package.tar.gz' ]"
}

testRepo() {
    assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
    $LUET_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/nofileconflicts_upgrade" \
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
    $LUET install --sync-repos -y --config $tmpdir/luet.yaml test1/noconflict@1.0
    installst=$?
    #assertTrue 'package installed' "[ -e '$tmpdir/testrootfs/c' ]"
}

testUpgrade() {
    out=$($LUET upgrade --sync-repos -y --config $tmpdir/luet.yaml)
    installst=$?
    assertEquals 'install test succeeded' "$installst" "0"
    assertNotContains 'does find conflicts' "$out" \
      "Error: file conflict found: found file test1 conflict between package"

    installed=$($LUET --config $tmpdir/luet.yaml search --installed)
    assertContains 'does upgrade' "$installed" "test1/noconflict-1.1"

}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

