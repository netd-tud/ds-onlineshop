# Reorder Service

The Reorder service provides functionality for automatically generating reorder requests based on stock alerts sent
by the Notification Service.

The service keeps a cache of the current stock alerts which need to be resolved. After a delay which can be set using the
environment variable `ALERT_EXPIRATION_MINUTES`, the service will generate a reorder request for the alerts which expired alerts.

## Proto Files

Generated proto files are placed inside `src/reorderservice/src/reorderservice/generated_proto` after running `genproto.sh`:

```
./genproto.sh
```

Original proto files should be placed at `../../protos` before executing the script.

## Build

From `src/reorderservice`, run:

```
docker build ./
```
