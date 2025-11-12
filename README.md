# Production-Ready Go & Bazel Monorepo Template

This repository provides a complete, production-ready, and highly scalable multi-service Go monorepo template. The core philosophy is **abstraction and reusability**, using custom Starlark macros to encapsulate the project's core workflows so that adding new services is trivial.

## Core Philosophy

- **Abstraction:** Complex workflows like API code generation and Kubernetes manifest creation are hidden behind simple, reusable Bazel macros (`go_api_library`, `k8s_environment`).
- **Reusability:** Shared code, such as database clients and loggers, is placed in the `/pkg` directory. The Kubernetes deployment model uses a single, generic Helm chart that is customized for each service and environment.
- **Trivial Onboarding:** Adding a new service is as simple as copying an existing service, updating a few lines of configuration, and running `bazel run //:gazelle`.

## Tech Stack

- **Build System:** **Bazel 8 LTS**
- **Dependency Management:** **Bzlmod** (`MODULE.bazel`)
- **Language:** Go 1.25.4
- **API:** gRPC, gRPC-Gateway, OpenAPI v2
- **Database ORM:** **GORM** (with PostgreSQL driver)
- **Containerization:** OCI Images (via `rules_oci`)
- **Deployment:** Helm + Kustomize (via `rules_helm` and `rules_kustomize`)
- **Linting:** `golangci-lint` (via `rules_lint`)

---

## Prerequisites

Before you begin, ensure you have the following installed:

- **Bazel:** Version 8.0.0 or higher. We recommend using [Bazelisk](https://github.com/bazelbuild/bazelisk) to automatically manage the Bazel version from the `.bazelversion` file.
- **Go:** Version 1.25.4 or higher.
- **Container registry access:** Needed only to push images with `oci_push`.

## First-Time Setup

To get your local development environment and IDE tooling (`gopls`) in sync, run the following command. This command uses Gazelle to analyze the Bzlmod dependencies in `MODULE.bazel` and updates the `go.mod` file accordingly.

```sh
bazel run //:gazelle-go-mod
```

## Core Workflows

### How to Build

Build all services, libraries, and generated code in the entire monorepo.

```sh
bazel build //...
```

### How to Test & Lint

Run all unit tests and lint checks across the entire monorepo.

```sh
bazel test //...
```

### How to Run a Service

To run a service locally (e.g., the `user-api`), use the `bazel run` command. This will build the service and its dependencies and execute the resulting binary.

```sh
# The user-api will be available on:
# gRPC: localhost:8080
# HTTP: localhost:8081
bazel run //services/user-api
```

### How to Add a New Service

1. **Copy Existing Service:** Copy an existing service directory (e.g., `cp -r services/product-api services/new-service`).
2. **Define Protobuf API:** Add a new `.proto` file in `/proto/acme/new-service/v1`.
3. **Update BUILD files:** Modify the `BUILD.bazel` files in the new directories to reflect the new service name and dependencies.
4. **Run Gazelle:** `bazel run //:gazelle` to update the Go build rules.
5. **Add Deployment Config:** Add new `values-new-service.yaml` files to the Kustomize overlays in `/k8s/overlays`.

---

## Deployment

### How to Generate Deployment YAML

You can generate a single, deployable YAML manifest for any environment using the `k8s_environment` macro.

```sh
# Generate the manifest for the 'dev' environment
bazel build //k8s:dev

# The output will be located at: bazel-bin/k8s/dev.yaml
# You can deploy it directly with kubectl:
kubectl apply -f bazel-bin/k8s/dev.yaml
```

### How to Push Artifacts

### Container Images (OCI)

To build and push a service's OCI image to the configured container registry (`harbor.example.com`), use the `:push` target. Images are built with `oci_image` and published with `oci_push`.

```sh
bazel run //services/user-api:push
bazel run //services/product-api:push
```

### Helm Chart

To push the base Helm chart as an OCI artifact to the registry, use the `:push-oci` target (unchanged).

```sh
bazel run //charts/app:push-oci
```

### Kubernetes Deployment Strategy (Helm + Kustomize)

This template uses a powerful combination of Helm and Kustomize for managing Kubernetes deployments.

1. **Base Helm Chart (`/charts/app`):** A single, generic "App" chart defines the common Kubernetes resources (`Deployment`, `Service`, `HPA`, etc.). It's highly configurable via a `values.yaml` file.
2. **Kustomize Overlays (`/k8s/overlays`):** Each environment (`dev`, `staging`, `prod`) has its own Kustomize overlay. The `kustomization.yaml` file in each overlay:

  - Specifies which services to deploy.
  - Points to environment-specific `values-*.yaml` files to configure each service's replica count, resources, etc.
  - Can apply strategic patches to the manifests (e.g., adding environment variables or sidecar containers).

This approach provides maximum flexibility and avoids duplicating YAML configuration.

### Secrets Management with External Secrets Operator (ESO)

For production-grade secrets management, we recommend using the [External Secrets Operator (ESO)](https://external-secrets.io/). This operator fetches secrets from an external provider (like AWS Secrets Manager, GCP Secret Manager, or HashiCorp Vault) and injects them as native Kubernetes `Secret` objects.

**Workflow:**

1. **Install ESO:** Install the External Secrets Operator in your Kubernetes cluster.
2. **Create an `ExternalSecret`:** You would create a manifest like the one below, which tells ESO to fetch a secret from your cloud provider and create a Kubernetes `Secret` named `db-credentials` with the key `DATABASE_URL`.

    ```yaml
    # Example: external-secret.yaml (This file would live outside this repo)
    apiVersion: external-secrets.io/v1beta1
    kind: ExternalSecret
    metadata:
      name: database-credentials
    spec:
      secretStoreRef:
        name: aws-secret-store # Assumes you have a SecretStore configured
        kind: ClusterSecretStore
      target:
        name: db-credentials # This is the name of the K8s Secret to create
      data:
      - secretKey: DATABASE_URL
        remoteRef:
          key: my-app/database/url # The key of the secret in AWS Secrets Manager
    ```

3. **Reference the Secret:** The Kustomize overlay in `/k8s/overlays/dev/kustomization.yaml` already includes a patch to reference this `db-credentials` secret and mount it as an environment variable in the `user-api` deployment. This keeps your application code clean and your deployment manifests free of sensitive information.
