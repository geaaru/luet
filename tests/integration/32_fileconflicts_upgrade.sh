#!/bin/bash

export LUET_NOLOCK=true

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
    mkdir $tmpdir/testbuild
    luet build --config $tmpdir/luet-build.yaml \
      --tree "$ROOT_DIR/tests/fixtures/fileconflicts_upgrade" \
      --destination $tmpdir/testbuild --compression gzip --all
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create packages' "[ -e '$tmpdir/testbuild/conflict-test1-1.0.package.tar.gz' ]"
    assertTrue 'create packages' "[ -e '$tmpdir/testbuild/conflict-test2-1.0.package.tar.gz' ]"
}

testRepo() {
    assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
    luet create-repo --config $tmpdir/luet-build.yaml \
    --tree "$ROOT_DIR/tests/fixtures/fileconflicts_upgrade" \
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
logging:
  enable_emoji: false
  color: false
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
    luet install -y --force --config $tmpdir/luet.yaml test1/conflict@1.0 test2/conflict@1.0
    #luet install -y --config $tmpdir/luet.yaml test/c@1.0 > /dev/null
    installst=$?
    assertEquals 'install test succeded' "$installst" "0"
    #assertTrue 'package installed' "[ -e '$tmpdir/testrootfs/c' ]"
}

testUpgrade() {
    out=$(luet upgrade -y --config $tmpdir/luet.yaml)
    installst=$?
    assertEquals 'install test succeeded' "$installst" "1"
    assertContains 'does find conflicts' "$out" \
      "Error: file conflict found: file test1 conflict between package"

    luet upgrade -y --config $tmpdir/luet.yaml --force
    #luet install -y --config $tmpdir/luet.yaml test/c@1.0 > /dev/null
    installst=$?
    assertEquals 'install test succeeded' "$installst" "0"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

