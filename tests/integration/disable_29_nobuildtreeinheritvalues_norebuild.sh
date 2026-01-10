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

testConfig() {
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping

    mkdir $tmpdir/testrootfs
    cat <<EOF > $tmpdir/anise.yaml
general:
  debug: true
system:
  rootfs: $tmpdir/testrootfs
  database_path: "/"
  database_engine: "boltdb"
config_from_host: true
repositories:
   - name: "main"
     type: "docker"
     enable: true
     cached: true
     urls:
       - "${TEST_DOCKER_IMAGE}"
EOF
    $ANISE config --config $tmpdir/anise.yaml
    res=$?
    assertEquals 'config test successfully' "$res" "0"
}

testBuild() {
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping

    # This will be ignored, as no --rebuild is supplied
    cat <<EOF > $tmpdir/default.yaml
extra: "an"
EOF

    mkdir $tmpdir/testbuild
    mkdir $tmpdir/empty
cp -rf  "$ROOT_DIR/tests/fixtures/docker_repo/interpolated" $tmpdir/empty/

    cat <<EOF > $tmpdir/empty/interpolated/definition.yaml
category: "test"
name: "interpolated"
version: "1.1"
foo: "bar"
EOF

    build_output=$($ANISE_BUILD build --pull --tree "$tmpdir/empty" \
    --config $tmpdir/anise.yaml --values $tmpdir/default.yaml --concurrency 1 \
    --from-repositories --destination $tmpdir/testbuild --compression zstd test/c@1.0 test/z test/interpolated)
    buildst=$?
    echo "$build_output"
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create package dep B' "[ -e '$tmpdir/testbuild/b-test-1.0.package.tar.zst' ]"
    assertTrue 'create package' "[ -e '$tmpdir/testbuild/c-test-1.0.package.tar.zst' ]"
    assertTrue 'create package Z' "[ -e '$tmpdir/testbuild/z-test-1.0+2.package.tar.zst' ]"
    assertTrue 'create package interpolated' "[ -e '$tmpdir/testbuild/interpolated-test-1.1.package.tar.zst' ]"
}

testRepo() {
    # Disable tests which require a DOCKER registry
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping

    $ANISE_BUILD create-repo \
    --output "${TEST_DOCKER_IMAGE}-2" \
    --packages $tmpdir/testbuild \
    --name "test" \
    --descr "Test Repo" \
    --urls $tmpdir/testrootfs \
    --tree-compression zstd \
    --tree-filename foo.tar \
    --tree "$tmpdir/empty" --config $tmpdir/anise.yaml --from-repositories \
    --meta-filename repository.meta.tar \
    --meta-compression zstd \
    --type docker --push-images --force-push --debug

    createst=$?
    assertEquals 'create repo successfully' "$createst" "0"
}

testConfigClient() {
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping

    cat <<EOF > $tmpdir/anise-client.yaml
general:
  debug: true
system:
  rootfs: $tmpdir/testrootfs
  database_path: "/"
  database_engine: "boltdb"
config_from_host: true
repositories:
   - name: "main"
     type: "docker"
     enable: true
     cached: true
     urls:
       - "${TEST_DOCKER_IMAGE}-2"
EOF
    $ANISE config --config $tmpdir/anise-client.yaml
    res=$?
    assertEquals 'config test successfully' "$res" "0"
}

testInstall() {
    # Disable tests which require a DOCKER registry
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping

    $ANISE install --sync-repos -y --config $tmpdir/anise-client.yaml test/c@1.0 test/z test/interpolated
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package installed' "[ -e '$tmpdir/testrootfs/c' ]"
    assertTrue 'package Z installed' "[ -e '$tmpdir/testrootfs/z' ]"
    ls -liah $tmpdir/testrootfs/
    assertTrue 'package interpolated installed' "[ -e '$tmpdir/testrootfs/interpolated-bar-an' ]"
}

testReInstall() {
    # Disable tests which require a DOCKER registry
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping

    output=$($ANISE install --sync-repos -y --config $tmpdir/anise-client.yaml  test/c@1.0)
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertContains 'contains warning' "$output" 'No packages to install'
}

testUnInstall() {
    # Disable tests which require a DOCKER registry
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping

    $ANISE uninstall -y --config $tmpdir/anise-client.yaml test/c@1.0
    installst=$?
    assertEquals 'uninstall test successfully' "$installst" "0"
    assertTrue 'package uninstalled' "[ ! -e '$tmpdir/testrootfs/c' ]"
}

testInstallAgain() {
    # Disable tests which require a DOCKER registry
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping

    assertTrue 'package uninstalled' "[ ! -e '$tmpdir/testrootfs/c' ]"
    output=$($ANISE install --sync-repos -y --config $tmpdir/anise-client.yaml test/c@1.0)
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertNotContains 'contains warning' "$output" 'No packages to install'
    assertTrue 'package installed' "[ -e '$tmpdir/testrootfs/c' ]"
    assertTrue 'package in cache' "[ -e '$tmpdir/testrootfs/packages/c-test-1.0.package.tar.zst' ]"
}

testCleanup() {
    [ -z "${TEST_DOCKER_IMAGE:-}" ] && startSkipping

    $ANISE cleanup --config $tmpdir/anise-client.yaml
    installst=$?
    assertEquals 'cleanup test successfully' "$installst" "0"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

