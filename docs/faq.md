1. Are there any caveats to know about this packaging

    Namenode caches journalnode ip address and doesn't change when address changes. You might want to patch your hdfs deployment with this patch [HADOOP-17068](https://issues.apache.org/jira/browse/HADOOP-17068)

1. Minimal command line tools required for this package to work correctly

    1. nslookup
    1. netcat
    1. curl

1. Minimal configuration for the package to work correctly

    1. Zookeeper should have `zookeeper.4lw.commands` enabled with bare minimum `stat` command

1. Tested versions for various components

    1. Hadoop: 3.1.x
    1. Hbase: 2.1.x - 2.4.x
    1. Zookeeper: 3.4.10+

1. Can I change service ports for components such as zookeeper, journalnode, etc. If not what are the defaults

    At present, ports are hard coded with packaging, will change this in the future.

    1. Zookeeper: 2181
    1. Journalnode: 8485
    1. Hmaster: 16000, 16010
    1. RegionServer: 16020, 16030
    1. Datanode: 9866
    1. Namenode: 8020, 50070
    1. Zkfc: 8019

1. Which shell is recommend bundled with hbase docker image

    All tests are done with `bash` shell and hence can't guarantee working with any other shell.
