# Dynamic Load Producer for Monitoring Demo

This component deploys a dynamic load producer and frontend demo to show simulate load throughout a day.
It includes a dynamic-load-producer for generating the load and frontend-demo which can be monitored in combination with the monitoring component.

## Quick Start

From the `kustomize/` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component components/dynamic-load-producer
```

This will update the `kustomize/kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/dynamic-load-producer
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.

## Components

### dynamic-load-producer
A simple batch script which sends GET Requests to the frontend-demo
- Request send to the /heavyLoad endpoint of the frontend-demo
- passes iters argument to define how much load should be generated

### frontend-demo
A duplicate of the frontend base component, which accepts the requests from the dynamic-load-producer

**Deployment Details:**
- uses the /heavyLoad endpoint to generate load
- **Access:** http://frontend-demo/heavyLoad?iters=10000000

**Configuration:**
- limited resources to easily show overload
  - *CPU*: 100m
  - *Memory*: 128Mi
