workspace(name = "io_kythe_llvmbzlgen")

load("//:setup.bzl", "llvmbzlgen_repositories")

llvmbzlgen_repositories()

load("//:external.bzl", "llvmbzlgen_dependencies")

llvmbzlgen_dependencies()

# gazelle:repo bazel_gazelle
# gazelle:repository_maco external.bzl%_gazelle_repositories
