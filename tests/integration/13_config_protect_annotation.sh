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
    $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/config_protect_annotation" --destination $tmpdir/testbuild --compression gzip test/a
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create package' "[ -e '$tmpdir/testbuild/a-test-1.0.package.tar.gz' ]"
}

testRepo() {
    assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
    $LUET_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/config_protect_annotation" \
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

    mkdir $tmpdir/config.protect.d

    cat <<EOF > $tmpdir/config.protect.d/conf1.yml
name: "protect1"
dirs:
- /etc/
EOF

    cat <<EOF > $tmpdir/luet.yaml
general:
  debug: true
system:
  rootfs: $tmpdir/testrootfs
  database_path: "/"
  database_engine: "boltdb"
config_protect_confdir:
    - $tmpdir/config.protect.d
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
    # Simulate previous installation
    mkdir $tmpdir/testrootfs/opt/etc -p
    echo "fakeconf" > $tmpdir/testrootfs/opt/etc/conf

    $LUET install --sync-repos -y --config $tmpdir/luet.yaml test/a
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"


    # Simulate config protect
    assertTrue 'package A installed' "[ -e '$tmpdir/testrootfs/c' ]"
    assertTrue 'config protect created' "[ -e '$tmpdir/testrootfs/opt/etc/._cfg0001_conf' ]"
}


testUnInstall() {
    $LUET uninstall -y --keep-protected-files --config $tmpdir/luet.yaml test/a
    installst=$?
    assertEquals 'uninstall test successfully' "$installst" "0"
    assertTrue 'package uninstalled' "[ ! -e '$tmpdir/testrootfs/c' ]"
    # TODO: we need remove it or not??
    assertTrue 'config protect created' "[ -e '$tmpdir/testrootfs/opt/etc/._cfg0001_conf' ]"
    assertTrue 'config protect maintains the protected files' "[ -e '$tmpdir/testrootfs/opt/etc/conf' ]"
}


testCleanup() {
    $LUET cleanup --config $tmpdir/luet.yaml
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed' "[ ! -e '$tmpdir/testrootfs/packages/a-test-1.0.package.tar.gz' ]"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

