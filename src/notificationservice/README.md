# Notification Service

The Notification service provides functionality for retrieving information about product stock alerts and recent orders per currency.
The service retrieves MQTT Events from the MQTT Broker and keeps an internal, in-memory storage, which is reached via gRPC.

It subscribes to the following topics:
1. `inventory/+/+/stock` for stock changes
2. `+/checkout/orders/completed` for recent orders

## Proto Files

Generated proto files are placed inside `src/notificationservice/genproto` after running `genproto.sh`:

```
./genproto.sh
```

Original proto files should be placed at `../../protos` before executing the script.

## Build

From `src/notificationservice`, run:

```
docker build ./
```
