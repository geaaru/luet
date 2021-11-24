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
      --tree "$ROOT_DIR/tests/fixtures/versioning" \
      --destination $tmpdir/testbuild \
      --compression gzip \
      --all
    buildst=$?
    assertEquals 'builds successfully' "0" "$buildst"

    luet build --config $tmpdir/luet-build.yaml \
      --tree "$ROOT_DIR/tests/fixtures/versioning" \
      --destination $tmpdir/testbuild \
      --compression gzip \
      media-libs/libsndfile
    buildst=$?
    assertEquals 'builds successfully' "0" "$buildst"


    luet build --config $tmpdir/luet-build.yaml \
      --tree "$ROOT_DIR/tests/fixtures/versioning" \
      --destination $tmpdir/testbuild \
      --compression gzip \
      '=dev-libs/libsigc++-2-2.10.1+1'
    buildst=$?
    assertEquals 'builds successfully' "0" "$buildst"
}

testRepo() {
    assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
    luet create-repo \
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
     urls:
       - "$tmpdir/testbuild"
EOF
    luet config --config $tmpdir/luet.yaml
    res=$?
    assertEquals 'config test successfully' "0" "$res"
}

testInstall() {
    luet install --sync-repos -y --config $tmpdir/luet.yaml media-libs/libsndfile
    installst=$?
    assertEquals 'install test successfully' "0" "$installst"
}

testInstall2() {
    luet install --sync-repos -y --config $tmpdir/luet.yaml '=dev-libs/libsigc++-2-2.10.1+1'
    installst=$?
    assertEquals 'install test successfully' "0" "$installst"
}


testCleanup() {
    luet cleanup --config $tmpdir/luet.yaml
    installst=$?
    assertEquals 'install test successfully' "0" "$installst"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2
