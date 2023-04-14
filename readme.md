# Namespace ResourceQuota Autoscaling

## About
The initial open-source release of this project is provided as-is. That implies that the codebase is only a slightly
modified version of what we at ING are using. In the future we would like to extend this Open Source repository with
the appropriate pipelines and contribution mechanics. This does imply that your journey for using this project for your
own purposes can use some improvement, and we will work on that.

This project is part of the ING Neoria. Neoria contains parts of the ING Container Hosting Platform (ICHP) stack
which is used to deliver Namespace-as-a-Service on top of OpenShift.

## The concept

This project is intended to be used for multi-tenant clusters, where each tenant of that cluster owns a namespace that
has limited resources via a ResourceQuota. The idea of Quota Autoscaling is to have Namespace ResourceQuota resize 
requests automatically issued to a resize endpoint based on resource usage of Pods. This endpoint should handle the
charging of that namespace and also the resize of that ResourceQuota. Basically, scaling Pods will trigger resize
requests for a namespaced ResourceQuota.

**You should change the resize endpoint of this program for it to function, read more below. A custom certificate 
for a custom resize API endpoint can be added in build/tls-ca-bundle.pem.**.

In the current implementation the QuotaScaler expects a 10x ratio between Namespace ResourceQuota CPU Requests and Limits.
Its aim is to encourage users to create burstable Pods. This value is a constant `REQ_LIM_RATIO` in `internal/watchers.go`

## Installation
In a nutshell:
- clone this repository
- implement your resize API in `internal/resize_api.go`
- build your own image
- update values file with that image name
- helm deploy to your target cluster

In `internal/resize_api.go` you can find the function pointer:
```go
var ResizeApiFunc = InvokeResizeApiStub // func(ns NamespaceResizeEvent) error
```
The `ResizeApiFunc` will be called with information about a resize when this namespace needs more/less resources.
By default, the function `InvokeResizeApiStub` is called. This stub function just patches the Namespace ResourceQuota
directly so that the component is functional without modification. In practise, this function should be replaced with
a custom Resize API functionality that resizes the ResourceQuota, and does charging for your stack. An example of this
is leftover in the `InvokeResizeApi` function, which calls an ING specific stack. A custom certificate
for a custom resize API endpoint can be added in build/tls-ca-bundle.pem (mounted under `/etc/pki/tls/certs/ca-bundle.crt`).

## Key features
- Offers a namespaced QuotaAutoscaler custom resource
- Operator monitors ResourceQuotas
- Operator monitors FailedCreate Pod Events
- Operator calls a (custom) resize endpoint based on QuotaAutoscaler defined behavior

## RBAC

The QuotaScaler needs the following cluster-scoped permissions:
- `watch, list` on `ichp.ing.net/quotaautoscalers` to be able to operate on the CRD
- `watch, list, get, patch` on `resourcequotas` to monitor namespace resource limits. Patch is needed for stub resize function, can be removed after custom resize API implementation.
- `get` on `replicasets, replicationcontrollers, statefulsets, daemonsets, jobs` to find out required resources after Pod `FailedCreate` event.
- `watch, list, get, create` on `events` to monitor Pod `FailedCreate` events, and to (optionally) produce resize events in the namespace.

## Quota-scaler usage for tenants

Users of a multi-tenant cluster can enable this functionality in their Namespace via an instance of
a QuotaAutoscaler custom resource, e.g. for a namespace called `saca-dev`:

```yaml

apiVersion: ichp.ing.net/v1
kind: QuotaAutoscaler
metadata:
  name: saca-dev-scaler
  namespace: saca-dev
spec:
  behavior: # Optional
    scaleDown:
      policies:
      - method: cpu
        value: 100
      - method: memory
        value: 100
  scaleUp:
    policies:
    - method: cpu
      value: 100
    - method: memory
      value: 100
  minCpu: "400m"      # Optional, this is the default and minimal value
  maxCpu: "35"        # Optional, this is the default and maximum value
  minMemory: "1G"     # Optional, this is the default and minimal value
  maxMemory: "150G"   # Optional, this is the default and maximum value
  resourceQuota: saca-dev-quota # Your ResourceQuota (note: NOT an object quota)
```

The QuotaAutoscaler watches a QuotaScaler object in your namespace.
Changing the min/max CPU and Memory values will act just as updating values
in a namespaced ResourceQuota (via the resize endpoint that you configure yourself).

On top of that, you can also specify a behavior. A behavior specifies in
what percentage of usage you would like to keep. For example setting a
scaleDown policy of 50 CPU and scaleUp policy of 70 CPU means that the
QuotaAutoscaler will keep your namespace ResourceQuota CPU between
50-70% usage. If your have less Pods running in your namespace the
QuotaScaler will resize your namespace so that the quota is at least
used for 50%. On the other hand, if you add more pods, and you use more
than 70%, the QuotaAutoscaler will scale up your namespace so that
maximally 70% if your quota is used.

Setting all scaleDown and scaleUp policies to 100% will ensure that the
namespace ResourceQuota is always fully utilized, there will not be any
“unspent” resources. The QuotaAutoscaler will detect when Pods cannot be
scheduled ( quota limit exceeded error message), and scale up your quota
based on those events. This is the value used when the QuotaScaler
object is generated for you when a new namespace is created.

The purpose of the QuotaAutoscaler is to dynamically charge based on the
resource usage of Pods without manual intervention. This opens up the
floor for scaling mechanisms such as Horizontal Pod Autoscalers.

DaemonSets are currently not supported by the QuotaAutoscaler.

## FAQ

### What is a ResourceQuota?

A ResourceQuota defines the maximum amount of resources that the Pods in
your namespaces can consume. The sum of all Pod resources cannot exceed
the ResourceQuota.

### The QuotaAutoscaler does not scale down?

Possible causes are:

-  Failure in resize endpoint
-  A resize request has already been issued recently. You can
   only scale down after a couple of minutes of inactivity with the QuotaAutoscaler.
   There is no time limitation on scaling up.

### How does this impact my prod deployments?

For production namespaces ING recommends to have a 50-70% behavior
policy in the QuotaAutoscaler. This will allow you to do rolling updates
without impact from Quota scaling. In addition, such a behavior policy
allows you to increase the ReplicaCount (Horizontal Pod Autoscaling) of
your Deployments without impact from Quota scaling. If you want
guarantees of a minimal ResourceQuota, you can configure this in the
QuotaAutoscaler object.


## Known issues

- Pod FailedCreate events are always added to the maximum quota, this may result in an excessive quota.
- `daemonsets` are not supported

## Integration tests

Integrations tests were pruned from this repository because they were too specific to ING's stack.