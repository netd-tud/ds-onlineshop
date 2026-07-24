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

# [START gke_recommendationservice_genproto]

# script to compile python protos
#
# requires gRPC tools:
#   pip install -r requirements.txt
outdir=proto

rm -rf $outdir
mkdir -p $outdir/recommendation

python -m grpc_tools.protoc -I../../protos \
    --python_out=./$outdir \
    --grpc_python_out=./$outdir \
    ../../protos/recommendation/recommendation.proto \
    ../../protos/productcatalog/productcatalog.proto \
    ../../protos/common/common.proto \

python -m grpc_tools.protoc -I../../protos --include_imports \
    --descriptor_set_out=/tmp/recommendation_descriptor.pb \
    ../../protos/recommendation/recommendation.proto \
    ../../protos/productcatalog/productcatalog.proto \
    ../../protos/common/common.proto \

python -m protoletariat --python-out ./$outdir --in-place raw /tmp/recommendation_descriptor.pb

touch $outdir/__init__.py
touch $outdir/recommendation/__init__.py
touch $outdir/productcatalog/__init__.py
touch $outdir/common/__init__.py
# [END gke_recommendationservice_genproto]
