#!/bin/bash

export ANISE_NOLOCK=true
export ANISE_BUILD=${ANISE_BUILD:-anise-build}
export ANISE=${ANISE:-anise}
export DEBUG_ENABLE=${DEBUG_ENABLE:-false}

export OUTPUT=${OUTPUT:-/dev/null}

if [ -n "${DEBUG}" ] ; then
  set -x
fi
