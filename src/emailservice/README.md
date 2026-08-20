# Email Service

The Email service provides functionality for sending emails to users as a confirmation for successfully completed orders.

## Proto Files

Generated proto files are placed inside `src/emailservice/proto` after running `genproto.sh`:

```
./genproto.sh
```

Original proto files should be placed at `../../protos` before executing the script.

## Build

From `src/emailservice`, run:

```
docker build ./
```
