# Kafka Component

This component provides a standardized Kafka cluster configuration. It simplifies the inclusion of Kafka as a distributed
streaming platform into the application environment by encapsulating the necessary Kubernetes resources and default configurations.

This component utilizes the `kafka.strimzi.io/v1` ([Strimzi](https://strimzi.io/)) operator, which provides a cloud-native way to deploy and manage Kafka clusters on Kubernetes.

## Quick Start

From the `kustomize/` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component components/kafka
```

This will update the `kustomize/kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/kafka
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.

## Usage
Before you deploy the Kafka Component you should also apply the strimzi setup which can be achieved by executing:
```
kubectl apply -k kustomize/infra/strimzi-operator
```

Microservices reach kafka at this address: `analytics-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092`
