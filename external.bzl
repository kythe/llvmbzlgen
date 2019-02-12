load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@io_kythe_llvmbzlgen//:setup.bzl", "maybe")

def llvmbzlgen_dependencies():
    go_rules_dependencies()
    go_register_toolchains()
    gazelle_dependencies()

    maybe(
        go_repository,
        name = "com_github_alecthomas_participle",
        commit = "bf8340a459bd383e5eb7d44a9a1b3af23b6cf8cd",
        importpath = "github.com/alecthomas/participle",
    )

    maybe(
        go_repository,
        name = "com_github_google_go_cmp",
        importpath = "github.com/google/go-cmp",
        tag = "v0.2.0",
    )
