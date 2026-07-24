#!/bin/bash -eu

PATH=$PATH:$(go env GOPATH)/bin
protodir=../../protos
outdir=./genproto

rm -rf genproto
mkdir -p genproto

protoc --proto_path=$protodir \
    --go_out=./$outdir --go_opt=paths=source_relative \
    --go-grpc_out=./$outdir --go-grpc_opt=paths=source_relative \
    --go_opt=Mcommon/common.proto=github.com/turt1z/microservices-demo/src/inventoryservice/genproto/common \
    --go-grpc_opt=Mcommon/common.proto=github.com/turt1z/microservices-demo/src/inventoryservice/genproto/common \
    $protodir/inventory/inventory.proto \
    $protodir/productcatalog/productcatalog.proto \
    $protodir/common/common.proto \
