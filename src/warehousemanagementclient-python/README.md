# Warehouse Management Client (Python)

The Warehouse Management Client connects to the warehousemanagement to execute operations and is supposed to be
an example/solution of a possible gRPC client implementation.

## Proto Files

Generated proto files are placed inside `src/warehousemanagementclient-python/proto` after running `genproto.sh`:

```
./genproto.sh
```

Original proto files should be placed at `../../protos` before executing the script.

## Run

You can configure weither a new product should be created or an existing product should be updated by changing the
`action` parameter in `config.json`. Further, you can select which distributed action variant should be used, in case
the inventory fails to create stock after the productcatalog objects has already been created, to rollback.
The three options are:
- naive (naive implementation of a rollback)
- saga (use [dtm](../../kustomize/components/dtm) and the saga pattern)
- xa (use [dtm](../../kustomize/components/dtm) and the xa pattern)

After configuring and going to `src/warehousemanagementclient-python`, run:

```
source venv/bin/activate
python client.py
```
