# Design notes

## Motivation

You cannot tell if you can successfully create PVs via the storage plugin until you try to create PVs.
So users may only notice the failure when they try to create PVs.

To avoid such a situation, the storage administrator should be aware of the problem before users realize it.

## Goal

- Verify that the storage driver is working properly
- Verify that the PV can be accessed
- Check for all specified StorageClasses
- Also check the storage plugin that creates node-local volumes on the specified node (e.g. TopoLVM)
- Output the monitoring results as Prometheus metrics

## Architecture

```mermaid
flowchart TB
    Prometheus[Prometheus, <br>VictoriaMetrics] -->|scrape| controller
    controller[controller]
    controller -->|create| cronjobA[CronJob] -->|create| probeAA
    controller -->|create| cronjobB[CronJob] -->|create| probeAB
    controller -->|create| cronjobC[CronJob] -->|create| probeBA
    controller -->|create| cronjobD[CronJob] -->|create| probeBB
    probeAA -->|use| volumeA[(Generic Ephemeral Volume)]
    probeAB -->|use| volumeB[(Generic Ephemeral Volume)]
    probeBA -->|use| volumeC[(Generic Ephemeral Volume)]
    probeBB -->|use| volumeD[(Generic Ephemeral Volume)]
    probeAA -->|post metrics| controller
    probeAB -->|post metrics| controller
    probeBA -->|post metrics| controller
    probeBB -->|post metrics| controller
    subgraph NodeA
        probeAA[Probe]
        probeAB[Probe]
    end
    subgraph NodeB
        probeBA[Probe]
        probeBB[Probe]
    end
    volumeA -.-|related| storageclassA[StorageClass A]
    volumeB -.-|related| storageclassB[StorageClass B]
    volumeC -.-|related| storageclassA[StorageClass A]
    volumeD -.-|related| storageclassB[StorageClass B]

    %% This is a workaround to make volumeA and volumeB closer.
    subgraph volumeAB [ ]
        volumeA
        volumeB
    end
    style volumeAB fill-opacity:0,stroke-width:0px

    %% This is a workaround to make volumeC and volumeD closer.
    subgraph volumeCD [ ]
        volumeC
        volumeD
    end
    style volumeCD fill-opacity:0,stroke-width:0px
```

### How pie works

pie works as follows:

1. The controller creates CronJobs for each node and StorageClass.
2. A CronJob periodically creates a probe pod.
3. A Probe pod requests to create a Generic Ephemeral Volume via the related StorageClass.
4. The controller monitors the pod creation events and measures how long it takes to create probe pods.
   (This indirectly measures the time required for volume provisioning.) Then it exposes the result as Prometheus metrics.
5. Once the prob pods are created, they try to read and write data from and to the Generic Ephemeral Volume, and measure the I/O latency. Then they post the result to the controller.
6. When the controller receives the requests from the probe pods, it exposes the result as Prometheus Metrics.

### Metrics design decision

As explained in [README.md](../README.md#prometheus-metrics), metrics related to PV creation are output in the form of whether the PV creation was completed within a certain time (denoted as `on_time` label of `create_probe_total`), not the time taken for the creation.

If you try to output the time taken to create a PV, the metrics would not be output until the PV is actually created.
Then, if the PV cannot be created due to some problems, the metric would not be output, and
you would not realize that there are some problems.

Therefore, if the PV is not created within a certain time, `create_probe_total` counter with `on_time=false` is incremented so that you can notice the problem even when the PV creation is completely stopped.

### Experimental Architecture using provision-probe and mount-probe

The current probe checks that both a new provisioning of a PV and its mounting succeed on every Node.
This guarantee is sufficient but not necessary; although mounting an already provisioned PV should succeed on every node, it is sufficient that a new provisioning succeeds on at least one Node.

To address the above issue, the new architecture has the following two types of probes:
- provision-probe, which checks that a new provision succeeds; and
- mount-probe, which checks that a PV (possibly already provisioned) can be successfully mounted on each Node.

In addition, PieProbe custom resource is introduced to group the probes that monitor a StorageClass. Each probe has an owner reference to a PieProbe and is GCed when the referenced PieProbe is deleted. See also issue [#50](https://github.com/topolvm/pie/issues/50).

```mermaid
flowchart TB
    Prometheus[Prometheus, <br>VictoriaMetrics] -->|scrape| controller
    controller[controller]
    controller -->|watch| pieprobeA[PieProbe<br />for StorageClass A] -->|own| ownedByPieProbeA
    controller -->|watch| pieprobeB[PieProbe<br />for StorageClass B] -->|own| ownedByPieProbeB
    controller -->|create| cronjobA[CronJob] -->|create| probeAA
    controller -->|create| cronjobB[CronJob] -->|create| probeAB
    controller -->|create| cronjobC[CronJob] -->|create| probeBA
    controller -->|create| cronjobD[CronJob] -->|create| probeBB
    controller -->|create| cronjobE[CronJob] -->|create| probeProvisionA
    controller -->|create| cronjobF[CronJob] -->|create| probeProvisionB
    probeAA -->|use| volumeA[(PersistentVolume)]
    probeAB -->|use| volumeB[(PersistentVolume)]
    probeBA -->|use| volumeC[(PersistentVolume)]
    probeBB -->|use| volumeD[(PersistentVolume)]
    probeProvisionA -->|use| volumeE[(Generic Ephemeral Volume)]
    probeProvisionB -->|use| volumeF[(Generic Ephemeral Volume)]
    probeAA -->|post metrics| controller
    probeAB -->|post metrics| controller
    probeBA -->|post metrics| controller
    probeBB -->|post metrics| controller
    probeProvisionA -->|post metrics| controller
    probeProvisionB -->|post metrics| controller
    subgraph ownedByPieProbeA[ ]
        subgraph Node A
            probeAA[mount-probe]
        end
        subgraph Node B
            probeBA[mount-probe]
        end
        cronjobA
        cronjobC
        cronjobE
        probeProvisionA[provision-probe]

        volumeA
        volumeC
        volumeE
    end
    subgraph ownedByPieProbeB[ ]
        subgraph Node A
            probeAB[mount-probe]
        end
        subgraph Node B
            probeBB[mount-probe]
        end
        cronjobB
        cronjobD
        cronjobF
        probeProvisionB[provision-probe]

        volumeB
        volumeD
        volumeF
    end
    volumeA -.-|related| storageclassA[StorageClass A]
    volumeB -.-|related| storageclassB[StorageClass B]
    volumeC -.-|related| storageclassA[StorageClass A]
    volumeD -.-|related| storageclassB[StorageClass B]
    volumeE -.-|related| storageclassA[StorageClass A]
    volumeF -.-|related| storageclassB[StorageClass B]
```

Each probe works as follows:
- provision-probe:
  1. The controller creates a provision-probe CronJob for each PieProbe.
  2. The CronJob periodically creates a provision-probe Pod.
  3. The Pod requests the creation of a Generic Ephemeral Volume via the related StorageClass.
  4. The controller monitors the Pod creation events and measures how long it takes to create the Pod.
  (This indirectly measures the time required for provisioning the volume.) Then it exposes the result as Prometheus metrics.
  5. Once the provision-probe Pod is created, it immediately exits normally. 
- mount-probe:
  1. The controller creates a mount-probe CronJob and a PVC for each Node and PieProbe.
  2. The CronJob periodically creates a mount-probe Pod.
  3. If the PVC is not yet bound, the Pod requests to provision a PV via the related StorageClass. Then, the Pod mounts the PV.
  4. The controller monitors the Pod creation events and measures how long it takes to create the Pod.
  (This indirectly measures the time required for mounting the volume.) Then it exposes the result as Prometheus metrics.
  5. Once the Pod is created, it tries to read and write data from and to the PV, and measures the I/O latency. Then it posts the result to the controller and exists normally.
  6. When the controller receives the request from the mount-probe Pod, it exposes the result as Prometheus metrics.
