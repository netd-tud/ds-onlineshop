# Authentication & Directory Services Component

This Kustomize component deploys an Identity and Access Management (IAM) stack.
It provisions an OpenLDAP server for identity storage, a Web Management UI, and a custom Go-Based Authentication Service
that processes user credentials via a Search-and-Bind workflow and issues stateless
JSON Web Tokens (JWT) for role-based access control.

## Deployment Instructions
From the `kustomize/` folder at the root level of this repository, execute these commands:

### Generate Local Cryptographic Key Pair
```bash
cd kustomize/components/auth

openssl genpkey -algorithm RSA -out certs/auth_private.pem -pkeyopt rsa_keygen_bits:2048

openssl rsa -pubout -in certs/auth_private.pem -out certs/auth_public.pem
```

### Enable the Component
```bash
kustomize edit add component components/auth
```

This will update the `kustomize/kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/auth
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.

## Access to LDAP UI
The LDAP management UI can be access over port `30380`

## Architectural Overview
1. Identity Storage (ldap.yaml): Runs an OpenLDAP instance hosting the dc=theonlineshop,dc=com directory tree.
2. Credential Verification (authservice.yaml): The Auth Service provides a gRPC endpoint (Login).
   It utilizes a secure administrative account (cn=admin) to search for the targeted entry, extracts its roles,
   and attempts a network socket bind using the user's plain-text password.
3. Stateless Token Issuance: Upon successful binding, the Auth Service signs an asymmetric RS256 token
   holding identity claims and enterprise role attributes. Downstream microservices parse and validate these tokens
   completely offline using the globally injected public key.

## Secret & ConfigMap Configuration
This component uses Kustomize generators to handle credentials and keys safely without hardcoding plaintext assets in configuration files.
- ldap-bootstrap-ldif: Injects structural mappings into the initial database directory pass.
- auth-public-key: Distributed globally as a plaintext ConfigMap file path to enable signature confirmation across downstream services.
- auth-private-key: Only used by the authservice itself for encrypting the JWT issued on login. (Should not be saved visible for all)
