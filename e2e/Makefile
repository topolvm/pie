KUBERNETES_VERSION ?= 1.25
# The container version of kind must be with the digest.
# ref. https://github.com/kubernetes-sigs/kind/releases
KIND_NODE_VERSION := kindest/node:v1.25.3@sha256:f52781bc0d7a19fb6c405c2af83abfeb311f130707a0e219175677e366cc45d1
KIND_CLUSTER_NAME := pie-test
KIND_VERSION := v0.17.0
HELM_VERSION := 3.10.2

PROJECT_ROOT := $(shell git rev-parse --show-toplevel)
BINDIR := $(PROJECT_ROOT)/bin

CURL := curl -sSLf

KIND := $(BINDIR)/kind-$(KIND_VERSION)
HELM := $(BINDIR)/helm-$(HELM_VERSION)

ifeq ($(KUBERNETES_VERSION), 1.24)
	KIND_NODE_VERSION := kindest/node:v1.24.7@sha256:577c630ce8e509131eab1aea12c022190978dd2f745aac5eb1fe65c0807eb315
else ifeq ($(KUBERNETES_VERSION), 1.23)
	KIND_NODE_VERSION := kindest/node:v1.23.13@sha256:ef453bb7c79f0e3caba88d2067d4196f427794086a7d0df8df4f019d5e336b61
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
