CHART_TESTING_VERSION := 3.10.1
CONTROLLER_TOOLS_VERSION := v0.13.0
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
# NOTE: the suffix .x means wildcard match so specifying the latest patch version.
ENVTEST_K8S_VERSION := 1.28.x
HELM_VERSION := 3.14.0
KIND_VERSION := v0.20.0
# It is set by CI using the environment variable, use conditional assignment.
KUBERNETES_VERSION ?= 1.28
KUSTOMIZE_VERSION := v5.3.0

# The container version of kind must be with the digest.
# ref. https://github.com/kubernetes-sigs/kind/releases
KIND_NODE_VERSION := kindest/node:v1.28.0@sha256:b7a4cad12c197af3ba43202d3efe03246b3f0793f162afb40a33c923952d5b31
ifeq ($(KUBERNETES_VERSION), 1.27)
	KIND_NODE_VERSION := kindest/node:v1.27.3@sha256:3966ac761ae0136263ffdb6cfd4db23ef8a83cba8a463690e98317add2c9ba72
else ifeq ($(KUBERNETES_VERSION), 1.26)
	KIND_NODE_VERSION := kindest/node:v1.26.6@sha256:6e2d8b28a5b601defe327b98bd1c2d1930b49e5d8c512e1895099e4504007adb
endif
