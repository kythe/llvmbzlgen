# gazelle:repo bazel_gazelle
workspace(name = "io_kythe_llvmbzlgen")

load("//:setup.bzl", "llvmbzlgen_repositories")

llvmbzlgen_repositories()

load("//:external.bzl", "_gazelle_repositories", "llvmbzlgen_dependencies")

llvmbzlgen_dependencies()

# gazelle:repository_macro external.bzl%_gazelle_repositories
_gazelle_repositories()
