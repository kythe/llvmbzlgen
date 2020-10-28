load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@io_kythe_llvmbzlgen//:shims.bzl", "go_repository")

def llvmbzlgen_dependencies():
    go_rules_dependencies()
    go_register_toolchains()
    gazelle_dependencies()
    _gazelle_repositories()

def _gazelle_repositories():
    go_repository(
        name = "com_github_alecthomas_participle",
        importpath = "github.com/alecthomas/participle",
        sum = "h1:Pvo8XUCQKgIywVjz/+Ci3IsjGg+g/TdKkMcfgghKCEw=",
        version = "v0.6.0",
    )
    go_repository(
        name = "com_github_alecthomas_repr",
        importpath = "github.com/alecthomas/repr",
        sum = "h1:GDQdwm/gAcJcLAKQQZGOJ4knlw+7rfEQQcmwTbt4p5E=",
        version = "v0.0.0-20181024024818-d37bc2a10ba1",
    )

    go_repository(
        name = "com_github_burntsushi_toml",
        importpath = "github.com/BurntSushi/toml",
        sum = "h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=",
        version = "v0.3.1",
    )

    go_repository(
        name = "com_github_creachadair_ini",
        importpath = "github.com/creachadair/ini",
        sum = "h1:5AJljey6DNMTKl4smh44ZZHErYvIBxDiWgB4QyxjRzY=",
        version = "v0.0.1",
    )

    go_repository(
        name = "com_github_creachadair_staticfile",
        importpath = "github.com/creachadair/staticfile",
        sum = "h1:RhyrMgi7IQn3GejgmGtFuCec58vboEMt5CH6N3ulRJk=",
        version = "v0.1.3",
    )

    go_repository(
        name = "com_github_davecgh_go_spew",
        importpath = "github.com/davecgh/go-spew",
        sum = "h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=",
        version = "v1.1.1",
    )

    go_repository(
        name = "com_github_google_go_cmp",
        importpath = "github.com/google/go-cmp",
        sum = "h1:crn/baboCvb5fXaQ0IJ1SGTsTVrWpDsCWC8EGETZijY=",
        version = "v0.3.0",
    )

    go_repository(
        name = "com_github_pmezard_go_difflib",
        importpath = "github.com/pmezard/go-difflib",
        sum = "h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_stretchr_objx",
        importpath = "github.com/stretchr/objx",
        sum = "h1:4G4v2dO3VZwixGIRoQ5Lfboy6nUhCyYzaqnIAPPhYs4=",
        version = "v0.1.0",
    )

    go_repository(
        name = "com_github_stretchr_testify",
        importpath = "github.com/stretchr/testify",
        sum = "h1:2E4SXV/wtOkTonXsotYi4li6zVWxYlZuYNCXe9XRJyk=",
        version = "v1.4.0",
    )
    go_repository(
        name = "in_gopkg_check_v1",
        importpath = "gopkg.in/check.v1",
        sum = "h1:yhCVgyC4o1eVCa2tZl7eS0r+SDo693bJlVdllGtEeKM=",
        version = "v0.0.0-20161208181325-20d25e280405",
    )
    go_repository(
        name = "in_gopkg_yaml_v2",
        importpath = "gopkg.in/yaml.v2",
        sum = "h1:ZCJp+EgiOT7lHqUV2J862kp8Qj64Jo6az82+3Td9dZw=",
        version = "v2.2.2",
    )

    go_repository(
        name = "org_bitbucket_creachadair_stringset",
        importpath = "bitbucket.org/creachadair/stringset",
        sum = "h1:L4vld9nzPt90UZNrXjNelTshD74ps4P5NGs3Iq6yN3o=",
        version = "v0.0.9",
    )
