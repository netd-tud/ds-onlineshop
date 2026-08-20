# Warehouse Management

The warehousemanagement provides grpc services for executing the following warehouse operations:
`UpdateProductStock` and `CreateNewProduct` wich internally connect to inventoryservice and prodcutcatalogservice
respectivly, to create and modify products and update stock values for prodcuts.
It is also possible to execute these operations via publishing mqtt events to appropriate topics.

## Proto Files

Generated proto files are placed inside `src/warehousemanagement/genproto` after running `genproto.sh`:

```
./genproto.sh
```

Original proto files should be placed at `../../protos` before executing the script.

## Build

From `src/warehousemanagement`, run:

```
docker build ./
```
