package com.flipkart.hbase;

import org.apache.curator.framework.CuratorFramework;
import org.apache.curator.framework.CuratorFrameworkFactory;
import org.apache.curator.framework.recipes.cache.TreeCache;
import org.apache.curator.retry.ExponentialBackoffRetry;
import org.apache.hadoop.conf.Configuration;
import org.apache.hadoop.fs.Path;
import org.apache.hadoop.hbase.HBaseConfiguration;
import org.apache.zookeeper.KeeperException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.BufferedWriter;
import java.io.File;
import java.io.FileWriter;
import java.net.InetAddress;

public class HbaseRackUtils implements Runnable {
    private static final Logger LOG = LoggerFactory.getLogger(HbaseRackUtils.class);
    private final static String ZK_QUORUM_STRING = "hbase.zookeeper.quorum";
    private final CuratorFramework client;
    private final String basePath;
    private final String topodataFile;

    public HbaseRackUtils(String hostPath, String basePath, String topodataFile) {
        this.basePath = basePath;
        this.topodataFile = topodataFile;
        client = CuratorFrameworkFactory.newClient(hostPath, new ExponentialBackoffRetry(1000, 3));
        client.getUnhandledErrorListenable().addListener((message, e) -> {
            LOG.error("Unhandled Error: " + message, e);
        });
        client.getConnectionStateListenable().addListener((c, newState) -> {
            LOG.info("Zookeeper State Change. New State: " + newState);
        });
        client.start();
    }

    public static void main(String[] args) throws Exception {
        if (args.length < 2) {
            LOG.error("Usage: java -jar rackutils.jar hbaseConfPath znode topologyfile");
            LOG.error("Example: java -jar rackutils.jar /etc/hbase /hbase-operator /tmp/rack_topology.data");
            System.exit(1);
        }

        Configuration configuration = HBaseConfiguration.create();
        configuration.addResource(new Path(args[0] + "/hbase-site.xml"));

        LOG.info("Connecting to: " + configuration.get(ZK_QUORUM_STRING) + " on znode " + args[1] + " for path " + args[2]);
        new HbaseRackUtils(configuration.get(ZK_QUORUM_STRING), args[1], args[2]).run();
        Thread.currentThread().join();
    }

    private void refreshRacks() throws Exception {
        try (BufferedWriter writer = new BufferedWriter(new FileWriter(new File(this.topodataFile).getAbsoluteFile()))) {
            for (String child : this.client.getChildren().forPath(this.basePath)) {
                byte[] value = this.client.getData().forPath(this.basePath + "/" + child);
                String address = null;
                try {
                    address = InetAddress.getByName(child).getHostAddress();
                    writer.write(address + " " + new String(value));
                    writer.newLine();
                } catch (Exception ex) {
                    LOG.error("Exception while resolution for address: {} with error: {}", address, ex.getMessage(), ex);
                    System.exit(1);
                }
            }
        } catch (KeeperException ex) {
            LOG.error("Failed with exception while watching. Error: " + ex.getMessage(), ex);
            System.exit(1);
        }
    }

    @Override
    public void run() {
        try {
            TreeCache cache = TreeCache.newBuilder(this.client, this.basePath).setCacheData(false).build();
            cache.getListenable().addListener((c, event) -> {
                if (event.getData() != null) {
                    LOG.info("Received update for znode of event type: " + event.getType() + " Path: " + event.getData().getPath());
                } else {
                    LOG.info("Received update for znode of event type: " + event.getType());
                }
                refreshRacks();
            });
            cache.start();
        } catch (Exception e) {
            LOG.error("Failed with exception while watching. Error: " + e.getMessage(), e);
            System.exit(1);
        }
    }
}

