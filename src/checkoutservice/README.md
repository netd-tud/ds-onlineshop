# Checkout Service

The Checkout service provides functionality for processing user checkout requests and managing the checkout flow.

## Proto Files

Generated proto files are placed inside `src/checkoutservice/genproto` after running `genproto.sh`:

```
./genproto.sh
```

Original proto files should be placed at `../../protos` before executing the script.

## Build

From `src/checkoutservice`, run:

```
docker build ./
```
