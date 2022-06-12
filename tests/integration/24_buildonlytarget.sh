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
    mkdir $tmpdir/testbuild1
    $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/simple_dep" --destination $tmpdir/testbuild1 test/c
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create package A 1.2' "[ -e '$tmpdir/testbuild1/a-test-1.2.package.tar' ]"
    assertTrue 'create package C 1.0' "[ -e '$tmpdir/testbuild1/c-test-1.0.package.tar' ]"
}

testBuildOnlyTarget() {
    mkdir $tmpdir/testbuild2
    $LUET_BUILD build --tree "$ROOT_DIR/tests/fixtures/simple_dep" --destination $tmpdir/testbuild2 --only-target-package test/c
    buildst=$?
    assertEquals 'builds successfully' "$buildst" "0"
    assertTrue 'create package A 1.2' "[ ! -e '$tmpdir/testbuild2/a-test-1.2.package.tar' ]"
    assertTrue 'create package C 1.0' "[ -e '$tmpdir/testbuild2/c-test-1.0.package.tar' ]"
}

# Load shUnit2.
. "$ROOT_DIR/tests/integration/shunit2"/shunit2

