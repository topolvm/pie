![GitHub release (latest by date)](https://img.shields.io/github/v/release/topolvm/pie?cacheSeconds=3600)
[![Main](https://github.com/topolvm/pie/actions/workflows/main.yaml/badge.svg)](https://github.com/topolvm/pie/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/topolvm/pie.svg)](https://pkg.go.dev/github.com/topolvm/pie)
[![Go Report Card](https://goreportcard.com/badge/github.com/topolvm/pie)](https://goreportcard.com/report/github.com/topolvm/pie)

# pie
An application that monitors the availability of Kubernetes storage in end-to-end manner.

## Description

pie verifies that PVs are successfully provisioned on the specified nodes for the specified storage classes and that the PVs can be successfully accessed. It outputs the results as metrics.

## Supported environments

- Kubernetes: 1.29, 1.28, 1.27

## Getting Started
### Running on the cluster

1. Create values.yaml. At least the following setting is mandatory.

    ```yaml
    controller:
      monitoringStorageClasses: [<storage_classes_to_be_monitored>]
    ```

2. Then you can install it using Helm.

    ```sh
    helm repo add pie https://topolvm.github.io/pie
    helm install pie --values values.yaml
    ```

## Prometheus metrics
### `pie_io_write_latency_seconds`
IO latency of write.

TYPE: gauge

### `pie_io_read_latency_seconds`
IO latency of read.

TYPE: gauge

### `pie_create_probe_total`
The number of attempts of the creation of the Pod object and the creation of the container.

TYPE: counter

### `pie_performance_probe_total`
The number of attempts of performing the IO benchmarks.

TYPE: counter

### `pie_io_write_latency_on_mount_probe_seconds`

_Experimental metrics._ IO latency of write, benchmarked on mount-probe Pods.

TYPE: gauge

### `pie_io_read_latency_on_mount_probe_seconds`

_Experimental metrics._ IO latency of read, benchmarked on mount-probe Pods.

TYPE: gauge

### `pie_mount_probe_total`

_Experimental metrics._ The number of attempts of the creation of the mount-probe Pod object and the creation of the container.

TYPE: counter

### `pie_performance_on_mount_probe_total`

_Experimental metrics._ The number of attempts of performing the IO benchmarks on mount-probe Pods.

TYPE: counter

### `pie_provision_probe_total`

_Experimental metrics._ The number of attempts of the creation of the provision-probe Pod object and the creation of the container.

TYPE: counter

## Contributing

### Test It Out
1. Run unit tests.
    ```sh
    make test
    ```

2. Run e2e test on the local cluster.
    ```sh
    make -C test/e2e create-cluster
    make -C test/e2e test
    ```
