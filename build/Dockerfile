FROM ubuntu:18.04

RUN apt-get update \
  && apt-get install -y wget curl unzip zip git-core python build-essential jq python-pip sudo iptables vim \
  && rm -rf /var/lib/apt/lists/*

ENV BAZEL_VERSION 0.21.0

RUN wget https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-installer-linux-x86_64.sh \
  && chmod +x bazel-${BAZEL_VERSION}-installer-linux-x86_64.sh \
  && ./bazel-${BAZEL_VERSION}-installer-linux-x86_64.sh \
  && bazel version \
  && rm -rf /root/.cache bazel-${BAZEL_VERSION}-installer-linux-x86_64.sh
