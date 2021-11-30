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

if  [[ -z ${TEST_REDIS_MODE} ]]; then
    export TEST_REDIS_MODE="all"
fi

if  [[ ${TEST_REDIS_MODE} == "all" ]]; then
    echo "cd ${GOPATH}/src/${REPO_PATH} && ginkgo -v --skip=$E2E_GINKGO_SKIP test/e2e/redisstandalone"
    cd ${GOPATH}/src/${REPO_PATH} && ginkgo -v --skip=${E2E_GINKGO_SKIP} test/e2e/redisstandalone
    echo "cd ${GOPATH}/src/${REPO_PATH} && ginkgo -v --skip=$E2E_GINKGO_SKIP test/e2e/redissentinel"
    cd ${GOPATH}/src/${REPO_PATH} && ginkgo -v --skip=${E2E_GINKGO_SKIP} test/e2e/redissentinel
elif [[ ${TEST_REDIS_MODE} == "sentinel" ]]; then
    echo "cd ${GOPATH}/src/${REPO_PATH} && ginkgo -v --skip=$E2E_GINKGO_SKIP test/e2e/redissentinel"
    cd ${GOPATH}/src/${REPO_PATH} && ginkgo -v --skip=${E2E_GINKGO_SKIP} test/e2e/redissentinel
elif [[ ${TEST_REDIS_MODE} == "standalone" ]]; then
    echo "cd ${GOPATH}/src/${REPO_PATH} && ginkgo -v --skip=$E2E_GINKGO_SKIP test/e2e/redisstandalone"
    cd ${GOPATH}/src/${REPO_PATH} && ginkgo -v --skip=${E2E_GINKGO_SKIP} test/e2e/redisstandalone
fi


