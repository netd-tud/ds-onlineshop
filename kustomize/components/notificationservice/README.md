# Show Stock Alerts and Recent orders to users

This component add a notification service that sends alerts when products are low on stock and keeps small list of recent orders.
These Alerts can be seend by users who authenticated with the system and possess associated roles.

From the `kustomize/` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component components/notificationservice
```

This will update the `kustomize/kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/notificationservice
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.

## Configuration
The amount of orders that is shown per currency on the notification page can me adjusted via the environment variable `ORDER_QUEUE_CAPACITY`.
