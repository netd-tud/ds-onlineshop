# Monitoring Stack for Online Boutique

This component deploys a comprehensive monitoring stack to collect, store, and visualize metrics from the Online Boutique microservices. It includes [Prometheus](https://prometheus.io/) for metrics collection and storage, [Grafana](https://grafana.com/) for visualization, [cAdvisor](https://github.com/google/cadvisor) for container metrics, and [Pushgateway](https://github.com/prometheus/pushgateway) for batch job metrics.

## Quick Start

From the `kustomize/` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component components/monitoring
```

This will update the `kustomize/kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/monitoring
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.

## Components

### Prometheus
A time-series database for storing and querying metrics collected from the microservices.

**Deployment Details:**
- **Port:** 9090
- **Storage:** 5Gi PersistentVolumeClaim
- **Access:** http://prometheus:9090/prometheus/

**Configuration:**
- Scrapes metrics from the Frontend service and cAdvisor endpoints every 10 seconds
- Collects batch job metrics from Pushgateway
- Uses Kubernetes service discovery (EndpointSlices) for dynamic target discovery
- Stores metrics in `/prometheus` with web interface accessible at the configured external URL

### Grafana
A visualization and monitoring platform for creating dashboards and alerts based on Prometheus metrics.

**Deployment Details:**
- **Port:** 3000
- **Storage:** 2Gi PersistentVolumeClaim
- **Access:** http://grafana:3000/grafana/

**Configuration:**
- Automatically provisioned with Prometheus as a data source
- Configured to serve from `/grafana/` path for proxy-based deployments
- Data source URL: `http://prometheus:9090/prometheus/`

### cAdvisor
A container metrics collector that runs as a DaemonSet to gather CPU, memory, and I/O metrics for every container on each node.

**Deployment Details:**
- **Type:** DaemonSet (runs on all nodes)
- **Port:** 8080 (metrics endpoint)
- **Access:** http://cadvisor:8080/metrics

**Configuration:**
- Connects to containerd socket at `/var/run/containerd/containerd.sock` or `/run/k3s/containerd/containerd.sock`
- Mounts host filesystem, `/sys`, and `/var/run` as read-only for metric collection
- Exposes metrics at `/metrics` endpoint for Prometheus scraping
- Service: `cadvisor` with headless ClusterIP (None)

### Pushgateway
A gateway for collecting metrics from batch jobs and other short-lived processes that cannot be scraped directly.

**Deployment Details:**
- **Port:** 9091
- **Location:** Runs in the same deployment pod as Prometheus
- **Access:** http://prometheus:9091/

**Configuration:**
- Allows push-based metrics collection for non-persistent jobs
- Accessible at `prometheus:9091` from within the cluster
- Metrics are scraped by Prometheus from this gateway

## Configuration Files

### prometheus.yml
Defines what metrics to scrape and how frequently:
- **Frontend:** Scrapes Spring Boot Actuator metrics from `frontend-demo:8080/actuator/prometheus`
- **cAdvisor:** Discovers cAdvisor endpoints via Kubernetes service discovery
- **Pushgateway:** Collects metrics pushed by batch jobs

### datasources.yml
Configures Grafana's connection to Prometheus as the default data source.

## RBAC Permissions

Prometheus has been granted ClusterRole `prometheus-endpoint-reader` with permissions to:
- List and watch Kubernetes services, pods, and nodes
- Access EndpointSlices for dynamic service discovery

This allows Prometheus to automatically discover scrape targets in the cluster.

## Next Steps

1. **View Metrics:** Query Prometheus directly at the web UI or via the HTTP API
2. **Create Dashboards:** Use Grafana to create custom dashboards visualizing your metrics
3. **Set Alerts:** Configure alerting rules in Prometheus for critical thresholds
4. **Push Custom Metrics:** Use Pushgateway to push metrics from batch jobs or scripts

For more information on configuring Prometheus scrape configs, see the [Prometheus documentation](https://prometheus.io/docs/prometheus/latest/configuration/configuration/).

For Grafana dashboard creation and data source configuration, see the [Grafana documentation](https://grafana.com/docs/).
