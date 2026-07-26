# Architecture

## Overview

ConfigMapWatcher is a Kubernetes operator built with Kubebuilder and controller-runtime.
It watches ConfigMapWatcher custom resources and related ConfigMaps, then sends event payloads to an external HTTP endpoint.

## API Group

- Group: apps.devops
- Version: v1alpha1
- Kind: ConfigMapWatcher

## Reconcile Flow

1. Read ConfigMapWatcher resource.
2. Read referenced ConfigMap from spec.configMapName and spec.configMapNamespace.
3. Compare ConfigMap resourceVersion with status.lastConfigMapVersion.
4. If unchanged, skip sending and wait for next reconcile.
5. Build event payload and send to spec.eventEndpoint.
6. Update status and Synced condition.

## Status Model

- lastEventStatus: Success or Failed
- lastSyncTime: time of successful sync
- lastConfigMapVersion: latest processed version
- lastEventSent: time of successful send
- observedGeneration: generation reflected by status
- conditions:
  - Type: Synced
  - Reason: SendSuccess or SendFailed

## Retry Strategy

Status update uses client-go RetryOnConflict with DefaultBackoff.
On each retry attempt:

1. Re-fetch latest resource.
2. Re-apply status fields and condition.
3. Call Status().Update.

This prevents transient 409 Conflict errors when the same resource is updated concurrently.

## Watches

- Primary watch: ConfigMapWatcher resources.
- Secondary watch: ConfigMap resources via map function.

## Current Behavior Notes

- Endpoint success is defined as HTTP 200.
- Endpoint failures return reconcile error and trigger requeue after 30 seconds.
- Missing ConfigMapWatcher or ConfigMap is treated as non-fatal and ignored.
