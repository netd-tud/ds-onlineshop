# Distributed Transaction Monitor

This component deploys the [dtm-labs/dtm](github.com/dtm-labs/dtm) **distributed transaction monitor** which can be used by some
application services to submit distributed transactions and ensure their correct execution.

The dtm service exposes two ports:
- `36789`: http connections and DTM UI
- `36790`: grpc connections

The component also deploys a postgres database to allow for correct execution even if dtm crashes.

## Quick Start

From inside your desired overlay directory e.g. `kustomize/overlays/k3s` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component ../../components/dtm
```

This will update the `kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- ../../base
components:
- ../../components/dtm
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.

