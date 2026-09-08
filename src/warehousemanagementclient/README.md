# Warehouse Management Client

The Warehouse Management Client connects to the warehousemanagement to execute operations and is supposed to be
an example/solution of a possible gRPC and MQTT client implementation.

## Proto Files

Generated proto files are placed inside `src/warehousemanagementclient/src/warehousemanagementclient/genproto` after running `genproto.sh`:

```
./genproto.sh
```

Original proto files should be placed at `../../protos` before executing the script.

## Run

You can configure weither a new product should be **create**d or an existing product should be **update**d by changing the
`action` parameter in `config.json`. Further, you can select which distributed action variant should be used, in case
the inventory fails to create stock after the productcatalog objects has already been created, to rollback.
The three options are:
- naive (naive implementation of a rollback)
- saga (use [dtm](../../kustomize/components/dtm) and the saga pattern)
- xa (use [dtm](../../kustomize/components/dtm) and the xa pattern)

It is also possible to choose weither a product should be created using synchronous gRPC or asynchronous MQTT by setting
`"connection-type"` to either `grpc` or `mqtt` in `config.json`.
> Note that currently only create is supported via MQTT. If you want to update product stock, please use gRPC.

After configuring and going to `src/warehousemanagementclient`, run:

```bash
source .venv/bin/activate
warehousemanagementclient

# or if you have uv installed

uv run warehousemanagementclient
```
