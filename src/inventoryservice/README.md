# Inventory Service

The Inventory service provides functionality for managing product inventory and stock levels.

The initial inventory state is defined in `inventory.json` and is not persistently stored in a database.

## Proto Files

Generated proto files are placed inside `src/inventoryservice/genproto` after running `genproto.sh`:

```
./genproto.sh
```

Original proto files should be placed at `../../protos` before executing the script.

## Build

From `src/inventoryservice`, run:

```
docker build ./
```
