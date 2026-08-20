# Frontend Service

The Frontend service provides the user interface for the application and provides functionality for
retrieving information from different services.

## Proto Files

Generated proto files are placed inside `src/frontend/genproto` after running `genproto.sh`:

```
./genproto.sh
```

Original proto files should be placed at `../../protos` before executing the script.

## Build

From `src/frontend`, run:

```
docker build ./
```
