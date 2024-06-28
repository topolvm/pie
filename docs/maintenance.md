Maintenance guide
=================

How to change the supported Kubernetes minor versions
-------------------------------------------

pie depends on some Kubernetes repositories like `k8s.io/client-go` and should support 3 consecutive Kubernetes versions at a time.

Issues and PRs related to the last upgrade task also help you understand how to upgrade the supported versions,
so checking them together with this guide is recommended when you do this task.

### Upgrade procedure

We should write down in the github issue of this task what are the important changes and the required actions to manage incompatibilities if exist.
The format is up to you.

Basically, we should pay attention to breaking changes and security fixes first.

#### Kubernetes

Choose the next version and check the [release note](https://kubernetes.io/docs/setup/release/notes/). e.g. 1.17, 1.18, 1.19 -> 1.18, 1.19, 1.20

To change the version, edit the following files.

- `.github/workflows/e2e.yaml`
- `README.md`
- `versions.mk`

We should also update go.mod by the following commands. Please note that Kubernetes v1 corresponds with v0 for the release tags. For example, v1.17.2 corresponds with the v0.17.2 tag.

```bash
$ VERSION=<upgrading Kubernetes release version>
$ go get k8s.io/api@v${VERSION} k8s.io/apimachinery@v${VERSION} k8s.io/client-go@v${VERSION} k8s.io/component-helpers@v${VERSION}
```

Read the [`controller-runtime`'s release note](https://github.com/kubernetes-sigs/controller-runtime/releases), and update to the newest version that is compatible with all supported kubernetes versions. If there are breaking changes, we should decide how to manage these changes.

```
$ VERSION=<upgrading controller-runtime version>
$ go get sigs.k8s.io/controller-runtime@v${VERSION}
```

Read the [`controller-tools`'s release note](https://github.com/kubernetes-sigs/controller-tools/releases), and update to the newest version that is compatible with all supported kubernetes versions. If there are breaking changes, we should decide how to manage these changes.
To change the version, edit `versions.mk`.

#### Go

Choose the same version of Go [used by the latest Kubernetes](https://github.com/kubernetes/kubernetes/blob/master/go.mod) supported by pie.

Edit the following files.

- `go.mod`
- `Dockerfile`

#### Depending tools

The following tools do not depend on other software, use latest versions.
To change their versions, edit `versions.mk`.
- [kind](https://github.com/kubernetes-sigs/kind/releases)
    - Update `KIND_NODE_VERSION` in `versions.mk`, too.
- [helm](https://github.com/helm/helm/releases)
- [kustomize](https://github.com/kubernetes-sigs/kustomize/releases)
- [chart-testing](https://github.com/helm/chart-testing/releases)

#### Depending modules

Read `kubernetes' go.mod`(https://github.com/kubernetes/kubernetes/blob/<upgrading Kubernetes release version\>/go.mod), and update the `prometheus/*` modules. Here is the example to update `prometheus/client_golang`.

```
$ VERSION=<upgrading prometheus-related libraries release version>
$ go get github.com/prometheus/client_golang@v${VERSION}
```

The following modules don't depend on other softwares, so use their latest versions:
- [github.com/onsi/ginkgo/v2](https://github.com/onsi/ginkgo/releases)
- [github.com/onsi/gomega](https://github.com/onsi/gomega/releases)
- [github.com/spf13/cobra](https://github.com/spf13/cobra/releases)
- [k8s.io/klog/v2](https://github.com/kubernetes/klog/releases)
- [sigs.k8s.io/yaml](https://github.com/kubernetes-sigs/yaml/releases)

Then, please tidy up the dependencies.

```bash
$ go mod tidy
```

Regenerate manifests using new controller-tools.

```console
$ make manifests
$ make generate
```

#### Final check

`git grep <the kubernetes version which support will be dropped>`, `git grep image:`, `git grep -i VERSION` and looking `versions.mk` might help to avoid overlooking necessary changes.
