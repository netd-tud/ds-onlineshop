# Replicated frontend-demo

This component increases the number of replicas of the frontend-demo defines in the dynamic load producer component. It is therefore only supposed to be used in combination with the aforemantioned.

## Quick Start

From the `kustomize/` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component components/frontend-demo-replicated
```

This will update the `kustomize/kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/dynamic-load-producer
- components/frontend-demo-replicated
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.
