#!/bin/bash -eu

OUTDIR="src/reorderservice/generated_proto"

rm -rf $OUTDIR
mkdir -p $OUTDIR/notification $OUTDIR/inventory $OUTDIR/checkout $OUTDIR/common $OUTDIR/payment $OUTDIR/auth

source .venv/bin/activate

python -m grpc_tools.protoc \
    -I ../../protos \
    --python_out=./$OUTDIR \
    --grpc_python_out=./$OUTDIR \
    ../../protos/notification/notification.proto \
    ../../protos/inventory/inventory.proto \
    ../../protos/checkout/checkout.proto \
    ../../protos/payment/payment.proto \
    ../../protos/cart/cart.proto \
    ../../protos/auth/auth.proto \
    ../../protos/common/common.proto

# Fix Python imports natively (Cross-platform sed)
if [[ "$OSTYPE" == "darwin"* ]]; then
  SED_INPLACE=(-i '')
else
  SED_INPLACE=(-i)
fi

MODULES=("checkout" "common" "inventory" "notification" "payment" "cart" "auth")

find "$OUTDIR" -type f -name "*.py" | while read -r file; do
  for mod in "${MODULES[@]}"; do
    # Replace: 'from X import X_pb2' -> 'from reorderservice.generated_proto.X import X_pb2'
    sed "${SED_INPLACE[@]}" "s/from $mod import ${mod}_pb2/from reorderservice.generated_proto.$mod import ${mod}_pb2/g" "$file"
  done
done

touch $OUTDIR/__init__.py
touch $OUTDIR/notification/__init__.py
touch $OUTDIR/inventory/__init__.py
touch $OUTDIR/checkout/__init__.py
touch $OUTDIR/common/__init__.py
touch $OUTDIR/payment/__init__.py
touch $OUTDIR/cart/__init__.py
touch $OUTDIR/auth/__init__.py
