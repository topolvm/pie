KUBERNETES_VERSION ?= 1.27
# The container version of kind must be with the digest.
# ref. https://github.com/kubernetes-sigs/kind/releases
KIND_NODE_VERSION := kindest/node:v1.27.1@sha256:b7d12ed662b873bd8510879c1846e87c7e676a79fefc93e17b2a52989d3ff42b
KIND_CLUSTER_NAME := pie-test
KIND_VERSION := v0.19.0
HELM_VERSION := 3.12.0

PROJECT_ROOT := $(shell git rev-parse --show-toplevel)
BINDIR := $(PROJECT_ROOT)/bin

CURL := curl -sSLf

KIND := $(BINDIR)/kind-$(KIND_VERSION)
HELM := $(BINDIR)/helm-$(HELM_VERSION)

ifeq ($(KUBERNETES_VERSION), 1.26)
	KIND_NODE_VERSION := kindest/node:v1.26.4@sha256:f4c0d87be03d6bea69f5e5dc0adb678bb498a190ee5c38422bf751541cebe92e
else ifeq ($(KUBERNETES_VERSION), 1.25)
	KIND_NODE_VERSION := kindest/node:v1.25.9@sha256:c08d6c52820aa42e533b70bce0c2901183326d86dcdcbedecc9343681db45161
endif

.PHONY: create-cluster
create-cluster: $(KIND) $(HELM)
	$(KIND) create cluster --name $(KIND_CLUSTER_NAME) --image $(KIND_NODE_VERSION)
	$(MAKE) -C ../ docker-build IMG=pie:dev
	$(KIND) load docker-image pie:dev --name $(KIND_CLUSTER_NAME)
	$(MAKE) $(KUBECONFIG)
	KUBECONFIG=$(KUBECONFIG) $(HELM) install --create-namespace -n e2e pie ../charts/pie/ -f values.yaml

.PHONY: test
test: $(KUBECONFIG)
	E2ETEST=true TEST_NAMESPACE=e2e  KUBECONFIG=$(KUBECONFIG) go test -v

KUBECONFIG := $(shell pwd)/.kubeconfig
.PHONY: $(KUBECONFIG)
$(KUBECONFIG): |$(KIND)
	$(KIND) export kubeconfig --name $(KIND_CLUSTER_NAME) --kubeconfig=$@

$(BINDIR):
	mkdir $@

$(KIND): |$(BINDIR)
	GOBIN=$(BINDIR) go install sigs.k8s.io/kind@$(KIND_VERSION)
	mv $(BINDIR)/kind $@

$(HELM): |$(BINDIR)
	$(CURL) https://get.helm.sh/helm-v$(HELM_VERSION)-linux-amd64.tar.gz \
		| tar xvz -C $(BINDIR) --strip-components 1 linux-amd64/helm
	mv $(BINDIR)/helm $@

.PHONY: clean
clean: $(KIND)
	rm -f $(KUBECONFIG)
	$(KIND) delete cluster --name $(KIND_CLUSTER_NAME)
