User manual
===========

This is the user manual for pie.

**Table of contents**

- [Stop and start the pie](#stop-and-start-the-pie)
  - [Stop the pie](#stop-the-pie)
  - [Start the pie](#start-the-pie)

Stop and start the pie
----------------------

### Stop the pie

To stop the pie, follow these steps:

1. Set the number of replicas of Deployments to 0.
   ```console
   $ NAMESPACE=<Set the namespace of the pie.>
   $ kubectl -n ${NAMESPACE} scale deployments --replicas=0 pie
   ```
2. Wait for Pods to stop.
   ```console
   $ watch kubectl -n ${NAMESPACE} get pods
   ```
3. Delete CronJobs (if exists).
   ```console
   $ CRONJOBS_LIST=$(kubectl -n ${NAMESPACE} get cronjobs --no-headers=true | awk '{print $1}')
   $ [ -n "${CRONJOBS_LIST}" ] && kubectl -n ${NAMESPACE} delete cronjobs ${CRONJOBS_LIST}
   ```

### Start the pie

To start the pie that was stopped in the above steps, follow these steps:

1. Set the number of replicas of Deployments to the original value (ex. 1).
   ```console
   $ NAMESPACE=<Set the namespace of the pie.>
   $ kubectl -n ${NAMESPACE} scale deployments --replicas=1 pie
   ```
2. Wait for Pods to be ready.
   ```console
   $ watch kubectl -n ${NAMESPACE} get pods
   ```
