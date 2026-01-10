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
  $ANISE_BUILD tree genidx --only-upper-level -t "$ROOT_DIR/tests/fixtures/docker"
  genidx=$?
  assertEquals 'genidx successfully' "$genidx" "0"

  mkdir $tmpdir/testbuild
  $ANISE_BUILD build --tree "$ROOT_DIR/tests/fixtures/docker" --destination $tmpdir/testbuild --compression gzip --all > ${OUTPUT}
  buildst=$?
  assertEquals 'builds successfully' "$buildst" "0"
  assertTrue 'create package' "[ -e '$tmpdir/testbuild/alpine-seed-1.0.package.tar.gz' ]"
}

testRepo() {
  assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
  $ANISE_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/docker" \
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
  rootfs: /
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
    $ANISE config --config $tmpdir/anise.yaml
    res=$?
    assertEquals 'config test successfully' "$res" "0"
}

# We test the Docker image generated with the current code that doesn't break
# from scratch installations of packages.
testInstall() {
    docker build --rm --no-cache -t anise:test .
    docker rm anise-runtime-test || true
    docker run --name anise-runtime-test \
       -v /tmp:/tmp \
       -v $tmpdir/anise.yaml:/etc/anise/anise.yaml:ro \
       anise:test install --sync-repos -y seed/alpine
    installst=$?
    assertEquals 'install test successfully' "0" "$installst"

    docker commit anise-runtime-test anise-runtime-test-image
    test=$(docker run --rm --entrypoint /bin/sh anise-runtime-test-image -c 'echo "ftw"')
    assertContains 'generated image runs successfully' "$test" "ftw"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

