"""
Custom Bazel macros for Protobuf and Go API generation.
"""

load("@rules_go//proto:def.bzl", "go_proto_library")
load("@rules_proto//proto:defs.bzl", "proto_library")

def go_api_library(name, srcs, deps = [], visibility = None):
    """
    A macro that generates a complete Go API library from a .proto file.

    This macro encapsulates the logic for generating:
    1. Go gRPC server/client code (`protoc-gen-go`, `protoc-gen-go-grpc`)
    2. Go gRPC-Gateway reverse-proxy code (`protoc-gen-grpc-gateway`)
    3. The OpenAPI v2 spec (`.swagger.json` file)
    4. Go validation code (`protoc-gen-validate`)
    """
    proto_library(
        name = name + "_proto",
        srcs = srcs,
        deps = deps + [
            "@com_github_envoyproxy_protoc_gen_validate//validate:validate_proto",
            "@com_github_grpc_ecosystem_grpc_gateway_v2//google/api:annotations_proto",
        ],
        visibility = ["//visibility:public"],
    )

    # 1. Go gRPC server/client code
    go_proto_library(
        name = name + "_go_grpc",
        compilers = [
            "@rules_go//proto:go_grpc",
        ],
        importpath = "go-monorepo-template/proto/" + native.package_name(),
        proto = ":" + name + "_proto",
        visibility = ["//visibility:public"],
    )

    # 2. Go gRPC-Gateway reverse-proxy code
    go_proto_library(
        name = name + "_go_grpc_gateway",
        compilers = [
            "@com_github_grpc_ecosystem_grpc_gateway_v2//protoc-gen-grpc-gateway:go_grpc_gateway",
        ],
        importpath = "go-monorepo-template/proto/" + native.package_name(),
        proto = ":" + name + "_proto",
        visibility = ["//visibility:public"],
    )

    # 3. OpenAPI v2 spec
    native.genrule(
        name = name + "_openapi_spec",
        srcs = [":" + name + "_proto"],
        outs = [name + ".swagger.json"],
        cmd = """
        $(execpath @com_github_grpc_ecosystem_grpc_gateway_v2//protoc-gen-openapiv2) \\
            --logtostderr=true \\
            --allow_merge=true \\
            --merge_file_name=$(location {out}) \\
            -I . \\
            -I $(GENDIR) \\
            $(locations srcs)
        """.format(out = name + ".swagger.json"),
        tools = ["@com_github_grpc_ecosystem_grpc_gateway_v2//protoc-gen-openapiv2"],
        visibility = ["//visibility:public"],
    )

    # 4. Go validation code
    go_proto_library(
        name = name + "_go_validate",
        compilers = [
            "@com_github_envoyproxy_protoc_gen_validate//:protoc-gen-validate_go",
        ],
        importpath = "go-monorepo-template/proto/" + native.package_name(),
        proto = ":" + name + "_proto",
        visibility = ["//visibility:public"],
    )
