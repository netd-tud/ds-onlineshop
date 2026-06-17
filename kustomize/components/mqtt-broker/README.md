# Add an MQTT Broker for Asynchronous Communication

This component deploys an MQTT broker (Eclipse Mosquitto) to facilitate asynchronous communication between microservices. Specifically, it configures the `checkoutservice` and `emailservice` to use the MQTT broker for event-driven interactions, replacing direct gRPC connections for order completion notifications.

From the `kustomize/` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component components/mqtt-broker
```

This will update the `kustomize/kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/mqtt-broker
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.
