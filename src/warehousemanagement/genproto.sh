#!/bin/bash -eu

PATH=$PATH:$(go env GOPATH)/bin
protodir=../../protos
outdir=./genproto

rm -rf $outdir
mkdir -p $outdir

protoc --proto_path=$protodir \
    --go_out=./$outdir --go_opt=paths=source_relative \
    --go-grpc_out=./$outdir --go-grpc_opt=paths=source_relative \
    --go-grpc_opt=Mcommon/common.proto=github.com/turt1z/microservices-demo/src/warehousemanagement/genproto/common \
    --go_opt=Mcommon/common.proto=github.com/turt1z/microservices-demo/src/warehousemanagement/genproto/common \
    --go-grpc_opt=Mproductcatalog/productcatalog.proto=github.com/turt1z/microservices-demo/src/warehousemanagement/genproto/productcatalog \
    --go_opt=Mproductcatalog/productcatalog.proto=github.com/turt1z/microservices-demo/src/warehousemanagement/genproto/productcatalog \
    --go-grpc_opt=Minventory/inventory.proto=github.com/turt1z/microservices-demo/src/warehousemanagement/genproto/inventory \
    --go_opt=Minventory/inventory.proto=github.com/turt1z/microservices-demo/src/warehousemanagement/genproto/inventory \
    $protodir/warehousemanagement/warehousemanagement.proto \
    $protodir/common/common.proto \
    $protodir/inventory/inventory.proto \
    $protodir/productcatalog/productcatalog.proto \
