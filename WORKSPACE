load("//bazel:workspace_mirror.bzl", "mirror")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "7be7dc01f1e0afdba6c8eb2b43d2fa01c743be1b9273ab1eaf6c233df078d705",
    urls = mirror("https://github.com/bazelbuild/rules_go/releases/download/0.16.5/rules_go-0.16.5.tar.gz"),
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "c0a5739d12c6d05b6c1ad56f2200cb0b57c5a70e03ebd2f7b87ce88cabf09c7b",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.16.0/bazel-gazelle-0.16.0.tar.gz"],
)

http_archive(
    name = "io_kubernetes_build",
    sha256 = "21160531ea8a9a4001610223ad815622bf60671d308988c7057168a495a7e2e8",
    strip_prefix = "repo-infra-b4bc4f1552c7fc1d4654753ca9b0e5e13883429f",
    urls = mirror("https://github.com/kubernetes/repo-infra/archive/b4bc4f1552c7fc1d4654753ca9b0e5e13883429f.tar.gz"),
)

http_archive(
    name = "bazel_skylib",
    sha256 = "bbccf674aa441c266df9894182d80de104cabd19be98be002f6d478aaa31574d",
    strip_prefix = "bazel-skylib-2169ae1c374aab4a09aa90e65efe1a3aad4e279b",
    urls = mirror("https://github.com/bazelbuild/bazel-skylib/archive/2169ae1c374aab4a09aa90e65efe1a3aad4e279b.tar.gz"),
)

load("@bazel_skylib//:lib.bzl", "versions")

versions.check(minimum_bazel_version = "0.17.2")

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

load("@io_bazel_rules_go//go:def.bzl", "go_download_sdk", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(
    go_version = "1.11",
)

load("//bazel:workspace_mirror.bzl", "export_urls")

export_urls("workspace_urls")
