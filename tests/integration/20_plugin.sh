#!/bin/bash

export LUET_NOLOCK=true
export PATH=$PATH:$ROOT_DIR/tests/fixtures/plugin


oneTimeSetUp() {
  export tmpdir="$(mktemp -d)"
  export EVENT_FILE=$tmpdir/events.txt
  export PAYLOAD_FILE=$tmpdir/payloads.txt
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
    mkdir $tmpdir/testbuild
    luet build --config $tmpdir/luet-build.yaml \
      --plugin test-foo \
      --tree "$ROOT_DIR/tests/fixtures/templatedfinalizers" \
      --destination $tmpdir/testbuild --compression gzip --all
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertContains 'event file contains corresponding event' "$(cat $EVENT_FILE)" 'package.pre.build'
    assertContains 'event file contains corresponding event' "$(cat $PAYLOAD_FILE)" 'alpine'
}

testRepo() {
    assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
    luet create-repo --config $tmpdir/luet-build.yaml \
      --plugin test-foo \
      --tree "$ROOT_DIR/tests/fixtures/templatedfinalizers" \
      --output $tmpdir/testbuild \
      --packages $tmpdir/testbuild \
      --name "test" \
      --descr "Test Repo" \
      --urls $tmpdir/testrootfs \
      --type disk > /dev/null

    createst=$?
    assertEquals 'create repo successfully' "$createst" "0"
    assertContains  'event file contains corresponding event' "$(cat $EVENT_FILE)" 'repository.pre.build'
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
     urls:
       - "$tmpdir/testbuild"
EOF
    luet config --config $tmpdir/luet.yaml
    res=$?
    assertEquals 'config test successfully' "$res" "0"
}

testInstall() {
    luet --plugin test-foo install --sync-repos -y --config $tmpdir/luet.yaml seed/alpine
    #luet install -y --config $tmpdir/luet.yaml test/c-1.0 > /dev/null
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed' "[ -e '$tmpdir/testrootfs/bin/busybox' ]"
    assertTrue 'finalizer runs' "[ -e '$tmpdir/testrootfs/tmp/foo' ]"
    assertEquals 'finalizer printed used shell' "$(cat $tmpdir/testrootfs/tmp/foo)" 'alpine'
        assertContains  'event file contains corresponding event'  "$(cat $EVENT_FILE)" 'package.install'

}


testCleanup() {
    luet cleanup --config $tmpdir/luet.yaml
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2
