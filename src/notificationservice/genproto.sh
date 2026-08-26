#!/bin/bash -eu

PATH=$PATH:$(go env GOPATH)/bin
protodir=../../protos
outdir=./genproto
module_path="github.com/netd-tud/ds-onlineshop/src/notificationservice/genproto"

rm -rf genproto
mkdir -p genproto

protoc --proto_path=$protodir \
    --go_out=./$outdir --go_opt=paths=source_relative \
    --go-grpc_out=./$outdir --go-grpc_opt=paths=source_relative \
    --go_opt=Mcommon/common.proto=$module_path/common \
    --go-grpc_opt=Mcommon/common.proto=$module_path/common \
    --go_opt=Mcart/cart.proto=$module_path/cart \
    --go-grpc_opt=Mcart/cart.proto=$module_path/cart \
    --go_opt=Mpayment/payment.proto=$module_path/payment \
    --go-grpc_opt=Mpayment/payment.proto=$module_path/payment \
    --go_opt=Mcheckout/checkout.proto=$module_path/checkout \
    --go-grpc_opt=Mcheckout/checkout.proto=$module_path/checkout \
    --go_opt=Mnotification/notification.proto=$module_path/notification \
    --go-grpc_opt=Mnotification/notification.proto=$module_path/notification \
    $protodir/common/common.proto \
    $protodir/cart/cart.proto \
    $protodir/payment/payment.proto \
    $protodir/checkout/checkout.proto \
    $protodir/notification/notification.proto
