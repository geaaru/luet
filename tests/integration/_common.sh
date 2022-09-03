#!/bin/bash

export LUET_NOLOCK=true
export LUET_BUILD=${LUET_BUILD:-luet-build}
export LUET=${LUET:-luet}

export OUTPUT=${OUTPUT:-/dev/null}

if [ -n "${DEBUG}" ] ; then
  set -x
fi
