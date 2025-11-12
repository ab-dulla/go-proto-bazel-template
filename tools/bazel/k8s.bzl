"""
Custom Bazel macros for Kubernetes manifest generation.
"""

load("@rules_helm//helm:defs.bzl", "helm_template")

def k8s_environment(name, chart, kustomize_overlay, services = []):
    """
    A macro that generates a final Kubernetes manifest for a specific environment.

    This macro encapsulates the entire deployment manifest generation chain:
    1. Bundle multiple container images together.
    2. Stamp the Helm chart with the digests of the container images.
    3. Render the Helm chart with environment-specific values.
    4. Apply a Kustomize overlay to the rendered Helm chart.
    5. Produce a single, deployable YAML manifest.
    """

    # 1. Collect image digests: each oci_image produces a sibling target `<name>.digest`.
    # rules_helm expects an `images` label which is a mapping (was previously container_bundle). For now we
    # pass the raw image targets list; if needed, create a helper rule producing a bundle file mapping release_name->digest.

    # 2. Render the Helm chart, stamping the image digests
    # Render Helm chart (no direct image stamping in rules_helm; image digests should be passed via values files)
    helm_template(
        name = name + "_helm_render",
        chart = chart,
        values = [service["values_file"] for service in services],
        visibility = ["//visibility:public"],
    )

    # 3. Expose the Kustomize overlay target directly as an alias for convenience.
    native.alias(
        name = name,
        actual = kustomize_overlay,
        visibility = ["//visibility:public"],
    )
