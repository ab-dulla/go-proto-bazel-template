"""
Custom Bazel macros for Kubernetes manifest generation.
"""

load("@rules_kustomize//kustomize:kustomize.bzl", "kustomize_build")
load("@rules_helm//helm:helm.bzl", "helm_template")
load("@rules_docker//container:container.bzl", "container_bundle")

def k8s_environment(name, chart, kustomize_overlay, services = {}):
    """
    A macro that generates a final Kubernetes manifest for a specific environment.

    This macro encapsulates the entire deployment manifest generation chain:
    1. Bundle multiple container images together.
    2. Stamp the Helm chart with the digests of the container images.
    3. Render the Helm chart with environment-specific values.
    4. Apply a Kustomize overlay to the rendered Helm chart.
    5. Produce a single, deployable YAML manifest.
    """

    # 1. Bundle images to get their digests for stamping
    container_bundle(
        name = name + "_images",
        images = {
            service["release_name"]: service["image_target"]
            for service in services
        },
    )

    # 2. Render the Helm chart, stamping the image digests
    helm_template(
        name = name + "_helm_render",
        chart = chart,
        images = ":" + name + "_images",
        stamp = "{{.image_tag}}",
        values = [service["values_file"] for service in services],
        visibility = ["//visibility:public"],
    )

    # 3. Apply Kustomize overlay
    kustomize_build(
        name = name,
        srcs = [
            ":" + name + "_helm_render",
        ],
        kustomization = kustomize_overlay,
        visibility = ["//visibility:public"],
    )
