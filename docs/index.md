## Repository Components

### Hbase Operator

Hbase k8s operatis is a custom kubernetes controller that uses custom resource to manage hbase applications and their components. There are 2 custom resources(CR) defined here. 1. HbaseCluster 2. HbaseTenant

#### Custom Resources

##### HbaseCluster

HbaseCluster custom resources spins up a cluster with following components in an ordered fashion

1. Zookeeper Quorum 
1. JournalNode Quorum
1. HA Namenodes
1. Cluster of Datanodes + Regionservers (single pod)
1. Hbase Masters

##### HbaseTenant

HbaseTenant custom resources spins up a cluster of following components.

1. Cluster of Datanodes + Regionservers (single pod)


### Helm Chart

Helm chart bundles the packaging aspects of hbase resource manifest in a simplified manner and can be used as dependency in your helm deployments.

Following are covered under helm chart

1. Entrypoint scripts for components such as `zookeeper`, `journalnode`, `namenode`, `hmaster`, `datanode`, `regionserver`. Where these entrypoints does necessary bootstrapping and trap SIGTERM to gracefully terminate application.

1. InitContainers for components such as

    * Ensure dns is resolvable well before statefulsets start
    * Rackawareness support with optional fault domain publisher to zookeeper
    * Namenode refresher on datanode start

1. SideCar Containers for components such as

    * RackUtils for constructing rack topology
    * MTL(Monitoring, Telemetry, Logs) publisher sidecars
