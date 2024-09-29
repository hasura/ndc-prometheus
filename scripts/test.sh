#!/bin/bash
set -eo pipefail

go test -v -coverpkg=./... -race -timeout 3m -coverprofile=coverage.out.tmp ./...
cat coverage.out.tmp | grep -v "main.go" > coverage.out
mkdir -p ./tmp

if [ ! -f ./tmp/ndc-test ]; then
  curl -L https://github.com/hasura/ndc-spec/releases/download/v0.1.6/ndc-test-x86_64-unknown-linux-gnu -o ./tmp/ndc-test
  chmod +x ./tmp/ndc-test
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

docker compose up -d prometheus node-exporter alertmanager ndc-prometheus
http_wait http://localhost:8080/health
http_wait http://localhost:9090/-/healthy

./tmp/ndc-test test --endpoint http://localhost:8080

# go tests
go test -v -coverpkg=./... -race -timeout 3m -coverprofile=coverage.out.tmp ./...
cat coverage.out.tmp | grep -v "main.go" > coverage.out