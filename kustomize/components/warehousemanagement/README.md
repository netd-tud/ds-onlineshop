# Wrapper to allow Product operations over catalog and inventory

This component adds a wrapper, which allows for operations on products which contain both catalog and inventory.
Three different variants of the wrapper are deployed, each serving a different rollback implementation
for creating a new product.

All three versions can be access over the following ports
- 30051: Naive Implementation
- 30052: Saga Implementation
- 30053: XA Implementation

From the `kustomize/` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component components/warehousemanagement
```

This will update the `kustomize/kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/warehousemanagement
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.
