"""Simplified proto macro: only gRPC Go code and placeholder swagger."""

load("@rules_go//proto:def.bzl", "go_proto_library")
load("@rules_proto//proto:defs.bzl", "proto_library")

def go_api_library(name, srcs, deps = [], visibility = ["//visibility:public"], importpath_prefix = "go-monorepo-template/"):
    """Create proto + Go gRPC code + placeholder OpenAPI spec (single target)."""
    proto_library(
        name = name + "_proto",
        srcs = srcs,
        deps = deps,
        visibility = visibility,
    )

    go_proto_library(
        name = name + "_go_grpc",
        compilers = [
            "@rules_go//proto:go_proto",
            "@rules_go//proto:go_grpc_v2",
        ],
        importpath = importpath_prefix + native.package_name(),
        proto = ":" + name + "_proto",
        visibility = visibility,
    )

    native.genrule(
        name = name + "_openapi_spec",
        srcs = [":" + name + "_proto"],
        outs = [name + ".swagger.json"],
        cmd = "echo '{\n  \"openapi\": \"3.0.0\",\n  \"info\": {\n    \"title\": \"" + name + " API\",\n    \"version\": \"v1\"\n  },\n  \"paths\": {}\n}' > $@",
        visibility = visibility,
    )
