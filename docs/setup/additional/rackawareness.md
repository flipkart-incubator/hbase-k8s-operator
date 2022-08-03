# Rackawareness

## How to enable

1. Build rack utils docker image as below

    ```sh
    docker build utilities/rackuitls/ --network host --build-arg AppName="RackUtils" -t "hbase-rack-utils:1.0.0"
    docker push hbase-rack-utils:1.0.0
    ```

1. Enable sidecar along with hmaster container as described below in values.yaml file

    ```yaml
    sidecarcontainers:
    - name: rackutils
      image: hbase-rack-utils:1.0.0
      cpuLimit: 0.2
      memoryLimit: 256Mi
      cpuRequest: 0.2
      memoryRequest: 256Mi
      runAsUser: 1011
      runAsGroup: 1011
      command: ["./entrypoint"]
      args: ["com.flipkart.hbase.HbaseRackUtils", "/etc/hbase", "/hbase-operator", "/opt/share/rack_topology.data"]
      volumeMounts:
      - name: data
        mountPath: /opt/share
    ```

    * `/etc/hbase` is where hbase configuration is mounted
    * `/hbase-operator` is zookeeper znode where rack topology information is stored for each datanode
    * `/opt/share/rack_topology.data` is path on hmaster container where topology information is stored

1. Command using which rack(fault domain) information can be fetched from each datanode

    ```sh
    commands:
      faultDomainCommand: "cat /etc/nodeinfo | grep 'smd' | sed 's/smd=//' | sed 's/\"//g'"
    ```

1. Add following configuration in hbase-site.xml

    ```xml
    <property>
      <name>net.topology.script.file.name</name>
      <value>/opt/scripts/rack_topology</value>
    </property>
    ```

## How it works

* Refer to previous section before reading further

* Rack state for each datanode is stored in zookeeper znode example: `/hbase-operator`

* Each datanode has init container refer: `chart/templates/meta/_faultdomain.tpl` which at the time of creation updates its latest faultdomain in znode

* Hmaster side container as described above, reads the znode `/hbase-operator` and constructs topology file `/opt/share/rack_topology.data`

* Topology file is configured in hbase-site.xml for hmaster to read each time a region assignment is made. Favored nodes of each region would be on different racks wherever possible.
