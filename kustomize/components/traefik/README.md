# Traefik Ingress Controller

This component deploys the [Traefik](https://doc.traefik.io/traefik/) ingress controller, which acts as a reverse proxy and load balancer
for routing traffic to the Online Boutique microservices.

## Quick Start

From inside your desired overlay directory e.g. `kustomize/overlays/k3s` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component ../../components/traefik
```

This will update the `kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- ../../base
components:
- ../../components/traefik
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.

## Configuration

### traefik-config.yaml
Defines all exposed services and their external ports

| Service                   | Port  |
| ------------------------- | ----- |
| MQTT Broker               | 31883 |
| LDAP                      | 30389 |
| AuthService               | 30060 |
| Warehousemanagement NAIVE | 30051 |
| Warehousemanagement SAGA  | 30052 |
| Warehousemanagement XA    | 30053 |
| dtm-ui                    | 30789 |
| flink-ui                  | 30181 |

### ingress-http.yaml
Defines subpaths for the `frontend`, `grafana` and `prometheus` service which are accessed over https
- `frontend` is exposed at `/`
- `grafana` is exposed at `/grafana`
- `prometheus` is exposed at `/prometheus`

Configures two IngressRoutes for `dtm-ui` and `flink-ui` which can't be configured with subpaths and need dedicated ports.

Also configures tls certificate and hostname.

### ingress-tcp.yaml
Defines IngressRoutes to services which accept TCP traffic, such as gRPC services and the MQTT Broker.
- MQTT Broker
- LDAP
- AuthService
- Warehousemanagement NAIVE
- Warehousemanagement SAGA
- Warehousemanagement XA
