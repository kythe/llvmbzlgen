load("@bazel_gazelle//:deps.bzl", _go_repository = "go_repository")
load("@bazel_tools//tools/build_defs/repo:utils.bzl", "maybe")

def go_repository(**kwargs):
    """Gazelle-compatible maybe-wrapped go_repository."""
    maybe(_go_repository, **kwargs)
