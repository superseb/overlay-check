overlay-check
========

## What is it

Status: alpha (hobby project with redundant code)

overlay-check is supposed to run as a DaemonSet in a Kubernetes cluster, will discover all pods in the created DaemonSet (via service account) and will ping each pod except itself to test overlay network (cross host).

## Building

`make`

This repository is based on [https://github.com/rancher/go-skel/](https://github.com/rancher/go-skel/)

## Running

```
---
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: overlay-check
spec:
  template:
    metadata:
      labels:
        app: overlay-check
    spec:
      containers:
        - name: overlay-check
          image: superseb/overlay-check:dev
          imagePullPolicy: Always
      serviceAccountName: overlay-check-sa
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: overlay-check-sa
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: overlay-check-read
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "watch", "list"]
- apiGroups: ["apps", "extensions"]
  resources: ["daemonsets"]
  verbs: ["get", "watch", "list"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: read-pods
subjects:
- kind: ServiceAccount
  name: overlay-check-sa
  namespace: default
roleRef:
  kind: ClusterRole
  name: overlay-check-read
  apiGroup: rbac.authorization.k8s.io
```

## License
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
