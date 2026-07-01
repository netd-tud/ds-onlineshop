# Send email confirmation over mqtt broker

This component facilitate asynchronous communication between microservices by using MQTT. Specifically, it configures the `checkoutservice` and `emailservice` to use the MQTT broker for event-driven interactions, replacing direct gRPC connections for order completion notifications.

From the `kustomize/` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component components/email-via-mqtt
```

This will update the `kustomize/kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/email-via-mqtt
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.
