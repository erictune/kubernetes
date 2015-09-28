<!-- BEGIN MUNGE: UNVERSIONED_WARNING -->

<!-- BEGIN STRIP_FOR_RELEASE -->

<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">

<h2>PLEASE NOTE: This document applies to the HEAD of the source tree</h2>

If you are using a released version of Kubernetes, you should
refer to the docs that go with that version.

<strong>
The latest 1.0.x release of this document can be found
[here](http://releases.k8s.io/release-1.0/docs/proposals/capabilities.md).

Documentation for other releases can be found at
[releases.k8s.io](http://releases.k8s.io).
</strong>
--

<!-- END STRIP_FOR_RELEASE -->

<!-- END MUNGE: UNVERSIONED_WARNING -->

# Capabilities Proposal

**Intended for discussion.  No need to merge or provide grammar/spelling/style comments at this time.*

Purpose of document:
1. Brian asked me to consider a "capabilities" model for security policy.  We have internal experience with that model.
1. Compare with OpenShift authorization model


## Capabilities Model

An internal capability mode that some of use have experience has these properties (heavily):
- ~50 capabilities
- Users create Pod-like objects that have the equivalent of a "pod.spec.serviceAccount".
- Users have permission to create Pods a those ServiceAccounts.
- ServiceAccounts imply a specific unix uid.
- ServiceAcccounts may have Capabilities.
- Some Capabilities map directly to Linux Kernel Capabilities.  Some are higher level concepts.

If we implement the *Capabilities* model, a Request would go through this
modified control flow:
1. U1 sends POST to `/api/v1/namespace/foo/pods/bar` where `spec.securityContext.privileged` is true.
1. A new CapabilitiesNeededDetector module in apiserver maps each request to a set of Capabilities
   needed to complete the request.  In this case `capabilitiesNeeded = []string{"CapabilityPrivilegedPod"}`.
1. The Authorization plugin is called.  As before, `authorization.Attributes` (see
   [here](../../pkg/auth/authorizer/interfaces.go))
   is filled in with the user, the Kind of the object, that the request is writing, etc.
   Additionally, now, the capabilitiesNeeded is also passed in.
1. As today, the Authorizer routine decides if the request should be allowed to proceed, based on all
   available attributes.


## Key use case: privileged

Use case: Allow user U1 (or all users in namespace N1) to create pods with
`spec.securityContext.privileged = true`; Do not allow user U2 (or all users in namespace
N2) to create pods with `spec.securityContext.privileged = true`.

### In OpenShift today

To implement the use case in OpenShift today you would create
this object:

```
kind: securityContextConstraints
metadata:
  name: allowU1ToMakePrivCont
a SecurityContextConstraints object  with `AllowPrivilegedContainer`

Part of this is proposed for OSS Kubernetes in #7893.

https://github.com/kubernetes/kubernetes/pull/7893/files
Two aspects to this: 
- how to refer (in speech and in config) to a pod with this bit set.
- specific authorication policy about who can create pods with this bit set.


### Common Control Flow with Capabilities

If we implement the *Capabilities* model, a Request would go through this
modified control flow:
1. U1 sends POST to `/api/v1/namespace/foo/pods/bar` where `spec.securityContext.privileged` is true.
1. A new CapabilitiesNeededDetector module in apiserver maps each request to a set of Capabilities
   needed to complete the request.  In this case `capabilitiesNeeded = []string{"CapabilityPrivilegedPod"}`.
1. The Authorization plugin is called.  As before, `authorization.Attributes` (see
   [here](https://github.com/kubernetes/kubernetes/blob/master/pkg/auth/authorizer/interfaces.go))
   is filled in with the user, the Kind of the object, that the request is writing, etc.
   Additionally, now, the capabilitiesNeeded is also passed in.
1. As today, the Authorizer routine decides if the request should be allowed to proceed, based on all
   available attributes.

Note that regardless of which Authorizer plugin is used, the set of _required_ capabilities
for the request is generated.  However, whether a given request also _can excercise_ those
capabilities is an implementation detail of the Authorizer implementation, of which there may 
be several.

### Authorizer Implementation Differences

This proposal is not specifying, and so a specific Authorizer implementation may differ in
how (or if it even supports):
- capabilities are given to a given user, group, or service account.
- which capabilities can or cannot be used in a cluster.
- which capabilities can or cannot be used in a particuler namespace.
- how precedence of above rules work (whitelist vs blacklist).

### Specific Authorizer implementation: Openstack Keystone

TODO

### Specific Authorizer implmentation: Openshift


### Use cases to consider

- allow the Daemon Controller to create privileged pods, even if such pods are not otherwise used in the cluster.
  But only allow it if the user that created the pod Template has permission to create such a pod in the first place.
- allow the service account of the ReplicationController control-loop to create any type of pods.
  But only allow it if the user that created the pod Template has permission to create such a pod in the first place.
- do not allow the replication controller to create privileged pods at all, as an extra protection, and on the
  assumption that such pods are only used with Daemon controller anyways.
- allow the Job controller to create any type of pods (e.g. batch jobs that do builds which require docker-in-docker).
  But only allow it if the user that created the pod Template has permission to create such a pod in the first place.

TODO: figure out delegation.
  
### How to implement in OpenShift with SecurityContextController



## General Design Questions for Capabilities

### Using Capabilities vs plain Policy

Which things require Capabilities, and which are left to the Authorizer implementation to implement (e.g. via its own Policies?)

Maybe a rule of thumb for deciding what is a capability is that *Capabilities are only be used for controlling use of specific fields*.  (Should they be called "field permissions"?)
They are not used when reading/writing an entire resource or sub-resource would suffice.  By this rule, using `pod.spec.securityContext.privileged` requires
a capability.  Binding pods does not, since that can be controlled via granting write permission to an entire resource (`/api/v1/namespace/{namespace}/bindings`).
Scaling a resource need not be a capability, since it can be implemented via granting write permission to a subresource (`/api/v1/namespace/{namespace}/replicationController/*/scale`).

This is not entirely satisfying.

## Capabilities Semantics

Modification:
 - if I can create a pod which uses Capability X, can I also modify it?  I think so.
 - if I cannot create a pod which uses Capability X, can I modify it?  Not directly, but maybe via a special verb.
However, up to each Authorizer implementation to implement this.

## Design Alternative: Special Verbs instead of Capabilities

Can Capabilities be implemented as "special verbs"?

We have a way to allow a user to read or modify an existing resource in a limited way,
using special-purpose resources, and sub-resources.
By special purpose resource, I mean like:
- `/api/v1/namespaces/{namespace}/endpoints`
- `/api/v1/namespaces/{namespace}/bindings`
You could give something permission to modify the hostname of a pod without giving
it permission to modify any other fields.
By sub-resources I mean like:
- `/api/v1/namespaces/{namesapce}/replicationControllers/{rcname}/scale`
You could give something permission to modify the scale of an RC without 
modifying the template (seems useful).

Can this model extend to cover the use case of privileged bit?  Maybe like this:
- Add an apiserver handler for endpoint `api/v1/namespace/{namespace}/pods/{podname}/setPrivileged`,
  which sets the `spec.securityContext.privileged` bit on a pod.  
- Then, write security policy (OpenShift style, but without SCC) that allows user U1 to call that
  special verb. (How does openshift support special verbs?)
- Then, U1 creates the pod, in some kind of paused state, but without the privileged bit. 
- Then POST to `api/v1/namespace/{namespace}/pods/{podname}/setPrivileged` to toggle that bit.
- Then modify the pod again to unpause it.
Pausing is needed to prevent pod from starting without privileged bit set.

But how would modification of the pod work?  Can an unprivileged user modify the pod with the privileged
bit? Usually you do not want that.  But we would allow a scheduler to bind a pod regardless of
the state of the privileged bit.

Open question: are users who cannot create a privileged pod prevented from modifying them?
Via the pods interface? Just to change a binding?

## And so on
How to do in:
- IAM-style
- Capabilities.

##  Design questions

- Given a request, is there exactly one set of capabilities that is necessary to complete the request, or
are the multiple combinations?  If the latter, how to express.
- Relationship to special verbs.
- How expressed in ABAC language?
- Having a capability vs asking to use it?
- Are capabilities always about allowing a certain field to have a certain value?

## Evaluation
Benefits of Capabilities Approach:
 - more pithy than a complex policy language can be.  Common language for talking about things
   that otherwise require a mouthful.
 - group together what would otherwise be ad-hoc policies into fewer categories with "equivalent security impact".
 - have terms that remain constant even if different kubernetes clusters have different 
 - has worked okay for Google before. 
 - familiar to users from Linux.
 - Unify various sources of authorization in OpenShift (SCC vs ...)
- Succinctly solve the "field permission" problem.
Concerns:
 - Can all authoriztion implementations implement capabilities (e.g. how would you do it in the space of GCP owner/editor/viewer
   scheme?


## Possible Capabilities

Candidates for Capabilities:
1. Boolean fields of Security Context
  1. Privileged
  1. to set RunAsNonRoot to false (vas distinct from unset).
1. Other fields of a Pod
  1. HostPort: ability to specify HostPort (any value, no subrange control).
  1. HostNet: ability to use Host network namespace (should this be grouped with HostPort?  Is it similar in power?)
  1. HostIPC: ability to use Host IPC ns.
  1. HostPath: ability to create pods with use HostPath dirs.  (would not filter which paths are allowed.  is that needed?)
1. Fields of a Service 
  1. Expose: ability to create a service which is exposed outside the cluster via a cloud load balancer.  (definition of "outside" varies by cluster?)
1. Fields of a Node
  1. Modify Node Schedulability (set/clear Unschedulable bit). node rolling upgrade controller needs this.
1. Subset of Linux Capabilities for which there is demonstrated need.  From experience, I think these may be added at some point:
  1. Linux CAP_IPC_LOCK.  Specifically, if `pod.spec.securityContext.Capabilities.Add[]` contains `CAP_IPC_LOCK`, then "CapabilityIPCLock" is used by the request. 
     Sometimes used by very latency sensitive applications to lock file pages into memory to prevent them being dropped under memory pressure.
  1. Linux CAP_SYS_NICE.  Could be used to allow very latency sensitive program access to real time thread scheduler, etc.  
1. Hypothetical stuff
  1. control Network Type-of-Service/TrafficClass bits in bare-metal networks
1. SELinux
  1. not sure whether these can be succinctly covered by a set of less than 10 capabilities.


# Misc

Namespace capability names like: "kubernetes.io/capability/privilegedPod", in order to allow
plugins to define own capabilities?

What if a request could be satisfied by multiple combinations of capabilities.
Express as list of sets (conjunctive normal form)?


# Evaluation

- Handle use case of: Control when one can create privileged pods.
- Avoid logical errors, duplication, and diversity in Authorizer implementation, of which we will have several.
- Have common language to talk about certain sensitive API actions.
- See if there is a plan which unifies SecurityContextController
  ([SCC](https://github.com/kubernetes/kubernetes/pull/7893)), Policy, various sensitive fields in Pods
  (not in SCC), etc (see [comment](https://github.com/kubernetes/kubernetes/pull/7893#issuecomment-106894040)).




[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/docs/proposals/capabilities.md?pixel)]()

<!-- BEGIN MUNGE: GENERATED_ANALYTICS -->
[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/docs/proposals/capabilities.md?pixel)]()
<!-- END MUNGE: GENERATED_ANALYTICS -->
