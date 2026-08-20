# Shipping Service

The Shipping service provides price quote, tracking IDs, and the impression of order fulfillment & shipping processes.

## Proto Files

Generated proto files are placed inside `src/shippingservice/genproto` after running `genproto.sh`:

```
./genproto.sh
```

Original proto files should be placed at `../../protos` before executing the script.

## Build

From `src/shippingservice`, run:

```
docker build ./
```
