load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def llvmbzlgen_dependencies():
    go_rules_dependencies()
    go_register_toolchains()
    gazelle_dependencies()
    _gazelle_repositories()

def _gazelle_repositories():
    go_repository(
        name = "com_github_alecthomas_participle",
        importpath = "github.com/alecthomas/participle",
        tag = "v0.2.1",
    )

    go_repository(
        name = "com_github_burntsushi_toml",
        importpath = "github.com/BurntSushi/toml",
        tag = "v0.3.1",
    )

    go_repository(
        name = "com_github_creachadair_ini",
        importpath = "github.com/creachadair/ini",
        tag = "v0.0.1",
    )

    go_repository(
        name = "com_github_creachadair_staticfile",
        importpath = "github.com/creachadair/staticfile",
        tag = "v0.0.3",
    )

    go_repository(
        name = "com_github_davecgh_go_spew",
        importpath = "github.com/davecgh/go-spew",
        tag = "v1.1.1",
    )

    go_repository(
        name = "com_github_google_go_cmp",
        importpath = "github.com/google/go-cmp",
        tag = "v0.3.0",
    )

    go_repository(
        name = "com_github_pmezard_go_difflib",
        importpath = "github.com/pmezard/go-difflib",
        tag = "v1.0.0",
    )

    go_repository(
        name = "com_github_stretchr_testify",
        importpath = "github.com/stretchr/testify",
        tag = "v1.2.2",
    )

    go_repository(
        name = "org_bitbucket_creachadair_stringset",
        importpath = "bitbucket.org/creachadair/stringset",
        tag = "v0.0.7",
    )
