# Deploy Hbase

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

    ```sh
    docker build . --network host -t hbase:2.4.8 && docker push hbase:2.4.8
    ```

## Hbase Standalone

### Package and Deploy Hbase Standalone

#### Helm Chart

1. A customisable base helm chart is available to make use of and simplify deployable helm charts. You can find `./helm-charts/hbase-chart/` under root folder of this repository

1. Build the base helm chart from root folder of this repository as follows

    ```sh
    helm package helm-charts/hbase-chart/
    ```

1. You can find package `hbase-chart-x.x.x.tgz` created under root folder of this repository. Otherwise you can publish chart to `jfrog` or `harbor` or any other chart registry. For manual testing, you can move `hbase-chart-x.x.x.tgz` under `examples/hbasestandalone-chart/charts/`

    ```sh
    mv hbase-chart-x.x.x.tgz examples/hbasestandalone-chart/charts/
    ```

1. Open `examples/hbasestandalone-chart/values.yaml`, and modify the values as per your requirement. Some of the recommended modifications are

    1. image: Docker image of hbase we built in previous section
    2. Memory limits / requests and CPU limits / request as per your requirements

1. You can deploy your helm package using following command

    ```sh
    helm upgrade --install --debug hbasestandalone-chart examples/hbasestandalone-chart/ -n hbase-standalone-ns
    ```

#### via Manifest

<details>
<summary>Sample Standalone yaml configuration</summary>

```yaml
# Source: hbasestandalone-chart/templates/hbasestandalone.yaml
apiVersion: kvstore.flipkart.com/v1
kind: HbaseStandalone
metadata:
  name: hbase-standalone
  namespace: hbase-standalone-ns
spec:
  baseImage: hbase:2.4.8
  fsgroup: 1011
  configuration:
    hbaseConfigName: hbase-config
    hbaseConfigMountPath: /etc/hbase
    hbaseConfig:
      hbase-site.xml: |
        <?xml version="1.0"?>
        <?xml-stylesheet type="text/xsl" href="configuration.xsl"?>
        <configuration>
          <property>
            <name>hbase.cluster.distributed</name>
            <value>false</value>
          </property>
          <property>
            <name>hbase.rootdir</name>
            <value>/grid/1/hbase</value>
          </property>
          <property>
            <name>hbase.tmp.dir</name>
            <value>/grid/1/tmp</value>
          </property>
          <property>
            <name>hbase.zookeeper.property.dataDir</name>
            <value>/grid/1/zookeeper</value>
          </property>
          <property>
            <name>hbase.unsafe.stream.capability.enforce</name>
            <value>false</value>
          </property>
          <property>
            <name>hbase.balancer.rsgroup.enabled</name>
            <value>true</value>
          </property>
          <property>
            <name>hbase.coprocessor.master.classes</name>
            <value>org.apache.hadoop.hbase.rsgroup.RSGroupAdminEndpoint</value>
          </property>
          <property>
            <name>hbase.master.loadbalancer.class</name>
            <value>org.apache.hadoop.hbase.rsgroup.RSGroupBasedLoadBalancer</value>
          </property>
        </configuration>
    hadoopConfigName: hadoop-config
    hadoopConfigMountPath: /etc/hadoop
    hadoopConfig:
      {}
  standalone:
      name: hbase-standalone-all
      size: 1
      isPodServiceRequired: false
      shareProcessNamespace: false
      terminateGracePeriod: 120
      volumeClaims:
      - name: data
        storageSize: 2Gi
        storageClassName: standard
      containers:
      - name: standalone
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m
          export HBASE_LOG_DIR=$0
          export HBASE_CONF_DIR=$1
          export HBASE_HOME=$2
          export USER=$(whoami)

          mkdir -p $HBASE_LOG_DIR
          ln -sf /dev/stdout $HBASE_LOG_DIR/hbase-$USER-master-$(hostname).out
          ln -sf /dev/stdout $HBASE_LOG_DIR/hbase-$USER-master-$(hostname).log

          function shutdown() {
            echo "Stopping Standalone"
            $HBASE_HOME/bin/hbase-daemon.sh stop master
          }

          trap shutdown SIGTERM
          exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start master &
          wait
        args:
        - /var/log/hbase
        - /etc/hbase
        - /opt/hbase
        - hbase-config
        ports:
        - port: 16000
          name: standalone-0
        - port: 16010
          name: standalone-1
        - port: 16030
          name: standalone-2
        - port: 16020
          name: standalone-3
        - port: 2181
          name: standalone-4
        livenessProbe:
          tcpPort: 16000
          initialDelay: 10
        readinessProbe:
          tcpPort: 16000
          initialDelay: 10
        cpuLimit: "0.5"
        memoryLimit: "2Gi"
        cpuRequest: "0.5"
        memoryRequest: "2Gi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
          addSysPtrace: false
        volumeMounts:
        - name: data
          mountPath: /grid/1
          readOnly: false

```

</details>

## Hbase Cluster

### Package and Deploy Hbase Cluster

#### Helm Chart

!!! danger "Changing namespace names would mean configuration having host names should also be changed such as zookeeper, namenode etc"

1. A customisable base helm chart is available to make use of and simplify deployable helm charts. You can find `./helm-charts/hbase-chart/` under root folder of this repository

1. Build the base helm chart from root folder of this repository as follows

    ```sh
    helm package helm-charts/hbase-chart/
    ```

1. You can find package `hbase-chart-x.x.x.tgz` created under root folder of this repository. Otherwise you can publish chart to `jfrog` or `harbor` or any other chart registry. For manual testing, you can move `hbase-chart-x.x.x.tgz` under `examples/hbasecluster-chart/charts/`

    ```sh
    mv hbase-chart-x.x.x.tgz examples/hbasecluster-chart/charts/
    ```

1. Open `examples/hbasecluster-chart/values.yaml`, and modify the values as per your requirement. Some of the recommended modifications are

    1. isBootstrap: Enable this flag first time you run this cluster. Which performs `hdfs format`, required at the time of cluster setup. Once cluster started, you can disable and upgrade the cluster again.
    1. image: Docker image of hbase we built in previous section
    1. annotations: In this examples, we have used to demonstrate MTL (Monitoring, Telemetry and Logging)
    1. Volume claims for your k8s can be fetched using `kubectl get storageclass`. Which can be used to replace `storageClass`
    1. `probeDelay`: This will affect both `liveness` and `readiness` alike
    1. Memory limits / requests and CPU limits / request as per your requirements

1. You can deploy your helm package using following command

    ```sh
    helm upgrade --install --debug hbasecluster-chart examples/hbasecluster-chart/ -n hbase-cluster-ns
    ```

#### via Manifest

<details>
<summary>Sample Cluster yaml configuration</summary>

```yaml
# Source: hbasecluster-chart/templates/hbasecluster.yaml
apiVersion: kvstore.flipkart.com/v1
kind: HbaseCluster
metadata:
  name: hbase-cluster
  namespace: hbase-cluster-ns
spec:
  baseImage: hbase:2.4.8
  isBootstrap: true
  fsgroup: 1011
  configuration:
    hbaseConfigName: hbase-config
    hbaseConfigMountPath: /etc/hbase
    hbaseConfig:
      hadoop-metrics2-hbase.properties: |
        # Licensed to the Apache Software Foundation (ASF) under one
        # or more contributor license agreements.  See the NOTICE file
        # distributed with this work for additional information
        # regarding copyright ownership.  The ASF licenses this file
        # to you under the Apache License, Version 2.0 (the
        # "License"); you may not use this file except in compliance
        # with the License.  You may obtain a copy of the License at
        #
        #     http://www.apache.org/licenses/LICENSE-2.0
        #
        # Unless required by applicable law or agreed to in writing, software
        # distributed under the License is distributed on an "AS IS" BASIS,
        # WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
        # See the License for the specific language governing permissions and
        # limitations under the License.

        # syntax: [prefix].[source|sink].[instance].[options]
        # See javadoc of package-info.java for org.apache.hadoop.metrics2 for details

        *.sink.file*.class=org.apache.hadoop.metrics2.sink.FileSink
        # default sampling period
        *.period=10

        # Below are some examples of sinks that could be used
        # to monitor different hbase daemons.

        # hbase.sink.file-all.class=org.apache.hadoop.metrics2.sink.FileSink
        # hbase.sink.file-all.filename=all.metrics

        # hbase.sink.file0.class=org.apache.hadoop.metrics2.sink.FileSink
        # hbase.sink.file0.context=hmaster
        # hbase.sink.file0.filename=master.metrics

        # hbase.sink.file1.class=org.apache.hadoop.metrics2.sink.FileSink
        # hbase.sink.file1.context=thrift-one
        # hbase.sink.file1.filename=thrift-one.metrics

        # hbase.sink.file2.class=org.apache.hadoop.metrics2.sink.FileSink
        # hbase.sink.file2.context=thrift-two
        # hbase.sink.file2.filename=thrift-one.metrics

        # hbase.sink.file3.class=org.apache.hadoop.metrics2.sink.FileSink
        # hbase.sink.file3.context=rest
        # hbase.sink.file3.filename=rest.metrics
      hbase-env.sh: "#\n#/**\n# * Licensed to the Apache Software Foundation (ASF) under one\n# * or more contributor license agreements.  See the NOTICE file\n# * distributed with this work for additional information\n# * regarding copyright ownership.  The ASF licenses this file\n# * to you under the Apache License, Version 2.0 (the\n# * \"License\"); you may not use this file except in compliance\n# * with the License.  You may obtain a copy of the License at\n# *\n# *     http://www.apache.org/licenses/LICENSE-2.0\n# *\n# * Unless required by applicable law or agreed to in writing, software\n# * distributed under the License is distributed on an \"AS IS\" BASIS,\n# * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\n# * See the License for the specific language governing permissions and\n# * limitations under the License.\n# */\n\n# Set environment variables here.\n\n# This script sets variables multiple times over the course of starting an hbase process,\n# so try to keep things idempotent unless you want to take an even deeper look\n# into the startup scripts (bin/hbase, etc.)\n\n# The java implementation to use.  Java 1.7+ required.\n#export JAVA_HOME=/usr/lib/jvm/j2sdk1.8-oracle\n\n# Extra Java CLASSPATH elements.  Optional.\n# export HBASE_CLASSPATH=\n\n# The maximum amount of heap to use. Default is left to JVM default.\n# export HBASE_HEAPSIZE=1G\n\n# Uncomment below if you intend to use off heap cache. For example, to allocate 8G of \n# offheap, set the value to \"8G\".\n# export HBASE_OFFHEAPSIZE=1G\n\n# Extra Java runtime options.\n# Below are what we set by default.  May only work with SUN JVM.\n# For more on why as well as other possible settings,\n# see http://wiki.apache.org/hadoop/PerformanceTuning\nexport HBASE_OPTS=\"-XX:+UseG1GC -XX:MaxGCPauseMillis=50 -XX:ParallelGCThreads=20 -Dsun.net.inetaddr.ttl=10 \"\n\n# Configure PermSize. Only needed in JDK7. You can safely remove it for JDK8+\n#export HBASE_MASTER_OPTS=\"$HBASE_MASTER_OPTS -XX:PermSize=128m -XX:MaxPermSize=128m\"\n#export HBASE_REGIONSERVER_OPTS=\"$HBASE_REGIONSERVER_OPTS -XX:PermSize=128m -XX:MaxPermSize=128m\"\n\n# Uncomment one of the below three options to enable java garbage collection logging for the server-side processes.\n\n# This enables basic gc logging to the .out file.\n# export SERVER_GC_OPTS=\"-verbose:gc -XX:+PrintGCDetails -XX:+PrintGCDateStamps\"\n\n# This enables basic gc logging to its own file.\n# If FILE-PATH is not replaced, the log file(.gc) would still be generated in the HBASE_LOG_DIR .\n# export SERVER_GC_OPTS=\"-verbose:gc -XX:+PrintGCDetails -XX:+PrintGCDateStamps -Xloggc:<FILE-PATH>\"\n\n# This enables basic GC logging to its own file with automatic log rolling. Only applies to jdk 1.6.0_34+ and 1.7.0_2+.\n# If FILE-PATH is not replaced, the log file(.gc) would still be generated in the HBASE_LOG_DIR .\nexport SERVER_GC_OPTS=\"-verbose:gc -XX:+PrintGCDetails -XX:+PrintGCDateStamps -Xloggc:<FILE-PATH> -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=1 -XX:GCLogFileSize=512M\"\n\n# Uncomment one of the below three options to enable java garbage collection logging for the client processes.\n\n# This enables basic gc logging to the .out file.\n# export CLIENT_GC_OPTS=\"-verbose:gc -XX:+PrintGCDetails -XX:+PrintGCDateStamps\"\n\n# This enables basic gc logging to its own file.\n# If FILE-PATH is not replaced, the log file(.gc) would still be generated in the HBASE_LOG_DIR .\n# export CLIENT_GC_OPTS=\"-verbose:gc -XX:+PrintGCDetails -XX:+PrintGCDateStamps -Xloggc:<FILE-PATH>\"\n\n# This enables basic GC logging to its own file with automatic log rolling. Only applies to jdk 1.6.0_34+ and 1.7.0_2+.\n# If FILE-PATH is not replaced, the log file(.gc) would still be generated in the HBASE_LOG_DIR .\n# export CLIENT_GC_OPTS=\"-verbose:gc -XX:+PrintGCDetails -XX:+PrintGCDateStamps -Xloggc:<FILE-PATH> -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=1 -XX:GCLogFileSize=512M\"\n\n# See the package documentation for org.apache.hadoop.hbase.io.hfile for other configurations\n# needed setting up off-heap block caching. \n\n# Uncomment and adjust to enable JMX exporting\n# See jmxremote.password and jmxremote.access in $JRE_HOME/lib/management to configure remote password access.\n# More details at: http://java.sun.com/javase/6/docs/technotes/guides/management/agent.html\n# NOTE: HBase provides an alternative JMX implementation to fix the random ports issue, please see JMX\n# section in HBase Reference Guide for instructions.\n\nexport HBASE_JMX_BASE=\"-Dsun.net.inetaddr.ttl=10 -Dcom.sun.management.jmxremote.ssl=false -Dcom.sun.management.jmxremote.authenticate=false\"\nexport HBASE_MASTER_OPTS=\"$HBASE_MASTER_OPTS $HBASE_JMX_BASE -Dcom.sun.management.jmxremote.port=10103  -Xms2048m -Xmx2048m \"\nexport HBASE_REGIONSERVER_OPTS=\"$HBASE_REGIONSERVER_OPTS $HBASE_JMX_BASE -Dcom.sun.management.jmxremote.port=10104 -Xms2048m -Xmx2048m  \"\n# export HBASE_THRIFT_OPTS=\"$HBASE_THRIFT_OPTS $HBASE_JMX_BASE -Dcom.sun.management.jmxremote.port=10103\"\nexport HBASE_ZOOKEEPER_OPTS=\"$HBASE_ZOOKEEPER_OPTS $HBASE_JMX_BASE -Dcom.sun.management.jmxremote.port=10105  -Xms1024m -Xmx1024m \"\n# export HBASE_REST_OPTS=\"$HBASE_REST_OPTS $HBASE_JMX_BASE -Dcom.sun.management.jmxremote.port=10105\"\n\n# File naming hosts on which HRegionServers will run.  $HBASE_HOME/conf/regionservers by default.\n# export HBASE_REGIONSERVERS=${HBASE_HOME}/conf/regionservers\n\n# Uncomment and adjust to keep all the Region Server pages mapped to be memory resident\n#HBASE_REGIONSERVER_MLOCK=true\n#HBASE_REGIONSERVER_UID=\"hbase\"\n\n# File naming hosts on which backup HMaster will run.  $HBASE_HOME/conf/backup-masters by default.\n# export HBASE_BACKUP_MASTERS=${HBASE_HOME}/conf/backup-masters\n\n# Extra ssh options.  Empty by default.\n# export HBASE_SSH_OPTS=\"-o ConnectTimeout=1 -o SendEnv=HBASE_CONF_DIR\"\n\n# Where log files are stored.  $HBASE_HOME/logs by default.\n# export HBASE_LOG_DIR=${HBASE_HOME}/logs\n\n# Enable remote JDWP debugging of major HBase processes. Meant for Core Developers \n# export HBASE_MASTER_OPTS=\"$HBASE_MASTER_OPTS -Xdebug -Xrunjdwp:transport=dt_socket,server=y,suspend=n,address=8070\"\n# export HBASE_REGIONSERVER_OPTS=\"$HBASE_REGIONSERVER_OPTS -Xdebug -Xrunjdwp:transport=dt_socket,server=y,suspend=n,address=8071\"\n# export HBASE_THRIFT_OPTS=\"$HBASE_THRIFT_OPTS -Xdebug -Xrunjdwp:transport=dt_socket,server=y,suspend=n,address=8072\"\n# export HBASE_ZOOKEEPER_OPTS=\"$HBASE_ZOOKEEPER_OPTS -Xdebug -Xrunjdwp:transport=dt_socket,server=y,suspend=n,address=8073\"\n\n# A string representing this instance of hbase. $USER by default.\n# export HBASE_IDENT_STRING=$USER\n\n# The scheduling priority for daemon processes.  See 'man nice'.\n# export HBASE_NICENESS=10\n\n# The directory where pid files are stored. /tmp by default.\nexport HBASE_PID_DIR=/var/run/hbase\n\n# Seconds to sleep between slave commands.  Unset by default.  This\n# can be useful in large clusters, where, e.g., slave rsyncs can\n# otherwise arrive faster than the master can service them.\n# export HBASE_SLAVE_SLEEP=0.1\n\n# Tell HBase whether it should manage it's own instance of Zookeeper or not.\nexport HBASE_MANAGES_ZK=false\n\n# The default log rolling policy is RFA, where the log file is rolled as per the size defined for the \n# RFA appender. Please refer to the log4j.properties file to see more details on this appender.\n# In case one needs to do log rolling on a date change, one should set the environment property\n# HBASE_ROOT_LOGGER to \"<DESIRED_LOG LEVEL>,DRFA\".\n# For example:\n# HBASE_ROOT_LOGGER=INFO,DRFA\n# The reason for changing default to RFA is to avoid the boundary case of filling out disk space as \n# DRFA doesn't put any cap on the log size. Please refer to HBase-5655 for more context.\n\nexport LD_LIBRARY_PATH=/opt/hadoop/lib/native\n"
      hbase-policy.xml: "<?xml version=\"1.0\"?>\n<?xml-stylesheet type=\"text/xsl\" href=\"configuration.xsl\"?>\n<!--\n/**\n * Licensed to the Apache Software Foundation (ASF) under one\n * or more contributor license agreements.  See the NOTICE file\n * distributed with this work for additional information\n * regarding copyright ownership.  The ASF licenses this file\n * to you under the Apache License, Version 2.0 (the\n * \"License\"); you may not use this file except in compliance\n * with the License.  You may obtain a copy of the License at\n *\n *     http://www.apache.org/licenses/LICENSE-2.0\n *\n * Unless required by applicable law or agreed to in writing, software\n * distributed under the License is distributed on an \"AS IS\" BASIS,\n * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\n * See the License for the specific language governing permissions and\n * limitations under the License.\n */\n-->\n\n<configuration>\n  <property>\n    <name>security.client.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for ClientProtocol and AdminProtocol implementations (ie. \n    clients talking to HRegionServers)\n    The ACL is a comma-separated list of user and group names. The user and \n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\". \n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.admin.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for HMasterInterface protocol implementation (ie. \n    clients talking to HMaster for admin operations).\n    The ACL is a comma-separated list of user and group names. The user and \n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\". \n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.masterregion.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for HMasterRegionInterface protocol implementations\n    (for HRegionServers communicating with HMaster)\n    The ACL is a comma-separated list of user and group names. The user and \n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\". \n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n</configuration>\n"
      hbase-site.xml: |+
        <?xml version="1.0"?>
        <?xml-stylesheet type="text/xsl" href="configuration.xsl"?>
        <configuration>

        <property>
        <name>dfs.client.retry.policy.enabled</name>
        <value>true</value>

        </property>

        <property>
        <name>dfs.client.retry.policy.spec</name>
        <value>1000,1</value>

        </property>

        <property>
        <name>dfs.client.read.shortcircuit</name>
        <value>true</value>

        </property>

        <property>
        <name>dfs.domain.socket.path</name>
        <value>/var/run/hadoop/dn._PORT</value>

        </property>

        <property>
        <name>hbase.assignment.usezk</name>
        <value>false</value>

        </property>

        <property>
        <name>hbase.oldwals.cleaner.thread.timeout.msec</name>
        <value>60000</value>

        </property>

        <property>
        <name>hbase.oldwals.cleaner.thread.check.interval.msec</name>
        <value>60000</value>

        </property>

        <property>
        <name>hbase.replication</name>
        <value>true</value>

        </property>

        <property>
        <name>hbase.master.logcleaner.plugins</name>
        <value>org.apache.hadoop.hbase.master.cleaner.TimeToLiveLogCleaner,org.apache.hadoop.hbase.master.cleaner.TimeToLiveProcedureWALCleaner,org.apache.hadoop.hbase.replication.master.ReplicationLogCleaner</value>

        </property>

        <property>
        <name>hbase.procedure.master.classes</name>
        <value>org.apache.hadoop.hbase.backup.master.LogRollMasterProcedureManager</value>

        </property>

        <property>
        <name>hbase.procedure.regionserver.classes</name>
        <value>org.apache.hadoop.hbase.backup.regionserver.LogRollRegionServerProcedureManager</value>

        </property>

        <property>
        <name>hbase.cluster.distributed</name>
        <value>true</value>

        </property>

        <property>
        <name>hbase.rootdir</name>
        <value>hdfs://hbase-store/hbase</value>

        </property>

        <property>
        <name>hbase.zookeeper.quorum</name>
        <value>hbase-cluster-zk-0.hbase-cluster.hbase-cluster-ns.svc.cluster.local,hbase-cluster-zk-1.hbase-cluster.hbase-cluster-ns.svc.cluster.local,hbase-cluster-zk-2.hbase-cluster.hbase-cluster-ns.svc.cluster.local</value>

        </property>

        <property>
        <name>zookeeper.znode.parent</name>
        <value>/hbase</value>

        </property>

        <property>
        <name>zookeeper.session.timeout</name>
        <value>30000</value>

        </property>

        <property>
        <name>hbase.hregion.memstore.flush.size</name>
        <value>128000000</value>

        </property>

        <property>
        <name>hbase.zookeeper.property.tickTime</name>
        <value>6000</value>

        </property>

        <property>
        <name>hbase.zookeeper.property.4lw.commands.whitelist</name>
        <value>*</value>

        </property>

        <property>
        <name>hbase.master.hfilecleaner.ttl</name>
        <value>600000</value>

        </property>

        <property>
        <name>hbase.balancer.period</name>
        <value>1800000</value>

        </property>

        <property>
        <name>hbase.master.logcleaner.ttl</name>
        <value>60000</value>

        </property>

        <property>
        <name>hbase.zookeeper.property.maxClientCnxns</name>
        <value>4000</value>

        </property>

        <property>
        <name>hbase.zookeeper.property.autopurge.purgeInterval</name>
        <value>1</value>

        </property>

        <property>
        <name>hbase.zookeeper.property.autopurge.snapRetainCount</name>
        <value>3</value>

        </property>

        <property>
        <name>hbase.master.balancer.stochastic.runMaxSteps</name>
        <value>true</value>

        </property>

        <property>
        <name>hbase.master.balancer.stochastic.minCostNeedBalance</name>
        <value>0.05f</value>

        </property>

        <property>
        <name>master.balancer.stochastic.maxSteps</name>
        <value>1000000</value>

        </property>

        <property>
        <name>hbase.zookeeper.property.dataDir</name>
        <value>/grid/1/zk</value>

        </property>

        <property>
        <name>hbase.security.authentication</name>
        <value>simple</value>

        </property>

        <property>
        <name>hbase.security.authorization</name>
        <value>true</value>

        </property>

        <property>
        <name>hbase.coprocessor.master.classes</name>
        <value>org.apache.hadoop.hbase.rsgroup.RSGroupAdminEndpoint,org.apache.hadoop.hbase.security.access.AccessController</value>

        </property>

        <property>
        <name>hbase.coprocessor.regionserver.classes</name>
        <value>org.apache.hadoop.hbase.security.access.AccessController</value>

        </property>

        <property>
        <name>hbase.master.loadbalancer.class</name>
        <value>org.apache.hadoop.hbase.rsgroup.RSGroupBasedLoadBalancer</value>

        </property>

        <property>
        <name>dfs.nameservices</name>
        <value>hbase-store</value>

        </property>

        <property>
        <name>dfs.ha.namenodes.hbase-store</name>
        <value>nn1,nn2</value>

        </property>

        <property>
        <name>dfs.namenode.rpc-address.hbase-store.nn1</name>
        <value>hbase-cluster-nn-0.hbase-cluster.hbase-cluster-ns.svc.cluster.local:8020</value>

        </property>

        <property>
        <name>dfs.namenode.http-address.hbase-store.nn1</name>
        <value>hbase-cluster-nn-0.hbase-cluster.hbase-cluster-ns.svc.cluster.local:50070</value>

        </property>

        <property>
        <name>dfs.namenode.rpc-address.hbase-store.nn2</name>
        <value>hbase-cluster-nn-1.hbase-cluster.hbase-cluster-ns.svc.cluster.local:8020</value>

        </property>

        <property>
        <name>dfs.namenode.http-address.hbase-store.nn2</name>
        <value>hbase-cluster-nn-1.hbase-cluster.hbase-cluster-ns.svc.cluster.local:50070</value>

        </property>

        <property>
        <name>dfs.client.failover.proxy.provider.hbase-store</name>
        <value>org.apache.hadoop.hdfs.server.namenode.ha.ConfiguredFailoverProxyProvider</value>

        </property>

        <property>
        <name>dfs.datanode.address</name>
        <value>0.0.0.0:9866</value>

        </property>

        <property>
        <name>hbase.hregion.majorcompaction</name>
        <value>0</value>

        </property>

        <property>
        <name>hbase.procedure.store.wal.use.hsync</name>
        <value>true</value>

        </property>

        <property>
        <name>net.topology.script.file.name</name>
        <value>/opt/scripts/rack_topology</value>

        </property>


        <property>
        <name>hbase.regionserver.hostname.disable.master.reversedns</name>
        <value>true</value>

        </property>

        <!-- appended configuration used in override -->

        </configuration>

      log4j.properties: "# Licensed to the Apache Software Foundation (ASF) under one\n# or more contributor license agreements.  See the NOTICE file\n# distributed with this work for additional information\n# regarding copyright ownership.  The ASF licenses this file\n# to you under the Apache License, Version 2.0 (the\n# \"License\"); you may not use this file except in compliance\n# with the License.  You may obtain a copy of the License at\n#\n#     http://www.apache.org/licenses/LICENSE-2.0\n#\n# Unless required by applicable law or agreed to in writing, software\n# distributed under the License is distributed on an \"AS IS\" BASIS,\n# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\n# See the License for the specific language governing permissions and\n# limitations under the License.\n\n# Define some default values that can be overridden by system properties\nhbase.root.logger=INFO,console\nhbase.security.logger=INFO,console\nhbase.log.dir=.\nhbase.log.file=hbase.log\n\n# Define the root logger to the system property \"hbase.root.logger\".\nlog4j.rootLogger=${hbase.root.logger}\n\n# Logging Threshold\nlog4j.threshold=ALL\n\n#\n# Daily Rolling File Appender\n#\nlog4j.appender.DRFA=org.apache.log4j.DailyRollingFileAppender\nlog4j.appender.DRFA.File=${hbase.log.dir}/${hbase.log.file}\n\n# Rollver at midnight\nlog4j.appender.DRFA.DatePattern=.yyyy-MM-dd\n\n# 30-day backup\n#log4j.appender.DRFA.MaxBackupIndex=30\nlog4j.appender.DRFA.layout=org.apache.log4j.PatternLayout\n\n# Pattern format: Date LogLevel LoggerName LogMessage\nlog4j.appender.DRFA.layout.ConversionPattern=%d{ISO8601} %-5p [%t] %c{2}: %m%n\n\n# Rolling File Appender properties\nhbase.log.maxfilesize=256MB\nhbase.log.maxbackupindex=5\n\n# Rolling File Appender\nlog4j.appender.RFA=org.apache.log4j.RollingFileAppender\nlog4j.appender.RFA.File=${hbase.log.dir}/${hbase.log.file}\n\nlog4j.appender.RFA.MaxFileSize=${hbase.log.maxfilesize}\nlog4j.appender.RFA.MaxBackupIndex=${hbase.log.maxbackupindex}\n\nlog4j.appender.RFA.layout=org.apache.log4j.PatternLayout\nlog4j.appender.RFA.layout.ConversionPattern=%d{ISO8601} %-5p [%t] %c{2}: %m%n\n\n#\n# Security audit appender\n#\nhbase.security.log.file=SecurityAuth.audit\nhbase.security.log.maxfilesize=256MB\nhbase.security.log.maxbackupindex=5\nlog4j.appender.RFAS=org.apache.log4j.RollingFileAppender\nlog4j.appender.RFAS.File=${hbase.log.dir}/${hbase.security.log.file}\nlog4j.appender.RFAS.MaxFileSize=${hbase.security.log.maxfilesize}\nlog4j.appender.RFAS.MaxBackupIndex=${hbase.security.log.maxbackupindex}\nlog4j.appender.RFAS.layout=org.apache.log4j.PatternLayout\nlog4j.appender.RFAS.layout.ConversionPattern=%d{ISO8601} %p %c: %m%n\nlog4j.category.SecurityLogger=${hbase.security.logger}\nlog4j.additivity.SecurityLogger=false\n#log4j.logger.SecurityLogger.org.apache.hadoop.hbase.security.access.AccessController=TRACE\n#log4j.logger.SecurityLogger.org.apache.hadoop.hbase.security.visibility.VisibilityController=TRACE\n\n#\n# Null Appender\n#\nlog4j.appender.NullAppender=org.apache.log4j.varia.NullAppender\n\n#\n# console\n# Add \"console\" to rootlogger above if you want to use this \n#\nlog4j.appender.console=org.apache.log4j.ConsoleAppender\nlog4j.appender.console.target=System.err\nlog4j.appender.console.layout=org.apache.log4j.PatternLayout\nlog4j.appender.console.layout.ConversionPattern=%d{ISO8601} %-5p [%t] %c{2}: %m%n\n\n# Custom Logging levels\n\nlog4j.logger.org.apache.zookeeper=INFO\n#log4j.logger.org.apache.hadoop.fs.FSNamesystem=DEBUG\nlog4j.logger.org.apache.hadoop.hbase=INFO\n# Make these two classes INFO-level. Make them DEBUG to see more zk debug.\nlog4j.logger.org.apache.hadoop.hbase.zookeeper.ZKUtil=INFO\nlog4j.logger.org.apache.hadoop.hbase.zookeeper.ZooKeeperWatcher=INFO\n#log4j.logger.org.apache.hadoop.dfs=DEBUG\n# Set this class to log INFO only otherwise its OTT\n# Enable this to get detailed connection error/retry logging.\n# log4j.logger.org.apache.hadoop.hbase.client.HConnectionManager$HConnectionImplementation=TRACE\n\n\n# Uncomment this line to enable tracing on _every_ RPC call (this can be a lot of output)\n#log4j.logger.org.apache.hadoop.ipc.HBaseServer.trace=DEBUG\n\n# Uncomment the below if you want to remove logging of client region caching'\n# and scan of hbase:meta messages\n# log4j.logger.org.apache.hadoop.hbase.client.HConnectionManager$HConnectionImplementation=INFO\n# log4j.logger.org.apache.hadoop.hbase.client.MetaScanner=INFO\n"
    hadoopConfigName: hadoop-config
    hadoopConfigMountPath: /etc/hadoop
    hadoopConfig:
      configuration.xsl: |
        <?xml version="1.0"?>
        <!--
           Licensed to the Apache Software Foundation (ASF) under one or more
           contributor license agreements.  See the NOTICE file distributed with
           this work for additional information regarding copyright ownership.
           The ASF licenses this file to You under the Apache License, Version 2.0
           (the "License"); you may not use this file except in compliance with
           the License.  You may obtain a copy of the License at

               http://www.apache.org/licenses/LICENSE-2.0

           Unless required by applicable law or agreed to in writing, software
           distributed under the License is distributed on an "AS IS" BASIS,
           WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
           See the License for the specific language governing permissions and
           limitations under the License.
        -->
        <xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">
        <xsl:output method="html"/>
        <xsl:template match="configuration">
        <html>
        <body>
        <table border="1">
        <tr>
         <td>name</td>
         <td>value</td>
         <td>description</td>
        </tr>
        <xsl:for-each select="property">
        <tr>
          <td><a name="{name}"><xsl:value-of select="name"/></a></td>
          <td><xsl:value-of select="value"/></td>
          <td><xsl:value-of select="description"/></td>
        </tr>
        </xsl:for-each>
        </table>
        </body>
        </html>
        </xsl:template>
        </xsl:stylesheet>
      core-site.xml: |
        <?xml version="1.0"?>
        <?xml-stylesheet type="text/xsl" href="configuration.xsl"?>
        <configuration>

        <property>
        <name>fs.trash.interval</name>
        <value>1440</value>
        </property>

        <property>
        <name>io.compression.codecs</name>
        <value>org.apache.hadoop.io.compress.GzipCodec,org.apache.hadoop.io.compress.DefaultCodec,org.apache.hadoop.io.compress.BZip2Codec,com.hadoop.compression.lzo.LzoCodec,com.hadoop.compression.lzo.LzopCodec</value>
        </property>

        <property>
        <name>io.compression.codec.lzo.class</name>
        <value>com.hadoop.compression.lzo.LzoCodec</value>
        </property>

        <property>
        <name>fs.default.name</name>
        <value>hdfs://hbase-store</value>
        </property>

        <property>
        <name>ha.zookeeper.quorum</name>
        <value>hbase-cluster-zk-0.hbase-cluster.hbase-cluster-ns.svc.cluster.local,hbase-cluster-zk-1.hbase-cluster.hbase-cluster-ns.svc.cluster.local,hbase-cluster-zk-2.hbase-cluster.hbase-cluster-ns.svc.cluster.local</value>
        </property>

        <property>
        <name>ha.zookeeper.parent-znode</name>
        <value>/hbase/hadoop-ha</value>
        </property>

        </configuration>
      dfs.exclude: ""
      dfs.include: |
        hbase-tenant-dn-0.hbase-tenant.hbase-tenant-ns.svc.cluster.local
        hbase-tenant-dn-1.hbase-tenant.hbase-tenant-ns.svc.cluster.local
        hbase-tenant-dn-2.hbase-tenant.hbase-tenant-ns.svc.cluster.local
        hbase-tenant-dn-3.hbase-tenant.hbase-tenant-ns.svc.cluster.local
        hbase-cluster-dn-0.hbase-cluster.hbase-cluster-ns.svc.cluster.local
        hbase-cluster-dn-1.hbase-cluster.hbase-cluster-ns.svc.cluster.local
        hbase-cluster-dn-2.hbase-cluster.hbase-cluster-ns.svc.cluster.local
      hadoop-env.sh: "# Licensed to the Apache Software Foundation (ASF) under one\n# or more contributor license agreements.  See the NOTICE file\n# distributed with this work for additional information\n# regarding copyright ownership.  The ASF licenses this file\n# to you under the Apache License, Version 2.0 (the\n# \"License\"); you may not use this file except in compliance\n# with the License.  You may obtain a copy of the License at\n#\n#     http://www.apache.org/licenses/LICENSE-2.0\n#\n# Unless required by applicable law or agreed to in writing, software\n# distributed under the License is distributed on an \"AS IS\" BASIS,\n# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\n# See the License for the specific language governing permissions and\n# limitations under the License.\n\n# Set Hadoop-specific environment variables here.\n\n# The only required environment variable is JAVA_HOME.  All others are\n# optional.  When running a distributed configuration it is best to\n# set JAVA_HOME in this file, so that it is correctly defined on\n# remote nodes.\n\n# The java implementation to use.\n#export JAVA_HOME=/usr/lib/jvm/j2sdk1.8-oracle\n\n# The jsvc implementation to use. Jsvc is required to run secure datanodes\n# that bind to privileged ports to provide authentication of data transfer\n# protocol.  Jsvc is not required if SASL is configured for authentication of\n# data transfer protocol using non-privileged ports.\n#export JSVC_HOME=${JSVC_HOME}\n\nexport HADOOP_CONF_DIR=/etc/hadoop\nexport HADOOP_PID_DIR=/var/run/hadoop\n\n# Extra Java CLASSPATH elements.  Automatically insert capacity-scheduler.\nfor f in $HADOOP_HOME/contrib/capacity-scheduler/*.jar; do\n  if [ \"$HADOOP_CLASSPATH\" ]; then\n    export HADOOP_CLASSPATH=$HADOOP_CLASSPATH:$f\n  else\n    export HADOOP_CLASSPATH=$f\n  fi\ndone\n\n# The maximum amount of heap to use, in MB. Default is 1000.\n#export HADOOP_HEAPSIZE=\"\"\n#export HADOOP_NAMENODE_INIT_HEAPSIZE=\"\"\n\n# Extra Java runtime options.  Empty by default.\n#export HADOOP_OPTS=\"$HADOOP_OPTS -Djava.net.preferIPv4Stack=true -Dcom.sun.management.jmxremote -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.port=1234 -Dcom.sun.management.jmxremote.ssl=false -XX:+UnlockCommercialFeatures -XX:+FlightRecorder\"\nexport HADOOP_OPTS=\"$HADOOP_OPTs  -Djava.net.preferIPv4Stack=true -Dsun.net.inetaddr.ttl=10 -XX:+UseG1GC -XX:MaxGCPauseMillis=50 -XX:ParallelGCThreads=8 \"\n\n# Command specific options appended to HADOOP_OPTS when specified\nexport HDFS_NAMENODE_OPTS=\"  -Xms2048m -Xmx2048m   -Dcom.sun.management.jmxremote -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.port=10102 -Dcom.sun.management.jmxremote.ssl=false -Dhadoop.security.logger=${HADOOP_SECURITY_LOGGER:-INFO,RFAS} -Dhdfs.audit.logger=${HDFS_AUDIT_LOGGER:-INFO,NullAppender} \"\nexport HDFS_DATANODE_OPTS=\"  -Xms2048m -Xmx2048m  -Dcom.sun.management.jmxremote -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.port=10101 -Dcom.sun.management.jmxremote.ssl=false -Dhadoop.security.logger=ERROR,RFAS \"\nexport HDFS_JOURNALNODE_OPTS=\" -Xms512m -Xmx512m   -Dcom.sun.management.jmxremote -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.port=10106 -Dcom.sun.management.jmxremote.ssl=false \"\nexport HDFS_ZKFC_OPTS=\"  -Dcom.sun.management.jmxremote -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.port=10107 -Dcom.sun.management.jmxremote.ssl=false \"\n\nexport HADOOP_SECONDARYNAMENODE_OPTS=\" -Xms2048m -Xmx2048m   -Dcom.sun.management.jmxremote -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.port=10102 -Dcom.sun.management.jmxremote.ssl=false -XX:+UnlockCommercialFeatures -XX:+FlightRecorder  -Dhadoop.security.logger=${HADOOP_SECURITY_LOGGER:-INFO,RFAS} -Dhdfs.audit.logger=${HDFS_AUDIT_LOGGER:-INFO,NullAppender} \"\n\nexport HADOOP_NFS3_OPTS=\"$HADOOP_NFS3_OPTS\"\nexport HADOOP_PORTMAP_OPTS=\"-Xmx512m $HADOOP_PORTMAP_OPTS\"\n\n# The following applies to multiple commands (fs, dfs, fsck, distcp etc)\nexport HADOOP_CLIENT_OPTS=\"-Xmx512m $HADOOP_CLIENT_OPTS\"\n#HADOOP_JAVA_PLATFORM_OPTS=\"-XX:-UsePerfData $HADOOP_JAVA_PLATFORM_OPTS\"\n\n# On secure datanodes, user to run the datanode as after dropping privileges.\n# This **MUST** be uncommented to enable secure HDFS if using privileged ports\n# to provide authentication of data transfer protocol.  This **MUST NOT** be\n# defined if SASL is configured for authentication of data transfer protocol\n# using non-privileged ports.\nexport HADOOP_SECURE_DN_USER=${HADOOP_SECURE_DN_USER}\n\n# Where log files are stored.  $HADOOP_HOME/logs by default.\n#export HADOOP_LOG_DIR=${HADOOP_LOG_DIR}/$USER\n\n# Where log files are stored in the secure data environment.\nexport HADOOP_SECURE_LOG_DIR=${HADOOP_LOG_DIR}/${HADOOP_HDFS_USER}\n\n###\n# HDFS Mover specific parameters\n###\n# Specify the JVM options to be used when starting the HDFS Mover.\n# These options will be appended to the options specified as HADOOP_OPTS\n# and therefore may override any similar flags set in HADOOP_OPTS\n#\n# export HADOOP_MOVER_OPTS=\"\"\n\n###\n# Advanced Users Only!\n###\n\n# The directory where pid files are stored. /tmp by default.\n# NOTE: this should be set to a directory that can only be written to by \n#       the user that will run the hadoop daemons.  Otherwise there is the\n#       potential for a symlink attack.\nexport HADOOP_PID_DIR=${HADOOP_PID_DIR}\nexport HADOOP_SECURE_PID_DIR=${HADOOP_PID_DIR}\n\n# A string representing this instance of hadoop. $USER by default.\nexport HADOOP_IDENT_STRING=$USER\n"
      hadoop-metrics.properties: |+
        # Configuration of the "dfs" context for null
        dfs.class=org.apache.hadoop.metrics.spi.NullContext

        # Configuration of the "dfs" context for file
        #dfs.class=org.apache.hadoop.metrics.file.FileContext
        #dfs.period=10
        #dfs.fileName=/tmp/dfsmetrics.log

        # Configuration of the "dfs" context for ganglia
        # Pick one: Ganglia 3.0 (former) or Ganglia 3.1 (latter)
        # dfs.class=org.apache.hadoop.metrics.ganglia.GangliaContext
        # dfs.class=org.apache.hadoop.metrics.ganglia.GangliaContext31
        # dfs.period=10
        # dfs.servers=localhost:8649


        # Configuration of the "mapred" context for null
        mapred.class=org.apache.hadoop.metrics.spi.NullContext

        # Configuration of the "mapred" context for file
        #mapred.class=org.apache.hadoop.metrics.file.FileContext
        #mapred.period=10
        #mapred.fileName=/tmp/mrmetrics.log

        # Configuration of the "mapred" context for ganglia
        # Pick one: Ganglia 3.0 (former) or Ganglia 3.1 (latter)
        # mapred.class=org.apache.hadoop.metrics.ganglia.GangliaContext
        # mapred.class=org.apache.hadoop.metrics.ganglia.GangliaContext31
        # mapred.period=10
        # mapred.servers=localhost:8649


        # Configuration of the "jvm" context for null
        #jvm.class=org.apache.hadoop.metrics.spi.NullContext

        # Configuration of the "jvm" context for file
        #jvm.class=org.apache.hadoop.metrics.file.FileContext
        #jvm.period=10
        #jvm.fileName=/tmp/jvmmetrics.log

        # Configuration of the "jvm" context for ganglia
        # jvm.class=org.apache.hadoop.metrics.ganglia.GangliaContext
        # jvm.class=org.apache.hadoop.metrics.ganglia.GangliaContext31
        # jvm.period=10
        # jvm.servers=localhost:8649

        # Configuration of the "rpc" context for null
        rpc.class=org.apache.hadoop.metrics.spi.NullContext

        # Configuration of the "rpc" context for file
        #rpc.class=org.apache.hadoop.metrics.file.FileContext
        #rpc.period=10
        #rpc.fileName=/tmp/rpcmetrics.log

        # Configuration of the "rpc" context for ganglia
        # rpc.class=org.apache.hadoop.metrics.ganglia.GangliaContext
        # rpc.class=org.apache.hadoop.metrics.ganglia.GangliaContext31
        # rpc.period=10
        # rpc.servers=localhost:8649


        # Configuration of the "ugi" context for null
        ugi.class=org.apache.hadoop.metrics.spi.NullContext

        # Configuration of the "ugi" context for file
        #ugi.class=org.apache.hadoop.metrics.file.FileContext
        #ugi.period=10
        #ugi.fileName=/tmp/ugimetrics.log

        # Configuration of the "ugi" context for ganglia
        # ugi.class=org.apache.hadoop.metrics.ganglia.GangliaContext
        # ugi.class=org.apache.hadoop.metrics.ganglia.GangliaContext31
        # ugi.period=10
        # ugi.servers=localhost:8649

      hadoop-metrics2.properties: "# syntax: [prefix].[source|sink].[instance].[options]\n# See javadoc of package-info.java for org.apache.hadoop.metrics2 for details\n\n*.sink.file.class=org.apache.hadoop.metrics2.sink.FileSink\n# default sampling period, in seconds\n*.period=10\n\n# The namenode-metrics.out will contain metrics from all context\n#namenode.sink.file.filename=namenode-metrics.out\n# Specifying a special sampling period for namenode:\n#namenode.sink.*.period=8\n\n#datanode.sink.file.filename=datanode-metrics.out\n\n#resourcemanager.sink.file.filename=resourcemanager-metrics.out\n\n#nodemanager.sink.file.filename=nodemanager-metrics.out\n\n#mrappmaster.sink.file.filename=mrappmaster-metrics.out\n\n#jobhistoryserver.sink.file.filename=jobhistoryserver-metrics.out\n\n# the following example split metrics of different\n# context to different sinks (in this case files)\n#nodemanager.sink.file_jvm.class=org.apache.hadoop.metrics2.sink.FileSink\n#nodemanager.sink.file_jvm.context=jvm\n#nodemanager.sink.file_jvm.filename=nodemanager-jvm-metrics.out\n#nodemanager.sink.file_mapred.class=org.apache.hadoop.metrics2.sink.FileSink\n#nodemanager.sink.file_mapred.context=mapred\n#nodemanager.sink.file_mapred.filename=nodemanager-mapred-metrics.out\n\n#\n# Below are for sending metrics to Ganglia\n#\n# for Ganglia 3.0 support\n# *.sink.ganglia.class=org.apache.hadoop.metrics2.sink.ganglia.GangliaSink30\n#\n# for Ganglia 3.1 support\n# *.sink.ganglia.class=org.apache.hadoop.metrics2.sink.ganglia.GangliaSink31\n\n# *.sink.ganglia.period=10\n\n# default for supportsparse is false\n# *.sink.ganglia.supportsparse=true\n\n#*.sink.ganglia.slope=jvm.metrics.gcCount=zero,jvm.metrics.memHeapUsedM=both\n#*.sink.ganglia.dmax=jvm.metrics.threadsBlocked=70,jvm.metrics.memHeapUsedM=40\n\n# Tag values to use for the ganglia prefix. If not defined no tags are used.\n# If '*' all tags are used. If specifiying multiple tags separate them with \n# commas. Note that the last segment of the property name is the context name.\n#\n#*.sink.ganglia.tagsForPrefix.jvm=ProcesName\n#*.sink.ganglia.tagsForPrefix.dfs=\n#*.sink.ganglia.tagsForPrefix.rpc=\n#*.sink.ganglia.tagsForPrefix.mapred=\n\n#namenode.sink.ganglia.servers=yourgangliahost_1:8649,yourgangliahost_2:8649\n\n#datanode.sink.ganglia.servers=yourgangliahost_1:8649,yourgangliahost_2:8649\n\n#resourcemanager.sink.ganglia.servers=yourgangliahost_1:8649,yourgangliahost_2:8649\n\n#nodemanager.sink.ganglia.servers=yourgangliahost_1:8649,yourgangliahost_2:8649\n\n#mrappmaster.sink.ganglia.servers=yourgangliahost_1:8649,yourgangliahost_2:8649\n\n#jobhistoryserver.sink.ganglia.servers=yourgangliahost_1:8649,yourgangliahost_2:8649\n"
      hadoop-policy.xml: "<?xml version=\"1.0\"?>\n<?xml-stylesheet type=\"text/xsl\" href=\"configuration.xsl\"?>\n<!--\n \n Licensed to the Apache Software Foundation (ASF) under one\n or more contributor license agreements.  See the NOTICE file\n distributed with this work for additional information\n regarding copyright ownership.  The ASF licenses this file\n to you under the Apache License, Version 2.0 (the\n \"License\"); you may not use this file except in compliance\n with the License.  You may obtain a copy of the License at\n\n     http://www.apache.org/licenses/LICENSE-2.0\n\n Unless required by applicable law or agreed to in writing, software\n distributed under the License is distributed on an \"AS IS\" BASIS,\n WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\n See the License for the specific language governing permissions and\n limitations under the License.\n\n-->\n\n<!-- Put site-specific property overrides in this file. -->\n\n<configuration>\n  <property>\n    <name>security.client.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for ClientProtocol, which is used by user code\n    via the DistributedFileSystem.\n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.client.datanode.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for ClientDatanodeProtocol, the client-to-datanode protocol\n    for block recovery.\n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.datanode.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for DatanodeProtocol, which is used by datanodes to\n    communicate with the namenode.\n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.inter.datanode.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for InterDatanodeProtocol, the inter-datanode protocol\n    for updating generation timestamp.\n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.namenode.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for NamenodeProtocol, the protocol used by the secondary\n    namenode to communicate with the namenode.\n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n <property>\n    <name>security.admin.operations.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for AdminOperationsProtocol. Used for admin commands.\n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.refresh.user.mappings.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for RefreshUserMappingsProtocol. Used to refresh\n    users mappings. The ACL is a comma-separated list of user and\n    group names. The user and group list is separated by a blank. For\n    e.g. \"alice,bob users,wheel\".  A special value of \"*\" means all\n    users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.refresh.policy.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for RefreshAuthorizationPolicyProtocol, used by the\n    dfsadmin and mradmin commands to refresh the security policy in-effect.\n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.ha.service.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for HAService protocol used by HAAdmin to manage the\n      active and stand-by states of namenode.</description>\n  </property>\n\n  <property>\n    <name>security.zkfc.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for access to the ZK Failover Controller\n    </description>\n  </property>\n\n  <property>\n    <name>security.qjournal.service.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for QJournalProtocol, used by the NN to communicate with\n    JNs when using the QuorumJournalManager for edit logs.</description>\n  </property>\n\n  <property>\n    <name>security.mrhs.client.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for HSClientProtocol, used by job clients to\n    communciate with the MR History Server job status etc. \n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <!-- YARN Protocols -->\n\n  <property>\n    <name>security.resourcetracker.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for ResourceTrackerProtocol, used by the\n    ResourceManager and NodeManager to communicate with each other.\n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.resourcemanager-administration.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for ResourceManagerAdministrationProtocol, for admin commands. \n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.applicationclient.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for ApplicationClientProtocol, used by the ResourceManager \n    and applications submission clients to communicate with each other.\n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.applicationmaster.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for ApplicationMasterProtocol, used by the ResourceManager \n    and ApplicationMasters to communicate with each other.\n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.containermanagement.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for ContainerManagementProtocol protocol, used by the NodeManager \n    and ApplicationMasters to communicate with each other.\n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.resourcelocalizer.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for ResourceLocalizer protocol, used by the NodeManager \n    and ResourceLocalizer to communicate with each other.\n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.job.task.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for TaskUmbilicalProtocol, used by the map and reduce\n    tasks to communicate with the parent tasktracker.\n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.job.client.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for MRClientProtocol, used by job clients to\n    communciate with the MR ApplicationMaster to query job status etc. \n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n\n  <property>\n    <name>security.applicationhistory.protocol.acl</name>\n    <value>*</value>\n    <description>ACL for ApplicationHistoryProtocol, used by the timeline\n    server and the generic history service client to communicate with each other.\n    The ACL is a comma-separated list of user and group names. The user and\n    group list is separated by a blank. For e.g. \"alice,bob users,wheel\".\n    A special value of \"*\" means all users are allowed.</description>\n  </property>\n</configuration>\n"
      hdfs-site.xml: "<?xml version=\"1.0\"?>\n<?xml-stylesheet type=\"text/xsl\" href=\"configuration.xsl\"?>\n<configuration>\n\n<property>\n<name>dfs.replication</name>\n<value>3</value>\n</property>\n\n<property>\n<name>dfs.replication.max</name>\n<value>3</value>\n</property>\n\n<property>\n<name>dfs.permissions</name>\n<value>true</value>\n</property>\n\n<property>\n<name>dfs.permissions.superusergroup</name>\n<value>hbase</value>\n</property>\n\n<property>\n<name>dfs.namenode.name.dir</name>\n<value>file:///grid/1/dfs/nn</value>\n</property>\n\n<property>\n<name>dfs.datanode.data.dir</name>\n<value>file:///grid/1/dfs/dn</value>\n</property>\n\n<property>\n<name>dfs.datanode.failed.volumes.tolerated</name>\n<value>0</value>\n</property>\n\n<property>\n<name>dfs.datanode.max.transfer.threads</name>\n<value>8192</value>\n</property>\n\n<property>\n<name>dfs.client.read.shortcircuit</name>\n<value>true</value>\n</property>\n\n<property>\n<name>dfs.block.local-path-access.user</name>\n<value>hbase</value>\n</property>\n\n<property>\n<name>dfs.domain.socket.path</name>\n<value>/var/run/hadoop/dn._PORT</value>\n</property>\n\n<property>\n<name>dfs.hosts</name>\n<value>/etc/hadoop/dfs.include</value>\n</property>\n\n<property>\n<name>dfs.hosts.exclude</name>\n<value>/etc/hadoop/dfs.exclude</value>\n</property>\n\n<property>\n<name>dfs.datanode.data.dir.perm</name>\n<value>700</value>\n</property>\n\n<property>\n<name>dfs.nameservices</name>\n<value>hbase-store</value>\n</property>\n\n<property>\n<name>dfs.namenodes.handler.count</name>\n<value>160</value>\n</property>\n\n<property>\n<name>dfs.ha.namenodes.hbase-store</name>\n<value>nn1,nn2</value>\n</property>\n\n<property>\n<name>dfs.namenode.rpc-address.hbase-store.nn1</name>\n<value>hbase-cluster-nn-0.hbase-cluster.hbase-cluster-ns.svc.cluster.local:8020</value>\n</property>\n\n<property>\n<name>dfs.namenode.http-address.hbase-store.nn1</name>\n<value>hbase-cluster-nn-0.hbase-cluster.hbase-cluster-ns.svc.cluster.local:50070</value>\n</property>\n\n<property>\n<name>dfs.namenode.rpc-address.hbase-store.nn2</name>\n<value>hbase-cluster-nn-1.hbase-cluster.hbase-cluster-ns.svc.cluster.local:8020</value>\n</property>\n\n<property>\n<name>dfs.namenode.http-address.hbase-store.nn2</name>\n<value>hbase-cluster-nn-1.hbase-cluster.hbase-cluster-ns.svc.cluster.local:50070</value>\n</property>\n\n<property>\n<name>dfs.namenode.http-bind-host</name>\n<value>0.0.0.0</value>\n</property>\n\n<property>\n<name>dfs.namenode.rpc-bind-host</name>\n<value>0.0.0.0</value>\n</property>\n\n<property>\n<name>dfs.namenode.shared.edits.dir</name>\n<value>qjournal://hbase-cluster-jn-0.hbase-cluster-ns.svc.cluster.local:8485;hbase-cluster-jn-1.hbase-cluster-ns.svc.cluster.local:8485;hbase-cluster-jn-2.hbase-cluster-ns.svc.cluster.local:8485/hbase-store</value>\n</property>\n\n<property>\n<name>dfs.journalnode.edits.dir</name>\n<value>/grid/1/dfs/jn</value>\n</property>\n\n<property>\n<name>dfs.client.failover.proxy.provider.hbase-store</name>\n<value>org.apache.hadoop.hdfs.server.namenode.ha.ConfiguredFailoverProxyProvider</value>\n</property>\n\n<property>\n<name>dfs.ha.fencing.methods</name>\n<value>sshfence\nshell(/bin/true)</value>\n</property>\n\n<property>\n<name>dfs.ha.fencing.ssh.private-key-files</name>\n<value>/var/lib/hadoop-hdfs/.ssh/id_dsa</value>\n</property>\n\n<property>\n<name>dfs.ha.fencing.ssh.connect-timeout</name>\n<value>20000</value>\n</property>\n\n<property>\n<name>dfs.ha.automatic-failover.enabled</name>\n<value>true</value>\n</property>\n\n<property>\n<name>dfs.webhdfs.enabled</name>\n<value>false</value>\n</property>\n\n<property>\n<name>dfs.datanode.block-pinning.enabled</name>\n<value>true</value>\n</property>\n\n<property>\n<name>dfs.namenode.avoid.read.stale.datanode</name>\n<value>true</value>\n</property>\n\n<property>\n<name>dfs.namenode.avoid.write.stale.datanode</name>\n<value>true</value>\n</property>\n\n<property>\n<name>dfs.datanode.du.reserved</name>\n<value>1073741824</value>\n</property>\n\n<property>\n<name>dfs.namenode.resource.du.reserved</name>\n<value>1073741824</value>\n</property>\n\n<!-- appendable config used to customize for override -->\n\n<property>\n<name>dfs.datanode.handler.count</name>\n<value>50</value>\n</property>\n\n<property>\n<name>dfs.client.retry.policy.enabled</name>\n<value>true</value>\n</property>\n\n<property>\n<name>dfs.client.retry.policy.spec</name>\n<value>1000,1</value>\n</property>\n\n<!--<property>\n  <name>dfs.datanode.use.datanode.hostname</name>\n  <value>true</value>\n</property>\n\n<property>\n\t<name>dfs.client.use.datanode.hostname</name>\n\t<value>true</value>\n</property>-->\n\n<!--<property>\n\t<name>dfs.namenode.datanode.registration.ip-hostnameeck</name>\n\t<value>false</value>\n</property>-->\n\n</configuration>\n"
      httpfs-log4j.properties: |
        #
        # Licensed under the Apache License, Version 2.0 (the "License");
        # you may not use this file except in compliance with the License.
        # You may obtain a copy of the License at
        #
        #    http://www.apache.org/licenses/LICENSE-2.0
        #
        # Unless required by applicable law or agreed to in writing, software
        # distributed under the License is distributed on an "AS IS" BASIS,
        # WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
        # See the License for the specific language governing permissions and
        # limitations under the License. See accompanying LICENSE file.
        #

        # If the Java System property 'httpfs.log.dir' is not defined at HttpFSServer start up time
        # Setup sets its value to '${httpfs.home}/logs'

        log4j.appender.httpfs=org.apache.log4j.DailyRollingFileAppender
        log4j.appender.httpfs.DatePattern='.'yyyy-MM-dd
        log4j.appender.httpfs.File=${httpfs.log.dir}/httpfs.log
        log4j.appender.httpfs.Append=true
        log4j.appender.httpfs.layout=org.apache.log4j.PatternLayout
        log4j.appender.httpfs.layout.ConversionPattern=%d{ISO8601} %5p %c{1} [%X{hostname}][%X{user}:%X{doAs}] %X{op} %m%n

        log4j.appender.httpfsaudit=org.apache.log4j.DailyRollingFileAppender
        log4j.appender.httpfsaudit.DatePattern='.'yyyy-MM-dd
        log4j.appender.httpfsaudit.File=${httpfs.log.dir}/httpfs-audit.log
        log4j.appender.httpfsaudit.Append=true
        log4j.appender.httpfsaudit.layout=org.apache.log4j.PatternLayout
        log4j.appender.httpfsaudit.layout.ConversionPattern=%d{ISO8601} %5p [%X{hostname}][%X{user}:%X{doAs}] %X{op} %m%n

        log4j.logger.httpfsaudit=INFO, httpfsaudit

        log4j.logger.org.apache.hadoop.fs.http.server=INFO, httpfs
        log4j.logger.org.apache.hadoop.lib=INFO, httpfs
      httpfs-signature.secret: |
        hadoop httpfs secret
      httpfs-site.xml: |
        <?xml version="1.0" encoding="UTF-8"?>
        <!--
          Licensed under the Apache License, Version 2.0 (the "License");
          you may not use this file except in compliance with the License.
          You may obtain a copy of the License at

          http://www.apache.org/licenses/LICENSE-2.0

          Unless required by applicable law or agreed to in writing, software
          distributed under the License is distributed on an "AS IS" BASIS,
          WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
          See the License for the specific language governing permissions and
          limitations under the License.
        -->
        <configuration>

        </configuration>
      kms-acls.xml: |
        <?xml version="1.0" encoding="UTF-8"?>
        <!--
          Licensed under the Apache License, Version 2.0 (the "License");
          you may not use this file except in compliance with the License.
          You may obtain a copy of the License at

          http://www.apache.org/licenses/LICENSE-2.0

          Unless required by applicable law or agreed to in writing, software
          distributed under the License is distributed on an "AS IS" BASIS,
          WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
          See the License for the specific language governing permissions and
          limitations under the License.
        -->
        <configuration>

          <!-- This file is hot-reloaded when it changes -->

          <!-- KMS ACLs -->

          <property>
            <name>hadoop.kms.acl.CREATE</name>
            <value>*</value>
            <description>
              ACL for create-key operations.
              If the user is not in the GET ACL, the key material is not returned
              as part of the response.
            </description>
          </property>

          <property>
            <name>hadoop.kms.acl.DELETE</name>
            <value>*</value>
            <description>
              ACL for delete-key operations.
            </description>
          </property>

          <property>
            <name>hadoop.kms.acl.ROLLOVER</name>
            <value>*</value>
            <description>
              ACL for rollover-key operations.
              If the user is not in the GET ACL, the key material is not returned
              as part of the response.
            </description>
          </property>

          <property>
            <name>hadoop.kms.acl.GET</name>
            <value>*</value>
            <description>
              ACL for get-key-version and get-current-key operations.
            </description>
          </property>

          <property>
            <name>hadoop.kms.acl.GET_KEYS</name>
            <value>*</value>
            <description>
              ACL for get-keys operations.
            </description>
          </property>

          <property>
            <name>hadoop.kms.acl.GET_METADATA</name>
            <value>*</value>
            <description>
              ACL for get-key-metadata and get-keys-metadata operations.
            </description>
          </property>

          <property>
            <name>hadoop.kms.acl.SET_KEY_MATERIAL</name>
            <value>*</value>
            <description>
              Complementary ACL for CREATE and ROLLOVER operations to allow the client
              to provide the key material when creating or rolling a key.
            </description>
          </property>

          <property>
            <name>hadoop.kms.acl.GENERATE_EEK</name>
            <value>*</value>
            <description>
              ACL for generateEncryptedKey CryptoExtension operations.
            </description>
          </property>

          <property>
            <name>hadoop.kms.acl.DECRYPT_EEK</name>
            <value>*</value>
            <description>
              ACL for decryptEncryptedKey CryptoExtension operations.
            </description>
          </property>

          <property>
            <name>default.key.acl.MANAGEMENT</name>
            <value>*</value>
            <description>
              default ACL for MANAGEMENT operations for all key acls that are not
              explicitly defined.
            </description>
          </property>

          <property>
            <name>default.key.acl.GENERATE_EEK</name>
            <value>*</value>
            <description>
              default ACL for GENERATE_EEK operations for all key acls that are not
              explicitly defined.
            </description>
          </property>

          <property>
            <name>default.key.acl.DECRYPT_EEK</name>
            <value>*</value>
            <description>
              default ACL for DECRYPT_EEK operations for all key acls that are not
              explicitly defined.
            </description>
          </property>

          <property>
            <name>default.key.acl.READ</name>
            <value>*</value>
            <description>
              default ACL for READ operations for all key acls that are not
              explicitly defined.
            </description>
          </property>


        </configuration>
      kms-log4j.properties: |-
        #
        # Licensed under the Apache License, Version 2.0 (the "License");
        # you may not use this file except in compliance with the License.
        # You may obtain a copy of the License at
        #
        #    http://www.apache.org/licenses/LICENSE-2.0
        #
        # Unless required by applicable law or agreed to in writing, software
        # distributed under the License is distributed on an "AS IS" BASIS,
        # WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
        # See the License for the specific language governing permissions and
        # limitations under the License. See accompanying LICENSE file.
        #

        # If the Java System property 'kms.log.dir' is not defined at KMS start up time
        # Setup sets its value to '${kms.home}/logs'

        log4j.appender.kms=org.apache.log4j.DailyRollingFileAppender
        log4j.appender.kms.DatePattern='.'yyyy-MM-dd
        log4j.appender.kms.File=${kms.log.dir}/kms.log
        log4j.appender.kms.Append=true
        log4j.appender.kms.layout=org.apache.log4j.PatternLayout
        log4j.appender.kms.layout.ConversionPattern=%d{ISO8601} %-5p %c{1} - %m%n

        log4j.appender.kms-audit=org.apache.log4j.DailyRollingFileAppender
        log4j.appender.kms-audit.DatePattern='.'yyyy-MM-dd
        log4j.appender.kms-audit.File=${kms.log.dir}/kms-audit.log
        log4j.appender.kms-audit.Append=true
        log4j.appender.kms-audit.layout=org.apache.log4j.PatternLayout
        log4j.appender.kms-audit.layout.ConversionPattern=%d{ISO8601} %m%n

        log4j.logger.kms-audit=INFO, kms-audit
        log4j.additivity.kms-audit=false

        log4j.rootLogger=ALL, kms
        log4j.logger.org.apache.hadoop.conf=ERROR
        log4j.logger.org.apache.hadoop=INFO
        log4j.logger.com.sun.jersey.server.wadl.generators.WadlGeneratorJAXBGrammarGenerator=OFF
      kms-site.xml: |
        <?xml version="1.0" encoding="UTF-8"?>
        <!--
          Licensed under the Apache License, Version 2.0 (the "License");
          you may not use this file except in compliance with the License.
          You may obtain a copy of the License at

          http://www.apache.org/licenses/LICENSE-2.0

          Unless required by applicable law or agreed to in writing, software
          distributed under the License is distributed on an "AS IS" BASIS,
          WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
          See the License for the specific language governing permissions and
          limitations under the License.
        -->
        <configuration>

          <!-- KMS Backend KeyProvider -->

          <property>
            <name>hadoop.kms.key.provider.uri</name>
            <value>jceks://file@/${user.home}/kms.keystore</value>
            <description>
              URI of the backing KeyProvider for the KMS.
            </description>
          </property>

          <property>
            <name>hadoop.security.keystore.JavaKeyStoreProvider.password</name>
            <value>none</value>
            <description>
              If using the JavaKeyStoreProvider, the password for the keystore file.
            </description>
          </property>

          <!-- KMS Cache -->

          <property>
            <name>hadoop.kms.cache.enable</name>
            <value>true</value>
            <description>
              Whether the KMS will act as a cache for the backing KeyProvider.
              When the cache is enabled, operations like getKeyVersion, getMetadata,
              and getCurrentKey will sometimes return cached data without consulting
              the backing KeyProvider. Cached values are flushed when keys are deleted
              or modified.
            </description>
          </property>

          <property>
            <name>hadoop.kms.cache.timeout.ms</name>
            <value>600000</value>
            <description>
              Expiry time for the KMS key version and key metadata cache, in
              milliseconds. This affects getKeyVersion and getMetadata.
            </description>
          </property>

          <property>
            <name>hadoop.kms.current.key.cache.timeout.ms</name>
            <value>30000</value>
            <description>
              Expiry time for the KMS current key cache, in milliseconds. This
              affects getCurrentKey operations.
            </description>
          </property>

          <!-- KMS Audit -->

          <property>
            <name>hadoop.kms.audit.aggregation.window.ms</name>
            <value>10000</value>
            <description>
              Duplicate audit log events within the aggregation window (specified in
              ms) are quashed to reduce log traffic. A single message for aggregated
              events is printed at the end of the window, along with a count of the
              number of aggregated events.
            </description>
          </property>

          <!-- KMS Security -->

          <property>
            <name>hadoop.kms.authentication.type</name>
            <value>simple</value>
            <description>
              Authentication type for the KMS. Can be either &quot;simple&quot;
              or &quot;kerberos&quot;.
            </description>
          </property>

          <property>
            <name>hadoop.kms.authentication.kerberos.keytab</name>
            <value>${user.home}/kms.keytab</value>
            <description>
              Path to the keytab with credentials for the configured Kerberos principal.
            </description>
          </property>

          <property>
            <name>hadoop.kms.authentication.kerberos.principal</name>
            <value>HTTP/localhost</value>
            <description>
              The Kerberos principal to use for the HTTP endpoint.
              The principal must start with 'HTTP/' as per the Kerberos HTTP SPNEGO specification.
            </description>
          </property>

          <property>
            <name>hadoop.kms.authentication.kerberos.name.rules</name>
            <value>DEFAULT</value>
            <description>
              Rules used to resolve Kerberos principal names.
            </description>
          </property>

          <!-- Authentication cookie signature source -->

          <property>
            <name>hadoop.kms.authentication.signer.secret.provider</name>
            <value>random</value>
            <description>
              Indicates how the secret to sign the authentication cookies will be
              stored. Options are 'random' (default), 'string' and 'zookeeper'.
              If using a setup with multiple KMS instances, 'zookeeper' should be used.
            </description>
          </property>

          <!-- Configuration for 'zookeeper' authentication cookie signature source -->

          <property>
            <name>hadoop.kms.authentication.signer.secret.provider.zookeeper.path</name>
            <value>/hadoop-kms/hadoop-auth-signature-secret</value>
            <description>
              The Zookeeper ZNode path where the KMS instances will store and retrieve
              the secret from.
            </description>
          </property>

          <property>
            <name>hadoop.kms.authentication.signer.secret.provider.zookeeper.connection.string</name>
            <value>#HOSTNAME#:#PORT#,...</value>
            <description>
              The Zookeeper connection string, a list of hostnames and port comma
              separated.
            </description>
          </property>

          <property>
            <name>hadoop.kms.authentication.signer.secret.provider.zookeeper.auth.type</name>
            <value>kerberos</value>
            <description>
              The Zookeeper authentication type, 'none' or 'sasl' (Kerberos).
            </description>
          </property>

          <property>
            <name>hadoop.kms.authentication.signer.secret.provider.zookeeper.kerberos.keytab</name>
            <value>/etc/hadoop/conf/kms.keytab</value>
            <description>
              The absolute path for the Kerberos keytab with the credentials to
              connect to Zookeeper.
            </description>
          </property>

          <property>
            <name>hadoop.kms.authentication.signer.secret.provider.zookeeper.kerberos.principal</name>
            <value>kms/#HOSTNAME#</value>
            <description>
              The Kerberos service principal used to connect to Zookeeper.
            </description>
          </property>

        </configuration>
      log4j.properties: "# Licensed to the Apache Software Foundation (ASF) under one\n# or more contributor license agreements.  See the NOTICE file\n# distributed with this work for additional information\n# regarding copyright ownership.  The ASF licenses this file\n# to you under the Apache License, Version 2.0 (the\n# \"License\"); you may not use this file except in compliance\n# with the License.  You may obtain a copy of the License at\n#\n#     http://www.apache.org/licenses/LICENSE-2.0\n#\n# Unless required by applicable law or agreed to in writing, software\n# distributed under the License is distributed on an \"AS IS\" BASIS,\n# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\n# See the License for the specific language governing permissions and\n# limitations under the License.\n\n# Define some default values that can be overridden by system properties\nhadoop.root.logger=INFO,console\nhadoop.log.dir=.\nhadoop.log.file=hadoop.log\n\n# Define the root logger to the system property \"hadoop.root.logger\".\nlog4j.rootLogger=${hadoop.root.logger}, EventCounter\n\n# Logging Threshold\nlog4j.threshold=ALL\n\n# Null Appender\nlog4j.appender.NullAppender=org.apache.log4j.varia.NullAppender\n\n#\n# Rolling File Appender - cap space usage at 5gb.\n#\nhadoop.log.maxfilesize=256MB\nhadoop.log.maxbackupindex=2\nlog4j.appender.RFA=org.apache.log4j.RollingFileAppender\nlog4j.appender.RFA.File=${hadoop.log.dir}/${hadoop.log.file}\n\nlog4j.appender.RFA.MaxFileSize=${hadoop.log.maxfilesize}\nlog4j.appender.RFA.MaxBackupIndex=${hadoop.log.maxbackupindex}\n\nlog4j.appender.RFA.layout=org.apache.log4j.PatternLayout\n\n# Pattern format: Date LogLevel LoggerName LogMessage\nlog4j.appender.RFA.layout.ConversionPattern=%d{ISO8601} %p %c: %m%n\n# Debugging Pattern format\n#log4j.appender.RFA.layout.ConversionPattern=%d{ISO8601} %-5p %c{2} (%F:%M(%L)) - %m%n\n\n\n#\n# Daily Rolling File Appender\n#\n\nlog4j.appender.DRFA=org.apache.log4j.DailyRollingFileAppender\nlog4j.appender.DRFA.File=${hadoop.log.dir}/${hadoop.log.file}\n\n# Rollover at midnight\nlog4j.appender.DRFA.DatePattern=.yyyy-MM-dd\n\nlog4j.appender.DRFA.layout=org.apache.log4j.PatternLayout\n\n# Pattern format: Date LogLevel LoggerName LogMessage\nlog4j.appender.DRFA.layout.ConversionPattern=%d{ISO8601} %p %c: %m%n\n# Debugging Pattern format\n#log4j.appender.DRFA.layout.ConversionPattern=%d{ISO8601} %-5p %c{2} (%F:%M(%L)) - %m%n\n\n\n#\n# console\n# Add \"console\" to rootlogger above if you want to use this \n#\n\nlog4j.appender.console=org.apache.log4j.ConsoleAppender\nlog4j.appender.console.target=System.err\nlog4j.appender.console.layout=org.apache.log4j.PatternLayout\nlog4j.appender.console.layout.ConversionPattern=%d{yy/MM/dd HH:mm:ss} %p %c{2}: %m%n\n\n#\n# TaskLog Appender\n#\n\n#Default values\nhadoop.tasklog.taskid=null\nhadoop.tasklog.iscleanup=false\nhadoop.tasklog.noKeepSplits=4\nhadoop.tasklog.totalLogFileSize=100\nhadoop.tasklog.purgeLogSplits=true\nhadoop.tasklog.logsRetainHours=12\n\nlog4j.appender.TLA=org.apache.hadoop.mapred.TaskLogAppender\nlog4j.appender.TLA.taskId=${hadoop.tasklog.taskid}\nlog4j.appender.TLA.isCleanup=${hadoop.tasklog.iscleanup}\nlog4j.appender.TLA.totalLogFileSize=${hadoop.tasklog.totalLogFileSize}\n\nlog4j.appender.TLA.layout=org.apache.log4j.PatternLayout\nlog4j.appender.TLA.layout.ConversionPattern=%d{ISO8601} %p %c: %m%n\n\n#\n# HDFS block state change log from block manager\n#\n# Uncomment the following to suppress normal block state change\n# messages from BlockManager in NameNode.\n#log4j.logger.BlockStateChange=WARN\n\n#\n#Security appender\n#\nhadoop.security.logger=INFO,NullAppender\nhadoop.security.log.maxfilesize=256MB\nhadoop.security.log.maxbackupindex=2\nlog4j.category.SecurityLogger=${hadoop.security.logger}\nhadoop.security.log.file=SecurityAuth-${user.name}.audit\nlog4j.appender.RFAS=org.apache.log4j.RollingFileAppender \nlog4j.appender.RFAS.File=${hadoop.log.dir}/${hadoop.security.log.file}\nlog4j.appender.RFAS.layout=org.apache.log4j.PatternLayout\nlog4j.appender.RFAS.layout.ConversionPattern=%d{ISO8601} %p %c: %m%n\nlog4j.appender.RFAS.MaxFileSize=${hadoop.security.log.maxfilesize}\nlog4j.appender.RFAS.MaxBackupIndex=${hadoop.security.log.maxbackupindex}\n\n#\n# Daily Rolling Security appender\n#\nlog4j.appender.DRFAS=org.apache.log4j.DailyRollingFileAppender \nlog4j.appender.DRFAS.File=${hadoop.log.dir}/${hadoop.security.log.file}\nlog4j.appender.DRFAS.layout=org.apache.log4j.PatternLayout\nlog4j.appender.DRFAS.layout.ConversionPattern=%d{ISO8601} %p %c: %m%n\nlog4j.appender.DRFAS.DatePattern=.yyyy-MM-dd\n\n#\n# hadoop configuration logging\n#\n\n# Uncomment the following line to turn off configuration deprecation warnings.\n# log4j.logger.org.apache.hadoop.conf.Configuration.deprecation=WARN\n\n#\n# hdfs audit logging\n#\nhdfs.audit.logger=INFO,NullAppender\nhdfs.audit.log.maxfilesize=256MB\nhdfs.audit.log.maxbackupindex=2\nlog4j.logger.org.apache.hadoop.hdfs.server.namenode.FSNamesystem.audit=${hdfs.audit.logger}\nlog4j.additivity.org.apache.hadoop.hdfs.server.namenode.FSNamesystem.audit=false\nlog4j.appender.RFAAUDIT=org.apache.log4j.RollingFileAppender\nlog4j.appender.RFAAUDIT.File=${hadoop.log.dir}/hdfs-audit.log\nlog4j.appender.RFAAUDIT.layout=org.apache.log4j.PatternLayout\nlog4j.appender.RFAAUDIT.layout.ConversionPattern=%d{ISO8601} %p %c{2}: %m%n\nlog4j.appender.RFAAUDIT.MaxFileSize=${hdfs.audit.log.maxfilesize}\nlog4j.appender.RFAAUDIT.MaxBackupIndex=${hdfs.audit.log.maxbackupindex}\n\n#\n# mapred audit logging\n#\nmapred.audit.logger=INFO,NullAppender\nmapred.audit.log.maxfilesize=256MB\nmapred.audit.log.maxbackupindex=2\nlog4j.logger.org.apache.hadoop.mapred.AuditLogger=${mapred.audit.logger}\nlog4j.additivity.org.apache.hadoop.mapred.AuditLogger=false\nlog4j.appender.MRAUDIT=org.apache.log4j.RollingFileAppender\nlog4j.appender.MRAUDIT.File=${hadoop.log.dir}/mapred-audit.log\nlog4j.appender.MRAUDIT.layout=org.apache.log4j.PatternLayout\nlog4j.appender.MRAUDIT.layout.ConversionPattern=%d{ISO8601} %p %c{2}: %m%n\nlog4j.appender.MRAUDIT.MaxFileSize=${mapred.audit.log.maxfilesize}\nlog4j.appender.MRAUDIT.MaxBackupIndex=${mapred.audit.log.maxbackupindex}\n\n# Custom Logging levels\n\n#log4j.logger.org.apache.hadoop.mapred.JobTracker=DEBUG\n#log4j.logger.org.apache.hadoop.mapred.TaskTracker=DEBUG\n#log4j.logger.org.apache.hadoop.hdfs.server.namenode.FSNamesystem.audit=DEBUG\n\n# Jets3t library\nlog4j.logger.org.jets3t.service.impl.rest.httpclient.RestS3Service=ERROR\n\n# AWS SDK & S3A FileSystem\nlog4j.logger.com.amazonaws=ERROR\nlog4j.logger.com.amazonaws.http.AmazonHttpClient=ERROR\nlog4j.logger.org.apache.hadoop.fs.s3a.S3AFileSystem=WARN\n\n#\n# Event Counter Appender\n# Sends counts of logging messages at different severity levels to Hadoop Metrics.\n#\nlog4j.appender.EventCounter=org.apache.hadoop.log.metrics.EventCounter\n\n#\n# Job Summary Appender \n#\n# Use following logger to send summary to separate file defined by \n# hadoop.mapreduce.jobsummary.log.file :\n# hadoop.mapreduce.jobsummary.logger=INFO,JSA\n# \nhadoop.mapreduce.jobsummary.logger=${hadoop.root.logger}\nhadoop.mapreduce.jobsummary.log.file=hadoop-mapreduce.jobsummary.log\nhadoop.mapreduce.jobsummary.log.maxfilesize=256MB\nhadoop.mapreduce.jobsummary.log.maxbackupindex=2\nlog4j.appender.JSA=org.apache.log4j.RollingFileAppender\nlog4j.appender.JSA.File=${hadoop.log.dir}/${hadoop.mapreduce.jobsummary.log.file}\nlog4j.appender.JSA.MaxFileSize=${hadoop.mapreduce.jobsummary.log.maxfilesize}\nlog4j.appender.JSA.MaxBackupIndex=${hadoop.mapreduce.jobsummary.log.maxbackupindex}\nlog4j.appender.JSA.layout=org.apache.log4j.PatternLayout\nlog4j.appender.JSA.layout.ConversionPattern=%d{yy/MM/dd HH:mm:ss} %p %c{2}: %m%n\nlog4j.logger.org.apache.hadoop.mapred.JobInProgress$JobSummary=${hadoop.mapreduce.jobsummary.logger}\nlog4j.additivity.org.apache.hadoop.mapred.JobInProgress$JobSummary=false\n\n#\n# Yarn ResourceManager Application Summary Log \n#\n# Set the ResourceManager summary log filename\nyarn.server.resourcemanager.appsummary.log.file=rm-appsummary.log\n# Set the ResourceManager summary log level and appender\nyarn.server.resourcemanager.appsummary.logger=${hadoop.root.logger}\n#yarn.server.resourcemanager.appsummary.logger=INFO,RMSUMMARY\n\n# To enable AppSummaryLogging for the RM, \n# set yarn.server.resourcemanager.appsummary.logger to \n# <LEVEL>,RMSUMMARY in hadoop-env.sh\n\n# Appender for ResourceManager Application Summary Log\n# Requires the following properties to be set\n#    - hadoop.log.dir (Hadoop Log directory)\n#    - yarn.server.resourcemanager.appsummary.log.file (resource manager app summary log filename)\n#    - yarn.server.resourcemanager.appsummary.logger (resource manager app summary log level and appender)\n\nlog4j.logger.org.apache.hadoop.yarn.server.resourcemanager.RMAppManager$ApplicationSummary=${yarn.server.resourcemanager.appsummary.logger}\nlog4j.additivity.org.apache.hadoop.yarn.server.resourcemanager.RMAppManager$ApplicationSummary=false\nlog4j.appender.RMSUMMARY=org.apache.log4j.RollingFileAppender\nlog4j.appender.RMSUMMARY.File=${hadoop.log.dir}/${yarn.server.resourcemanager.appsummary.log.file}\nlog4j.appender.RMSUMMARY.MaxFileSize=256MB\nlog4j.appender.RMSUMMARY.MaxBackupIndex=2\nlog4j.appender.RMSUMMARY.layout=org.apache.log4j.PatternLayout\nlog4j.appender.RMSUMMARY.layout.ConversionPattern=%d{ISO8601} %p %c{2}: %m%n\n\n# HS audit log configs\n#mapreduce.hs.audit.logger=INFO,HSAUDIT\n#log4j.logger.org.apache.hadoop.mapreduce.v2.hs.HSAuditLogger=${mapreduce.hs.audit.logger}\n#log4j.additivity.org.apache.hadoop.mapreduce.v2.hs.HSAuditLogger=false\n#log4j.appender.HSAUDIT=org.apache.log4j.DailyRollingFileAppender\n#log4j.appender.HSAUDIT.File=${hadoop.log.dir}/hs-audit.log\n#log4j.appender.HSAUDIT.layout=org.apache.log4j.PatternLayout\n#log4j.appender.HSAUDIT.layout.ConversionPattern=%d{ISO8601} %p %c{2}: %m%n\n#log4j.appender.HSAUDIT.DatePattern=.yyyy-MM-dd\n\n# Http Server Request Logs\n#log4j.logger.http.requests.namenode=INFO,namenoderequestlog\n#log4j.appender.namenoderequestlog=org.apache.hadoop.http.HttpRequestLogAppender\n#log4j.appender.namenoderequestlog.Filename=${hadoop.log.dir}/jetty-namenode-yyyy_mm_dd.log\n#log4j.appender.namenoderequestlog.RetainDays=3\n\n#log4j.logger.http.requests.datanode=INFO,datanoderequestlog\n#log4j.appender.datanoderequestlog=org.apache.hadoop.http.HttpRequestLogAppender\n#log4j.appender.datanoderequestlog.Filename=${hadoop.log.dir}/jetty-datanode-yyyy_mm_dd.log\n#log4j.appender.datanoderequestlog.RetainDays=3\n\n#log4j.logger.http.requests.resourcemanager=INFO,resourcemanagerrequestlog\n#log4j.appender.resourcemanagerrequestlog=org.apache.hadoop.http.HttpRequestLogAppender\n#log4j.appender.resourcemanagerrequestlog.Filename=${hadoop.log.dir}/jetty-resourcemanager-yyyy_mm_dd.log\n#log4j.appender.resourcemanagerrequestlog.RetainDays=3\n\n#log4j.logger.http.requests.jobhistory=INFO,jobhistoryrequestlog\n#log4j.appender.jobhistoryrequestlog=org.apache.hadoop.http.HttpRequestLogAppender\n#log4j.appender.jobhistoryrequestlog.Filename=${hadoop.log.dir}/jetty-jobhistory-yyyy_mm_dd.log\n#log4j.appender.jobhistoryrequestlog.RetainDays=3\n\n#log4j.logger.http.requests.nodemanager=INFO,nodemanagerrequestlog\n#log4j.appender.nodemanagerrequestlog=org.apache.hadoop.http.HttpRequestLogAppender\n#log4j.appender.nodemanagerrequestlog.Filename=${hadoop.log.dir}/jetty-nodemanager-yyyy_mm_dd.log\n#log4j.appender.nodemanagerrequestlog.RetainDays=3\n"
  tenantNamespaces: [hbase-tenant-ns]
  deployments:
    zookeeper:
      name: hbase-cluster-zk
      size: 3
      isPodServiceRequired: true
      shareProcessNamespace: false
      terminateGracePeriod: 120
      volumeClaims:
      - name: data
        storageSize: 2Gi
        storageClassName: standard
      volumes:
      - name: nodeinfo
        volumeSource: HostPath
        path: /etc/nodeinfo
      initContainers:
      - name: init-dnslookup
        isBootstrap: false
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m

          i=0
          while true; do
            echo "$i iteration"
            dig +short $(hostname -f) | grep -v -e '^$'
            if [ $? == 0 ]; then
              sleep 30 # 30 seconds default dns caching
              echo "Breaking..."
              break
            fi
            i=$((i + 1))
            sleep 1
          done
        cpuLimit: "0.2"
        memoryLimit: "128Mi"
        cpuRequest: "0.2"
        memoryRequest: "128Mi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
      containers:
      - name: zookeeper
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m -x

          export HBASE_LOG_DIR=$0
          export HBASE_CONF_DIR=$1
          export HBASE_HOME=$2
          export USER=$(whoami)

          mkdir -p $HBASE_LOG_DIR
          ln -sf /dev/stdout $HBASE_LOG_DIR/hbase-$USER-zookeeper-$(hostname).log
          ln -sf /dev/stdout $HBASE_LOG_DIR/hbase-$USER-zookeeper-$(hostname).out

          function shutdown() {
            echo "Stopping Zookeeper"
            $HBASE_HOME/bin/hbase-daemon.sh stop zookeeper
          }

          trap shutdown SIGTERM
          exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start zookeeper &
          wait
        args:
        - /var/log/hbase
        - /etc/hbase
        - /opt/hbase
        ports:
        - port: 2181
          name: zookeeper-0
        - port: 2888
          name: zookeeper-1
        - port: 3888
          name: zookeeper-2
        startupProbe:
          initialDelay: 30
          timeout: 60
          failureThreshold: 10
          command:
          - /bin/bash
          - -c
          - |
            #! /bin/bash
            set -m

            export HBASE_LOG_DIR=$0
            export HBASE_CONF_DIR=$1
            export HBASE_HOME=$2

            #TODO: Find better alternative
            IFS=',' read -ra ZKs <<< $($HBASE_HOME/bin/hbase zkcli quit 2> /dev/null | grep "Connecting to" | sed 's/Connecting to //')
            visited=""
            quorum=""
            myhost="localhost 2181"
            for zk in "${ZKs[@]}"; do
              if [[ $(echo $zk | grep $(hostname -f) | wc -l) == 1 ]]; then
                myhost=$(echo $zk | sed 's/:/ /')
              fi

              if [[ $(echo "stat" | nc $(echo $zk | sed 's/:/ /') | grep "Mode: " | wc -l) == 1 ]]; then
                quorum="present"
              fi
              visited="true"
            done

            if [[ -n $visited && -z $quorum ]]; then
              echo "Quorum is absent, disabling startup checks..."
              sleep 5
              exit 0
            fi

            if [[ $(echo "stat" | nc $myhost | grep "Mode: " | wc -l) == 1 ]]; then
              exit 0
            else
              echo "zookeeper is not able to connect to quorum"
              exit 1
            fi
          - /var/log/hbase
          - /etc/hbase
          - /opt/hbase
        livenessProbe:
          tcpPort: 2181
          initialDelay: 20
        readinessProbe:
          tcpPort: 2181
          initialDelay: 20
        cpuLimit: "0.5"
        memoryLimit: "2Gi"
        cpuRequest: "0.5"
        memoryRequest: "2Gi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
          addSysPtrace: false
        volumeMounts:
        - name: data
          mountPath: /grid/1
          readOnly: false
    journalnode:
      name: hbase-cluster-jn
      size: 3
      isPodServiceRequired: true
      shareProcessNamespace: false
      terminateGracePeriod: 120
      volumeClaims:
      - name: data
        storageSize: 2Gi
        storageClassName: standard
      volumes:
      - name: nodeinfo
        volumeSource: HostPath
        path: /etc/nodeinfo
      initContainers:
      - name: init-dnslookup
        isBootstrap: false
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m

          i=0
          while true; do
            echo "$i iteration"
            dig +short $(hostname -f) | grep -v -e '^$'
            if [ $? == 0 ]; then
              sleep 30 # 30 seconds default dns caching
              echo "Breaking..."
              break
            fi
            i=$((i + 1))
            sleep 1
          done
        cpuLimit: "0.2"
        memoryLimit: "128Mi"
        cpuRequest: "0.2"
        memoryRequest: "128Mi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
      containers:
      - name: journalnode
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m

          export HADOOP_LOG_DIR=$0
          export HADOOP_CONF_DIR=$1
          export HADOOP_HOME=$2

          function shutdown() {
            echo "Stopping Journalnode"
            $HADOOP_HOME/bin/hdfs --daemon stop journalnode
          }

          trap shutdown SIGTERM
          exec $HADOOP_HOME/bin/hdfs journalnode start &
          wait
        args:
        - /var/log/hadoop
        - /etc/hadoop
        - /opt/hadoop
        ports:
        - port: 8485
          name: journalnode-0
        - port: 8480
          name: journalnode-1
        livenessProbe:
          tcpPort: 8485
          initialDelay: 40
        readinessProbe:
          tcpPort: 8485
          initialDelay: 40
        cpuLimit: "0.5"
        memoryLimit: "1Gi"
        cpuRequest: "0.5"
        memoryRequest: "1Gi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
          addSysPtrace: false
        volumeMounts:
        - name: data
          mountPath: /grid/1
          readOnly: false
    hmaster:
      name: hbase-cluster-hmaster
      size: 2
      isPodServiceRequired: false
      shareProcessNamespace: false
      terminateGracePeriod: 120
      volumes:
      - name: data
        volumeSource: EmptyDir
      initContainers:
      - name: init-dnslookup
        isBootstrap: false
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m

          i=0
          while true; do
            echo "$i iteration"
            dig +short $(hostname -f) | grep -v -e '^$'
            if [ $? == 0 ]; then
              sleep 30 # 30 seconds default dns caching
              echo "Breaking..."
              break
            fi
            i=$((i + 1))
            sleep 1
          done
        cpuLimit: "0.2"
        memoryLimit: "128Mi"
        cpuRequest: "0.2"
        memoryRequest: "128Mi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
      sidecarContainers:
      - name: rackutils
        image: hbase-rack-utils:1.0.1
        command: [./entrypoint]
        args: [com.flipkart.hbase.HbaseRackUtils /etc/hbase /hbase-operator /opt/share/rack_topology.data]
        cpuLimit: "0.2"
        memoryLimit: "256Mi"
        cpuRequest: "0.2"
        memoryRequest: "256Mi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
        volumeMounts:
        - name: data
          mountPath: /opt/share
          readOnly: false
      containers:
      - name: hmaster
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m
          export HBASE_LOG_DIR=$0
          export HBASE_CONF_DIR=$1
          export HBASE_HOME=$2
          export USER=$(whoami)

          mkdir -p $HBASE_LOG_DIR
          ln -sf /dev/stdout $HBASE_LOG_DIR/hbase-$USER-master-$(hostname).out
          ln -sf /dev/stdout $HBASE_LOG_DIR/hbase-$USER-master-$(hostname).log

          function shutdown() {
            echo "Stopping Hmaster"
            $HBASE_HOME/bin/hbase-daemon.sh stop master
          }

          trap shutdown SIGTERM
          exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start master &
          wait
        args:
        - /var/log/hbase
        - /etc/hbase
        - /opt/hbase
        ports:
        - port: 16000
          name: hmaster-0
        - port: 16010
          name: hmaster-1
        livenessProbe:
          tcpPort: 16000
          initialDelay: 10
        readinessProbe:
          tcpPort: 16000
          initialDelay: 10
        cpuLimit: "0.3"
        memoryLimit: "3Gi"
        cpuRequest: "0.3"
        memoryRequest: "3Gi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
          addSysPtrace: false
        volumeMounts:
        - name: data
          mountPath: /opt/share
          readOnly: false
    datanode:
      name: hbase-cluster-dn
      size: 3
      isPodServiceRequired: false
      shareProcessNamespace: true
      terminateGracePeriod: 120
      volumeClaims:
      - name: data
        storageSize: 10Gi
        storageClassName: standard
      volumes:
      - name: lifecycle
        volumeSource: EmptyDir
      - name: nodeinfo
        volumeSource: HostPath
        path: /etc/nodeinfo
      initContainers:
      - name: init-dnslookup
        isBootstrap: false
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m

          i=0
          while true; do
            echo "$i iteration"
            dig +short $(hostname -f) | grep -v -e '^$'
            if [ $? == 0 ]; then
              sleep 30 # 30 seconds default dns caching
              echo "Breaking..."
              break
            fi
            i=$((i + 1))
            sleep 1
          done
        cpuLimit: "0.2"
        memoryLimit: "128Mi"
        cpuRequest: "0.2"
        memoryRequest: "128Mi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
      - name: init-faultdomain
        isBootstrap: false
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m -x

          export HBASE_LOG_DIR=/var/log/hbase
          export HBASE_CONF_DIR=/etc/hbase
          export HBASE_HOME=/opt/hbase

          # Make it optional
          FAULT_DOMAIN_COMMAND="cat /etc/nodeinfo | grep 'smd' | sed 's/smd=//' | sed 's/\"//g'"
          HOSTNAME=$(hostname -f)

          echo "Running command to get fault domain: $FAULT_DOMAIN_COMMAND"
          SMD=$(eval $FAULT_DOMAIN_COMMAND)
          echo "SMD value: $SMD"

          if [[ -n "$FAULT_DOMAIN_COMMAND" ]]; then
            echo "create /hbase-operator $SMD" | $HBASE_HOME/bin/hbase zkcli 2> /dev/null || true
            echo "create /hbase-operator/$HOSTNAME $SMD" | $HBASE_HOME/bin/hbase zkcli 2> /dev/null
            echo ""
            echo "Completed"
          fi
        cpuLimit: "0.1"
        memoryLimit: "386Mi"
        cpuRequest: "0.1"
        memoryRequest: "386Mi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
        volumeMounts:
        - name: nodeinfo
          mountPath: /etc/nodeinfo
          readOnly: true
      - name: init-refreshnn
        isBootstrap: false
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -x -m

          export HADOOP_LOG_DIR=/var/log/hadoop
          export HADOOP_CONF_DIR=/etc/hadoop
          export HADOOP_HOME=/opt/hadoop

          $HADOOP_HOME/bin/hdfs dfsadmin -refreshNodes

        cpuLimit: "0.2"
        memoryLimit: "256Mi"
        cpuRequest: "0.2"
        memoryRequest: "256Mi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
      containers:
      - name: datanode
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -x -m

          export HADOOP_LOG_DIR=$0
          export HADOOP_CONF_DIR=$1
          export HADOOP_HOME=$2

          function shutdown() {
            while true; do
              #TODO: Kill it beyond certain wait time
              if [[ -f "/lifecycle/rs-terminated" ]]; then
                echo "Stopping datanode"
                sleep 3
                $HADOOP_HOME/bin/hdfs --daemon stop datanode
                break
              fi
              echo "Waiting for regionserver to die"
              sleep 2
            done
          }

          trap shutdown SIGTERM
          exec $HADOOP_HOME/bin/hdfs datanode &
          PID=$!

          #TODO: Correct way to identify if process is up
          touch /lifecycle/dn-started

          wait $PID
        args:
        - /var/log/hadoop
        - /etc/hadoop
        - /opt/hadoop
        ports:
        - port: 9866
          name: datanode-0
        startupProbe:
          initialDelay: 30
          timeout: 60
          failureThreshold: 10
          command:
          - /bin/bash
          - -c
          - |
            #! /bin/bash
            set -m

            export HADOOP_LOG_DIR=$0
            export HADOOP_CONF_DIR=$1
            export HADOOP_HOME=$2

            while :
            do
              if [[ $($HADOOP_HOME/bin/hdfs dfsadmin -report -live | grep "$(hostname -f)" | wc -l) == 2 ]]; then
                echo "datanode is listed as live under namenode. Exiting..."
                exit 0
              else
                echo "datanode is still not listed as live under namenode"
                exit 1
              fi
            done
            exit 1
          - /var/log/hadoop
          - /etc/hadoop
          - /opt/hadoop
        livenessProbe:
          tcpPort: 9866
          initialDelay: 60
        readinessProbe:
          tcpPort: 9866
          initialDelay: 60
        cpuLimit: "0.5"
        memoryLimit: "3Gi"
        cpuRequest: "0.5"
        memoryRequest: "3Gi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
          addSysPtrace: true
        volumeMounts:
        - name: data
          mountPath: /grid/1
          readOnly: false
        - name: lifecycle
          mountPath: /lifecycle
          readOnly: false
      - name: regionserver
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m
          export HBASE_LOG_DIR=$0
          export HBASE_CONF_DIR=$1
          export HBASE_HOME=$2
          export USER=$(whoami)

          FAULT_DOMAIN_COMMAND=$3

          mkdir -p $HBASE_LOG_DIR
          #TODO: logfile names
          ln -sf /dev/stdout $HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).out
          ln -sf /dev/stdout $HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).log

          function shutdown() {
            echo "Stopping Regionserver"
            host=`hostname -f`
            #TODO: Needs to be addressed
            $HBASE_HOME/bin/hbase org.apache.hadoop.hbase.util.RSGroupAwareRegionMover -m 6 -r $host -o unload
            touch /lifecycle/rs-terminated
            $HBASE_HOME/bin/hbase-daemon.sh stop regionserver
          }

          while true; do
            if [[ -f "/lifecycle/dn-started" ]]; then
              echo "Starting rs"
              sleep 5
              break
            fi
            echo "Waiting for datanode to start"
            sleep 2
          done

          trap shutdown SIGTERM
          exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start regionserver &
          wait
        args:
        - /var/log/hbase
        - /etc/hbase
        - /opt/hbase
        ports:
        - port: 16020
          name: regionserver-0
        - port: 16030
          name: regionserver-1
        livenessProbe:
          tcpPort: 16020
          initialDelay: 60
        readinessProbe:
          tcpPort: 16020
          initialDelay: 60
        cpuLimit: "0.5"
        memoryLimit: "5Gi"
        cpuRequest: "0.5"
        memoryRequest: "5Gi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
          addSysPtrace: true
        volumeMounts:
        - name: lifecycle
          mountPath: /lifecycle
          readOnly: false
        - name: nodeinfo
          mountPath: /etc/nodeinfo
          readOnly: true
    namenode:
      name: hbase-cluster-nn
      size: 2
      isPodServiceRequired: true
      shareProcessNamespace: false
      terminateGracePeriod: 120
      volumeClaims:
      - name: data
        storageSize: 4Gi
        storageClassName: standard
      volumes:
      - name: lifecycle
        volumeSource: EmptyDir
      - name: nodeinfo
        volumeSource: HostPath
        path: /etc/nodeinfo
      initContainers:
      - name: init-dnslookup
        isBootstrap: false
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m

          i=0
          while true; do
            echo "$i iteration"
            dig +short $(hostname -f) | grep -v -e '^$'
            if [ $? == 0 ]; then
              sleep 30 # 30 seconds default dns caching
              echo "Breaking..."
              break
            fi
            i=$((i + 1))
            sleep 1
          done
        cpuLimit: "0.2"
        memoryLimit: "128Mi"
        cpuRequest: "0.2"
        memoryRequest: "128Mi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
      - name: init-namenode
        isBootstrap: true
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m -x

          export HADOOP_LOG_DIR=$0
          export HADOOP_CONF_DIR=$1
          export HADOOP_HOME=$2

          echo "N" | $HADOOP_HOME/bin/hdfs namenode -format $($HADOOP_HOME/bin/hdfs getconf -confKey dfs.nameservices) || true
        args:
        - /var/log/hadoop
        - /etc/hadoop
        - /opt/hadoop
        cpuLimit: "0.5"
        memoryLimit: "3Gi"
        cpuRequest: "0.5"
        memoryRequest: "3Gi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
        volumeMounts:
        - name: data
          mountPath: /grid/1
          readOnly: false
      - name: init-zkfc
        isBootstrap: true
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m

          export HADOOP_LOG_DIR=$0
          export HADOOP_CONF_DIR=$1
          export HADOOP_HOME=$2

          echo "N" | $HADOOP_HOME/bin/hdfs zkfc -formatZK || true
        args:
        - /var/log/hadoop
        - /etc/hadoop
        - /opt/hadoop
        cpuLimit: "0.5"
        memoryLimit: "512Mi"
        cpuRequest: "0.5"
        memoryRequest: "512Mi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
      containers:
      - name: namenode
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m -x

          export HADOOP_LOG_DIR=$0
          export HADOOP_CONF_DIR=$1
          export HADOOP_HOME=$2

          function shutdown() {
            echo "Stopping Namenode"
            is_active=$($HADOOP_HOME/bin/hdfs haadmin -getAllServiceState | grep "$(hostname -f)" | grep "active" | wc -l)

            if [[ $is_active == 1 ]]; then
              for i in $(echo $NNS | tr "," "\n"); do
                if [[ $($HADOOP_HOME/bin/hdfs haadmin -getServiceState $i | grep "standby" | wc -l) == 1 ]]; then
                  STANDBY_SERVICE=$i
                  break
                fi
              done

              echo "Is Active. Transitioning to standby"
              if [[ -n "$MY_SERVICE" && -n "$STANDBY_SERVICE" && $MY_SERVICE != $STANDBY_SERVICE ]]; then
                echo "Failing over from $MY_SERVICE to $STANDBY_SERVICE"
                $HADOOP_HOME/bin/hdfs haadmin -failover $MY_SERVICE $STANDBY_SERVICE
              else
                echo "$MY_SERVICE or $STANDBY_SERVICE is not defined or same. Cannot failover. Exitting..."
              fi
            else
             echo "Is not active"
            fi
            sleep 60
            echo "Completed shutdown cleanup"
            touch /lifecycle/nn-terminated
            $HADOOP_HOME/bin/hdfs --daemon stop namenode
          }

          NAMESERVICES=$($HADOOP_HOME/bin/hdfs getconf -confKey dfs.nameservices)
          NNS=$($HADOOP_HOME/bin/hdfs getconf -confKey dfs.ha.namenodes.$NAMESERVICES)
          MY_SERVICE=""
          HTTP_ADDR=""
          for i in $(echo $NNS | tr "," "\n"); do
            if [[ $($HADOOP_HOME/bin/hdfs getconf -confKey dfs.namenode.rpc-address.$NAMESERVICES.$i | sed 's/:[0-9]\+$//' | grep $(hostname -f) | wc -l ) == 1 ]]; then
              MY_SERVICE=$i
              HTTP_ADDR=$($HADOOP_HOME/bin/hdfs getconf -confKey dfs.namenode.http-address.$NAMESERVICES.$i)
            fi
          done

          echo "My Service: $MY_SERVICE"

          trap shutdown SIGTERM
          echo "N" | $HADOOP_HOME/bin/hdfs namenode -bootstrapStandby || true
          exec $HADOOP_HOME/bin/hdfs namenode &
          wait
        args:
        - /var/log/hadoop
        - /etc/hadoop
        - /opt/hadoop
        ports:
        - port: 8020
          name: namenode-0
        - port: 9870
          name: namenode-1
        - port: 50070
          name: namenode-2
        - port: 9000
          name: namenode-3
        startupProbe:
          initialDelay: 30
          timeout: 60
          failureThreshold: 10
          command:
          - /bin/bash
          - -c
          - |
            #! /bin/bash
            set -m

            export HADOOP_LOG_DIR=$0
            export HADOOP_CONF_DIR=$1
            export HADOOP_HOME=$2

            if [[ $($HADOOP_HOME/bin/hdfs dfsadmin -safemode get | grep "Safe mode is OFF" | wc -l) == 0 ]]; then
              echo "Looks like there is no namenode with safemode off. Skipping checks..."
              exit 0
            elif [[ $($HADOOP_HOME/bin/hdfs dfsadmin -safemode get | grep "$(hostname -f)" | grep "Safe mode is OFF" | wc -l) == 1 ]]; then
              echo "Namenode is out of safemode. Exiting..."
              exit 0
            else
              echo "Namenode is still in safemode. Failing..."
              exit 1
            fi

            echo "Something unexpected happened at startup probe. Failing..."
            exit 1
          - /var/log/hadoop
          - /etc/hadoop
          - /opt/hadoop
        livenessProbe:
          tcpPort: 8020
          initialDelay: 60
        readinessProbe:
          tcpPort: 8020
          initialDelay: 60
        cpuLimit: "0.5"
        memoryLimit: "3Gi"
        cpuRequest: "0.5"
        memoryRequest: "3Gi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
          addSysPtrace: false
        volumeMounts:
        - name: data
          mountPath: /grid/1
          readOnly: false
        - name: lifecycle
          mountPath: /lifecycle
          readOnly: false
      - name: zkfc
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m

          export HADOOP_LOG_DIR=$0
          export HADOOP_CONF_DIR=$1
          export HADOOP_HOME=$2

          function shutdown() {
            while true; do
              if [[ -f "/lifecycle/nn-terminated" ]]; then
                echo "Stopping zkfc"
                sleep 10
                $HADOOP_HOME/bin/hdfs --daemon stop zkfc
                break
              fi
              echo "Waiting for namenode to die"
              sleep 2
            done
          }

          trap shutdown SIGTERM
          exec $HADOOP_HOME/bin/hdfs zkfc &
          wait
        args:
        - /var/log/hadoop
        - /etc/hadoop
        - /opt/hadoop
        ports:
        - port: 8019
          name: zkfc-0
        livenessProbe:
          tcpPort: 8019
          initialDelay: 30
        readinessProbe:
          tcpPort: 8019
          initialDelay: 30
        cpuLimit: "0.2"
        memoryLimit: "512Mi"
        cpuRequest: "0.2"
        memoryRequest: "512Mi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
          addSysPtrace: false
        volumeMounts:
        - name: lifecycle
          mountPath: /lifecycle
          readOnly: false

```

</details>

## Hbase Tenant

### Operator Side

1. Add additional namespaces to watch for. Here specific namespace to be onboarded for a particular tenant

    ```sh
    vim operator/config/custom/config/hbase-operator-config.yaml
    ```

1. Create configmap with command

    ```sh
     kubectl apply -f operator/config/custom/config/hbase-operator-config.yaml -n hbase-operator-ns
    ```

1. Deploy operator

### Package and Deploy Hbase Tenant

#### Helm Chart

!!! danger "Changing namespace names would mean configuration having host names should also be changed such as zookeeper, namenode etc"

1. A customisable base helm chart is available to make use of and simplify deployable helm charts. You can find `./helm-charts/hbase-chart/` under root folder of this repository

1. Build the base helm chart from root folder of this repository as follows

    ```sh
    helm package helm-charts/hbase-chart/
    ```

1. You can find package `hbase-chart-x.x.x.tgz` created under root folder of this repository. Otherwise you can publish chart to `jfrog` or `harbor` or any other chart registry. For manual testing, you can move `hbase-chart-x.x.x.tgz` under `examples/hbasetenant-chart/charts/`

    ```sh
    mv hbase-chart-x.x.x.tgz examples/hbasetenant-chart/charts/
    ```

1. Open `examples/hbasetenant-chart/values.yaml`, and modify the values as per your requirement. Some of the recommended modifications are

    1. image: Docker image of hbase we built in previous section
    1. annotations: In this examples, we have used to demonstrate MTL (Monitoring, Telemetry and Logging)
    1. Volume claims for your k8s can be fetched using `kubectl get storageclass`. Which can be used to replace `storageClass`
    1. `probeDelay`: This will affect both `liveness` and `readiness` alike
    1. Memory limits / requests and CPU limits / request as per your requirements

1. You can deploy your helm package using following command

    ```sh
    helm upgrade --install --debug hbasetenant-chart examples/hbasetenant-chart/ -n hbase-tenant-ns
    ```

#### via Manifest

<details>
<summary>Sample Tenant yaml configuration</summary>

```yaml
# Source: hbasetenant-chart/templates/hbasetenant.yaml
apiVersion: kvstore.flipkart.com/v1
kind: HbaseTenant
metadata:
  name: hbase-tenant
  namespace: hbase-tenant-ns
spec:
  baseImage: hbase:2.4.8
  fsgroup: 1011
  configuration:
    hbaseConfigName: hbase-config
    hbaseConfigMountPath: /etc/hbase
    hbaseConfig:
      {}
    hadoopConfigName: hadoop-config
    hadoopConfigMountPath: /etc/hadoop
    hadoopConfig:
      {}
  datanode:
      name: hbase-tenant-dn
      size: 4
      isPodServiceRequired: false
      shareProcessNamespace: true
      terminateGracePeriod: 120
      volumeClaims:
      - name: data
        storageSize: 10Gi
        storageClassName: standard
      volumes:
      - name: lifecycle
        volumeSource: EmptyDir
      - name: nodeinfo
        volumeSource: HostPath
        path: /etc/nodeinfo
      initContainers:
      - name: init-dnslookup
        isBootstrap: false
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m

          i=0
          while true; do
            echo "$i iteration"
            dig +short $(hostname -f) | grep -v -e '^$'
            if [ $? == 0 ]; then
              sleep 30 # 30 seconds default dns caching
              echo "Breaking..."
              break
            fi
            i=$((i + 1))
            sleep 1
          done
        cpuLimit: "0.2"
        memoryLimit: "128Mi"
        cpuRequest: "0.2"
        memoryRequest: "128Mi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
      - name: init-faultdomain
        isBootstrap: false
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m -x

          export HBASE_LOG_DIR=/var/log/hbase
          export HBASE_CONF_DIR=/etc/hbase
          export HBASE_HOME=/opt/hbase

          # Make it optional
          FAULT_DOMAIN_COMMAND="cat /etc/nodeinfo | grep 'smd' | sed 's/smd=//' | sed 's/\"//g'"
          HOSTNAME=$(hostname -f)

          echo "Running command to get fault domain: $FAULT_DOMAIN_COMMAND"
          SMD=$(eval $FAULT_DOMAIN_COMMAND)
          echo "SMD value: $SMD"

          if [[ -n "$FAULT_DOMAIN_COMMAND" ]]; then
            echo "create /hbase-operator $SMD" | $HBASE_HOME/bin/hbase zkcli 2> /dev/null || true
            echo "create /hbase-operator/$HOSTNAME $SMD" | $HBASE_HOME/bin/hbase zkcli 2> /dev/null
            echo ""
            echo "Completed"
          fi
        cpuLimit: "0.1"
        memoryLimit: "386Mi"
        cpuRequest: "0.1"
        memoryRequest: "386Mi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
        volumeMounts:
        - name: nodeinfo
          mountPath: /etc/nodeinfo
          readOnly: true
      - name: init-refreshnn
        isBootstrap: false
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -x -m

          export HADOOP_LOG_DIR=/var/log/hadoop
          export HADOOP_CONF_DIR=/etc/hadoop
          export HADOOP_HOME=/opt/hadoop

          $HADOOP_HOME/bin/hdfs dfsadmin -refreshNodes

        cpuLimit: "0.2"
        memoryLimit: "256Mi"
        cpuRequest: "0.2"
        memoryRequest: "256Mi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
      containers:
      - name: datanode
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m

          export HADOOP_LOG_DIR=$0
          export HADOOP_CONF_DIR=$1
          export HADOOP_HOME=$2
          export HADOOP_CONF_NAME=$3

          function shutdown() {
            while true; do
              #TODO: Kill it beyond certain wait time
              if [[ -f "/lifecycle/rs-terminated" ]]; then
                echo "Stopping datanode"
                sleep 3
                $HADOOP_HOME/bin/hdfs --daemon stop datanode
                break
              fi
              echo "Waiting for regionserver to die"
              sleep 2
            done
          }

          #move this to init container
          curl -sX GET http://127.0.0.1:8802/v1/configmaps/$HADOOP_CONF_NAME | jq '.data | to_entries[] | .key, .value' | while IFS= read -r key; read -r value; do echo $value | jq -r '.' | tee $(echo $key | jq -r '.' | xargs -I {} echo $HADOOP_CONF_DIR/{}) > /dev/null; done

          sleep 1

          trap shutdown SIGTERM
          exec $HADOOP_HOME/bin/hdfs datanode &
          PID=$!

          #TODO: Correct way to identify if process is up
          touch /lifecycle/dn-started

          wait $PID
        args:
        - /var/log/hadoop
        - /etc/hadoop
        - /opt/hadoop
        - hadoop-config
        ports:
        - port: 9866
          name: datanode-0
        startupProbe:
          initialDelay: 30
          timeout: 60
          failureThreshold: 10
          command:
          - /bin/bash
          - -c
          - |
            #! /bin/bash
            set -m

            export HADOOP_LOG_DIR=$0
            export HADOOP_CONF_DIR=$1
            export HADOOP_HOME=$2

            while :
            do
              if [[ $($HADOOP_HOME/bin/hdfs dfsadmin -report -live | grep "$(hostname -f)" | wc -l) == 2 ]]; then
                echo "datanode is listed as live under namenode. Exiting..."
                exit 0
              else
                echo "datanode is still not listed as live under namenode"
                exit 1
              fi
            done
            exit 1
          - /var/log/hadoop
          - /etc/hadoop
          - /opt/hadoop
          - hadoop-config
        livenessProbe:
          tcpPort: 9866
          initialDelay: 60
        readinessProbe:
          tcpPort: 9866
          initialDelay: 60
        cpuLimit: "0.3"
        memoryLimit: "2Gi"
        cpuRequest: "0.3"
        memoryRequest: "2Gi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
          addSysPtrace: true
        volumeMounts:
        - name: data
          mountPath: /grid/1
          readOnly: false
        - name: lifecycle
          mountPath: /lifecycle
          readOnly: false
        - name: nodeinfo
          mountPath: /etc/nodeinfo
          readOnly: true
      - name: regionserver
        command:
        - /bin/bash
        - -c
        - |
          #! /bin/bash
          set -m
          export HBASE_LOG_DIR=$0
          export HBASE_CONF_DIR=$1
          export HBASE_HOME=$2
          export HBASE_CONF_NAME=$3
          export USER=$(whoami)

          mkdir -p $HBASE_LOG_DIR
          ln -sf /dev/stdout $HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).out
          ln -sf /dev/stdout $HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).log

          function shutdown() {
            echo "Stopping Regionserver"
            host=`hostname -f`
            $HBASE_HOME/bin/hbase org.apache.hadoop.hbase.util.RegionMover -m 6 -r $host -o unload
            touch /lifecycle/rs-terminated
            $HBASE_HOME/bin/hbase-daemon.sh stop regionserver
          }

          while true; do
            if [[ -f "/lifecycle/dn-started" ]]; then
              echo "Starting rs"
              sleep 5
              break
            fi
            echo "Waiting for datanode to start"
            sleep 2
          done

          curl -sX GET http://127.0.0.1:8802/v1/configmaps/$HBASE_CONF_NAME | jq '.data | to_entries[] | .key, .value' | while IFS= read -r key; read -r value; do echo $value | jq -r '.' | tee $(echo $key | jq -r '.' | xargs -I {} echo $HBASE_CONF_DIR/{}) > /dev/null; done

          sleep 1

          trap shutdown SIGTERM
          exec $HBASE_HOME/bin/hbase-daemon.sh foreground_start regionserver &
          wait
        args:
        - /var/log/hbase
        - /etc/hbase
        - /opt/hbase
        - hbase-config
        ports:
        - port: 16030
          name: regionserver-0
        - port: 16020
          name: regionserver-1
        livenessProbe:
          tcpPort: 16030
          initialDelay: 60
        readinessProbe:
          tcpPort: 16030
          initialDelay: 60
        cpuLimit: "0.4"
        memoryLimit: "3Gi"
        cpuRequest: "0.4"
        memoryRequest: "3Gi"
        securityContext:
          runAsUser: 1011
          runAsGroup: 1011
          addSysPtrace: true
        volumeMounts:
        - name: lifecycle
          mountPath: /lifecycle
          readOnly: false
        - name: nodeinfo
          mountPath: /etc/nodeinfo
          readOnly: true

```

</details>
