#!/bin/bash -eu
outdir=proto

rm -rf $outdir
mkdir -p $outdir/email $outdir/common

python -m grpc_tools.protoc -I../../protos \
    --python_out=./$outdir \
    --grpc_python_out=./$outdir \
    ../../protos/email/email.proto \
    ../../protos/common/common.proto

python -m grpc_tools.protoc -I../../protos --include_imports \
    --descriptor_set_out=/tmp/email_descriptor.pb \
    ../../protos/email/email.proto \
    ../../protos/common/common.proto

python -m protoletariat --python-out ./$outdir --in-place raw /tmp/email_descriptor.pb

touch $outdir/__init__.py
touch $outdir/email/__init__.py
touch $outdir/common/__init__.py
