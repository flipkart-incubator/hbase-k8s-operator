package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	kvstorev1 "github.com/flipkart-incubator/hbase-k8s-operator/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"
)

const (
	tenantName1 = "yak-tenant-test-1"
)

// TestHbaseTenantReconciler_ResNotFound tests the Reconcile method for a HbaseTenant object that is not found
func TestHbaseTenantReconciler_ResNotFound(t *testing.T) {
	k8sMockClient, reconciler, ctx, req := doTenantTestSetup()

	k8sMockClient.On("Get", ctx, req.NamespacedName, mock.Anything).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	k8sMockClient.AssertExpectations(t)
}

// TestHbaseTenantReconciler_ErrorGettingRes tests the Reconcile method when error is returned while getting the HbaseTenant object
func TestHbaseTenantReconciler_ErrorGettingRes(t *testing.T) {
	k8sMockClient, reconciler, ctx, req := doTenantTestSetup()
	k8sMockClient.On("Get", ctx, req.NamespacedName, mock.Anything).Return(assert.AnError)

	result, err := reconciler.Reconcile(ctx, req)
	assert.Error(t, err)
	assert.Equal(t, ctrl.Result{RequeueAfter: time.Second * 5}, result)

	k8sMockClient.AssertExpectations(t)
}

// TestHbaseTenantReconciler_SuccessfulReconciliation_ObjectsNotFound tests the Reconcile method
// when all objects are not found and created successfully
func TestHbaseTenantReconciler_SuccessfulReconciliation_ObjectsNotFound(t *testing.T) {
	//mock hbase tenant object
	hbasetenant := getMockHbaseTenant()

	k8sMockClient, reconciler, ctx, req := doTenantTestSetup()

	k8sMockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseTenant{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseTenant)
			*arg = *hbasetenant
		}).
		Return(nil)

	mockCfgHb := buildConfigMap(hbasetenant.Spec.Configuration.HbaseConfigName, hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.Configuration.HbaseConfig, hbasetenant.Spec.Configuration.HbaseTenantConfig, reconciler.Log)
	ctrl.SetControllerReference(hbasetenant, mockCfgHb, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHb.Name, Namespace: mockCfgHb.Namespace}, &corev1.ConfigMap{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mockCfgHb, []client.CreateOption(nil)).Return(nil)

	mockCfgHd := buildConfigMap(hbasetenant.Spec.Configuration.HadoopConfigName, hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.Configuration.HadoopConfig, hbasetenant.Spec.Configuration.HadoopTenantConfig, reconciler.Log)
	ctrl.SetControllerReference(hbasetenant, mockCfgHd, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHd.Name, Namespace: mockCfgHd.Namespace}, &corev1.ConfigMap{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mockCfgHd, []client.CreateOption(nil)).Return(nil)

	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: hbasetenant.Spec.Datanode.Name, Namespace: hbasetenant.Namespace}, &appsv1.StatefulSet{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))

	mockStsSvc := buildService(hbasetenant.Name, hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.ServiceLabels, hbasetenant.Spec.ServiceSelectorLabels, []kvstorev1.HbaseClusterDeployment{hbasetenant.Spec.Datanode}, true)
	ctrl.SetControllerReference(hbasetenant, mockStsSvc, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockStsSvc.Name, Namespace: hbasetenant.Namespace}, &corev1.Service{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mockStsSvc, []client.CreateOption(nil)).Return(nil)

	mockStsZK := buildStatefulSet(hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.BaseImage, false,
		hbasetenant.Spec.Configuration, "", hbasetenant.Spec.FSGroup, hbasetenant.Spec.Datanode, reconciler.Log, false)
	ctrl.SetControllerReference(hbasetenant, mockStsZK, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: hbasetenant.Spec.Datanode.Name, Namespace: hbasetenant.Namespace}, &appsv1.StatefulSet{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mockStsZK, []client.CreateOption(nil)).Return(nil)

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, result)

	// AssertExpectations asserts that everything specified with On and Return was in fact called as expected.
	k8sMockClient.AssertExpectations(t)
}

// TestHbaseTenantReconciler_SuccessfulReconciliation_AllObjectsFound tests the Reconcile method
// when all objects are found and updated successfully
func TestHbaseTenantReconciler_SuccessfulReconciliation_AllObjectsFound(t *testing.T) {
	//mock hbase tenant object
	hbasetenant := getMockHbaseTenant()

	k8sMockClient, reconciler, ctx, req := doTenantTestSetup()

	k8sMockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseTenant{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseTenant)
			*arg = *hbasetenant
		}).
		Return(nil)

	mockCfgHb := buildConfigMap(hbasetenant.Spec.Configuration.HbaseConfigName, hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.Configuration.HbaseConfig, hbasetenant.Spec.Configuration.HbaseTenantConfig, reconciler.Log)
	ctrl.SetControllerReference(hbasetenant, mockCfgHb, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHb.Name, Namespace: mockCfgHb.Namespace}, &corev1.ConfigMap{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.ConfigMap)
			*arg = *mockCfgHb
		}).
		Return(nil)

	mockCfgHd := buildConfigMap(hbasetenant.Spec.Configuration.HadoopConfigName, hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.Configuration.HadoopConfig, hbasetenant.Spec.Configuration.HadoopTenantConfig, reconciler.Log)
	ctrl.SetControllerReference(hbasetenant, mockCfgHd, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHd.Name, Namespace: mockCfgHd.Namespace}, &corev1.ConfigMap{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.ConfigMap)
			*arg = *mockCfgHd
		}).
		Return(nil)
	k8sMockClient.On("Update", ctx, mock.Anything, []client.UpdateOption(nil)).Return(nil)

	mockStsSvc := buildService(hbasetenant.Name, hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.ServiceLabels, hbasetenant.Spec.ServiceSelectorLabels, []kvstorev1.HbaseClusterDeployment{hbasetenant.Spec.Datanode}, true)
	ctrl.SetControllerReference(hbasetenant, mockStsSvc, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockStsSvc.Name, Namespace: hbasetenant.Namespace}, &corev1.Service{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.Service)
			*arg = *mockStsSvc
		}).
		Return(nil)
	k8sMockClient.On("Update", ctx, mockStsSvc, []client.UpdateOption(nil)).Return(nil)

	mockSts := buildStatefulSet(hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.BaseImage, false,
		hbasetenant.Spec.Configuration, "", hbasetenant.Spec.FSGroup, hbasetenant.Spec.Datanode, reconciler.Log, false)
	ctrl.SetControllerReference(hbasetenant, mockSts, reconciler.Scheme)

	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: hbasetenant.Spec.Datanode.Name, Namespace: hbasetenant.Namespace}, &appsv1.StatefulSet{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*appsv1.StatefulSet)
			*arg = *mockSts
		}).Return(nil).Times(2)

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: time.Second * 20}, result)

	// AssertExpectations asserts that everything specified with On and Return was in fact called as expected.
	k8sMockClient.AssertExpectations(t)
}

// TestHbaseTenantReconciler_SuccessfulReconciliation_AllObjectsFoundRestFlow tests the Reconcile method happy flow
// This test will fail if ran as individual as it depends on hashstore impl.
// when ran along with other tests it will pass as hashstore will have values filled and update method will not be called.
func TestHbaseTenantReconciler_SuccessfulReconciliation_AllObjectsFoundRestFlow(t *testing.T) {
	//mock hbase tenant object
	hbasetenant := getMockHbaseTenant()

	k8sMockClient, reconciler, ctx, req := doTenantTestSetup()

	k8sMockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseTenant{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseTenant)
			*arg = *hbasetenant
		}).
		Return(nil)

	mockCfgHb := buildConfigMap(hbasetenant.Spec.Configuration.HbaseConfigName, hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.Configuration.HbaseConfig, hbasetenant.Spec.Configuration.HbaseTenantConfig, reconciler.Log)
	ctrl.SetControllerReference(hbasetenant, mockCfgHb, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHb.Name, Namespace: mockCfgHb.Namespace}, &corev1.ConfigMap{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.ConfigMap)
			*arg = *mockCfgHb
		}).
		Return(nil)

	mockCfgHd := buildConfigMap(hbasetenant.Spec.Configuration.HadoopConfigName, hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.Configuration.HadoopConfig, hbasetenant.Spec.Configuration.HadoopTenantConfig, reconciler.Log)
	ctrl.SetControllerReference(hbasetenant, mockCfgHd, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHd.Name, Namespace: mockCfgHd.Namespace}, &corev1.ConfigMap{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.ConfigMap)
			*arg = *mockCfgHd
		}).
		Return(nil)

	mockStsSvc := buildService(hbasetenant.Name, hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.ServiceLabels, hbasetenant.Spec.ServiceSelectorLabels, []kvstorev1.HbaseClusterDeployment{hbasetenant.Spec.Datanode}, true)
	ctrl.SetControllerReference(hbasetenant, mockStsSvc, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockStsSvc.Name, Namespace: hbasetenant.Namespace}, &corev1.Service{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.Service)
			*arg = *mockStsSvc
		}).
		Return(nil)

	mockSts := buildStatefulSet(hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.BaseImage, false,
		hbasetenant.Spec.Configuration, "", hbasetenant.Spec.FSGroup, hbasetenant.Spec.Datanode, reconciler.Log, false)
	ctrl.SetControllerReference(hbasetenant, mockSts, reconciler.Scheme)
	mockSts.Status.ReadyReplicas = hbasetenant.Spec.Datanode.Size
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: hbasetenant.Spec.Datanode.Name, Namespace: hbasetenant.Namespace}, &appsv1.StatefulSet{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*appsv1.StatefulSet)
			*arg = *mockSts
		}).Return(nil)

	mockPdb := buildPodDisruptionBudget(hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.Datanode, reconciler.Log)
	ctrl.SetControllerReference(hbasetenant, mockPdb, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: hbasetenant.Spec.Datanode.Name + "-pdb", Namespace: hbasetenant.Namespace}, &policyv1.PodDisruptionBudget{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mockPdb, []client.CreateOption(nil)).Return(nil)

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, result)

	// AssertExpectations asserts that everything specified with On and Return was in fact called as expected.
	k8sMockClient.AssertExpectations(t)
}

// TestHbaseTenantReconciler_Failure_EventPublish tests the Reconcile method when error is returned while publishing event
func TestHbaseTenantReconciler_Failure_EventPublish(t *testing.T) {
	//mock hbase tenant object
	hbasetenant := getInvalidConfigHbasetenant()

	k8sMockClient, reconciler, ctx, req := doTenantTestSetup()

	k8sMockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseTenant{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseTenant)
			*arg = *hbasetenant
		}).
		Return(nil)

	k8sMockClient.On("Get", ctx, types.NamespacedName{Namespace: hbasetenant.Namespace, Name: "ConfigValidateFailed"}, &corev1.Event{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mock.Anything, []client.CreateOption(nil)).Return(nil)

	result, err := reconciler.Reconcile(ctx, req)
	assert.Error(t, err)
	assert.Equal(t, ctrl.Result{Requeue: false, RequeueAfter: 0}, result)

	// AssertExpectations asserts that everything specified with On and Return was in fact called as expected.
	k8sMockClient.AssertExpectations(t)
}

func getMockClientAndTenantReconciler() (*K8sMockClient, *HbaseTenantReconciler) {
	mockClient := new(K8sMockClient)
	scheme := runtime.NewScheme()
	_ = kvstorev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	reconciler := &HbaseTenantReconciler{
		Client: mockClient,
		Log:    ctrl.Log.WithName("controllers").WithName("HbaseTenant"),
		Scheme: scheme,
	}
	return mockClient, reconciler
}

func getMockHbaseTenant() *kvstorev1.HbaseTenant {
	tenant := &kvstorev1.HbaseTenant{}
	hbaseTenantJson := "{\"apiVersion\":\"kvstore.abc.com/v1\",\"kind\":\"HbaseTenant\",\"metadata\":{\"labels\":{\"app.kubernetes.io/managed-by\":\"Helm\"},\"name\":\"yak-tenant-test-1\",\"namespace\":\"yak-tenant-test-1-ns\"},\"spec\":{\"baseImage\":\"test-image\",\"configuration\":{\"hadoopConfig\":{\"core-site.xml\":\"<?xmlversion=\\\"1.0\\\"?>\\n<?xml-stylesheettype=\\\"text/xsl\\\"href=\\\"configuration.xsl\\\"?>\\n<!--Generatedbyconfdon2021-03-0911:46:01.973761409+0530ISTm=+0.012850605-->\\n<configuration>\\n<property>\\n<name>fs.trash.interval</name>\\n<value>1440</value>\\n</property>\\n</configuration>\\n\",\"hadoop-env.sh\":\"exportHADOOP_CONF_DIR=/etc/hadoop\\nexportHADOOP_PID_DIR=/var/run/hadoop\\nforfin$HADOOP_HOME/contrib/capacity-scheduler/*.jar;do\\nif[\\\"$HADOOP_CLASSPATH\\\"];then\\nexportHADOOP_CLASSPATH=$HADOOP_CLASSPATH:$f\\nelse\\nexportHADOOP_CLASSPATH=$f\\nfi\\ndone\\nexportHADOOP_OPTS=\\\"$HADOOP_OPTs-Djava.net.preferIPv4Stack=true-Dnetworkaddress.cache.ttl=60-Dsun.net.inetaddr.ttl=60-XX:+UseZGC\\\"\\nexportHDFS_NAMENODE_OPTS=\\\"-Xms4096m-Xmx4096m-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10102-Dcom.sun.management.jmxremote.ssl=false-Dhadoop.security.logger=${HADOOP_SECURITY_LOGGER:-INFO,RFAS}-Dhdfs.audit.logger=${HDFS_AUDIT_LOGGER:-INFO,NullAppender}\\\"\\nexportHDFS_DATANODE_OPTS=\\\"-Xms3072m-Xmx3072m-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10101-Dcom.sun.management.jmxremote.ssl=false-Dhadoop.security.logger=ERROR,RFAS\\\"\\nexportHDFS_JOURNALNODE_OPTS=\\\"-Xms512m-Xmx512m-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10106-Dcom.sun.management.jmxremote.ssl=false\\\"\\nexportHDFS_ZKFC_OPTS=\\\"-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10107-Dcom.sun.management.jmxremote.ssl=false\\\"\\nexportHADOOP_SECONDARYNAMENODE_OPTS=\\\"-Xms4096m-Xmx4096m-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10102-Dcom.sun.management.jmxremote.ssl=false-XX:+UnlockCommercialFeatures-XX:+FlightRecorder-Dhadoop.security.logger=${HADOOP_SECURITY_LOGGER:-INFO,RFAS}-Dhdfs.audit.logger=${HDFS_AUDIT_LOGGER:-INFO,NullAppender}\\\"\\nexportHADOOP_NFS3_OPTS=\\\"$HADOOP_NFS3_OPTS\\\"\\nexportHADOOP_PORTMAP_OPTS=\\\"-Xmx512m$HADOOP_PORTMAP_OPTS\\\"\\nexportHADOOP_CLIENT_OPTS=\\\"-Xmx512m$HADOOP_CLIENT_OPTS\\\"\\nexportHADOOP_SECURE_DN_USER=${HADOOP_SECURE_DN_USER}\\nexportHADOOP_SECURE_LOG_DIR=${HADOOP_LOG_DIR}/${HADOOP_HDFS_USER}\\nexportHADOOP_PID_DIR=${HADOOP_PID_DIR}\\nexportHADOOP_SECURE_PID_DIR=${HADOOP_PID_DIR}\\nexportHADOOP_IDENT_STRING=$USER\\n\",\"hdfs-site.xml\":\"<?xmlversion=\\\"1.0\\\"?>\\n<?xml-stylesheettype=\\\"text/xsl\\\"href=\\\"configuration.xsl\\\"?>\\n<!--Generatedbyconfdon2021-03-0911:46:01.976938018+0530ISTm=+0.016027218-->\\n<configuration>\\n<property>\\n<name>dfs.replication</name>\\n<value>3</value>\\n</property>\\n</configuration>\\n\"},\"hadoopConfigMountPath\":\"/etc/hadoop\",\"hadoopConfigName\":\"hadoop-config\",\"hbaseConfig\":{\"hbase-env.sh\":\"exportHBASE_OPTS=\\\"-XX:+UseZGC-Dsun.net.inetaddr.ttl=60-Djava.net.preferIPv4Stack=true-Dnetworkaddress.cache.ttl=60\\\"\\nexportSERVER_GC_OPTS=\\\"-verbose:gc-XX:+PrintGCDetails-Xloggc:<FILE-PATH>\\\"\\nexportHBASE_JMX_BASE=\\\"-Dnetworkaddress.cache.ttl=60-Dnetworkaddress.cache.negative.ttl=0-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.ssl=false-Dcom.sun.management.jmxremote.authenticate=false\\\"\\nexportHBASE_MASTER_OPTS=\\\"$HBASE_MASTER_OPTS$HBASE_JMX_BASE-Dcom.sun.management.jmxremote.port=10103-Xms6g-Xmx6g\\\"\\nexportHBASE_REGIONSERVER_OPTS=\\\"$HBASE_REGIONSERVER_OPTS$HBASE_JMX_BASE-Dcom.sun.management.jmxremote.port=10104-Xms8g-Xmx8g-XX:MaxDirectMemorySize=12g-Djute.maxbuffer=536870912\\\"\\nexportHBASE_ZOOKEEPER_OPTS=\\\"$HBASE_ZOOKEEPER_OPTS$HBASE_JMX_BASE-Dcom.sun.management.jmxremote.port=10105-Xms1g-Xmx1g-Djute.maxbuffer=536870912\\\"\\nexportHBASE_PID_DIR=/var/run/hbase\\nexportHBASE_MANAGES_ZK=false\\nexportLD_LIBRARY_PATH=/opt/hadoop/lib/native\",\"hbase-policy.xml\":\"<?xmlversion=\\\"1.0\\\"?>\\n<?xml-stylesheettype=\\\"text/xsl\\\"href=\\\"configuration.xsl\\\"?>\\n<!--\\n/**\\n*LicensedtotheApacheSoftwareFoundation(ASF)underone\\n*ormorecontributorlicenseagreements.SeetheNOTICEfile\\n*distributedwiththisworkforadditionalinformation\\n*regardingcopyrightownership.TheASFlicensesthisfile\\n*toyouundertheApacheLicense,Version2.0(the\\n*\\\"License\\\");youmaynotusethisfileexceptincompliance\\n*withtheLicense.YoumayobtainacopyoftheLicenseat\\n*\\n*http://www.apache.org/licenses/LICENSE-2.0\\n*\\n*Unlessrequiredbyapplicablelaworagreedtoinwriting,software\\n*distributedundertheLicenseisdistributedonan\\\"ASIS\\\"BASIS,\\n*WITHOUTWARRANTIESORCONDITIONSOFANYKIND,eitherexpressorimplied.\\n*SeetheLicenseforthespecificlanguagegoverningpermissionsand\\n*limitationsundertheLicense.\\n*/\\n-->\\n\\n<configuration>\\n<property>\\n<name>security.client.protocol.acl</name>\\n<value>*</value>\\n<description>ACLforClientProtocolandAdminProtocolimplementations(ie.\\nclientstalkingtoHRegionServers)\\nTheACLisacomma-separatedlistofuserandgroupnames.Theuserand\\ngrouplistisseparatedbyablank.Fore.g.\\\"alice,bobusers,wheel\\\".\\nAspecialvalueof\\\"*\\\"meansallusersareallowed.</description>\\n</property>\\n\\n<property>\\n<name>security.admin.protocol.acl</name>\\n<value>*</value>\\n<description>ACLforHMasterInterfaceprotocolimplementation(ie.\\nclientstalkingtoHMasterforadminoperations).\\nTheACLisacomma-separatedlistofuserandgroupnames.Theuserand\\ngrouplistisseparatedbyablank.Fore.g.\\\"alice,bobusers,wheel\\\".\\nAspecialvalueof\\\"*\\\"meansallusersareallowed.</description>\\n</property>\\n\\n<property>\\n<name>security.masterregion.protocol.acl</name>\\n<value>*</value>\\n<description>ACLforHMasterRegionInterfaceprotocolimplementations\\n(forHRegionServerscommunicatingwithHMaster)\\nTheACLisacomma-separatedlistofuserandgroupnames.Theuserand\\ngrouplistisseparatedbyablank.Fore.g.\\\"alice,bobusers,wheel\\\".\\nAspecialvalueof\\\"*\\\"meansallusersareallowed.</description>\\n</property>\\n</configuration>\\n\",\"hbase-site.xml\":\"<?xmlversion=\\\"1.0\\\"?>\\n<?xml-stylesheettype=\\\"text/xsl\\\"href=\\\"configuration.xsl\\\"?>\\n<!--Generatedbyconfdon2021-03-0911:46:01.975303151+0530ISTm=+0.014392356-->\\n<configuration>\\n<property>\\n<name>cluster.replication.sink.manager</name>\\n<value>org.apache.hadoop.hbase.rsgroup.replication.RSGroupAwareReplicationSinkManager</value>\\n</property>\\n</configuration>\\n\"},\"hbaseConfigMountPath\":\"/etc/hbase\",\"hbaseConfigName\":\"hbase-config\"},\"datanode\":{\"podDisruptionBudget\":{\"maxUnavailable\":1},\"containers\":[{\"args\":[\"/var/log/abc/yak/hadoop\",\"/etc/hadoop\",\"/opt/hadoop\",\"hadoop-config\"],\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\n\\nexportHADOOP_LOG_DIR=$0\\nexportHADOOP_CONF_DIR=$1\\nexportHADOOP_HOME=$2\\nexportHADOOP_CONF_NAME=$3\\nexportUSER=$(whoami)\\nexportHADOOP_LOG_FILE=$HADOOP_LOG_DIR/hadoop-$USER-datanode-$(hostname).log\\n\\nmkdir-p$HADOOP_LOG_DIR\\ntouch$HADOOP_LOG_FILE\\n\\nfunctionshutdown(){\\nwhiletrue;do\\n#TODO:Killitbeyondcertainwaittime\\nif[[-f\\\"/lifecycle/rs-terminated\\\"]];then\\necho\\\"Stoppingdatanode\\\"\\nsleep3\\n$HADOOP_HOME/bin/hdfs--daemonstopdatanode\\nbreak\\nfi\\necho\\\"Waitingforregionservertodie\\\"\\nsleep2\\ndone\\n}\\n\\n#movethistoinitcontainer\\ncurl-sXGEThttp://127.0.0.1:8802/v1/configmaps/$HADOOP_CONF_NAME|jq'.data|to_entries[]|.key,.value'|whileIFS=read-rkey;read-rvalue;doecho$value|jq-r'.'|tee$(echo$key|jq-r'.'|xargs-I{}echo$HADOOP_CONF_DIR/{})>/dev/null;done\\n\\nsleep1\\n\\ntrapshutdownSIGTERM\\nexec$HADOOP_HOME/bin/hdfsdatanode2>&1|tee-a$HADOOP_LOG_FILE&\\nPID=$!\\n\\n#TODO:Correctwaytoidentifyifprocessisup\\ntouch/lifecycle/dn-started\\n\\nwait$PID\\n\"],\"cpuLimit\":\"1\",\"cpuRequest\":\"1\",\"livenessProbe\":{\"initialDelay\":60,\"tcpPort\":9866},\"memoryLimit\":\"4Gi\",\"memoryRequest\":\"4Gi\",\"name\":\"datanode\",\"ports\":[{\"name\":\"datanode-0\",\"port\":9866}],\"readinessProbe\":{\"initialDelay\":60,\"tcpPort\":9866},\"securityContext\":{\"addSysPtrace\":true,\"runAsGroup\":1011,\"runAsUser\":1011},\"startupProbe\":{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\n\\nexportHADOOP_LOG_DIR=$0\\nexportHADOOP_CONF_DIR=$1\\nexportHADOOP_HOME=$2\\n\\nwhile:\\ndo\\nif[[$($HADOOP_HOME/bin/hdfsdfsadmin-report-live|grep\\\"$(hostname-f)\\\"|wc-l)==2]];then\\necho\\\"datanodeislistedasliveundernamenode.Exiting...\\\"\\nexit0\\nelse\\necho\\\"datanodeisstillnotlistedasliveundernamenode\\\"\\nexit1\\nfi\\ndone\\nexit1\\n\",\"/var/log/abc/yak/hadoop\",\"/etc/hadoop\",\"/opt/hadoop\",\"hadoop-config\"],\"failureThreshold\":10,\"initialDelay\":30,\"timeout\":60},\"volumeMounts\":[{\"mountPath\":\"/grid/1\",\"name\":\"data\",\"readOnly\":false},{\"mountPath\":\"/lifecycle\",\"name\":\"lifecycle\",\"readOnly\":false},{\"mountPath\":\"/var/run/hadoop\",\"name\":\"hadooprun\",\"readOnly\":false},{\"mountPath\":\"/etc/nodeinfo\",\"name\":\"nodeinfo\",\"readOnly\":true}]},{\"args\":[\"/var/log/abc/yak/hbase\",\"/etc/hbase\",\"/opt/hbase\",\"hbase-config\"],\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\nexportHBASE_LOG_DIR=$0\\nexportHBASE_CONF_DIR=$1\\nexportHBASE_HOME=$2\\nexportHBASE_CONF_NAME=$3\\nexportUSER=$(whoami)\\n\\nmkdir-p$HBASE_LOG_DIR\\ntouch$HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).log&&tail-F$HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).log&\\ntouch$HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).out&&tail-F$HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).out&\\n\\nfunctionshutdown(){\\necho\\\"StoppingRegionserver\\\"\\nhost=`hostname-f`\\n$HBASE_HOME/bin/hbaseorg.apache.hadoop.hbase.rsgroup.util.RSGroupAwareRegionMover-m6-r$host-ounload\\ntouch/lifecycle/rs-terminated\\n$HBASE_HOME/bin/hbase-daemon.shstopregionserver\\n}\\n\\nwhiletrue;do\\nif[[-f\\\"/lifecycle/dn-started\\\"]];then\\necho\\\"Startingrs\\\"\\nsleep5\\nbreak\\nfi\\necho\\\"Waitingfordatanodetostart\\\"\\nsleep2\\ndone\\n\\ncurl-sXGEThttp://127.0.0.1:8802/v1/configmaps/$HBASE_CONF_NAME|jq'.data|to_entries[]|.key,.value'|whileIFS=read-rkey;read-rvalue;doecho$value|jq-r'.'|tee$(echo$key|jq-r'.'|xargs-I{}echo$HBASE_CONF_DIR/{})>/dev/null;done\\n\\nsleep1\\n\\ntrapshutdownSIGTERM\\nexec$HBASE_HOME/bin/hbase-daemon.shforeground_startregionserver&\\nwait\\n\"],\"cpuLimit\":\"9\",\"cpuRequest\":\"9\",\"livenessProbe\":{\"initialDelay\":60,\"tcpPort\":16030},\"memoryLimit\":\"26Gi\",\"memoryRequest\":\"26Gi\",\"name\":\"regionserver\",\"ports\":[{\"name\":\"regionserver-0\",\"port\":16030},{\"name\":\"regionserver-1\",\"port\":16020}],\"readinessProbe\":{\"initialDelay\":60,\"tcpPort\":16030},\"securityContext\":{\"addSysPtrace\":true,\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/grid/1\",\"name\":\"data\",\"readOnly\":false},{\"mountPath\":\"/lifecycle\",\"name\":\"lifecycle\",\"readOnly\":false},{\"mountPath\":\"/var/run/hadoop\",\"name\":\"hadooprun\",\"readOnly\":false},{\"mountPath\":\"/etc/nodeinfo\",\"name\":\"nodeinfo\",\"readOnly\":true},{\"mountPath\":\"/etc/secrets\",\"name\":\"secret-volume\",\"readOnly\":true}]}],\"dnsConfig\":{\"options\":[{\"name\":\"use-vc\",\"value\":\"\"}]},\"initContainers\":[{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\n\\ni=0\\nwhiletrue;do\\necho\\\"$iiteration\\\"\\ndig+short$(hostname-f)|grep-v-e'^$'\\nif[$?==0];then\\nsleep30#30secondsdefaultdnscaching\\necho\\\"Breaking...\\\"\\nbreak\\nfi\\ni=$((i+1))\\nsleep1\\ndone\\n\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"isBootstrap\":false,\"memoryLimit\":\"128Mi\",\"memoryRequest\":\"128Mi\",\"name\":\"init-dnslookup\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}},{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m-x\\n\\nexportHBASE_LOG_DIR=/var/log/abc/yak/hbase\\nexportHBASE_CONF_DIR=/etc/hbase\\nexportHBASE_HOME=/opt/hbase\\n\\n#Makeitoptional\\nFAULT_DOMAIN_COMMAND=\\\"cat/etc/nodeinfo|grep'smd'|sed's/smd=//'|sed's/\\\\\\\"//g'\\\"\\nHOSTNAME=$(hostname-f)\\n\\necho\\\"Runningcommandtogetfaultdomain:$FAULT_DOMAIN_COMMAND\\\"\\nSMD=$(eval$FAULT_DOMAIN_COMMAND)\\necho\\\"SMDvalue:$SMD\\\"\\n\\nif[[-n\\\"$FAULT_DOMAIN_COMMAND\\\"]];then\\necho\\\"create/hbase-operator$SMD\\\"|$HBASE_HOME/bin/hbasezkcli2>/dev/null||true\\necho\\\"create/hbase-operator/$HOSTNAME$SMD\\\"|$HBASE_HOME/bin/hbasezkcli2>/dev/null\\necho\\\"\\\"\\necho\\\"Completed\\\"\\nfi\\n\"],\"cpuLimit\":\"0.1\",\"cpuRequest\":\"0.1\",\"isBootstrap\":false,\"memoryLimit\":\"386Mi\",\"memoryRequest\":\"386Mi\",\"name\":\"init-faultdomain\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/etc/nodeinfo\",\"name\":\"nodeinfo\",\"readOnly\":true}]},{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-x-m\\n\\nexportHADOOP_LOG_DIR=/var/log/abc/yak/hadoop\\nexportHADOOP_CONF_DIR=/etc/hadoop\\nexportHADOOP_HOME=/opt/hadoop\\n\\n$HADOOP_HOME/bin/hdfsdfsadmin-refreshNodes||true\\n\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"isBootstrap\":false,\"memoryLimit\":\"256Mi\",\"memoryRequest\":\"256Mi\",\"name\":\"init-refreshnn\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}},{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-exmopipefail\\n\\n#TODORemove\\n#ip=$(ifconfig|grep\\\"inet\\\"|grep-Fv127.0.0.1|awk'{print$2}'|head-1)\\nhostn=$(hostname-f)\\nzone=$FCP_ZONE\\nvpc=$FCP_VPC\\n\\n\\nrun_command(){\\necho\\\"Operation:${1}\\\"\\ncmd_output=$(eval${3})\\nempty_ok=${4}\\necho\\\"\\\"\\n\\nif[[$empty_ok!=\\\"true\\\"]];then\\niftest-z\\\"$cmd_output\\\"\\nthen\\necho\\\"Failedtodooperation:${2}.Exitting...\\\"\\nexit2\\nfi\\nfi\\n}\\n\\nindex=$(($((0x$(sha1sum<<<\\\"$hostn\\\"|cut-c1-2)))%2))\\netc_hosts_key=\\\"etc-hosts-$index\\\"\\n\\nforVARIABLEin12345\\ndo\\nrun_command\\\"Getsmdmappingbucketdata\\\"\\\"SMDbucketdata\\\"\\\"curl-s-XGET\\\\\\\"http://${config_endpoint}/v1/buckets/${topology_bucket}\\\\\\\"\\\"\\nbucket_data=$cmd_output\\n\\nrun_command\\\"Parseversionfrombucketdata\\\"\\\"parseversion\\\"\\\"echo'$bucket_data'|python3-c\\\\\\\"importsys,json;print(json.dumps(json.load(sys.stdin)['metadata']['version']))\\\\\\\"\\\"\\nexisting_version=$cmd_output\\n\\nrun_command\\\"Parsemappingdatafrombucketdata\\\"\\\"parsemapping\\\"\\\"echo'$bucket_data'|python3-c\\\\\\\"importsys,json;print(json.dumps(json.load(sys.stdin)['keys']['$etc_hosts_key']))\\\\\\\"\\\"\\netc_hosts=$cmd_output\\n\\nif[[\\\"$etc_hosts\\\"==*\\\\\\\"\\\"$hostn\\\"\\\\\\\"*]];then\\nrun_command\\\"Updateexistingmappinginbucketdata\\\"\\\"updatemapping\\\"\\\"echo'$etc_hosts'|python3-c\\\\\\\"importsys,json;value=json.load(sys.stdin);[(item)foriteminvalueif'$hostn'initem][0]['$hostn']='$ip';print(json.dumps(value,sort_keys=True))\\\\\\\"\\\"\\nelse\\nrun_command\\\"Addmappingtobucketdata\\\"\\\"addmapping\\\"\\\"echo'$etc_hosts'|python3-c\\\\\\\"importsys,json;value=json.load(sys.stdin);item={u'$hostn':u'$ip'};value.append(item);print(json.dumps(value,sort_keys=True))\\\\\\\"\\\"\\nfi\\nnew_etc_hosts=$cmd_output\\ndone\\n\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"isBootstrap\":false,\"memoryLimit\":\"256Mi\",\"memoryRequest\":\"256Mi\",\"name\":\"publish-myip\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}}],\"isPodServiceRequired\":false,\"name\":\"yak-tenant-test-1-dn\",\"podManagementPolicy\":\"Parallel\",\"shareProcessNamespace\":true,\"size\":5,\"terminateGracePeriod\":120,\"volumeClaims\":[{\"name\":\"data\",\"storageClassName\":\"test-strg\",\"storageSize\":\"184Gi\"}],\"volumes\":[{\"name\":\"lifecycle\",\"volumeSource\":\"EmptyDir\"},{\"name\":\"hadooprun\",\"volumeSource\":\"EmptyDir\"},{\"name\":\"nodeinfo\",\"path\":\"/etc/nodeinfo\",\"volumeSource\":\"HostPath\"}]},\"serviceLabels\":{\"hbase-operator.cfg-statefulset-update/enable\":\"config-only\"},\"fsgroup\":1011}}"
	err := json.Unmarshal([]byte(hbaseTenantJson), tenant)
	if err != nil {
		fmt.Println(err)
	}
	return tenant
}

func doTenantTestSetup() (*K8sMockClient, *HbaseTenantReconciler, context.Context, ctrl.Request) {
	k8sMockClient, reconciler := getMockClientAndTenantReconciler()
	ctx := context.TODO()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      tenantName1,
			Namespace: tenantNamespace1,
		},
	}
	return k8sMockClient, reconciler, ctx, req
}

func getInvalidConfigHbasetenant() *kvstorev1.HbaseTenant {
	out, err := os.ReadFile("../testdata/test_invalid_hbase_tenant.json")
	if err != nil {
		fmt.Println(err)
	}
	tenant := &kvstorev1.HbaseTenant{}
	unmarshalErr := json.Unmarshal(out, tenant)
	if unmarshalErr != nil {
		fmt.Println(unmarshalErr)
	}
	return tenant
}
