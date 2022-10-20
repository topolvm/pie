KIND_NODE_VERSION := v1.24.6@sha256:97e8d00bc37a7598a0b32d1fabd155a96355c49fa0d4d4790aab0f161bf31be1
KIND_CLUSTER_NAME := pie-test
# https://github.com/kubernetes-sigs/kind/releases
KIND_VERSION := v0.16.0
HELM_VERSION := 3.10.1

PROJECT_ROOT := $(shell git rev-parse --show-toplevel)
BINDIR := $(PROJECT_ROOT)/bin

CURL := curl -sSLf

KIND := $(BINDIR)/kind-$(KIND_VERSION)
HELM := $(BINDIR)/helm-$(HELM_VERSION)

.PHONY: create-cluster
create-cluster: $(KIND) $(HELM)
	$(KIND) create cluster --name $(KIND_CLUSTER_NAME) --image kindest/node:$(KIND_NODE_VERSION)
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
