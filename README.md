TODO: add badges

# pie
An application that monitors the availability of Kubernetes storage in end-to-end manner.

## Description

pie verifies that PVs are successfully provisioned on the specified nodes for the specified storage classes and that the PVs can be successfully accessed. It outputs the results as metrics.

## Getting Started
### Running on the cluster

1. Create values.yaml. At least the following setting is mandatory.

    ```yaml
    controller:
      monitoringStorageClasses: [<storage_classes_to_be_monitored>]
    ```

2. Then you can install it using Helm.

    ```sh
    helm repo add pie
    helm install pie --values values.yaml
    ```

## Prometheus metrics
### `io_write_latency_seconds`
IO latency of write.

TYPE: gauge

### `io_read_latency_seconds`
IO latency of read.

TYPE: gauge

### `create_probe_fast_total`
The number of attempts that take less time between the creation of the Pod object and the creation of the container than the threshold.

TYPE: counter

### `create_probe_slow_total`
The number of attempts that take more time between the creation of the Pod object and the creation of the container than the threshold.

TYPE: counter

## Contributing

### Test It Out
1. Run unit tests.
    ```sh
    make test
    ```

2. Run e2e test on the local cluster.
    ```sh
    make -C e2e create-cluster
    make -C e2e test
    ```
