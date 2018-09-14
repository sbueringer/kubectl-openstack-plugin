load("//bazel:workspace_mirror.bzl", "mirror")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
http_archive(
    name = "io_bazel_rules_go",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.15.3/rules_go-0.15.3.tar.gz"],
    sha256 = "97cf62bdef33519412167fd1e4b0810a318a7c234f5f8dc4f53e2da86241c492",
)
http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.14.0/bazel-gazelle-0.14.0.tar.gz"],
    sha256 = "c0a5739d12c6d05b6c1ad56f2200cb0b57c5a70e03ebd2f7b87ce88cabf09c7b",
)

http_archive(
    name = "io_kubernetes_build",
    sha256 = "1188feb932cefad328b0a3dd75b3ebd1d79dd26dbdd723f019ceb760e27ba6d8",
    strip_prefix = "repo-infra-84d52408a061e87d45aebf5a0867246bdf66d180",
    urls = mirror("https://github.com/kubernetes/repo-infra/archive/84d52408a061e87d45aebf5a0867246bdf66d180.tar.gz"),
)

http_archive(
    name = "bazel_skylib",
    sha256 = "bbccf674aa441c266df9894182d80de104cabd19be98be002f6d478aaa31574d",
    strip_prefix = "bazel-skylib-2169ae1c374aab4a09aa90e65efe1a3aad4e279b",
    urls = mirror("https://github.com/bazelbuild/bazel-skylib/archive/2169ae1c374aab4a09aa90e65efe1a3aad4e279b.tar.gz"),
)

load("@bazel_skylib//:lib.bzl", "versions")

versions.check(minimum_bazel_version = "0.16.0")

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

load("@io_bazel_rules_go//go:def.bzl", "go_download_sdk", "go_register_toolchains", "go_rules_dependencies")
load("@io_bazel_rules_docker//docker:docker.bzl", "docker_pull", "docker_repositories")

go_rules_dependencies()

go_register_toolchains(
    go_version = "1.11",
)

load("//bazel:workspace_mirror.bzl", "export_urls")

export_urls("workspace_urls")
