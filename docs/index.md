# Hbase Operator

## About Hbase

HBase is an open-source non-relational distributed database modeled after Google's Bigtable and written in Java. It is developed as part of Apache Software Foundation's Apache Hadoop project and runs on top of HDFS (Hadoop Distributed File System) or Alluxio, providing Bigtable-like capabilities for Hadoop. That is, it provides a fault-tolerant way of storing large quantities of sparse data.

## Operator Details

1. This operator is designed to be Namespace scoped. Single kubernetes cluster can run multiple instances of this operator in separate namespaces listening for multiple other namespaces

1. This operator generic enough to be able to run wide range of hbase versions. Tested across several 2.x versions

1. Multi tenant hbase cluster which can span across multiple namespaces, where tenants can be from different teams owning their own infra and maintenance.

1. A helm chart wrapper is present along with operator aiming to standardise deployments and avoid common pitfalls such as writing complex probes, startup and shutdown scripts, etc

1. Generic enough to extend it with any customisations such as metric sidecars, istio sidecars, annotations, init containers etc

1. Supports for rack awareness where fault domain can be fed in from multiple options such as file in a pod, env variable, etc and state is stored in zookeeper as a central store for building rack topology

## Repository Components

### Hbase Operator

Hbase k8s operatis is a custom kubernetes controller that uses custom resource to manage hbase applications and their components. There are 3 custom resources(CR) defined here.

#### Custom Resources

##### HbaseCluster

HbaseCluster CRD's spins up a cluster with following components in an ordered fashion

1. Zookeeper Quorum
1. JournalNode Quorum
1. HA Namenodes
1. Cluster of Datanodes + Regionservers (single pod to enable short circuiting)
1. Hbase Masters

##### HbaseTenant

HbaseTenant CRD is capable of bringing up a group of datandes along with regionservers to form a rsgroup which can be grouped under different namespace. 

1. Cluster of Datanodes + Regionservers (single pod)

##### HbaseStandalone

HbaseStandalone CRD is capable of bringup a single pod hbase primarily used for testing purposes.

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
