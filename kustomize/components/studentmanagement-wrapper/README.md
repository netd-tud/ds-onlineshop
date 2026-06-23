# Wrapper to allow Product operations over catalog and inventory

This component adds a wrapper, which allows for operations on products which contain both catalog and inventory.

From the `kustomize/` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component components/studentmanagement-wrapper
```

This will update the `kustomize/kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/studentmanagement-wrapper
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.
