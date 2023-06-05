#!/bin/bash

testsourcedir=$(dirname "${BASH_SOURCE[0]}")
source ${testsourcedir}/_common.sh

oneTimeSetUp() {
    export tmpdir="$(mktemp -d)"
    docker images --filter='reference=luet/cache' --format='{{.Repository}}:{{.Tag}}' | xargs -r docker rmi
}

oneTimeTearDown() {
    rm -rf "$tmpdir"
    docker images --filter='reference=luet/cache' --format='{{.Repository}}:{{.Tag}}' | xargs -r docker rmi
}

testBuild() {
    cat <<EOF > $tmpdir/default.yaml
bb: "ttt"
EOF

  $LUET_BUILD tree genidx --only-upper-level -t "$ROOT_DIR/tests/fixtures/build_values"
  genidx=$?
  assertEquals 'genidx successfully' "$genidx" "0"

  mkdir $tmpdir/testbuild
  $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/build_values" --values $tmpdir/default.yaml --destination $tmpdir/testbuild --compression gzip  distro/a distro/b test/foo distro/c
  buildst=$?
  assertEquals 'builds successfully' "$buildst" "0"
  assertTrue 'create package B' "[ -e '$tmpdir/testbuild/b-distro-0.3.package.tar.gz' ]"
  assertTrue 'create package A' "[ -e '$tmpdir/testbuild/a-distro-0.1.package.tar.gz' ]"
  assertTrue 'create package C' "[ -e '$tmpdir/testbuild/c-distro-0.3.package.tar.gz' ]"
  assertTrue 'create package foo' "[ -e '$tmpdir/testbuild/foo-test-1.1.package.tar.gz' ]"
}

testRepo() {
  assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild/repository.yaml' ]"
  $LUET_BUILD create-repo --tree "$ROOT_DIR/tests/fixtures/build_values" \
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
    cat <<EOF > $tmpdir/luet.yaml
general:
  debug: true
system:
  rootfs: $tmpdir/testrootfs
  database_engine: "memory"
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

testBuildWithNoTree() {
    mkdir $tmpdir/testbuild2
    mkdir $tmpdir/emptytree
    $LUET_BUILD build --from-repositories --tree $tmpdir/emptytree --config $tmpdir/luet.yaml distro/c --destination $tmpdir/testbuild2 --compression gzip distro/a distro/b test/foo distro/c
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create package B' "[ -e '$tmpdir/testbuild2/b-distro-0.3.package.tar.gz' ]"
    assertTrue 'create package A' "[ -e '$tmpdir/testbuild2/a-distro-0.1.package.tar.gz' ]"
    assertTrue 'create package C' "[ -e '$tmpdir/testbuild2/c-distro-0.3.package.tar.gz' ]"
    assertTrue 'create package foo' "[ -e '$tmpdir/testbuild2/foo-test-1.1.package.tar.gz' ]"
}

testRepo2() {
    assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild2/repository.yaml' ]"
    $LUET_BUILD create-repo --config $tmpdir/luet.yaml --from-repositories --tree $tmpdir/emptytree \
    --output $tmpdir/testbuild2 \
    --packages $tmpdir/testbuild2 \
    --name "test" \
    --descr "Test Repo" \
    --urls $tmpdir/testrootfs \
    --type disk

    createst=$?
    assertEquals 'create repo successfully' "$createst" "0"
    assertTrue 'create repository' "[ -e '$tmpdir/testbuild2/repository.yaml' ]"
}

testCleanup() {
    $LUET cleanup --config $tmpdir/luet.yaml
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"
    assertTrue 'package cleaned' "[ ! -e '$tmpdir/testrootfs/packages/c-test-1.0.package.tar.gz' ]"
}

testInstall2() {

    cat <<EOF > $tmpdir/luet2.yaml
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
       - "$tmpdir/testbuild2"
EOF
    $LUET install --sync-repos -y --config $tmpdir/luet2.yaml distro/a
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    assertTrue 'package installed A' "[ -e '$tmpdir/testrootfs/a' ]"
    # Build time can interpolate on fields which aren't package properties.
    assertTrue 'extra field on A' "[ -e '$tmpdir/testrootfs/build-extra-baz' ]"
    assertTrue 'package installed A interpolated with values' "[ -e '$tmpdir/testrootfs/a-ttt' ]"
    # Finalizers can interpolate only on package field. No extra fields are allowed at this time.
    assertTrue 'finalizer executed on A' "[ -e '$tmpdir/testrootfs/finalize-a' ]"

    installed=$($LUET --config $tmpdir/luet2.yaml search --installed .)
    searchst=$?
    assertEquals 'search exists successfully' "$searchst" "0"

    assertContains 'contains distro/a-0.1' "$installed" 'distro/a-0.1'

    $LUET uninstall -y --config $tmpdir/luet2.yaml distro/a
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    # We do the same check for the others
    $LUET install -y --sync-repos --config $tmpdir/luet2.yaml distro/b
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    assertTrue 'package installed B' "[ -e '$tmpdir/testrootfs/b' ]"
    assertTrue 'package installed B interpolated with values' "[ -e '$tmpdir/testrootfs/b-ttt' ]"
    assertTrue 'extra field on B' "[ -e '$tmpdir/testrootfs/build-extra-f' ]"
    assertTrue 'finalizer executed on B' "[ -e '$tmpdir/testrootfs/finalize-b' ]"

    installed=$($LUET --config $tmpdir/luet2.yaml search --installed .)
    searchst=$?
    assertEquals 'search exists successfully' "$searchst" "0"

    assertContains 'contains distro/b-0.3' "$installed" 'distro/b-0.3'

    $LUET uninstall -y --config $tmpdir/luet2.yaml distro/b
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    $LUET install --sync-repos -y --config $tmpdir/luet2.yaml distro/c
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    assertTrue 'package installed C' "[ -e '$tmpdir/testrootfs/c' ]"
    assertTrue 'extra field on C' "[ -e '$tmpdir/testrootfs/build-extra-bar' ]"
    assertTrue 'package installed C interpolated with values' "[ -e '$tmpdir/testrootfs/c-ttt' ]"
    assertTrue 'finalizer executed on C' "[ -e '$tmpdir/testrootfs/finalize-c' ]"

    installed=$($LUET --config $tmpdir/luet2.yaml search --installed .)
    searchst=$?
    assertEquals 'search exists successfully' "$searchst" "0"

    assertContains 'contains distro/c-0.3' "$installed" 'distro/c-0.3'

    $LUET uninstall -y --config $tmpdir/luet2.yaml distro/c
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    $LUET install --sync-repos -y --config $tmpdir/luet2.yaml test/foo
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    assertTrue 'package installed foo' "[ -e '$tmpdir/testrootfs/foo' ]"
    assertTrue 'package installed foo interpolated with values' "[ -e '$tmpdir/testrootfs/foo-ttt' ]"
}


testBuildWithNoTree3() {
    cat <<EOF > $tmpdir/default.yaml
bb: "newinterpolation"
foo: "sq"
EOF
    mkdir $tmpdir/testbuild3
    mkdir $tmpdir/emptytree
    $LUET_BUILD build --from-repositories --values $tmpdir/default.yaml --tree $tmpdir/emptytree --config $tmpdir/luet.yaml distro/c --destination $tmpdir/testbuild3 --compression gzip distro/a distro/b test/foo
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create package B' "[ -e '$tmpdir/testbuild3/b-distro-0.3.package.tar.gz' ]"
    assertTrue 'create package A' "[ -e '$tmpdir/testbuild3/a-distro-0.1.package.tar.gz' ]"
    assertTrue 'create package C' "[ -e '$tmpdir/testbuild3/c-distro-0.3.package.tar.gz' ]"
    assertTrue 'create package foo' "[ -e '$tmpdir/testbuild3/foo-test-1.1.package.tar.gz' ]"
}

testRepo3() {
    assertTrue 'no repository' "[ ! -e '$tmpdir/testbuild3/repository.yaml' ]"
    $LUET_BUILD create-repo --config $tmpdir/luet.yaml --from-repositories --tree $tmpdir/emptytree \
    --output $tmpdir/testbuild3 \
    --packages $tmpdir/testbuild3 \
    --name "test" \
    --descr "Test Repo" \
    --urls $tmpdir/testrootfs \
    --type disk

    createst=$?
    assertEquals 'create repo successfully' "$createst" "0"
    assertTrue 'create repository' "[ -e '$tmpdir/testbuild3/repository.yaml' ]"
}

testInstall3() {
    mkdir $tmpdir/testrootfs3

    cat <<EOF > $tmpdir/luet2.yaml
general:
  debug: true
system:
  rootfs: $tmpdir/testrootfs3
  database_path: "/"
  database_engine: "boltdb"
config_from_host: true
repositories:
   - name: "main"
     type: "disk"
     enable: true
     cached: true
     urls:
       - "$tmpdir/testbuild3"
EOF
    $LUET install --sync-repos -y --config $tmpdir/luet2.yaml distro/a
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    assertTrue 'package installed A' "[ -e '$tmpdir/testrootfs3/a' ]"
    # Build time can interpolate on fields which aren't package properties.
    assertTrue 'extra field on A' "[ -e '$tmpdir/testrootfs3/build-extra-sq' ]"
    assertTrue 'package installed A interpolated with values' "[ -e '$tmpdir/testrootfs3/a-newinterpolation' ]"
    # Finalizers can interpolate only on package field. No extra fields are allowed at this time.
    assertTrue 'finalizer executed on A' "[ -e '$tmpdir/testrootfs3/finalize-a' ]"
    installed=$($LUET --config $tmpdir/luet2.yaml search --installed .)
    searchst=$?
    assertEquals 'search exists successfully' "$searchst" "0"

    assertContains 'contains distro/a-0.1' "$installed" 'distro/a-0.1'

    $LUET uninstall -y --config $tmpdir/luet2.yaml distro/a
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    # We do the same check for the others
    $LUET install -y --sync-repos --config $tmpdir/luet2.yaml distro/b
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    assertTrue 'package installed B' "[ -e '$tmpdir/testrootfs3/b' ]"
    assertTrue 'package installed B interpolated with values' "[ -e '$tmpdir/testrootfs3/b-newinterpolation' ]"
    assertTrue 'extra field on B' "[ -e '$tmpdir/testrootfs3/build-extra-sq' ]"
    assertTrue 'finalizer executed on B' "[ -e '$tmpdir/testrootfs3/finalize-b' ]"
    installed=$($LUET --config $tmpdir/luet2.yaml search --installed .)
    searchst=$?
    assertEquals 'search exists successfully' "$searchst" "0"

    assertContains 'contains distro/b-0.3' "$installed" 'distro/b-0.3'

    $LUET uninstall -y --config $tmpdir/luet2.yaml distro/b
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    $LUET install --sync-repos -y --config $tmpdir/luet2.yaml distro/c
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    assertTrue 'package installed C' "[ -e '$tmpdir/testrootfs3/c' ]"
    assertTrue 'extra field on C' "[ -e '$tmpdir/testrootfs3/build-extra-sq' ]"
    assertTrue 'package installed C interpolated with values' "[ -e '$tmpdir/testrootfs3/c-newinterpolation' ]"
    assertTrue 'finalizer executed on C' "[ -e '$tmpdir/testrootfs3/finalize-c' ]"

    installed=$($LUET --config $tmpdir/luet2.yaml search --installed .)
    searchst=$?
    assertEquals 'search exists successfully' "$searchst" "0"

    assertContains 'contains distro/c-0.3' "$installed" 'distro/c-0.3'

    $LUET uninstall -y --config $tmpdir/luet2.yaml distro/c
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    $LUET install --sync-repos -y --config $tmpdir/luet2.yaml test/foo
    installst=$?
    assertEquals 'install test successfully' "$installst" "0"

    assertTrue 'package installed foo' "[ -e '$tmpdir/testrootfs3/foo' ]"
    assertTrue 'package installed foo interpolated with values' "[ -e '$tmpdir/testrootfs3/foo-newinterpolation' ]"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2
