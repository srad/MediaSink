load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "for_core_protos_go_proto",
    srcs = ["bfc_memory_map.pb.go"],
    importpath = "github.com/wamuir/graft/tensorflow/xla/tsl/protobuf/for_core_protos_go_proto",
    visibility = ["//visibility:public"],
    deps = [
        "@org_golang_google_protobuf//reflect/protoreflect",
        "@org_golang_google_protobuf//runtime/protoimpl",
    ],
)

alias(
    name = "go_default_library",
    actual = ":for_core_protos_go_proto",
    visibility = ["//visibility:public"],
)
