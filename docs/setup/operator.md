
## Deploy Operator

Hbase operator is written to understand hbase tenant and hbase cluster Custom Resource Definitions of kubernetes.

### Build Docker image for Hbase Operator

!!! danger "navigate to `/operator` under parent directory of this repository"

1. You can build operator using following command

    ```
    make docker-build IMG=hbase-operator:v1.0.0
    ```
    or

    ```
    docker build -f Dockerfile -t hbase-operator:v1.0.0 .
    ```

1. Push docker image to remote registry

    ```
    docker push hbase-operator:v1.0.0
    ```

### Deploy Operator

#### Via Makefile

!!! danger "navigate to `/operator` under parent directory of this repository"

1. Deploy operator image in your kubernetes. Use `-n` optionally to specify namespace

    ```
    make deploy IMG=hbase-operator:v1.0.0 -n hbase-operator-ns
    ```

1. UnDeploy operator. Use `-n` optionally to specify namespace

    ```
    make undeploy IMG=hbase-operator:v1.0.0 -n hbase-operator-ns
    ```

1. Hbase operator is verbose enough on any operations performed. You can check container logs using `kubectl` or other mechanism. Example

    ```
    kubectl logs hbase-operator-controller-manager-76b4455b76-t4bbb -c manager -f -n hbase-operator-ns
    ```

#### Via Helm Chart

1. **Base Helm Chart:** You can find base helm chart which packages all the necessary manifests into single package. Navigate to `helm-charts/operator-chart` from root directory of this repository. You can build the package using following command

    ```
    helm package helm-charts/operator-chart/
    ```

1. **Deploy Helm Chart:**: You can find example helm chart to deploy operator under `examples/operator-chart` 

    ```
    helm upgrade --install --debug example examples/operator-chart/ -n hbase-operator-ns
    ```

1. Hbase operator is verbose enough on any operations performed. You can check container logs using `kubectl` or other mechanism. Example

    ```
    kubectl logs hbase-operator-controller-manager-76b4455b76-t4bbb -c manager -f -n hbase-operator-ns
    ```
