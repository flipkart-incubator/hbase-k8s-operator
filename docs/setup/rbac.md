!!! warning "This is just for illustration, deploying hbase cluster/tenant/standalone comes bundled with rbac permissions"

1. RBAC for multi namespace deployment (Operator is deployed in its own namespace different from either cluster or tenant namesapces)


    1. Create `ClusterRole` with permissions required for operator to apply on namespaces. Assuming operator is on different namespace from hbasecluster and or tenant. Modify `Role` to `ClusterRole` in config/rbac/role.yaml in case you want to have global scope or else apply hbase-cluster-ns or hbase-tenant-ns namespace without any changes
        ```
        kubectl apply -f config/rbac/role.yaml
        ```

        **Or**

        Apply contents from `config/rbac/role.yaml` using some automation tool

    1. Create RoleBilding under namespace which is hosting either `hbase-tenant-ns` or `hbase-cluster-ns` such as follows. Where `hbase-tenant-ns` and `hbase-cluster-ns` are the namespace on which you would deploy your resources
        ```
        kubectl apply -f config/rbac/role_binding.yaml -n hbase-cluster-ns
        kubectl apply -f config/rbac/role_binding.yaml -n hbase-tenant-ns
        ```

        !!! Danger "Service Account and roleRef particulars should match with which operator will be run along with namespace"

1. RBAC for single namespace deployment (Operator is deployed along with hbase cluster/tenant in single namespace)

    1. Create `Role` with permissions required for operator to apply on namespaces.
        ```
        kubectl apply -f config/rbac/role.yaml
        ```

    1. Create RoleBilding under same namespace.
        ```
        kubectl apply -f config/rbac/role_binding.yaml
        ``` 

        !!! Danger "Service Account and roleRef particulars should match with which operator will be run along with namespace"
