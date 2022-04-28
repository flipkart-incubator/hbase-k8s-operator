
# Deploy Operator

Hbase operator is written to understand hbase tenant and hbase cluster Custom Resource Definitions of kubernetes.

## Build Docker image for Hbase Operator

!!! danger "navigate to `/operator` under parent directory of this repository"

1. You can build operator using following command

    ```sh
    make docker-build IMG=hbase-operator:v1.0.0
    ```

    or

    ```sh
    docker build -f Dockerfile -t hbase-operator:v1.0.0 .
    ```

1. Push docker image to remote registry

    ```sh
    docker push hbase-operator:v1.0.0
    ```

    or to run in minikube

    ```sh
    docker save hbase-operator:v1.0.0 | pv | (eval $(minikube docker-env) && docker load)
    ```

## Deploy Operator

### Via Makefile

!!! danger "navigate to `/operator` under parent directory of this repository"

1. Deploy operator image in your kubernetes. Use `-n` optionally to specify namespace

    ```sh
    make deploy IMG=hbase-operator:v1.0.0 -n hbase-operator-ns
    ```

1. UnDeploy operator. Use `-n` optionally to specify namespace

    ```sh
    make undeploy IMG=hbase-operator:v1.0.0 -n hbase-operator-ns
    ```

1. Hbase operator is verbose enough on any operations performed. You can check container logs using `kubectl` or other mechanism. Example

    ```sh
    kubectl logs hbase-operator-controller-manager-76b4455b76-t4bbb -c manager -f -n hbase-operator-ns
    ```

### Via Helm Chart

1. Modify namespaces to watch for under `examples/operator-chart/values.yaml`. This ensures only those namespaces are watched on which objects to be created

1. **Base Helm Chart:** You can find base helm chart which packages all the necessary manifests into single package. Navigate to `helm-charts/operator-chart` from root directory of this repository. You can build the package using following command

    ```sh
    helm package helm-charts/operator-chart/
    ```

1. **Deploy Helm Chart:**: You can find example helm chart to deploy operator under `examples/operator-chart`

    ```sh
    helm upgrade --install --debug example examples/operator-chart/ -n hbase-operator-ns
    ```

1. Hbase operator is verbose enough on any operations performed. You can check container logs using `kubectl` or other mechanism. Example

    ```sh
    kubectl logs hbase-operator-controller-manager-76b4455b76-t4bbb -c manager -f -n hbase-operator-ns
    ```
