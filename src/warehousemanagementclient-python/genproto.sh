#!/bin/bash -eu
outdir=proto

rm -rf $outdir
mkdir -p $outdir/common

python -m grpc_tools.protoc -I../../protos \
    --python_out=./$outdir \
    --grpc_python_out=./$outdir \
    ../../protos/common/common.proto \
    ../../protos/inventory/inventory.proto \
    ../../protos/productcatalog/productcatalog.proto \
    ../../protos/warehousemanagement/warehousemanagement.proto

python -m grpc_tools.protoc -I../../protos --include_imports \
    --descriptor_set_out=/tmp/warehouseclient_descriptor.pb \
    ../../protos/common/common.proto \
    ../../protos/inventory/inventory.proto \
    ../../protos/productcatalog/productcatalog.proto \
    ../../protos/warehousemanagement/warehousemanagement.proto

python -m protoletariat --python-out ./$outdir --in-place raw /tmp/warehouseclient_descriptor.pb

touch $outdir/__init__.py
touch $outdir/common/__init__.py
touch $outdir/inventory/__init__.py
touch $outdir/productcatalog/__init__.py
touch $outdir/warehousemanagement/__init__.py

