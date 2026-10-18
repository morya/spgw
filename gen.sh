#!/bin/bash

set -e

cd $(dirname $0)

go run scripts/protofmt.go spgwpb/spgw.proto

cd spgwpb

# protoc --proto_path=. --go_out=. \
#     --go_opt=paths=source_relative \
#     --micro_out=. spgw.proto

protoc --proto_path=. --go_out=. \
    --micro_out=. spgw.proto
