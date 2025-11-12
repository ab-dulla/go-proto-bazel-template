"""
Custom Bazel macros for Protobuf and Go API generation.
"""

load("@rules_go//proto:def.bzl", "go_proto_library")
load("@rules_proto//proto:defs.bzl", "proto_library")

def _proto_link_impl(ctx):
    # Gather generated Go sources from output group.
    generated = []
    group = ctx.attr.target[OutputGroupInfo]
    for f in group.go_generated_srcs.to_list():
        p = f.path
        if p.endswith(".pb.go") or p.endswith("_grpc.pb.go") or p.endswith(".gw.go") or p.endswith("_validate.pb.go"):
            generated.append(f)

    script = ctx.actions.declare_file(ctx.label.name + "_copy.sh")
    pkg = ctx.label.package  # e.g. proto/acme/product/v1
    lines = [
        "#!/usr/bin/env bash",
        "set -euo pipefail",
        "dest=\"$BUILD_WORKSPACE_DIRECTORY/%s\"" % pkg,
        "mkdir -p \"$dest\"",
    ]
    for f in generated:
        lines.append("cp \"%s\" \"$dest/$(basename %s)\"" % (f.path, f.path))
    lines.append("echo 'Linked %d generated files to %s'" % (len(generated), pkg))
    ctx.actions.write(script, "\n".join(lines), is_executable = True)
    return [DefaultInfo(executable = script, runfiles = ctx.runfiles(files = generated))]

proto_link = rule(
    implementation = _proto_link_impl,
    attrs = {"target": attr.label(mandatory = True, doc = "go_proto_library to link")},
    executable = True,
    doc = "Executable rule that copies generated Go proto sources into the workspace for IDEs.",
)

def go_api_library(name, srcs, deps = [], visibility = ["//visibility:public"]):
    """Generate Go API artifacts (gRPC, gateway, OpenAPI, validate) from protos.

    Args:
      name: Base name for targets.
      srcs: List of .proto source files.
      deps: Additional proto_library dependencies.
      visibility: Visibility list applied to generated proto + go targets.
    """
    proto_library(
        name = name + "_proto",
        srcs = srcs,
        deps = deps + [
            "@com_github_envoyproxy_protoc_gen_validate//validate:validate_proto",
            "@com_github_grpc_ecosystem_grpc_gateway_v2//google/api:annotations_proto",
        ],
        visibility = visibility,
    )

    # 1. Go gRPC server/client code
    go_proto_library(
        name = name + "_go_grpc",
        compilers = [
            "@rules_go//proto:go_grpc",
        ],
        importpath = "go-monorepo-template/proto/" + native.package_name(),
        proto = ":" + name + "_proto",
        visibility = visibility,
    )

    # 2. Go gRPC-Gateway reverse-proxy code
    go_proto_library(
        name = name + "_go_grpc_gateway",
        compilers = [
            "@com_github_grpc_ecosystem_grpc_gateway_v2//protoc-gen-grpc-gateway:go_grpc_gateway",
        ],
        importpath = "go-monorepo-template/proto/" + native.package_name(),
        proto = ":" + name + "_proto",
        visibility = visibility,
    )

    # 3. OpenAPI v2 spec
    native.genrule(
        name = name + "_openapi_spec",
        srcs = [":" + name + "_proto"],
        outs = [name + ".swagger.json"],
        cmd = """
$(execpath @com_github_grpc_ecosystem_grpc_gateway_v2//protoc-gen-openapiv2) \
    --logtostderr=true \
    --allow_merge=true \
    --merge_file_name=$(location {out}) \
    -I . \
    -I $(GENDIR) \
    $(locations srcs)
""".format(out = name + ".swagger.json"),
        tools = ["@com_github_grpc_ecosystem_grpc_gateway_v2//protoc-gen-openapiv2"],
        visibility = visibility,
    )

    # 4. Go validation code
    go_proto_library(
        name = name + "_go_validate",
        compilers = [
            "@com_github_envoyproxy_protoc_gen_validate//:protoc-gen-validate_go",
        ],
        importpath = "go-monorepo-template/proto/" + native.package_name(),
        proto = ":" + name + "_proto",
        visibility = visibility,
    )

    # Convenience: link main generated grpc code into workspace for IDEs.
    proto_link(
        name = name + "_link",
        target = ":" + name + "_go_grpc",
    )
