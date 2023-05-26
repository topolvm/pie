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

- `Makefile`
- `README.md`
- `e2e/Makefile`
- `.github/workflows/e2e.yaml`

We should also update go.mod by the following commands. Please note that Kubernetes v1 corresponds with v0 for the release tags. For example, v1.17.2 corresponds with the v0.17.2 tag.

```bash
$ VERSION=<upgrading Kubernetes release version>
$ go get k8s.io/api@v${VERSION} k8s.io/apimachinery@v${VERSION} k8s.io/client-go@v${VERSION}
```

Read the [`controller-runtime`'s release note](https://github.com/kubernetes-sigs/controller-runtime/releases), and update to the newest version that is compatible with all supported kubernetes versions. If there are breaking changes, we should decide how to manage these changes.

```
$ VERSION=<upgrading controller-runtime version>
$ go get sigs.k8s.io/controller-runtime@v${VERSION}
```

Read the [`controller-tools`'s release note](https://github.com/kubernetes-sigs/controller-tools/releases), and update to the newest version that is compatible with all supported kubernetes versions. If there are breaking changes, we should decide how to manage these changes.
To change the version, edit `Makefile`. 

#### Go

Choose the same version of Go [used by the latest Kubernetes](https://github.com/kubernetes/kubernetes/blob/master/go.mod) supported by pie.

Edit the following files.

- go.mod
- Dockerfile

#### Depending tools

The following tools do not depend on other software, use latest versions.
- [kind](https://github.com/kubernetes-sigs/kind/releases)
  - To change the version, edit the following files.
    - `e2e/Makefile`
- [helm](https://github.com/helm/helm/releases)
  - To change the version, edit the following files.
    - `e2e/Makefile`
- [kustomize](https://github.com/kubernetes-sigs/kustomize/releases)
  - To change the version, edit the following files.
    - `Makefile`

#### Depending modules

Read `kubernetes' go.mod`(https://github.com/kubernetes/kubernetes/blob/\<upgrading Kubernetes release version\>/go.mod), and update the `prometheus/*` modules. Here is the example to update `prometheus/client_golang`.

```
$ VERSION=<upgrading prometheus-related libraries release version>
$ go get github.com/prometheus/client_golang@$VERSION
```

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

`git grep <the kubernetes version which support will be dropped>`, `git grep image:`, and `git grep -i VERSION` might help to avoid overlooking necessary changes.
