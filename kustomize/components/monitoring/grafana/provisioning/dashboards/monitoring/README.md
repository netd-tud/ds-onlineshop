# Monitoring Dashboards
These Dashboards visualize the cpu usage of pods in relation to request being sent.
It showcases cpu exhaustion, throttling and how it affects the performance.

Two deployments are shown here, one with only a single computational pod and one with multiple pods, which are created
dynamically to showcase the difference in performance, that is possible when replicating a service.

## Non-replicated
This dashboard show the metrics for the first deployment with a single running instance of the service.

## Replicated
This dashboard show the metrics for the second deployment with a replicated set of the service.

## Combined
This dashboard show the metrics for both deployments in the same diagrams to allow for easy comparison.
