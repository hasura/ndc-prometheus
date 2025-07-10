#!/bin/bash
set -eo pipefail

trap 'docker compose down' EXIT

mkdir -p ./tmp

NDC_TEST_VERSION=v0.2.4
NDC_TEST_PATH=./tmp/ndc-test

if [ ! -f "$NDC_TEST_PATH" ]; then
  if [ "$(uname -m)" == "arm64" ]; then
    curl -L https://github.com/hasura/ndc-spec/releases/download/$NDC_TEST_VERSION/ndc-test-aarch64-apple-darwin -o $NDC_TEST_PATH
  elif [ $(uname) == "Darwin" ]; then
    curl -L https://github.com/hasura/ndc-spec/releases/download/$NDC_TEST_VERSION/ndc-test-x86_64-apple-darwin -o $NDC_TEST_PATH
  else
    curl -L https://github.com/hasura/ndc-spec/releases/download/$NDC_TEST_VERSION/ndc-test-x86_64-unknown-linux-gnu -o $NDC_TEST_PATH
  fi

  chmod +x $NDC_TEST_PATH
fi


http_wait() {
  printf "$1:\t "
  for i in {1..120};
  do
    local code="$(curl -s -o /dev/null -m 2 -w '%{http_code}' $1)"
    if [ "$code" != "200" ]; then
      printf "."
      sleep 1
    else
      printf "\r\033[K$1:\t ${GREEN}OK${NC}\n"
      return 0
    fi
  done
  printf "\n${RED}ERROR${NC}: cannot connect to $1.\n"
  exit 1
}

docker compose -f ./compose.base.yaml up -d prometheus node-exporter alertmanager ndc-prometheus
http_wait http://localhost:8080/health
http_wait http://admin:test@localhost:9090/-/healthy

$NDC_TEST_PATH test --endpoint http://localhost:8080

# go tests
go test -v -coverpkg=./connector/... -race -timeout 3m -coverprofile=coverage.out ./...
