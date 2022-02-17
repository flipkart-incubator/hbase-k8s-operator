!!! danger "change directory to `/operator` under parent directory of this repository"

!!! warning "You should install `kustomize`, `kubectl`, `minikube` for below examples to work"

## Create required resource

1. Extract crds and apply it on your k8s cluster

    1. Apply crds on your cluster using kubectl

        ```
        kustomize build config/crd | kubectl apply -f -
        ```

    1. **Or** generate the crd as follows and apply using some automation tool

        ```
        kustomize build config/crd
        ```

1. Create namespaces if not already created. lets keep `hbase-operator-ns` for namespace on which operator will be deployed, `hbase-cluster-ns` for namespace on which hbase cluster will be deployed and `hbase-tenant-ns` for namespace on which tenant will be deployed.

    ```
    kubectl create namespace hbase-operator-ns
    kubectl create namespace hbase-cluster-ns
    kubectl create namespace hbase-tenant-ns
    kubectl create namespace hbase-standalone-ns
    ```
