# Automatically reorder products low on stock

This component adds a reorderservice, which connects to the notificationservice to retrieve StockAlerts and reorder products
by using an RPC in inventoryservice.

From the `kustomize/` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component components/reorderservice
```

This will update the `kustomize/kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/reorderservice
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.

## Configuration
The behavior of the reorderservice can be modified by setting the following environment variables:
- `LOW_ALERT_EXPIRATION_MINUTES`: The time in minutes that needs to pass before a reorder is executed to resolve a StockAlert with `low` Severity
- `CRITICAL_ALERT_EXPIRATION_MINUTES`: The time in minutes that needs to pass before a reorder is executed to resolve a StockAlert with `critical` Severity
- `LOWEST_REORDER_AMOUNT`: The minimum quantity to reorder when a StockAlert is received
- `HIGHEST_REORDER_AMOUNT`: The maximum quantity to reorder when a StockAlert is received

The reorder amount is randomly calculated as a number between `LOWEST_REORDER_AMOUNT` and `HIGHEST_REORDER_AMOUNT`.
