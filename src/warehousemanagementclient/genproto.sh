#!/bin/bash -eu

OUTDIR="src/warehousemanagementclient/genproto"

rm -rf $OUTDIR
mkdir -p $OUTDIR

source .venv/bin/activate

python -m grpc_tools.protoc -I../../protos \
    --python_out=./$OUTDIR \
    --grpc_python_out=./$OUTDIR \
    ../../protos/common/common.proto \
    ../../protos/inventory/inventory.proto \
    ../../protos/productcatalog/productcatalog.proto \
    ../../protos/warehousemanagement/warehousemanagement.proto \
    ../../protos/auth/auth.proto

# Fix Python imports natively (Cross-platform sed)
if [[ "$OSTYPE" == "darwin"* ]]; then
  SED_INPLACE=(-i '')
else
  SED_INPLACE=(-i)
fi

MODULES=("common" "inventory" "productcatalog" "warehousemanagement" "auth")

find "$OUTDIR" -type f -name "*.py" | while read -r file; do
  for mod in "${MODULES[@]}"; do
    # Replace: 'from X import X_pb2' -> 'from warehousemanagementclient.genproto.X import X_pb2'
    sed "${SED_INPLACE[@]}" "s/from $mod import ${mod}_pb2/from warehousemanagementclient.genproto.$mod import ${mod}_pb2/g" "$file"
  done
done

touch $OUTDIR/__init__.py
touch $OUTDIR/common/__init__.py
touch $OUTDIR/inventory/__init__.py
touch $OUTDIR/productcatalog/__init__.py
touch $OUTDIR/warehousemanagement/__init__.py
touch $OUTDIR/auth/__init__.py


