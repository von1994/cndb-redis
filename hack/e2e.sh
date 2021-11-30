#!/bin/bash

set -e

readonly REPO_PATH=github.com/von1994/cndb-redis

if [[ -z ${STORAGECLASSNAME} ]]; then
    echo "env STORAGECLASSNAME not set"
    exit 1
fi

if [[ -z ${KUBECONFIG} ]]; then
    echo "env KUBECONFIG not set"
#    exit 1
fi

if [[ -z ${E2E_GINKGO_SKIP} ]]; then
    export E2E_GINKGO_SKIP=""
fi

echo "run e2e tests..."
echo "cd ${GOPATH}/src/${REPO_PATH} && ginkgo -v --skip=$E2E_GINKGO_SKIP test/e2e/RedisSentinel"
cd ${GOPATH}/src/${REPO_PATH} && ginkgo -v --skip=${E2E_GINKGO_SKIP} test/e2e/RedisSentinel

