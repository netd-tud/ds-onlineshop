# Use a REST Service to Display Ratings on Product pages

This component adds a ratingservice, which create a simple REST Server to retrieve associated data and display it on the product page.

From the `kustomize/` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component components/ratingservice
```

This will update the `kustomize/kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/ratingservice
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.
