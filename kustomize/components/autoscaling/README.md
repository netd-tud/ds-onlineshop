# Frontend Autoscaling

This component configures a HorizontalPodAutoscaler (HPA) for the frontend service, which automatically scales the
number of frontend pods based on CPU utilization.

## Quick Start

From inside your desired overlay directory e.g. `kustomize/overlays/k3s` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component ../../components/autoscaling
```

This will update the `kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- ../../base
components:
- ../../components/autoscaling
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.
