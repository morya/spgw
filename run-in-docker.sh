#!/bin/bash

set -x
set -o errexit
set -o nounset

image=sms-spgw:latest

export GOOS=linux
export GOARCH=amd64
export GOPROXY=https://goproxy.cn

swag init --propertyStrategy pascalcase
go build -o app

docker build -t $image -f Dockerfile-dev .

docker compose down; docker compose up -d # --build

rm -rf app