---
title: Snapshot single PVCs
sidebar_class_name: auto-sidebar-hide
keywords:
  - 'snapshots, sinple PVC, kubernetes, k8s'
description: >-
  Instructions on taking snapshots of single PVCs and restoring PVCs from the
  snapshots
series: k8s-local-snap
aliases:
  - >-
    /docs/portworx-enterprise/operations/operate-kubernetes/storage-operations/create-snapshots/on-demand/snaps-single-pvc/
sidebar_position: 800
---
import Comment from '/src/components/Comment'

This document will show you how to create a snapshot of a PVC backed by a Portworx volume.

## Creating snapshot within a single namespace

1. If you have a PVC called jenkins-home-jenkins-master-0, in the jenkins namespace, you can create a snapshot for that PVC by using the following spec:

    ```yaml
    apiVersion: v1
    kind: Namespace 
    metadata:
      name: jenkins
    ---
    apiVersion: volumesnapshot.external-storage.k8s.io/v1
    kind: VolumeSnapshot
    metadata:
      name: jenkins-home-jenkins-master-0
      namespace: jenkins
    spec:
      persistentVolumeClaimName: jenkins-home-jenkins-master-0
    ```

2. Once you apply the above object you can check the status of the snapshots using `kubectl`:

    ```bash
    kubectl get -n jenkins volumesnapshot
    ```

    ```output
    NAME                                                      AGE
    jenkins-jobs-jenkins-master-0-snapshot-2019-03-20-snap1   6m
    ```

    ```bash
    kubectl get -n jenkins volumesnapshotdatas
    ```

    ```output
    NAME                                                       AGE
    k8s-volume-snapshot-ab059f02-4b5e-11e9-bca9-0242ac110002   8m
    ```

3. The creation of the volumesnapshotdatas object indicates that the snapshot has
been created. If you describe the volumesnapshotdatas object you can see the
Portworx Volume Snapshot ID and the PVC for which the snapshot was created:

  ```bash
  kubectl describe volumesnapshotdatas
  ```

  ```output
  Name:         k8s-volume-snapshot-ab059f02-4b5e-11e9-bca9-0242ac110002
  Namespace:
  Labels:       <none>
  Annotations:  <none>
  API Version:  volumesnapshot.external-storage.k8s.io/v1
  Kind:         VolumeSnapshotData
  Metadata:
    Creation Timestamp:  2019-03-20T22:22:37Z
    Generation:          1
    Resource Version:    56596513
    Self Link:           /apis/volumesnapshot.external-storage.k8s.io/v1/volumesnapshotdatas/k8s-volume-snapshot-ab059f02-4b5e-11e9-bca9-0242ac110002
    UID:                 ab07a5c9-4b5e-11e9-9693-0cc47ab5f9a2
  Spec:
    Persistent Volume Ref:
      Kind:  PersistentVolume
      Name:  pvc-9b609a88-3f5e-11e8-83b6-0cc47ab5f9a2
    Portworx Volume:
      Snapshot Id:    411710013297550893
      Snapshot Type:  local
    Volume Snapshot Ref:
      Kind:  VolumeSnapshot
       Name:  jenkins/jenkins-jobs-jenkins-master-0-snapshot-2019-03-20-snap1-aa53d9a3-4b5e-11e9-9693-0cc47ab5f9a2
  Status:
    Conditions:
      Last Transition Time:  2019-03-20T22:22:37Z
      Message:               Snapshot created successfully and it is ready
      Reason:
      Status:                True
      Type:                  Ready
    Creation Timestamp:      <nil>
  Events:                    <none>
  ```

4. You can use the `storkctl` command to verify that the snapshot was created successfully:

    ```bash
    storkctl -n jenkins get snap
    ```

    ```output
    NAME                                                      PVC                             STATUS    CREATED               COMPLETED             TYPE
    jenkins-jobs-jenkins-master-0-snapshot-2019-03-20-snap1   jenkins-jobs-jenkins-master-0   Ready     20 Mar 19 15:22 PDT   20 Mar 19 15:22 PDT   local
    ```

For details about how you can restore a snapshot to a new PVC or the original PVC, see the [Restore snapshots](/operations/operate-kubernetes/storage-operations/create-snapshots/on-demand/snaps-local.md) section.

## Creating snapshots across namespaces

When creating snapshots, you can provide comma separated regexes with `stork.libopenstorage.org/snapshot-restore-namespaces` annotation to specify which namespaces the snapshot can be restored to.
When creating PVC from snapshots, if a snapshot exists in another namespace, the snapshot namespace should be specified with `stork.libopenstorage.org/snapshot-source-namespace` annotation.

Let's take an example where we have 2 namespaces _dev_ and _prod_. We will create a PVC and snapshot in the _dev_ namespace and then create a PVC in the _prod_ namespace from the snapshot.

1. Create the namespaces

    ```yaml
    apiVersion: v1
    kind: Namespace
    metadata:
      name: dev
      labels:
        name: dev
    ---
    apiVersion: v1
    kind: Namespace
    metadata:
      name: prod
      labels:
        name: prod
    ```

2. Create the PVC

    ```yaml
    kind: PersistentVolumeClaim
    apiVersion: v1
    metadata:
      name: mysql-data
      namespace: dev
      annotations:
        volume.beta.kubernetes.io/storage-class: px-mysql-sc
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 2Gi
    ---
    kind: StorageClass
    apiVersion: storage.k8s.io/v1
    metadata:
      name: px-mysql-sc
    provisioner: kubernetes.io/portworx-volume
    parameters:
      repl: "2"
    ```

3. Create the snapshot

    ```yaml
    apiVersion: volumesnapshot.external-storage.k8s.io/v1
    kind: VolumeSnapshot
    metadata:
      name: mysql-snapshot
      namespace: dev
      annotations:
        stork.libopenstorage.org/snapshot-restore-namespaces: "prod"
    spec:
      persistentVolumeClaimName: mysql-data

    ```

4. Create a PVC in a different namespace from the snapshot

    ```yaml
    apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      name: mysql-clone
      namespace: prod
      annotations:
        snapshot.alpha.kubernetes.io/snapshot: mysql-snapshot
        stork.libopenstorage.org/snapshot-source-namespace: dev
    spec:
      accessModes:
        - ReadWriteOnce
      storageClassName: stork-snapshot-sc
      resources:
        requests:
          storage: 2Gi
    ```

<Comment> 

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: jenkins
  labels:
    name: jenkins
---
apiVersion: v1
kind: Namespace
metadata:
  name: jenkins-prod
  labels:
    name: jenkins-prod
``` 
```yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: px-jenkins-sc
provisioner: kubernetes.io/portworx-volume
parameters:
  repl: "2"
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: jenkins-home-jenkins-master-0
  namespace: jenkins
  annotations:
    volume.beta.kubernetes.io/storage-class: px-jenkins-sc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi
``` 
```yaml
apiVersion: volumesnapshot.external-storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: jenkins-home-jenkins-master-0-snap
  namespace: jenkins
  annotations:
    stork.libopenstorage.org/snapshot-restore-namespaces: "jenkins-prod"
spec:
  persistentVolumeClaimName: jenkins-home-jenkins-master-0
```
```bash
kubectl get -n jenkins volumesnapshot jenkins-home-jenkins-master-0-snap
``` 
```bash
kubectl get -n jenkins volumesnapshotdatas
```
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: jenkins-clone
  namespace: jenkins-prod
  annotations:
    snapshot.alpha.kubernetes.io/snapshot: jenkins-home-jenkins-master-0-snap
    stork.libopenstorage.org/snapshot-source-namespace: jenkins
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: stork-snapshot-sc
  resources:
    requests:
      storage: 2Gi      
``` 
```bash
kubectl get pvc -n jenkins-prod
```
```bash
kubectl delete ns jenkins
```
```bash
kubectl delete ns jenkins-prod
``` 
</Comment>

