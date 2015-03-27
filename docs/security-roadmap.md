# Kubernetes Security Roadmap

Relevant issues [labeled with area-security](https://github.com/GoogleCloudPlatform/kubernetes/labels/area%2Fsecurity).

Security priority levels:
   1. Keep outsiders out of the cluster
   1. Tie for second:
     - Keep tenants away from each other.
     - Finer-grained permissions among users of the same tenant.
   1. Protect the cluster from itself (least privilege for cluster components)


Features
 - All ports secured
   - Etcd and SSL #5987
   - https://github.com/GoogleCloudPlatform/kubernetes/issues/6067
   - ...

 - Trusted Kubernetes Code
   - Kubernetes examples probably should not rely on containers owned by other people #1827

 - Cert-based node auth
   - Client Cert Authentication #2591

 - Are we still running containers as "Root"?
   - Need to support running as non-root #1859

 - Volume permissions
   - Volumes are created in container with root ownership and strict permissions #2630

 - Stop exposing readonly unauthorised access to apiserver
   - Existing kubernetes-ro use cases #5921
   - kube-proxy #5917
   - #4567


 - Bring trailing cloud-providers up to latest security
   - #5247
   - #4947
   - neverending work

 - Restrict access to cadvisor #5028

 - Restrict use of container escapes
     - hostPort #4849
     - hostDir  and privileged container #2502
     - docker socket (such as for docker build) #1567

 - Usability improvements for secrets
   - Improve user experience for creating / updating secrets #4822
   - Fine-grained control of secret data in volume #4789
   - Expose secrets to containers in environment variables #4710
   - Key/Cert rotation for Kubernetes clients. #4672

 - Security improvements for secrets
   - Kubelet secret volume plugin should store data in tmpfs #4602

 - Authentication
   - Bedrock auth sources #2202
   - Make sure we support multiple CAs for client TLS auth
   - Remaining work for using tokens instead of basic auth  ? #4007


 - Trusted images
   - Authorization Policy should be able to require "trusted images" #3889
   - Limit which docker repos a namespace allows to pull from.
   - 

 - Authorization
   - Basic Requirements
     - Built-in Policy language as default for controlling cluster bootstrap, and letting users kick tires.
     - Demonstrated Extensibility
       - authorizer plugins for two or more platforms, such as GCP, Openshift, AWS IAM.
     - Related Issues:
       - ABAC Policy should have operation as input attribute #2877
       - Make Policy be a proper REST object with its own registry #2212
       - Use groups in authz policy #2205
     - Distribute auth tokens to pods #1907
       - Also used for self-hosting.
   - More features
     - Policies on Labels and Selectors #2211
     - Fine-grained authorization for Secrets: #4957
     - Expiration of Authz Policies #2204
     - Audit logs #2203
     - Determine how namespaces and labels and access control interact. #1491

 - Walls between Pods
   - Add information about securing the Kubelet to security.md #1053
   - Ability to set SELinux labels for volumes #699

 - Trusting images 
   - Private Registry Authentication #499

 - Unorganized
  1. Per pod (vs current per-kubelet) docker credentials: Pod A is allowed to use docker credential X but pod B is not,
  even though they are in same cluster and may be on same kubelet. May implement with secrets. (This last bit goes in
  the issue though.)

  1. Limit who can schedule on master node.

 - `secrets` kind.
  1. Control which users/pods have access to which secrets.
  1. Expose which resourceVersion of a secret is used by a running pod.
  1. More real-world examples using secrets.  With example of how to do rotation.
  1. Quota or other limit on pe-apiserver and per-kubelet memory usage of secrets.
  1. Least-privilege for system components.
    1. Kubelets should only have read-only access to secrets, and only for ones they need.
    1. Kubelets should only be able to pull secrets, pods bound to their node.

  
 - Authorization
  1. Policy as objects
  1. Story on how to delegate policy to another form of policy store (Amazon IAM, etc.)
  1. 

 - User Authentication
 - Authorization
 - Clustering
 - Resource management and isolation



 - Users can trust certain distros of kubernetes.
  1. security tests in conformance/e2e test, run against key distros
  1. examples use google-controlled images?
  1. sign releases?
  1. That CVE issue.
  1. 

  Remove scopes from master nodes and use tokens instead.

- Auditability


Experimental endpoint to make service account (user and token) and to make a secret with that token and make it not
readable via api and not leave cluster.

  Then make policy for that acct.  Then make one for.scheduler. And then for cm.

  Could.make for admin user too? But readable.  And other human users.  


  Users.requirements: test users.  Svc users.  Sync from other source.(gke).


  Eric Tune <etune@google.com>
  Mar 17 (2 days ago)

  to me 
  Need token and user storage in api.to make this.work. 

  if.service.account already exists in IAM.and auth is delegated then what happens? Creation is delegated or a noop.


-----------

PRs:
x509 request authenticator #2793
:w
:wq

