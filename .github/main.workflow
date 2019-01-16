workflow "Build" {
  on = "push"
  resolves = ["bazel"]
}

action "bazel" {
  uses = "docker://sbueringer/bazel"
  runs = "bazel"
  args = "build //cmd/kubectl-os:kubectl-os"
}
