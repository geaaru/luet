#!/bin/bash

testsourcedir=$(dirname "${BASH_SOURCE[0]}")
source ${testsourcedir}/_common.sh

oneTimeSetUp() {
    export tmpdir="$(mktemp -d)"
    docker images --filter='reference=anise/cache' --format='{{.Repository}}:{{.Tag}}' | xargs -r docker rmi
}

oneTimeTearDown() {
    rm -rf "$tmpdir"
    docker images --filter='reference=anise/cache' --format='{{.Repository}}:{{.Tag}}' | xargs -r docker rmi
}

testBuild() {
    # Disable tests which require a DOCKER registry
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping
    cat <<EOF > $tmpdir/default.yaml
extra: "bar"
foo: "baz"
EOF

    $ANISE_BUILD tree genidx --only-upper-level -t "$ROOT_DIR/tests/fixtures/docker_repo"
    genidx=$?
    assertEquals 'genidx successfully' "$genidx" "0"

    mkdir $tmpdir/testbuild
    $ANISE_BUILD build --tree "$ROOT_DIR/tests/fixtures/docker_repo" \
               --destination $tmpdir/testbuild --concurrency 1 \
               --image-repository "${TEST_DOCKER_IMAGE}-cache" --push \
               --compression zstd --values $tmpdir/default.yaml \
               test/c@1.0 test/z test/interpolated #> /dev/null
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create package dep B' "[ -e '$tmpdir/testbuild/b-test-1.0.package.tar.zst' ]"
    assertTrue 'create package' "[ -e '$tmpdir/testbuild/c-test-1.0.package.tar.zst' ]"
    assertTrue 'create package z' "[ -e '$tmpdir/testbuild/z-test-1.0+2.package.tar.zst' ]"
    assertTrue 'create package interpolated' "[ -e '$tmpdir/testbuild/interpolated-test-1.0+2.package.tar.zst' ]"
}

testRepo() {
    # Disable tests which require a DOCKER registry
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping

    createres=$($ANISE_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/docker_repo" \
    --output "${TEST_DOCKER_IMAGE}" \
    --packages $tmpdir/testbuild \
    --name "test" \
    --descr "Test Repo" \
    --urls $tmpdir/testrootfs \
    --tree-compression zstd \
    --tree-filename foo.tar.zst \
    --type docker --push-images --force-push)

    createst=$?
    assertEquals 'create repo successfully' "$createst" "0"
    assertContains 'contains image push' "$createres" 'Pushed image: quay.io/mocaccinoos/integration-test:z-test-1.0-2'
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
repos_confdir:
  - "$tmpdir/etc/anise/repos.conf.d"
config_protect_confdir:
  - "$tmpdir/etc/anise/config.protect.d"
subsets_defdir:
  - "$tmpdir/etc/anise/subsets.conf.d"
repositories:
   - name: "main"
     type: "docker"
     cached: true
     enable: true
     urls:
       - "${TEST_DOCKER_IMAGE}"
EOF
    $ANISE config --config $tmpdir/anise.yaml
    res=$?
    assertEquals 'config test successfully' "$res" "0"
}

testInstall() {
    # Disable tests which require a DOCKER registry
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping

    $ANISE install -y --sync-repos --config $tmpdir/anise.yaml test/c@1.0 test/z test/interpolated
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed' "[ -e '$tmpdir/testrootfs/c' ]"
    assertTrue 'package Z installed' "[ -e '$tmpdir/testrootfs/z' ]"
    assertTrue 'package interpolated installed' "[ -e '$tmpdir/testrootfs/interpolated-baz-bar' ]"
}

testReInstall() {
    # Disable tests which require a DOCKER registry
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping

    output=$($ANISE install -y --config $tmpdir/anise.yaml  test/c@1.0)
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertContains 'contains warning' "$output" 'No packages to install'
}

testUnInstall() {
    # Disable tests which require a DOCKER registry
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping

    $ANISE uninstall -y --config $tmpdir/anise.yaml test/c@1.0 test/z
    installst=$?
    assertEquals 'uninstall test successfully' "$installst" "0"
    assertTrue 'package uninstalled' "[ ! -e '$tmpdir/testrootfs/c' ]"
    assertTrue 'package Z uninstalled' "[ ! -e '$tmpdir/testrootfs/z' ]"
}

testInstallAgain() {
    # Disable tests which require a DOCKER registry
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping

    assertTrue 'package uninstalled' "[ ! -e '$tmpdir/testrootfs/c' ]"
    output=$($ANISE install -y --config $tmpdir/anise.yaml test/c@1.0 test/z)
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertNotContains 'contains warning' "$output" 'No packages to install'
    assertTrue 'package installed' "[ -e '$tmpdir/testrootfs/c' ]"
    assertTrue 'package Z installed' "[ -e '$tmpdir/testrootfs/z' ]"
    assertTrue 'package Z in cache' "[ -e '$tmpdir/testrootfs/packages/z-test-1.0+2.package.tar.zst' ]"
    assertTrue 'package in cache' "[ -e '$tmpdir/testrootfs/packages/c-test-1.0.package.tar.zst' ]"
}

testCleanup() {
    $ANISE cleanup --config $tmpdir/anise.yaml --purge-repos
    installst=$?
    assertEquals 'cleanup test successfully' "$installst" "0"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

