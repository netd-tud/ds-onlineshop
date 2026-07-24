#!/bin/bash -eu
#
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# [START gke_checkoutservice_genproto]

PATH=$PATH:$(go env GOPATH)/bin
protodir=../../protos
outdir=./genproto

rm -rf $outdir
mkdir -p $outdir

protoc --proto_path=$protodir \
    --go_out=./$outdir --go_opt=paths=source_relative \
    --go-grpc_out=./$outdir --go-grpc_opt=paths=source_relative \
    --go_opt=Mcommon/common.proto=github.com/turt1z/microservices-demo/src/checkoutservice/genproto/common \
    --go-grpc_opt=Mcommon/common.proto=github.com/turt1z/microservices-demo/src/checkoutservice/genproto/common \
    --go_opt=Mpayment/payment.proto=github.com/turt1z/microservices-demo/src/checkoutservice/genproto/payment \
    --go-grpc_opt=Mpayment/payment.proto=github.com/turt1z/microservices-demo/src/checkoutservice/genproto/payment \
    --go_opt=Mcart/cart.proto=github.com/turt1z/microservices-demo/src/checkoutservice/genproto/cart \
    --go-grpc_opt=Mcart/cart.proto=github.com/turt1z/microservices-demo/src/checkoutservice/genproto/cart \
    $protodir/checkout/checkout.proto \
    $protodir/common/common.proto \
    $protodir/payment/payment.proto \
    $protodir/productcatalog/productcatalog.proto \
    $protodir/cart/cart.proto \
    $protodir/currency/currency.proto \
    $protodir/email/email.proto \
    $protodir/shipping/shipping.proto

# [END gke_checkoutservice_genproto]
