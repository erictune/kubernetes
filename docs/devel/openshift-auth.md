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
[here](http://releases.k8s.io/release-1.0/docs/devel/openshift-auth.md).

Documentation for other releases can be found at
[releases.k8s.io](http://releases.k8s.io).
</strong>
--

<!-- END STRIP_FOR_RELEASE -->

<!-- END MUNGE: UNVERSIONED_WARNING -->

# OpenShift Authorization Summary

**Intended for discussion.  No need to merge or provide grammar/spelling/style comments at this time.*

Purpose of document:
1. Summarize how authorization is handled in OpenShift, as it is proposed to upstream it to K8s.
1. Get feedback from people who otherwise would not read the PRs.

## Current State of OpenShift

### Summary of Roles, RoleBindings, etc

Authorization for creating entire objects is done via the types defined
[here](https://github.com/openshift/origin/blob/master/pkg/authorization/api/v1/types.go).

Summary of API types:

- a Role object lists various actions that someone/thing acting in that role can do.  There are predefined roles, and custom ones are possible.
- the allowed actions of a Role are expressed as a list of PolicyRules.
- PolicyRules can allow or deny access to all resources of a certain kind, or to specific resources.
- PolicyRules can allow specific actions ("verbs") on an object.  Verbs include: "create", "delete", "get",
  "list", "update", "watch".
- PolicyRules have an escape valve called an AttributeRestriction which allows arbitrary conditions to be specified
  if a corresponding "plugin" is plugged into the apiserver.  For example, one AttributeRestriction checks that the
  Host of the requestor (e.g. kubelet) is the same as the Host of the object (e.g. Pod) being requested.
  computes some function is a set of allowed actions to do on the apiserver (e.g. get pods).
- A RoleBinding says that a User or Group of users can act in a Role.
- A Role and a RoleBinding are for namespaced objects and apply to objects in its same namespace.
- Objects with no namespace and cluster-wide policies that apply regardless of namespace, are
  held in corresponding ClusterRole and ClusterRoleBinding objects.

s that apply to requests which have no associated namespace (node, some events, etc).
- There are half a dozen other types whose purpose is not relevant or I do not understand yet.
-

Notes on behavior and implementation
- implmented by `openshiftAuthorizer`.
- Authorize method first check if an action is globally allowed. func (a *openshiftAuthorizer) authorizeWithNamespaceRules(ctx kapi.Context, passedAttributes AuthorizationAttributes)

### SecurityContexts and SecurityContextConstraints

PolicyRules authorize actions on entire objects.  However, permission to use specific fields in an
`pod.spec.container[].securityContext` are controlled by a `securityContextConstraints` object.

Summary of that type:

- `securityContextConstraints` is defined [here
](https://github.com/openshift/origin/blob/70015e4a6f9b821ccd5a01134054286e3813b4eb/Godeps/_workspace/src/k8s.io/kubernetes/pkg/api/v1/types.go#L2050).
- it is documented [here](https://github.com/openshift/origin/blob/9bcf45dcdbaccb0b02edee795c91a7c810f349a3/docs/proposals/security-context-constraints.md).
- for each filed in a SecurityContext object, there is a corresponding field in SecurityContextConstraints (SCC) which control use of former.
- also, there are fields in SecurityContext which control which users/groups it applies to
- various admission controllers forbid creation of Pods whose SecurityContexts violate SecurityContextConstraints.
- you can modify an SCC object to change what is allowed
- the combination of SCs, SCCs, and the admission controllers for them allow OpenShift to do things like
  allocate a unique unix UID to each service account in the system, while still allowing a superuser to override that
  allocation to run a pod with an arbitrary SecurityContext.
- SCCs allow assigning a specific unix UID to pods in a certain namespace or with a specific service account.
- ServiceAccounts can have SCCs and the system may allocate unique unix UIDs to service accounts.

### Commentary:

- The unix uid allocation thing is cool, and important to OpenShift, and helps with pod security, and important
  to running in an environment with things like NFS volumes that have files belonging to various unix uids.
- Wonder why UID allocation is not
A user can create a pod with a certain security context in a certain namespace if:
1. user is permitted to create pods in that namespace
1. the pod has a service account (omitting description of case when this is not true)
1. the user has the ability to use the security context constraints of the service account, which means:
  1. intersect the constraints implied by SecurityContextConstraint for user and SecurityContextConstraint for the service account.
     if empty intersection, disallow creation.
If allowed, a SecurityContextConstraintsAllocator to create the security context for the pod (e.g. allocates a unix uid appropriate
to the pod.
[PR 7893](https://github.com/kubernetes/kubernetes/pull/7893) proposes to merge the SecurityContextConstraints object definition,
control, and client support to Kubernetes.

Let us write to write SCC for SecurityContextConstraints from now on.


### SCC in OSS K8s and GKE

How it might be used in plain Kubernetes:
1. Cluster has default SCC named "default".
1. if I want to allow user U1 to create privileged pods, I modify "default" using `kubectl replace securitycontextcontstraints default`
with one like:

```
kind: securitycontextcontstraints
...
allowPrivilegedContainer: true
...
users: [ "U1" ]
```

How it might be used in GKE:
- same way as above (assuming GKE had an ability to set "users")

### Evaluation

- SecurityContexts/SecurityContextConstraints and Roles/RoleBindings are independent mechanisms.
- SecurityContextConstraints only apply to pods.  But, we may want to limit values in certain fields of other objects (e.g. limit who can make a service that has an external load balancer).

## 

## Comparison

## Detailed Exploration of Capabilities

### Possible Capabilities


Proposal from OpenShift

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

### 



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


## Possible use cases and Capabilities


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

-----

- not necessarily easy for users to reason about the security implications of a Pod or Service setting.  
- could have general matching language for Auth like in amazon IAM but awkward to integrate for people who want simpler authz systems.
- there will be multiple authz systems due to integration with existing on prem or cloud provider systems.

Define capabilities, which are just words.

Builtin code in apiserver maps a Pod, Service or other thing to a list of required capabilities.
Pass these to Authorizer routines.
They can chose to use these in their decision, and use these strings in their various policy files.


Is there a lattice or tree of capabilities and is it useful to find the meet/join of it.
.

Namespace with kubernetes.io/capability/foo
what about a subtree for linux OS capabilities.

What if there are multiple capabilities required but person only has one, but others cover others?
Avoid overlap?

Cases:
- privileged
- override a default security context specified user
- anything in security context
- volume stuff?


They represent common things, but some auth impls can still authorize ignoring the caps, and looking at particulars.

## Evaluation

- Handle use case of: Control when one can create privileged pods.
- Avoid logical errors, duplication, and diversity in Authorizer implementation, of which we will have several.
- Have common language to talk about certain sensitive API actions.
- See if there is a plan which unifies SecurityContextController
  ([SCC](https://github.com/kubernetes/kubernetes/pull/7893)), Policy, various sensitive fields in Pods
  (not in SCC), etc (see [comment](https://github.com/kubernetes/kubernetes/pull/7893#issuecomment-106894040)).


### TODO integrate this from Clayton

Here's the policy engine design doc:

https://github.com/openshift/origin/blob/master/docs/proposals/policy.md

It was modeled after keystone (more or less) - it's RBAC on top of ABAC and
is additive only (no negated rules).

Here's the user doc
https://docs.openshift.org/latest/admin_guide/manage_authorization_policy.html

The engine today plugs in under the Authorizer interface, although it has a
number of caches for performance that make setup complex.

Here's the docs for the oauth and user auth module:
https://docs.openshift.org/latest/admin_guide/configuring_authentication.html

It plugs in under the Authenticator interface.

Pretty soon we need to start the discussion going on the full feature list,
how we can modulator it for inclusion upstream, and what pieces are complex
or optional.

Here is the high level tracking issue:

https://github.com/GoogleCloudPlatform/kubernetes/issues/10408

We've discussed the design in various proposals (dating back to the very
first user / ABAC proposal), but I don't think there's a canonical
reference issue right now.  David and Jordan will be helping boot that up
starting soon.

I'm not sure how well documented it is, but the authenticators are responsible for determining the Identity on a request (distinct from user) and then mapping that identity to a user.Info (https://github.com/GoogleCloudPlatform/kubernetes/blob/master/pkg/auth/user/user.go#L20). The userInfo (but not the identity) is stored in the context for subsequent handlers in the chain. You can see a sample OpenID authenticator that determines identity and maps users based on claims here in openshift: https://github.com/openshift/origin/blob/master/pkg/auth/oauth/external/openid/openid.go.

Right now, the authorizer implementations between OpenShift and Kubernetes have drifted slightly. Essentially, any openshift authorizer can be adapted to a kube authorizer, but not the other way around. See https://github.com/openshift/origin/blob/master/pkg/authorization/authorizer/interfaces.go#L11-L14 and https://github.com/GoogleCloudPlatform/kubernetes/blob/master/pkg/auth/authorizer/interfaces.go#L49-L51. The big difference is the GetAllowedSubjects method. We discovered that API was important for auditing (who has access to this resource) and for efficient reverse lookups (which namespaces can I view).

Being able to give the authorizer enough authority in Keystone to be able to fulfill those two methods against Keystone for a particular user would allow a very clean integration with the layers. Basically, your plugin has to answer: "can this user (not necessarily me) perform ActionX" and "which users and groups can perform ActionX".
:wq



[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/docs/devel/openshift-auth.md?pixel)]()

<!-- BEGIN MUNGE: GENERATED_ANALYTICS -->
[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/docs/devel/openshift-auth.md?pixel)]()
<!-- END MUNGE: GENERATED_ANALYTICS -->
