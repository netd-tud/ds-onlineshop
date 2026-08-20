# Recommendation Service

The Recommendation service provides functionality for generating product recommendations based on user behavior and preferences.

## Proto Files

Generated proto files are placed inside `src/recommendationservice/proto` after running `genproto.sh`:

```
./genproto.sh
```

Original proto files should be placed at `../../protos` before executing the script.

## Build

From `src/recommendationservice`, run:

```
docker build ./
```
