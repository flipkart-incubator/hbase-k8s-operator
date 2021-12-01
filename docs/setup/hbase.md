!!! danger "change directory to `/examples` under parent directory of this repository"

## Build Docker Image for Hbase

### Maven POM

1. `pom.xml` is written to bundle all of the dependencies to package in docker. Such has downloading hadoop and hbase binaries from either public repos or private repos of organisation. 

1. Optionally you can customise the package to suit your needs. Some Examples are: optional libraries for change propogation, modified hbase base libraries, repair libraries, etc.

1. Additionally you can have modified `repositories`, `distributionManagement`, `pluginRepositories` so as to download dependencies from private repositories

### Dockerfile

1. Dockerfile package hbase and hadoop binaries, which can be downloaded from public mirrors or private mirrors.

1. User, Group, Directories required are created and given sufficient permissions for hbase to run.

1. Optionally can modify or add additional libraries from hbase or hadoop packages

1. Optionally utilities can be installed required such as dnsutils, netcat, etc

1. Base image should be kept smaller, builder image can include jdk image

1. Build docker image and publish to a repository. 

    ```
    docker build . --network host -t hbase:2.4.2 && docker push hbase:2.4.2
    ```

## Hbase Standalone

### Package and Deploy Hbase Standalone

#### Helm Chart

1. A customisable base helm chart is available to make use of and simplify deployable helm charts. You can find `./helm-charts/hbase-chart/` under root folder of this repository

1. Build the base helm chart from root folder of this repository as follows
    ```
    helm package helm-charts/hbase-chart/
    ```

1. You can find package `hbase-chart-x.x.x.tgz` created under root folder of this repository. Otherwise you can publish chart to `jfrog` or `harbor` or any other chart registry. For manual testing, you can move `hbase-chart-x.x.x.tgz` under `examples/hbasestandalone-chart/charts/`
    ```
    cd hbase-operator && mv hbase-chart-x.x.x.tgz examples/hbasestandalone-chart/charts/ 
    ```

1. Open `examples/hbasestandalone-chart/values.yaml`, and modify the values as per your requirement. Some of the recommended modifications are

    1. image: Docker image of hbase we built in previous section
    1. Memory limits / requests and CPU limits / request as per your requirements

1. You can deploy your helm package using following command
    ```
    helm upgrade --install --debug hbasestandalone-chart hbasestandalone-chart/ -n hbase_standalone
    ```

## Hbase Cluster

### Package and Deploy Hbase Cluster

#### Helm Chart

!!! danger "Changing namespace names would mean configuration having host names should also be changed such as zookeeper, namenode etc"

1. A customisable base helm chart is available to make use of and simplify deployable helm charts. You can find `./helm-charts/hbase-chart/` under root folder of this repository

1. Build the base helm chart from root folder of this repository as follows
    ```
    helm package helm-charts/hbase-chart/
    ```

1. You can find package `hbase-chart-x.x.x.tgz` created under root folder of this repository. Otherwise you can publish chart to `jfrog` or `harbor` or any other chart registry. For manual testing, you can move `hbase-chart-x.x.x.tgz` under `examples/hbasecluster-chart/charts/`
    ```
    cd hbase-operator && mv hbase-chart-x.x.x.tgz examples/hbasecluster-chart/charts/ 
    ```

1. Open `examples/hbasecluster-chart/values.yaml`, and modify the values as per your requirement. Some of the recommended modifications are

    1. isBootstrap: Enable this flag first time you run this cluster. Which performs `hdfs format`, required at the time of cluster setup. Once cluster started, you can disable and upgrade the cluster again. 
    1. image: Docker image of hbase we built in previous section
    1. annotations: In this examples, we have used to demonstrate MTL (Monitoring, Telemetry and Logging)
    1. Volume claims for your k8s can be fetched using `kubectl get storageclass`. Which can be used to replace `storageClass`
    1. `probeDelay`: This will affect both `liveness` and `readiness` alike
    1. Memory limits / requests and CPU limits / request as per your requirements

1. You can deploy your helm package using following command
    ```
    helm upgrade --install --debug hbasecluster-chart hbasecluster-chart/ -n hbase_cluster
    ```


## Hbase Tenant

### Operator Side

1. Add additional namespaces to watch for. Here specific namespace to be onboarded for a particular tenant
    ```
    vim operator/config/custom/config/hbase-operator-config.yaml
    ```

1. Create configmap with command
    ```
     kubectl apply -f operator/config/custom/config/hbase-operator-config.yaml -n hbase_operator
    ```

1. Deploy operator

### Tenant Side

1. Create Rolebinding under namespace which is hosting either hbasetenant or hbasecluster such as follows. Where `hbase_tenant` is the namespace on which you would deploy your resources

    ```
    ./testbin/bin/kubectl apply -f config/rbac/role_binding.yaml -n hbase_tenant
    ```

### Package and Deploy Hbase Tenant

#### Helm Chart

!!! danger "Changing namespace names would mean configuration having host names should also be changed such as zookeeper, namenode etc"

1. A customisable base helm chart is available to make use of and simplify deployable helm charts. You can find `./helm-charts/hbase-chart/` under root folder of this repository

1. Build the base helm chart from root folder of this repository as follows
    ```
    helm package helm-charts/hbase-chart/
    ```

1. You can find package `hbase-chart-x.x.x.tgz` created under root folder of this repository. Otherwise you can publish chart to `jfrog` or `harbor` or any other chart registry. For manual testing, you can move `hbase-chart-x.x.x.tgz` under `examples/hbasetenant-chart/charts/`
    ```
    cd hbase-operator && mv hbase-chart-x.x.x.tgz examples/hbasetenant-chart/charts/ 
    ```

1. Open `examples/hbasetenant-chart/values.yaml`, and modify the values as per your requirement. Some of the recommended modifications are

    1. image: Docker image of hbase we built in previous section
    1. annotations: In this examples, we have used to demonstrate MTL (Monitoring, Telemetry and Logging)
    1. Volume claims for your k8s can be fetched using `kubectl get storageclass`. Which can be used to replace `storageClass`
    1. `probeDelay`: This will affect both `liveness` and `readiness` alike
    1. Memory limits / requests and CPU limits / request as per your requirements

1. You can deploy your helm package using following command

    ```
    helm upgrade --install --debug hbasetenant-chart hbasetenant-chart/ -n hbase_tenant
    ```
