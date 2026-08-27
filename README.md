**Online Boutique** is a cloud-first microservices demo application.  The application is a
web-based e-commerce app where users can browse items, add them to the cart, and purchase them.

## Architecture

**Online Boutique** is composed of 14 microservices, written in different languages,
that communicate with each other mostly over gRPC. These business services are complemented by several
infrastructure services, along with load generators used to demonstrate the application's behavior under load.

[![Architecture of microservices](/docs/img/architecture-diagram-deployment.png)](/docs/img/architecture-diagram-deployment.png)

### Microservices
Find **Protocol Buffers Descriptions** at the [`./protos` directory](/protos).

| Service                                             | Language      | Description                                                                                                                                       |
|-----------------------------------------------------|---------------|---------------------------------------------------------------------------------------------------------------------------------------------------|
| [frontend](/src/frontend)                           | Go            | Exposes an HTTP server to serve the website. Does not require signup/login and generates session IDs for all users automatically.                 |
| [productcatalogservice](/src/productcatalogservice) | Go            | Provides the list of products from a JSON file and ability to search products and get individual products.                                        |
| [inventoryservice](/src/inventoryservice)           | Go            | Manages the inventory of products.                                                                                                                |
| [checkoutservice](/src/checkoutservice)             | Go            | Retrieves user cart, prepares order and orchestrates the payment, shipping and the email notification.                                            |
| [warehousemanagement](/src/warehousemanagement)     | Go            | Acts as a wrapper around inventory- and productcatalogservice and allows to connect and perform operations via gRPC/MQTT from outside the cluster |
| [cartservice](/src/cartservice)                     | C#            | Stores the items in the user's shopping cart in Redis and retrieves it.                                                                           |
| [authservice](/src/authservice)                     | Go            | Handles user authentication and authorization by searching for user information in a LDAP directory. Issues JWT for auth in other services        |
| [currencyservice](/src/currencyservice)             | Node.js       | Converts one money amount to another currency. Uses real values fetched from European Central Bank. It's the highest QPS service.                 |
| [ratingservice](/src/ratingservice)                 | Go            | Provides product ratings and reviews.                                                                                                             |
| [notificationservice](/src/notificationservice)     | Go            | Provide low stock alerts and list of recent orders.                                                                                               |
| [paymentservice](/src/paymentservice)               | Node.js       | Charges the given credit card info (mock) with the given amount and returns a transaction ID.                                                     |
| [shippingservice](/src/shippingservice)             | Go            | Gives shipping cost estimates based on the shopping cart. Ships items to the given address (mock)                                                 |
| [emailservice](/src/emailservice)                   | Python        | Sends users an order confirmation email (mock).                                                                                                   |
| [recommendationservice](/src/recommendationservice) | Python        | Recommends other products based on what's given in the cart.                                                                                      |
| [adservice](/src/adservice)                         | Java          | Provides text ads based on given context words.                                                                                                   |

### Infrastructure Services
| Service                                                                   | Image/Version                             | Purpose                                                                                           |
|---------------------------------------------------------------------------|-------------------------------------------|---------------------------------------------------------------------------------------------------|
| [MQTT broker](kustomize/base/mqtt-broker.yaml)                            | `eclipse-mosquitto:2.1-alpine`            | Pub/sub message broker for MQTT events                                                            |
| [LDAP](kustomize/components/auth/ldap.yaml)                               | `osixia/openldap:2.6.10-alpha`            | User Directory service / authentication                                                           |
| [DTM](kustomize/components/dtm/dtm.yaml)                                  | `yedf/dtm:1.19.0`                         | Distributed transaction coordination (SAGA, XA)                                                   |
| [cAdvisor](kustomize/components/monitoring/cadvisor-daemonset.yaml)       | `google/cadvisor:0.57`                    | Collects per-container resource metrics and exposes them to Prometheus                            |
| [Pushgateway](kustomize/components/monitoring/prometheus-deployment.yaml) | `prom/pushgateway:v1.9.0`                 | Relays load values from dynamic-load-producer to Prometheus                                       |
| [Prometheus](kustomize/components/monitoring/prometheus-deployment.yaml)  | `prom/prometheus:v3.12.0`                 | Metrics collection and storage                                                                    |
| [Grafana](kustomize/components/monitoring/grafana-deployment.yaml)        | `grafana/grafana:13.1`                    | Dashboards for visualizing load metrics and analytics data                                        |
| [Kafka (Strimzi)](kustomize/components/kafka)                             | Strimzi `1.1.0`, Kafka `4.3.0`            | Event streaming backbone for analytics pipeline                                                   |
| [Flink](kustomize/components/flink)                                       | `flink:1.19.0`                            | Stream processing (SQL jobs consuming Kafka, writing to PostgreSQL                                |
| [PostgreSQL](kustomize/components/flink/postgres.yaml)                    | `postgres:18-alpine`                      | Persistent sink for the Flink analytics pipeline                                                  |
| [Traefik](kustomize/components/traefik)                                   | `rancher/mirrored-library-traefik:3.6.13` | Ingress controller: HTTP(S) routing and TCP passthrough for non-HTTP protocols (MQTT, LDAP, gRPC) |                                                                     |

### Load Generators
| Service                                             | Language      | Description                                                                          |
|-----------------------------------------------------|---------------|--------------------------------------------------------------------------------------|
| [loadgenerator](/src/loadgenerator)                 | Python/Locust | Continuously sends requests imitating realistic user shopping flows to the frontend. |
| [dynamic-load-producer](/src/dynamic-load-producer) | Bash          | Creates dynamic load for the monitoring task showcasing overloading.                 |

## Screenshots

| Home Page                                                                                                             | Checkout Screen                                                                                                        |
|-----------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------|
| [![Screenshot of store homepage](/docs/img/online-boutique-frontend-1.png)](/docs/img/online-boutique-frontend-1.png) | [![Screenshot of checkout screen](/docs/img/online-boutique-frontend-2.png)](/docs/img/online-boutique-frontend-2.png) |
