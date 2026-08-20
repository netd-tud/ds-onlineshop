# Auth Service

The Auth service provides authentication and authorization functionality for the application. It connects to a ldap database
to verify provided credentials. Upon successful authentication, it returns a JWT token that can be used to access other services functions.

## Proto Files

Generated proto files are placed inside `src/authservice/genproto` after running `genproto.sh`:

```
./genproto.sh
```

Original proto files should be placed at `../../protos` before executing the script.

## Build

From `src/authservice`, run:

```
docker build ./
```
