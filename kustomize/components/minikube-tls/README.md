# Minikube Local deploy with TLS/SSL

Ensure Minikube is configured to access the demo over HTTP by following the [Development Guide](https://github.com/turt1z/microservices-demo/blob/main/docs/development-guide.md)

### 1. Create Local Self Signed Certificate

Generate an X.509 certificate with the appropriate Subject Alternative Name (SAN) extension required by modern browsers:
```bash
openssl req -x509 -out localhost.crt -keyout localhost.key \
  -newkey rsa:2048 -nodes -sha256 \
  -subj '/CN=localhost' -extensions EXT -config <( \
   printf "[dn]\nCN=localhost\n[req]\ndistinguished_name = dn\n[EXT]\nsubjectAltName=DNS:localhost\nkeyUsage=digitalSignature\nextendedKeyUsage=serverAuth")
```

Export the certificate to PKCS#12 format for browser import:
```bash
openssl pkcs12 -export -out localhost.pfx -inkey localhost.key -in localhost.crt
```
**_NOTE:_**
Import `localhost.pfx` into your browsers trusted certificat store to avoid security warnings when accessing the demo over HTTPS.

### 2. Enable and configure Minikube Ingress

Enable the built-in NGINX Ingress controller add-on:
```bash
minikube addons enable ingress
```

Patch the controller service to expose it via a local `LoadBalancer`:
```bash
kubectl patch svc ingress-nginx-controller \
  -n ingress-nginx \
  -p '{"spec": {"type": "LoadBalancer"}}'
```

### 3.  Create Secrets and Apply kustomize Component

Create the TLS secret in the cluster using the generated certificate and key:
```bash
kubectl create secret tls frontend-tls \
  --cert=localhost.crt \
  --key=localhost.key
```

Apply Kustomize overlay
```bash
cd kustomize && \
kustomize edit add component components/minikube-tls && \
kubectl apply -k . && \
cd ..
```

### 4. Port Forwarding
```bash
sudo KUBECONFIG={$HOME}/.kube/config \
  kubectl port-forward -n ingress-nginx \
  svc/ingress-nginx-controller \
  443:443 80:80
```
You can now access the demo at https://localhost
