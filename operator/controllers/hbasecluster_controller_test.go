package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strconv"
	"testing"
	"time"

	kvstorev1 "github.com/flipkart-incubator/hbase-k8s-operator/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace    = "test-namespace"
	testCluster      = "test-cluster"
	tenantNamespace1 = "yak-tenant-test-1"
	tenantNamespace2 = "yak-tenant-test-2"
)

type MockClient struct {
	mock.Mock
	client.Client
}

func (m *MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	args := m.Called(ctx, key, obj)
	return args.Error(0)
}

func (m *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func TestHbaseClusterReconciler_ResNotFound(t *testing.T) {
	mockClient, reconciler := getMockClientAndReconciler()
	ctx := context.TODO()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testCluster,
			Namespace: testNamespace,
		},
	}

	mockClient.On("Get", ctx, req.NamespacedName, mock.Anything).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	mockClient.AssertExpectations(t)
}

func TestHbaseClusterReconciler_ErrorGettingRes(t *testing.T) {
	mockClient, reconciler := getMockClientAndReconciler()
	ctx := context.TODO()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testCluster,
			Namespace: testNamespace,
		},
	}
	mockClient.On("Get", ctx, req.NamespacedName, mock.Anything).Return(assert.AnError)

	result, err := reconciler.Reconcile(ctx, req)
	assert.Error(t, err)
	assert.Equal(t, ctrl.Result{RequeueAfter: time.Second * 5}, result)

	mockClient.AssertExpectations(t)

}

func TestHbaseClusterReconciler_SuccessfulReconciliation_ObjectsNotFound(t *testing.T) {
	//mock hbase cluster object
	hbasecluster := getMockHbaseCluster()

	mockClient, reconciler := getMockClientAndReconciler()
	ctx := context.TODO()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testCluster,
			Namespace: testNamespace,
		},
	}

	deployments := []kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Journalnode, hbasecluster.Spec.Deployments.Namenode, hbasecluster.Spec.Deployments.Datanode, hbasecluster.Spec.Deployments.Hmaster}
	if hbasecluster.Spec.Deployments.Zookeeper.Size != 0 {
		deployments = append([]kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Zookeeper}, deployments...)
	}

	mockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseCluster{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseCluster)
			*arg = *hbasecluster
		}).
		Return(nil)

	mockSvc := buildService(hbasecluster.Name, hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.ServiceLabels, hbasecluster.Spec.ServiceSelectorLabels, deployments, true)
	assert.Equal(t, testCluster, mockSvc.Name)
	assert.Equal(t, testCluster, mockSvc.Spec.Selector["hbasecluster_cr"])

	mockClient.On("Get", ctx, req.NamespacedName, &corev1.Service{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	ctrl.SetControllerReference(hbasecluster, mockSvc, reconciler.Scheme)
	mockClient.On("Create", ctx, mockSvc, []client.CreateOption(nil)).Return(nil)

	for _, namespace := range []string{tenantNamespace1, tenantNamespace2, testNamespace} {
		mockCfgHb := buildConfigMap(hbasecluster.Spec.Configuration.HbaseConfigName, hbasecluster.Name, namespace, hbasecluster.Spec.Configuration.HbaseConfig, hbasecluster.Spec.Configuration.HbaseTenantConfig, reconciler.Log)
		ctrl.SetControllerReference(hbasecluster, mockCfgHb, reconciler.Scheme)
		mockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHb.Name, Namespace: mockCfgHb.Namespace}, &corev1.ConfigMap{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
		mockClient.On("Create", ctx, mockCfgHb, []client.CreateOption(nil)).Return(nil)

		mockCfgHd := buildConfigMap(hbasecluster.Spec.Configuration.HadoopConfigName, hbasecluster.Name, namespace, hbasecluster.Spec.Configuration.HadoopConfig, hbasecluster.Spec.Configuration.HadoopTenantConfig, reconciler.Log)
		ctrl.SetControllerReference(hbasecluster, mockCfgHd, reconciler.Scheme)
		mockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHd.Name, Namespace: mockCfgHd.Namespace}, &corev1.ConfigMap{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
		mockClient.On("Create", ctx, mockCfgHd, []client.CreateOption(nil)).Return(nil)
	}

	mockClient.On("Get", ctx, types.NamespacedName{Name: hbasecluster.Spec.Deployments.Datanode.Name, Namespace: hbasecluster.Namespace}, &appsv1.StatefulSet{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))

	// only one component zk is mocked here as sts reconcile method requeues the call after sts create
	// other component's reconcile does not happen unless for previous one it ensures to have ready replica same as desired.
	if hbasecluster.Spec.Deployments.Zookeeper.IsPodServiceRequired {
		var name string
		var index int32 = 0
		for index < hbasecluster.Spec.Deployments.Zookeeper.Size {
			name = hbasecluster.Spec.Deployments.Zookeeper.Name + "-" + strconv.Itoa(int(index))
			mockJNPodSvc := buildService(name, hbasecluster.Name, hbasecluster.Namespace, nil, nil, []kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Zookeeper}, false)
			ctrl.SetControllerReference(hbasecluster, mockJNPodSvc, reconciler.Scheme)
			fmt.Println("mocking GET svc with parameters: ", name, hbasecluster.Namespace)
			mockClient.On("Get", ctx, types.NamespacedName{Name: name, Namespace: hbasecluster.Namespace}, &corev1.Service{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
			fmt.Println("mocking CREATE svc with parameters: ", mockJNPodSvc)
			mockClient.On("Create", ctx, mockJNPodSvc, []client.CreateOption(nil)).Return(nil)
			index += 1
		}
	}

	mockStsZK := buildStatefulSet(hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.BaseImage,
		hbasecluster.Spec.IsBootstrap, hbasecluster.Spec.Configuration, "",
		hbasecluster.Spec.FSGroup, hbasecluster.Spec.Deployments.Zookeeper, reconciler.Log, true)
	ctrl.SetControllerReference(hbasecluster, mockStsZK, reconciler.Scheme)
	mockClient.On("Get", ctx, types.NamespacedName{Name: hbasecluster.Spec.Deployments.Zookeeper.Name, Namespace: hbasecluster.Namespace}, &appsv1.StatefulSet{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	mockClient.On("Create", ctx, mockStsZK, []client.CreateOption(nil)).Return(nil)

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, result)

	// AssertExpectations asserts that everything specified with On and Return was in fact called as expected.
	mockClient.AssertExpectations(t)
}

func TestHbaseClusterReconciler_SuccessfulReconciliation_AllObjectsFound(t *testing.T) {
	//mock hbase cluster object
	hbasecluster := getMockHbaseCluster()

	mockClient, reconciler := getMockClientAndReconciler()
	ctx := context.TODO()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testCluster,
			Namespace: testNamespace,
		},
	}

	deployments := []kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Journalnode, hbasecluster.Spec.Deployments.Namenode, hbasecluster.Spec.Deployments.Datanode, hbasecluster.Spec.Deployments.Hmaster}
	if hbasecluster.Spec.Deployments.Zookeeper.Size != 0 {
		deployments = append([]kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Zookeeper}, deployments...)
	}

	mockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseCluster{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseCluster)
			*arg = *hbasecluster
		}).
		Return(nil)

	mockSvc := buildService(hbasecluster.Name, hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.ServiceLabels, hbasecluster.Spec.ServiceSelectorLabels, deployments, true)
	assert.Equal(t, testCluster, mockSvc.Name)
	assert.Equal(t, testCluster, mockSvc.Spec.Selector["hbasecluster_cr"])

	ctrl.SetControllerReference(hbasecluster, mockSvc, reconciler.Scheme)
	mockClient.On("Get", ctx, req.NamespacedName, &corev1.Service{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.Service)
			*arg = *mockSvc
		}).
		Return(nil)
	mockClient.On("Update", ctx, mockSvc, []client.UpdateOption(nil)).Return(nil)

	for _, namespace := range []string{tenantNamespace1, tenantNamespace2, testNamespace} {
		mockCfgHb := buildConfigMap(hbasecluster.Spec.Configuration.HbaseConfigName, hbasecluster.Name, namespace, hbasecluster.Spec.Configuration.HbaseConfig, hbasecluster.Spec.Configuration.HbaseTenantConfig, reconciler.Log)
		ctrl.SetControllerReference(hbasecluster, mockCfgHb, reconciler.Scheme)
		mockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHb.Name, Namespace: mockCfgHb.Namespace}, &corev1.ConfigMap{}).
			Run(func(args mock.Arguments) {
				arg := args.Get(2).(*corev1.ConfigMap)
				*arg = *mockCfgHb
			}).
			Return(nil)
		mockClient.On("Update", ctx, mockCfgHb, []client.UpdateOption(nil)).Return(nil)

		mockCfgHd := buildConfigMap(hbasecluster.Spec.Configuration.HadoopConfigName, hbasecluster.Name, namespace, hbasecluster.Spec.Configuration.HadoopConfig, hbasecluster.Spec.Configuration.HadoopTenantConfig, reconciler.Log)
		ctrl.SetControllerReference(hbasecluster, mockCfgHd, reconciler.Scheme)
		mockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHd.Name, Namespace: mockCfgHd.Namespace}, &corev1.ConfigMap{}).
			Run(func(args mock.Arguments) {
				arg := args.Get(2).(*corev1.ConfigMap)
				*arg = *mockCfgHd
			}).
			Return(nil)
		mockClient.On("Update", ctx, mockCfgHd, []client.UpdateOption(nil)).Return(nil)
	}

	mockClient.On("Get", ctx, types.NamespacedName{Name: hbasecluster.Spec.Deployments.Datanode.Name, Namespace: hbasecluster.Namespace}, &appsv1.StatefulSet{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))

	// only one component zk is mocked here as sts reconcile method requeues the call after sts create
	// other component's reconcile does not happen unless for previous one it ensures to have ready replica same as desired.
	if hbasecluster.Spec.Deployments.Zookeeper.IsPodServiceRequired {
		var name string
		var index int32 = 0
		for index < hbasecluster.Spec.Deployments.Zookeeper.Size {
			name = hbasecluster.Spec.Deployments.Zookeeper.Name + "-" + strconv.Itoa(int(index))
			mockZKPodSvc := buildService(name, hbasecluster.Name, hbasecluster.Namespace, nil, nil, []kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Zookeeper}, false)
			ctrl.SetControllerReference(hbasecluster, mockZKPodSvc, reconciler.Scheme)
			//fmt.Println("mocking GET svc with parameters: ", name, hbasecluster.Namespace)
			mockClient.On("Get", ctx, types.NamespacedName{Name: name, Namespace: hbasecluster.Namespace}, &corev1.Service{}).
				Run(func(args mock.Arguments) {
					arg := args.Get(2).(*corev1.Service)
					*arg = *mockZKPodSvc
				}).
				Return(nil)
			//fmt.Println("mocking CREATE svc with parameters: ", mockJNPodSvc)
			mockClient.On("Update", ctx, mockZKPodSvc, []client.UpdateOption(nil)).Return(nil)
			index += 1
		}
	}

	mockStsZK := buildStatefulSet(hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.BaseImage,
		hbasecluster.Spec.IsBootstrap, hbasecluster.Spec.Configuration, "",
		hbasecluster.Spec.FSGroup, hbasecluster.Spec.Deployments.Zookeeper, reconciler.Log, true)
	ctrl.SetControllerReference(hbasecluster, mockStsZK, reconciler.Scheme)
	mockClient.On("Get", ctx, types.NamespacedName{Name: hbasecluster.Spec.Deployments.Zookeeper.Name, Namespace: hbasecluster.Namespace}, &appsv1.StatefulSet{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*appsv1.StatefulSet)
			*arg = *mockStsZK
		}).
		Return(nil)
	mockClient.On("Update", ctx, mockStsZK, []client.UpdateOption(nil)).Return(nil)

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: time.Second * 20}, result)

	// AssertExpectations asserts that everything specified with On and Return was in fact called as expected.
	mockClient.AssertExpectations(t)
}

func TestHbaseClusterReconciler_SuccessfulReconciliation_AllObjectsFoundRestFlow(t *testing.T) {
	TestHbaseClusterReconciler_SuccessfulReconciliation_AllObjectsFound(t)
	//mock hbase cluster object
	hbasecluster := getMockHbaseCluster()

	mockClient, reconciler := getMockClientAndReconciler()
	ctx := context.TODO()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testCluster,
			Namespace: testNamespace,
		},
	}

	deployments := []kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Journalnode, hbasecluster.Spec.Deployments.Namenode, hbasecluster.Spec.Deployments.Datanode, hbasecluster.Spec.Deployments.Hmaster}
	if hbasecluster.Spec.Deployments.Zookeeper.Size != 0 {
		deployments = append([]kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Zookeeper}, deployments...)
	}

	mockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseCluster{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseCluster)
			*arg = *hbasecluster
		}).
		Return(nil)

	mockSvc := buildService(hbasecluster.Name, hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.ServiceLabels, hbasecluster.Spec.ServiceSelectorLabels, deployments, true)
	assert.Equal(t, testCluster, mockSvc.Name)
	assert.Equal(t, testCluster, mockSvc.Spec.Selector["hbasecluster_cr"])

	ctrl.SetControllerReference(hbasecluster, mockSvc, reconciler.Scheme)
	mockClient.On("Get", ctx, req.NamespacedName, &corev1.Service{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.Service)
			*arg = *mockSvc
		}).
		Return(nil)

	for _, namespace := range []string{tenantNamespace1, tenantNamespace2, testNamespace} {
		mockCfgHb := buildConfigMap(hbasecluster.Spec.Configuration.HbaseConfigName, hbasecluster.Name, namespace, hbasecluster.Spec.Configuration.HbaseConfig, hbasecluster.Spec.Configuration.HbaseTenantConfig, reconciler.Log)
		ctrl.SetControllerReference(hbasecluster, mockCfgHb, reconciler.Scheme)
		mockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHb.Name, Namespace: mockCfgHb.Namespace}, &corev1.ConfigMap{}).
			Run(func(args mock.Arguments) {
				arg := args.Get(2).(*corev1.ConfigMap)
				*arg = *mockCfgHb
			}).
			Return(nil)

		mockCfgHd := buildConfigMap(hbasecluster.Spec.Configuration.HadoopConfigName, hbasecluster.Name, namespace, hbasecluster.Spec.Configuration.HadoopConfig, hbasecluster.Spec.Configuration.HadoopTenantConfig, reconciler.Log)
		ctrl.SetControllerReference(hbasecluster, mockCfgHd, reconciler.Scheme)
		mockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHd.Name, Namespace: mockCfgHd.Namespace}, &corev1.ConfigMap{}).
			Run(func(args mock.Arguments) {
				arg := args.Get(2).(*corev1.ConfigMap)
				*arg = *mockCfgHd
			}).
			Return(nil)
	}

	mockClient.On("Get", ctx, types.NamespacedName{Name: hbasecluster.Spec.Deployments.Datanode.Name, Namespace: hbasecluster.Namespace}, &appsv1.StatefulSet{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))

	// only one component zk is mocked here as sts reconcile method requeues the call after sts create
	// other component's reconcile does not happen unless for previous one it ensures to have ready replica same as desired.
	if hbasecluster.Spec.Deployments.Zookeeper.IsPodServiceRequired {
		var name string
		var index int32 = 0
		for index < hbasecluster.Spec.Deployments.Zookeeper.Size {
			name = hbasecluster.Spec.Deployments.Zookeeper.Name + "-" + strconv.Itoa(int(index))
			mockZKPodSvc := buildService(name, hbasecluster.Name, hbasecluster.Namespace, nil, nil, []kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Zookeeper}, false)
			ctrl.SetControllerReference(hbasecluster, mockZKPodSvc, reconciler.Scheme)
			//fmt.Println("mocking GET svc with parameters: ", name, hbasecluster.Namespace)
			mockClient.On("Get", ctx, types.NamespacedName{Name: name, Namespace: hbasecluster.Namespace}, &corev1.Service{}).
				Run(func(args mock.Arguments) {
					arg := args.Get(2).(*corev1.Service)
					*arg = *mockZKPodSvc
				}).
				Return(nil)
			//fmt.Println("mocking CREATE svc with parameters: ", mockJNPodSvc)
			index += 1
		}
	}

	mockStsZK := buildStatefulSet(hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.BaseImage,
		hbasecluster.Spec.IsBootstrap, hbasecluster.Spec.Configuration, "",
		hbasecluster.Spec.FSGroup, hbasecluster.Spec.Deployments.Zookeeper, reconciler.Log, true)
	ctrl.SetControllerReference(hbasecluster, mockStsZK, reconciler.Scheme)
	mockStsZK.Status.ReadyReplicas = hbasecluster.Spec.Deployments.Zookeeper.Size
	mockClient.On("Get", ctx, types.NamespacedName{Name: hbasecluster.Spec.Deployments.Zookeeper.Name, Namespace: hbasecluster.Namespace}, &appsv1.StatefulSet{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*appsv1.StatefulSet)
			*arg = *mockStsZK
		}).
		Return(nil)

	mockPdbZk := buildPodDisruptionBudget(hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.Deployments.Zookeeper, reconciler.Log)
	ctrl.SetControllerReference(hbasecluster, mockPdbZk, reconciler.Scheme)
	mockClient.On("Get", ctx, types.NamespacedName{Name: mockPdbZk.Name, Namespace: mockPdbZk.Namespace}, &policyv1.PodDisruptionBudget{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	mockClient.On("Create", ctx, mockPdbZk, []client.CreateOption(nil)).Return(nil)

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, result)

	// AssertExpectations asserts that everything specified with On and Return was in fact called as expected.
	mockClient.AssertExpectations(t)
}

func getMockClientAndReconciler() (*MockClient, *HbaseClusterReconciler) {
	mockClient := new(MockClient)
	scheme := runtime.NewScheme()
	_ = kvstorev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	reconciler := &HbaseClusterReconciler{
		Client: mockClient,
		Log:    ctrl.Log.WithName("controllers").WithName("HbaseCluster"),
		Scheme: scheme,
	}
	return mockClient, reconciler
}

func getMockHbaseCluster() *kvstorev1.HbaseCluster {
	cluster := &kvstorev1.HbaseCluster{}
	hbaseclusterJson := "{\"apiVersion\":\"kvstore.flipkart.com/v1\",\"kind\":\"HbaseCluster\",\"metadata\":{\"name\":\"test-cluster\",\"namespace\":\"test-namespace\",\"resourceVersion\":\"18282894059\",\"uid\":\"0c7eec61-be8a-4c90-819d-d4568e3fca65\"},\"spec\":{\"baseImage\":\"edge.fkinternal.com/indradhanush/yak-base:2.5.3-08-rc7\",\"configuration\":{\"hadoopConfig\":{\"core-site.xml\":\"<?xmlversion=\\\"1.0\\\"?>\\n<?xml-stylesheettype=\\\"text/xsl\\\"href=\\\"configuration.xsl\\\"?>\\n<!--Generatedbyconfdon2021-03-0911:46:01.973761409+0530ISTm=+0.012850605-->\\n<configuration>\\n</configuration>\\n\",\"dfs.exclude\":\"\",\"dfs.include\":\"\",\"hadoop-env.sh\":\"exportHADOOP_CONF_DIR=/etc/hadoop\\nexportHADOOP_PID_DIR=/var/run/hadoop\\nforfin$HADOOP_HOME/contrib/capacity-scheduler/*.jar;do\\nif[\\\"$HADOOP_CLASSPATH\\\"];then\\nexportHADOOP_CLASSPATH=$HADOOP_CLASSPATH:$f\\nelse\\nexportHADOOP_CLASSPATH=$f\\nfi\\ndone\\nexportHADOOP_OPTS=\\\"$HADOOP_OPTs-Djava.net.preferIPv4Stack=true-Dnetworkaddress.cache.ttl=60-Dsun.net.inetaddr.ttl=60-XX:+UseZGC\\\"\\nexportHDFS_NAMENODE_OPTS=\\\"-Xms4096m-Xmx4096m-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10102-Dcom.sun.management.jmxremote.ssl=false-Dhadoop.security.logger=${HADOOP_SECURITY_LOGGER:-INFO,RFAS}-Dhdfs.audit.logger=${HDFS_AUDIT_LOGGER:-INFO,NullAppender}\\\"\\nexportHDFS_DATANODE_OPTS=\\\"-Xms4096m-Xmx4096m-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10101-Dcom.sun.management.jmxremote.ssl=false-Dhadoop.security.logger=ERROR,RFAS\\\"\\nexportHDFS_JOURNALNODE_OPTS=\\\"-Xms512m-Xmx512m-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10106-Dcom.sun.management.jmxremote.ssl=false\\\"\\nexportHDFS_ZKFC_OPTS=\\\"-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10107-Dcom.sun.management.jmxremote.ssl=false\\\"\\nexportHADOOP_SECONDARYNAMENODE_OPTS=\\\"-Xms4096m-Xmx4096m-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10102-Dcom.sun.management.jmxremote.ssl=false-XX:+UnlockCommercialFeatures-XX:+FlightRecorder-Dhadoop.security.logger=${HADOOP_SECURITY_LOGGER:-INFO,RFAS}-Dhdfs.audit.logger=${HDFS_AUDIT_LOGGER:-INFO,NullAppender}\\\"\\nexportHADOOP_NFS3_OPTS=\\\"$HADOOP_NFS3_OPTS\\\"\\nexportHADOOP_PORTMAP_OPTS=\\\"-Xmx512m$HADOOP_PORTMAP_OPTS\\\"\\nexportHADOOP_CLIENT_OPTS=\\\"-Xmx512m$HADOOP_CLIENT_OPTS\\\"\\nexportHADOOP_SECURE_DN_USER=${HADOOP_SECURE_DN_USER}\\nexportHADOOP_SECURE_LOG_DIR=${HADOOP_LOG_DIR}/${HADOOP_HDFS_USER}\\nexportHADOOP_PID_DIR=${HADOOP_PID_DIR}\\nexportHADOOP_SECURE_PID_DIR=${HADOOP_PID_DIR}\\nexportHADOOP_IDENT_STRING=$USER\\n\",\"hdfs-site.xml\":\"<?xmlversion=\\\"1.0\\\"?>\\n<?xml-stylesheettype=\\\"text/xsl\\\"href=\\\"configuration.xsl\\\"?>\\n<!--Generatedbyconfdon2021-03-0911:46:01.976938018+0530ISTm=+0.016027218-->\\n<configuration>\\n<property>\\n<name>dfs.replication</name>\\n<value>3</value>\\n</property>\\n\\n<property>\\n<name>dfs.replication.max</name>\\n<value>3</value>\\n</property>\\n</configuration>\\n\"},\"hadoopConfigMountPath\":\"/etc/hadoop\",\"hadoopConfigName\":\"hadoop-config\",\"hadoopTenantConfig\":[{\"hadoop-env.sh\":\"exportHADOOP_CONF_DIR=/etc/hadoop\\nexportHADOOP_PID_DIR=/var/run/hadoop\\nforfin$HADOOP_HOME/contrib/capacity-scheduler/*.jar;do\\nif[\\\"$HADOOP_CLASSPATH\\\"];then\\nexportHADOOP_CLASSPATH=$HADOOP_CLASSPATH:$f\\nelse\\nexportHADOOP_CLASSPATH=$f\\nfi\\ndone\\nexportHADOOP_OPTS=\\\"$HADOOP_OPTs-Djava.net.preferIPv4Stack=true-Dnetworkaddress.cache.ttl=60-Dsun.net.inetaddr.ttl=60-XX:+UseZGC\\\"\\nexportHDFS_NAMENODE_OPTS=\\\"-Xms4096m-Xmx4096m-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10102-Dcom.sun.management.jmxremote.ssl=false-Dhadoop.security.logger=${HADOOP_SECURITY_LOGGER:-INFO,RFAS}-Dhdfs.audit.logger=${HDFS_AUDIT_LOGGER:-INFO,NullAppender}\\\"\\nexportHDFS_DATANODE_OPTS=\\\"-Xms2048m-Xmx2048m-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10101-Dcom.sun.management.jmxremote.ssl=false-Dhadoop.security.logger=ERROR,RFAS\\\"\\nexportHDFS_JOURNALNODE_OPTS=\\\"-Xms512m-Xmx512m-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10106-Dcom.sun.management.jmxremote.ssl=false\\\"\\nexportHDFS_ZKFC_OPTS=\\\"-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10107-Dcom.sun.management.jmxremote.ssl=false\\\"\\nexportHADOOP_SECONDARYNAMENODE_OPTS=\\\"-Xms4096m-Xmx4096m-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.authenticate=false-Dcom.sun.management.jmxremote.port=10102-Dcom.sun.management.jmxremote.ssl=false-XX:+UnlockCommercialFeatures-XX:+FlightRecorder-Dhadoop.security.logger=${HADOOP_SECURITY_LOGGER:-INFO,RFAS}-Dhdfs.audit.logger=${HDFS_AUDIT_LOGGER:-INFO,NullAppender}\\\"\\nexportHADOOP_NFS3_OPTS=\\\"$HADOOP_NFS3_OPTS\\\"\\nexportHADOOP_PORTMAP_OPTS=\\\"-Xmx512m$HADOOP_PORTMAP_OPTS\\\"\\nexportHADOOP_CLIENT_OPTS=\\\"-Xmx512m$HADOOP_CLIENT_OPTS\\\"\\nexportHADOOP_SECURE_DN_USER=${HADOOP_SECURE_DN_USER}\\nexportHADOOP_SECURE_LOG_DIR=${HADOOP_LOG_DIR}/${HADOOP_HDFS_USER}\\nexportHADOOP_PID_DIR=${HADOOP_PID_DIR}\\nexportHADOOP_SECURE_PID_DIR=${HADOOP_PID_DIR}\\nexportHADOOP_IDENT_STRING=$USER\\n\",\"namespace\":\"yak-tenant-oms-snpt-prod\"}],\"hbaseConfig\":{\"hbase-site.xml\":\"<?xmlversion=\\\"1.0\\\"?>\\n<?xml-stylesheettype=\\\"text/xsl\\\"href=\\\"configuration.xsl\\\"?>\\n<!--Generatedbyconfdon2021-03-0911:46:01.975303151+0530ISTm=+0.014392356-->\\n<configuration>\\n<property>\\n<name>cluster.replication.sink.manager</name>\\n<value>org.apache.hadoop.hbase.rsgroup.replication.RSGroupAwareReplicationSinkManager</value>\\n</property>\\n</configuration>\\n\"},\"hbaseConfigMountPath\":\"/etc/hbase\",\"hbaseConfigName\":\"hbase-config\",\"hbaseTenantConfig\":[{\"hbase-env.sh\":\"exportHBASE_OPTS=\\\"-XX:+UseZGC-Dsun.net.inetaddr.ttl=60-Djava.net.preferIPv4Stack=true-Dnetworkaddress.cache.ttl=60\\\"\\nexportSERVER_GC_OPTS=\\\"-verbose:gc-XX:+PrintGCDetails-Xloggc:<FILE-PATH>\\\"\\nexportHBASE_JMX_BASE=\\\"-Dnetworkaddress.cache.ttl=60-Dnetworkaddress.cache.negative.ttl=0-Dcom.sun.management.jmxremote-Dcom.sun.management.jmxremote.ssl=false-Dcom.sun.management.jmxremote.authenticate=false\\\"\\nexportHBASE_MASTER_OPTS=\\\"$HBASE_MASTER_OPTS$HBASE_JMX_BASE-Dcom.sun.management.jmxremote.port=10103-Xms8g-Xmx8g\\\"\\nexportHBASE_REGIONSERVER_OPTS=\\\"$HBASE_REGIONSERVER_OPTS$HBASE_JMX_BASE-Dcom.sun.management.jmxremote.port=10104-Xms4g-Xmx4g-XX:MaxDirectMemorySize=2g-Djute.maxbuffer=536870912\\\"\\nexportHBASE_ZOOKEEPER_OPTS=\\\"$HBASE_ZOOKEEPER_OPTS$HBASE_JMX_BASE-Dcom.sun.management.jmxremote.port=10105-Xms2g-Xmx2g-Djute.maxbuffer=536870912\\\"\\nexportHBASE_PID_DIR=/var/run/hbase\\nexportHBASE_MANAGES_ZK=false\\nexportLD_LIBRARY_PATH=/opt/hadoop/lib/native\\n\",\"namespace\":\"2c10gConfig\"}]},\"deployments\":{\"datanode\":{\"annotations\":{\"fcp.k8s.mtl/cosmos-jmx\":\"enabled\",\"fcp.k8s.mtl/cosmos-statsd\":\"disabled\",\"fcp.k8s.mtl/cosmos-tail\":\"disabled\",\"fcp.k8s.mtl/mtl-config\":\"mtl-config-2\",\"fcp.k8s.mtl/mtl-config-map\":\"mtl-dn\",\"fcp.k8s/webhook-inject-fcp-dns\":\"No\"},\"containers\":[{\"args\":[\"/var/log/flipkart/yak/hadoop\",\"/etc/hadoop\",\"/opt/hadoop\"],\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-x-m\\n\\nexportHADOOP_LOG_DIR=$0\\nexportHADOOP_CONF_DIR=$1\\nexportHADOOP_HOME=$2\\nexportUSER=$(whoami)\\nexportHADOOP_LOG_FILE=$HADOOP_LOG_DIR/hadoop-$USER-datanode-$(hostname).log\\n\\nmkdir-p$HADOOP_LOG_DIR\\ntouch$HADOOP_LOG_FILE\\n\\nfunctionshutdown(){\\nwhile[[!-f\\\"/lifecycle/rs-terminated\\\"]];doecho\\\"Waitingforregionservertodie\\\";sleep2;done\\necho\\\"Stoppingdatanode\\\"\\nsleep10\\n$HADOOP_HOME/bin/hdfs--daemonstopdatanode\\n}\\n\\ntrapshutdownSIGTERM\\nexec$HADOOP_HOME/bin/hdfsdatanode2>&1|tee-a$HADOOP_LOG_FILE&\\nPID=$!\\n\\nDOMAIN_SOCKET=$($HADOOP_HOME/bin/hdfsgetconf-confKeydfs.domain.socket.path)\\nDOMAIN_SOCKET=$(echo$DOMAIN_SOCKET|sed-e's/_PORT/*/g')\\nwhile[!-e${DOMAIN_SOCKET}];dosleep1;done\\ntouch/lifecycle/dn-started\\n\\nwait$PID\\n\"],\"cpuLimit\":\"3\",\"cpuRequest\":\"3\",\"livenessProbe\":{\"initialDelay\":60,\"tcpPort\":9866},\"memoryLimit\":\"10Gi\",\"memoryRequest\":\"10Gi\",\"name\":\"datanode\",\"ports\":[{\"name\":\"datanode-0\",\"port\":9866}],\"readinessProbe\":{\"initialDelay\":60,\"tcpPort\":9866},\"securityContext\":{\"addSysPtrace\":true,\"runAsGroup\":1011,\"runAsUser\":1011},\"startupProbe\":{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\n\\nexportHADOOP_LOG_DIR=$0\\nexportHADOOP_CONF_DIR=$1\\nexportHADOOP_HOME=$2\\n\\nwhile:\\ndo\\nif[[$($HADOOP_HOME/bin/hdfsdfsadmin-report-live|grep\\\"$(hostname-f)\\\"|wc-l)==2]];then\\necho\\\"datanodeislistedasliveundernamenode.Exiting...\\\"\\nexit0\\nelse\\necho\\\"datanodeisstillnotlistedasliveundernamenode\\\"\\nexit1\\nfi\\ndone\\nexit1\\n\",\"/var/log/flipkart/yak/hadoop\",\"/etc/hadoop\",\"/opt/hadoop\"],\"failureThreshold\":10,\"initialDelay\":30,\"timeout\":60},\"volumeMounts\":[{\"mountPath\":\"/grid/1\",\"name\":\"data\",\"readOnly\":false},{\"mountPath\":\"/lifecycle\",\"name\":\"lifecycle\",\"readOnly\":false},{\"mountPath\":\"/var/run/hadoop\",\"name\":\"hadooprun\",\"readOnly\":false},{\"mountPath\":\"/etc/nodeinfo\",\"name\":\"nodeinfo\",\"readOnly\":true},{\"mountPath\":\"/var/log/flipkart/yak\",\"name\":\"app-log\",\"readOnly\":false}]},{\"args\":[\"/var/log/flipkart/yak/hbase\",\"/etc/hbase\",\"/opt/hbase\"],\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\nexportHBASE_LOG_DIR=$0\\nexportHBASE_CONF_DIR=$1\\nexportHBASE_HOME=$2\\nexportUSER=$(whoami)\\n\\nFAULT_DOMAIN_COMMAND=$3\\n\\nmkdir-p$HBASE_LOG_DIR\\ntouch$HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).log&&tail-F$HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).log&\\ntouch$HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).out&&tail-F$HBASE_LOG_DIR/hbase-$USER-regionserver-$(hostname).out&\\n\\nfunctionshutdown(){\\necho\\\"StoppingRegionserver\\\"\\nhost=`hostname-f`\\nexportHBASE_STOP_TIMEOUT=20\\necho\\\"swtichoffbalancer\\\"\\necho\\\"balance_switchfalse\\\"|$HBASE_HOME/bin/hbaseshell&>/tmp/null\\n$HBASE_HOME/bin/hbaseorg.apache.hadoop.hbase.rsgroup.util.RSGroupAwareRegionMover-m6-r$host-ounload\\nsleep5\\necho\\\"swtichonbalancer\\\"\\necho\\\"balance_switchtrue\\\"|$HBASE_HOME/bin/hbaseshell&>/tmp/null\\ntouch/lifecycle/rs-terminated\\necho\\\"stoppingservernow\\\"\\n$HBASE_HOME/bin/hbase-daemon.shstopregionserver\\n}\\n\\nwhiletrue;do\\nif[[-f\\\"/lifecycle/dn-started\\\"]];then\\necho\\\"Startingrs\\\"\\nsleep5\\nbreak\\nfi\\necho\\\"Waitingfordatanodetostart\\\"\\nsleep2\\ndone\\n\\ntrapshutdownSIGTERM\\nexec$HBASE_HOME/bin/hbase-daemon.shforeground_startregionserver&\\nwait\\n\"],\"cpuLimit\":\"10\",\"cpuRequest\":\"10\",\"livenessProbe\":{\"initialDelay\":60,\"tcpPort\":16020},\"memoryLimit\":\"35Gi\",\"memoryRequest\":\"35Gi\",\"name\":\"regionserver\",\"ports\":[{\"name\":\"regionserver-0\",\"port\":16020},{\"name\":\"regionserver-1\",\"port\":16030}],\"readinessProbe\":{\"initialDelay\":60,\"tcpPort\":16020},\"securityContext\":{\"addSysPtrace\":true,\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/grid/1\",\"name\":\"data\",\"readOnly\":false},{\"mountPath\":\"/lifecycle\",\"name\":\"lifecycle\",\"readOnly\":false},{\"mountPath\":\"/var/run/hadoop\",\"name\":\"hadooprun\",\"readOnly\":false},{\"mountPath\":\"/etc/nodeinfo\",\"name\":\"nodeinfo\",\"readOnly\":true},{\"mountPath\":\"/var/log/flipkart/yak\",\"name\":\"app-log\",\"readOnly\":false}]}],\"initContainers\":[{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\n\\ni=0\\nwhiletrue;do\\necho\\\"$iiteration\\\"\\ndig+short$(hostname-f)|grep-v-e'^$'\\nif[$?==0];then\\nsleep30#30secondsdefaultdnscaching\\necho\\\"Breaking...\\\"\\nbreak\\nfi\\ni=$((i+1))\\nsleep1\\ndone\\n\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"isBootstrap\":false,\"memoryLimit\":\"128Mi\",\"memoryRequest\":\"128Mi\",\"name\":\"init-dnslookup\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}},{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m-x\\n\\nexportHBASE_LOG_DIR=/var/log/flipkart/yak/hbase\\nexportHBASE_CONF_DIR=/etc/hbase\\nexportHBASE_HOME=/opt/hbase\\n\\n#Makeitoptional\\nFAULT_DOMAIN_COMMAND=\\\"cat/etc/nodeinfo|grep'smd'|sed's/smd=//'|sed's/\\\\\\\"//g'\\\"\\nHOSTNAME=$(hostname-f)\\n\\necho\\\"Runningcommandtogetfaultdomain:$FAULT_DOMAIN_COMMAND\\\"\\nSMD=$(eval$FAULT_DOMAIN_COMMAND)\\necho\\\"SMDvalue:$SMD\\\"\\n\\nif[[-n\\\"$FAULT_DOMAIN_COMMAND\\\"]];then\\necho\\\"create/hbase-operator$SMD\\\"|$HBASE_HOME/bin/hbasezkcli2>/dev/null||true\\necho\\\"create/hbase-operator/$HOSTNAME$SMD\\\"|$HBASE_HOME/bin/hbasezkcli2>/dev/null\\necho\\\"\\\"\\necho\\\"Completed\\\"\\nfi\\n\"],\"cpuLimit\":\"0.1\",\"cpuRequest\":\"0.1\",\"isBootstrap\":false,\"memoryLimit\":\"386Mi\",\"memoryRequest\":\"386Mi\",\"name\":\"init-faultdomain\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/etc/nodeinfo\",\"name\":\"nodeinfo\",\"readOnly\":true}]},{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-x-m\\n\\nexportHADOOP_LOG_DIR=/var/log/flipkart/yak/hadoop\\nexportHADOOP_CONF_DIR=/etc/hadoop\\nexportHADOOP_HOME=/opt/hadoop\\n\\n$HADOOP_HOME/bin/hdfsdfsadmin-refreshNodes||true\\n\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"isBootstrap\":false,\"memoryLimit\":\"256Mi\",\"memoryRequest\":\"256Mi\",\"name\":\"init-refreshnn\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}},{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-exmopipefail\\n\\ntopology_bucket=$FCP_APPID-k8s-hosts\\nauthn_client_id=prod-id-yak-ch-confsvc\\nauthn_secret=38XeZZirsXuqcyjxq6hPBW2aCJJ97r7JokK/XNEnV+ub8ZmK\\nip=$(hostname-i)\\n#TODORemove\\n#ip=$(ifconfig|grep\\\"inet\\\"|grep-Fv127.0.0.1|awk'{print$2}'|head-1)\\n\\nhostn=$(hostname-f)\\nzone=$FCP_ZONE\\nvpc=$FCP_VPC\\n\\nauthn_endpoint=https://service.authn-prod.fkcloud.in/\\nif[[$vpc==\\\"Fk-Preprod\\\"]]\\nthen\\nconfig_endpoint=10.24.2.28\\ntarget_client_id=http://10.24.2.28:80\\nelif[[$zone==\\\"in-hyderabad-1\\\"]]\\nthen\\nconfig_endpoint=10.24.0.32\\ntarget_client_id=http://10.24.0.32:80\\nelif[[$zone==\\\"in-chennai-1\\\"]]\\nthen\\nconfig_endpoint=10.47.0.101\\ntarget_client_id=http://10.47.0.101:80\\nelif[[$zone==\\\"in-chennai-2\\\"]]\\nthen\\nconfig_endpoint=10.83.47.156\\ntarget_client_id=cfg-api-calvin-ch\\nelif[[$zone==\\\"asia-south1\\\"]]\\nthen\\nconfig_endpoint=api.aso1.cfgsvc-prod.fkcloud.in\\ntarget_client_id=http://api.aso1.cfgsvc-prod.fkcloud.in:80\\nelse\\necho\\\"Invalidzone:$zone\\\"\\nexit1\\nfi\\n\\nrun_command(){\\necho\\\"Operation:${1}\\\"\\ncmd_output=$(eval${3})\\nempty_ok=${4}\\necho\\\"\\\"\\n\\nif[[$empty_ok!=\\\"true\\\"]];then\\niftest-z\\\"$cmd_output\\\"\\nthen\\necho\\\"Failedtodooperation:${2}.Exitting...\\\"\\nexit2\\nfi\\nfi\\n}\\n\\n#Usethiswhentherearehugenumberofentriesinsinglebucket\\nindex=$(($((0x$(sha1sum<<<\\\"$hostn\\\"|cut-c1-2)))%2))\\netc_hosts_key=\\\"etc-hosts-$index\\\"\\n\\netc_hosts_key=\\\"etc-hosts\\\"\\n\\nforVARIABLEin12345\\ndo\\nrun_command\\\"Getsmdmappingbucketdata\\\"\\\"SMDbucketdata\\\"\\\"curl-s-XGET\\\\\\\"http://${config_endpoint}/v1/buckets/${topology_bucket}\\\\\\\"\\\"\\nbucket_data=$cmd_output\\n\\nrun_command\\\"Parseversionfrombucketdata\\\"\\\"parseversion\\\"\\\"echo'$bucket_data'|python3-c\\\\\\\"importsys,json;print(json.dumps(json.load(sys.stdin)['metadata']['version']))\\\\\\\"\\\"\\nexisting_version=$cmd_output\\n\\nrun_command\\\"Parsemappingdatafrombucketdata\\\"\\\"parsemapping\\\"\\\"echo'$bucket_data'|python3-c\\\\\\\"importsys,json;print(json.dumps(json.load(sys.stdin)['keys']['$etc_hosts_key']))\\\\\\\"\\\"\\netc_hosts=$cmd_output\\n\\nif[[\\\"$etc_hosts\\\"==*\\\\\\\"\\\"$hostn\\\"\\\\\\\"*]];then\\nrun_command\\\"Updateexistingmappinginbucketdata\\\"\\\"updatemapping\\\"\\\"echo'$etc_hosts'|python3-c\\\\\\\"importsys,json;value=json.load(sys.stdin);[(item)foriteminvalueif'$hostn'initem][0]['$hostn']='$ip';print(json.dumps(value,sort_keys=True))\\\\\\\"\\\"\\nelse\\nrun_command\\\"Addmappingtobucketdata\\\"\\\"addmapping\\\"\\\"echo'$etc_hosts'|python3-c\\\\\\\"importsys,json;value=json.load(sys.stdin);item={u'$hostn':u'$ip'};value.append(item);print(json.dumps(value,sort_keys=True))\\\\\\\"\\\"\\nfi\\nnew_etc_hosts=$cmd_output\\n\\ndata='{'\\\\\\\"\\\"$etc_hosts_key\\\"\\\\\\\"':'\\\"$new_etc_hosts\\\"'}'\\nif[\\\"$new_etc_hosts\\\"=\\\"$etc_hosts\\\"];then\\necho\\\"NochangesinSMD.Notupdating\\\"\\nexit0\\nelse\\niftest-z\\\"$config_svc_token\\\";then\\nrun_command\\\"Generatingauthntokenfortalkingtoconfigbucket\\\"\\\"TokenConfigsvc\\\"\\\"curl-s-XPOST-F\\\\\\\"client_id=${authn_client_id}\\\\\\\"-F\\\\\\\"client_secret=${authn_secret}\\\\\\\"-F\\\\\\\"grant_type=client_credentials\\\\\\\"-F\\\\\\\"target_client_id=${target_client_id}\\\\\\\"${authn_endpoint}/oauth/token|sed's/,/\\\\n/g'|grep\\\\\\\"access_token\\\\\\\"|sed's/\\\\\\\"//g'|sed's/.*://g'\\\"\\nconfig_svc_token=$cmd_output\\nfi\\n\\necho\\\"ExistingMapping:$etc_hosts,NewMapping:$new_etc_hostsforexistingversion:$existing_version\\\"\\noutput=$(curl-XPOST-s\\\"http://${config_endpoint}/v1/buckets/${topology_bucket}/keys?message=updated_by_$ip\\\"-H\\\"X-Config-Bucket-Version:${existing_version}\\\"-H\\\"Content-Type:application/json\\\"--data-binary\\\"$data\\\"-H\\\"Authorization:Bearer${config_svc_token}\\\")\\nfinal_result=$?\\nfailed_message=\\\"Updatefailed\\\"\\nif[[$final_result-eq0&&\\\"$output\\\"==*\\\"$topology_bucket\\\"*&&\\\"$output\\\"==*\\\"version\\\"*&&\\\"$output\\\"==*\\\"created\\\"*&&\\\"$output\\\"==*\\\"lastUpdated\\\"*]];then\\necho$output\\nexit0\\nelse\\necho\\\"Failedtoupdateip.exiting..\\\"\\nexit1\\nfi\\nfi\\ndone\\n\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"isBootstrap\":false,\"memoryLimit\":\"256Mi\",\"memoryRequest\":\"256Mi\",\"name\":\"publish-myip\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}}],\"isPodServiceRequired\":false,\"labels\":{\"mcs.discovery.fcp.io/enable\":\"true\"},\"name\":\"test-cluster-dn\",\"podManagementPolicy\":\"Parallel\",\"shareProcessNamespace\":true,\"sidecarContainers\":[{\"cpuLimit\":\"100m\",\"cpuRequest\":\"100m\",\"image\":\"edge.fkinternal.com/indradhanush/fk-yak-filebeat:8.11.3-fk1\",\"memoryLimit\":\"128Mi\",\"memoryRequest\":\"128Mi\",\"name\":\"filebeat\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/var/log/flipkart/yak\",\"name\":\"app-log\",\"readOnly\":false},{\"mountPath\":\"/etc/filebeat\",\"name\":\"filebeat\",\"readOnly\":false}]}],\"size\":5,\"terminateGracePeriod\":120,\"volumeClaims\":[{\"name\":\"data\",\"storageClassName\":\"local-extreme-1a-ext4\",\"storageSize\":\"256Gi\"}],\"volumes\":[{\"name\":\"lifecycle\",\"volumeSource\":\"EmptyDir\"},{\"name\":\"hadooprun\",\"volumeSource\":\"EmptyDir\"},{\"name\":\"nodeinfo\",\"path\":\"/etc/nodeinfo\",\"volumeSource\":\"HostPath\"},{\"name\":\"app-log\",\"volumeSource\":\"EmptyDir\"},{\"configName\":\"filebeat\",\"name\":\"filebeat\",\"volumeSource\":\"ConfigMap\"}]},\"hmaster\":{\"annotations\":{\"fcp.k8s.mtl/cosmos-jmx\":\"enabled\",\"fcp.k8s.mtl/cosmos-statsd\":\"disabled\",\"fcp.k8s.mtl/cosmos-tail\":\"disabled\",\"fcp.k8s.mtl/mtl-config\":\"mtl-config-2\",\"fcp.k8s.mtl/mtl-config-map\":\"mtl-hmaster\",\"fcp.k8s/webhook-inject-fcp-dns\":\"No\"},\"containers\":[{\"args\":[\"/var/log/flipkart/yak/hbase\",\"/etc/hbase\",\"/opt/hbase\"],\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\nexportHBASE_LOG_DIR=$0\\nexportHBASE_CONF_DIR=$1\\nexportHBASE_HOME=$2\\nexportUSER=$(whoami)\\n\\nmkdir-p$HBASE_LOG_DIR\\ntouch$HBASE_LOG_DIR/hbase-$USER-master-$(hostname).log&&tail-F$HBASE_LOG_DIR/hbase-$USER-master-$(hostname).log&\\ntouch$HBASE_LOG_DIR/hbase-$USER-master-$(hostname).out&&tail-F$HBASE_LOG_DIR/hbase-$USER-master-$(hostname).out&\\n\\nfunctionshutdown(){\\necho\\\"StoppingHmaster\\\"\\n$HBASE_HOME/bin/hbase-daemon.shstopmaster\\n}\\n\\ntrapshutdownSIGTERM\\nexec$HBASE_HOME/bin/hbase-daemon.shforeground_startmaster&\\nwait\\n\"],\"cpuLimit\":\"6\",\"cpuRequest\":\"6\",\"livenessProbe\":{\"initialDelay\":10,\"tcpPort\":16000},\"memoryLimit\":\"40Gi\",\"memoryRequest\":\"40Gi\",\"name\":\"hmaster\",\"ports\":[{\"name\":\"hmaster-0\",\"port\":16000},{\"name\":\"hmaster-1\",\"port\":16010}],\"readinessProbe\":{\"initialDelay\":10,\"tcpPort\":16000},\"securityContext\":{\"addSysPtrace\":false,\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/opt/share\",\"name\":\"data\",\"readOnly\":false},{\"mountPath\":\"/var/log/flipkart/yak\",\"name\":\"app-log\",\"readOnly\":false},{\"mountPath\":\"/etc/nodeinfo\",\"name\":\"nodeinfo\",\"readOnly\":true}]}],\"initContainers\":[{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\n\\ni=0\\nwhiletrue;do\\necho\\\"$iiteration\\\"\\ndig+short$(hostname-f)|grep-v-e'^$'\\nif[$?==0];then\\nsleep30#30secondsdefaultdnscaching\\necho\\\"Breaking...\\\"\\nbreak\\nfi\\ni=$((i+1))\\nsleep1\\ndone\\n\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"isBootstrap\":false,\"memoryLimit\":\"128Mi\",\"memoryRequest\":\"128Mi\",\"name\":\"init-dnslookup\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}},{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-exmopipefail\\n\\ntopology_bucket=$FCP_APPID-k8s-hosts\\nauthn_client_id=prod-id-yak-ch-confsvc\\nauthn_secret=38XeZZirsXuqcyjxq6hPBW2aCJJ97r7JokK/XNEnV+ub8ZmK\\nip=$(hostname-i)\\n#TODORemove\\n#ip=$(ifconfig|grep\\\"inet\\\"|grep-Fv127.0.0.1|awk'{print$2}'|head-1)\\n\\nhostn=$(hostname-f)\\nzone=$FCP_ZONE\\nvpc=$FCP_VPC\\n\\nauthn_endpoint=https://service.authn-prod.fkcloud.in/\\nif[[$vpc==\\\"Fk-Preprod\\\"]]\\nthen\\nconfig_endpoint=10.24.2.28\\ntarget_client_id=http://10.24.2.28:80\\nelif[[$zone==\\\"in-hyderabad-1\\\"]]\\nthen\\nconfig_endpoint=10.24.0.32\\ntarget_client_id=http://10.24.0.32:80\\nelif[[$zone==\\\"in-chennai-1\\\"]]\\nthen\\nconfig_endpoint=10.47.0.101\\ntarget_client_id=http://10.47.0.101:80\\nelif[[$zone==\\\"in-chennai-2\\\"]]\\nthen\\nconfig_endpoint=10.83.47.156\\ntarget_client_id=cfg-api-calvin-ch\\nelif[[$zone==\\\"asia-south1\\\"]]\\nthen\\nconfig_endpoint=api.aso1.cfgsvc-prod.fkcloud.in\\ntarget_client_id=http://api.aso1.cfgsvc-prod.fkcloud.in:80\\nelse\\necho\\\"Invalidzone:$zone\\\"\\nexit1\\nfi\\n\\nrun_command(){\\necho\\\"Operation:${1}\\\"\\ncmd_output=$(eval${3})\\nempty_ok=${4}\\necho\\\"\\\"\\n\\nif[[$empty_ok!=\\\"true\\\"]];then\\niftest-z\\\"$cmd_output\\\"\\nthen\\necho\\\"Failedtodooperation:${2}.Exitting...\\\"\\nexit2\\nfi\\nfi\\n}\\n\\n#Usethiswhentherearehugenumberofentriesinsinglebucket\\nindex=$(($((0x$(sha1sum<<<\\\"$hostn\\\"|cut-c1-2)))%2))\\netc_hosts_key=\\\"etc-hosts-$index\\\"\\n\\netc_hosts_key=\\\"etc-hosts\\\"\\n\\nforVARIABLEin12345\\ndo\\nrun_command\\\"Getsmdmappingbucketdata\\\"\\\"SMDbucketdata\\\"\\\"curl-s-XGET\\\\\\\"http://${config_endpoint}/v1/buckets/${topology_bucket}\\\\\\\"\\\"\\nbucket_data=$cmd_output\\n\\nrun_command\\\"Parseversionfrombucketdata\\\"\\\"parseversion\\\"\\\"echo'$bucket_data'|python3-c\\\\\\\"importsys,json;print(json.dumps(json.load(sys.stdin)['metadata']['version']))\\\\\\\"\\\"\\nexisting_version=$cmd_output\\n\\nrun_command\\\"Parsemappingdatafrombucketdata\\\"\\\"parsemapping\\\"\\\"echo'$bucket_data'|python3-c\\\\\\\"importsys,json;print(json.dumps(json.load(sys.stdin)['keys']['$etc_hosts_key']))\\\\\\\"\\\"\\netc_hosts=$cmd_output\\n\\nif[[\\\"$etc_hosts\\\"==*\\\\\\\"\\\"$hostn\\\"\\\\\\\"*]];then\\nrun_command\\\"Updateexistingmappinginbucketdata\\\"\\\"updatemapping\\\"\\\"echo'$etc_hosts'|python3-c\\\\\\\"importsys,json;value=json.load(sys.stdin);[(item)foriteminvalueif'$hostn'initem][0]['$hostn']='$ip';print(json.dumps(value,sort_keys=True))\\\\\\\"\\\"\\nelse\\nrun_command\\\"Addmappingtobucketdata\\\"\\\"addmapping\\\"\\\"echo'$etc_hosts'|python3-c\\\\\\\"importsys,json;value=json.load(sys.stdin);item={u'$hostn':u'$ip'};value.append(item);print(json.dumps(value,sort_keys=True))\\\\\\\"\\\"\\nfi\\nnew_etc_hosts=$cmd_output\\n\\ndata='{'\\\\\\\"\\\"$etc_hosts_key\\\"\\\\\\\"':'\\\"$new_etc_hosts\\\"'}'\\nif[\\\"$new_etc_hosts\\\"=\\\"$etc_hosts\\\"];then\\necho\\\"NochangesinSMD.Notupdating\\\"\\nexit0\\nelse\\niftest-z\\\"$config_svc_token\\\";then\\nrun_command\\\"Generatingauthntokenfortalkingtoconfigbucket\\\"\\\"TokenConfigsvc\\\"\\\"curl-s-XPOST-F\\\\\\\"client_id=${authn_client_id}\\\\\\\"-F\\\\\\\"client_secret=${authn_secret}\\\\\\\"-F\\\\\\\"grant_type=client_credentials\\\\\\\"-F\\\\\\\"target_client_id=${target_client_id}\\\\\\\"${authn_endpoint}/oauth/token|sed's/,/\\\\n/g'|grep\\\\\\\"access_token\\\\\\\"|sed's/\\\\\\\"//g'|sed's/.*://g'\\\"\\nconfig_svc_token=$cmd_output\\nfi\\n\\necho\\\"ExistingMapping:$etc_hosts,NewMapping:$new_etc_hostsforexistingversion:$existing_version\\\"\\noutput=$(curl-XPOST-s\\\"http://${config_endpoint}/v1/buckets/${topology_bucket}/keys?message=updated_by_$ip\\\"-H\\\"X-Config-Bucket-Version:${existing_version}\\\"-H\\\"Content-Type:application/json\\\"--data-binary\\\"$data\\\"-H\\\"Authorization:Bearer${config_svc_token}\\\")\\nfinal_result=$?\\nfailed_message=\\\"Updatefailed\\\"\\nif[[$final_result-eq0&&\\\"$output\\\"==*\\\"$topology_bucket\\\"*&&\\\"$output\\\"==*\\\"version\\\"*&&\\\"$output\\\"==*\\\"created\\\"*&&\\\"$output\\\"==*\\\"lastUpdated\\\"*]];then\\necho$output\\nexit0\\nelse\\necho\\\"Failedtoupdateip.exiting..\\\"\\nexit1\\nfi\\nfi\\ndone\\n\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"isBootstrap\":false,\"memoryLimit\":\"256Mi\",\"memoryRequest\":\"256Mi\",\"name\":\"publish-myip\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}}],\"isPodServiceRequired\":false,\"labels\":{\"mcs.discovery.fcp.io/enable\":\"true\"},\"name\":\"test-cluster-hmaster\",\"podManagementPolicy\":\"Parallel\",\"shareProcessNamespace\":false,\"sidecarContainers\":[{\"args\":[\"com.flipkart.hbase.HbaseRackUtils\",\"/etc/hbase\",\"/hbase-operator\",\"/opt/share/rack_topology.data\"],\"command\":[\"./entrypoint\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"image\":\"edge.fkinternal.com/operator-hbase/hbase-rack-utils:1.0.3\",\"memoryLimit\":\"256Mi\",\"memoryRequest\":\"256Mi\",\"name\":\"rackutils\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/opt/share\",\"name\":\"data\",\"readOnly\":false}]},{\"cpuLimit\":\"100m\",\"cpuRequest\":\"100m\",\"image\":\"edge.fkinternal.com/indradhanush/fk-yak-filebeat:8.11.3-fk1\",\"memoryLimit\":\"128Mi\",\"memoryRequest\":\"128Mi\",\"name\":\"filebeat\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/var/log/flipkart/yak\",\"name\":\"app-log\",\"readOnly\":false},{\"mountPath\":\"/etc/filebeat\",\"name\":\"filebeat\",\"readOnly\":false}]}],\"size\":2,\"terminateGracePeriod\":120,\"volumes\":[{\"name\":\"data\",\"volumeSource\":\"EmptyDir\"},{\"name\":\"app-log\",\"volumeSource\":\"EmptyDir\"},{\"configName\":\"filebeat\",\"name\":\"filebeat\",\"volumeSource\":\"ConfigMap\"},{\"name\":\"nodeinfo\",\"path\":\"/etc/nodeinfo\",\"volumeSource\":\"HostPath\"}]},\"journalnode\":{\"annotations\":{\"fcp.k8s.mtl/cosmos-jmx\":\"enabled\",\"fcp.k8s.mtl/cosmos-statsd\":\"disabled\",\"fcp.k8s.mtl/cosmos-tail\":\"disabled\",\"fcp.k8s.mtl/mtl-config\":\"mtl-config-2\",\"fcp.k8s.mtl/mtl-config-map\":\"mtl-jn\",\"fcp.k8s/webhook-inject-fcp-dns\":\"No\"},\"containers\":[{\"args\":[\"/var/log/flipkart/yak/hadoop\",\"/etc/hadoop\",\"/opt/hadoop\"],\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\n\\nexportHADOOP_LOG_DIR=$0\\nexportHADOOP_CONF_DIR=$1\\nexportHADOOP_HOME=$2\\nexportUSER=$(whoami)\\nexportHADOOP_LOG_FILE=$HADOOP_LOG_DIR/hadoop-$USER-journalnode-$(hostname).log\\n\\nmkdir-p$HADOOP_LOG_DIR\\ntouch$HADOOP_LOG_FILE\\n\\nfunctionshutdown(){\\necho\\\"StoppingJournalnode\\\"\\n$HADOOP_HOME/bin/hdfs--daemonstopjournalnode\\n}\\n\\ntrapshutdownSIGTERM\\nexec$HADOOP_HOME/bin/hdfsjournalnodestart2>&1|tee-a$HADOOP_LOG_FILE&\\nwait\\n\"],\"cpuLimit\":\"2\",\"cpuRequest\":\"2\",\"livenessProbe\":{\"initialDelay\":40,\"tcpPort\":8485},\"memoryLimit\":\"5Gi\",\"memoryRequest\":\"5Gi\",\"name\":\"journalnode\",\"ports\":[{\"name\":\"journalnode-0\",\"port\":8485},{\"name\":\"journalnode-1\",\"port\":8480}],\"readinessProbe\":{\"initialDelay\":40,\"tcpPort\":8485},\"securityContext\":{\"addSysPtrace\":false,\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/grid/1\",\"name\":\"data\",\"readOnly\":false},{\"mountPath\":\"/var/log/flipkart/yak\",\"name\":\"app-log\",\"readOnly\":false},{\"mountPath\":\"/etc/nodeinfo\",\"name\":\"nodeinfo\",\"readOnly\":true}]}],\"initContainers\":[{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\n\\ni=0\\nwhiletrue;do\\necho\\\"$iiteration\\\"\\ndig+short$(hostname-f)|grep-v-e'^$'\\nif[$?==0];then\\nsleep30#30secondsdefaultdnscaching\\necho\\\"Breaking...\\\"\\nbreak\\nfi\\ni=$((i+1))\\nsleep1\\ndone\\n\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"isBootstrap\":false,\"memoryLimit\":\"128Mi\",\"memoryRequest\":\"128Mi\",\"name\":\"init-dnslookup\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}},{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-exmopipefail\\n\\ntopology_bucket=$FCP_APPID-k8s-hosts\\nauthn_client_id=prod-id-yak-ch-confsvc\\nauthn_secret=38XeZZirsXuqcyjxq6hPBW2aCJJ97r7JokK/XNEnV+ub8ZmK\\nip=$(hostname-i)\\n#TODORemove\\n#ip=$(ifconfig|grep\\\"inet\\\"|grep-Fv127.0.0.1|awk'{print$2}'|head-1)\\n\\nhostn=$(hostname-f)\\nzone=$FCP_ZONE\\nvpc=$FCP_VPC\\n\\nauthn_endpoint=https://service.authn-prod.fkcloud.in/\\nif[[$vpc==\\\"Fk-Preprod\\\"]]\\nthen\\nconfig_endpoint=10.24.2.28\\ntarget_client_id=http://10.24.2.28:80\\nelif[[$zone==\\\"in-hyderabad-1\\\"]]\\nthen\\nconfig_endpoint=10.24.0.32\\ntarget_client_id=http://10.24.0.32:80\\nelif[[$zone==\\\"in-chennai-1\\\"]]\\nthen\\nconfig_endpoint=10.47.0.101\\ntarget_client_id=http://10.47.0.101:80\\nelif[[$zone==\\\"in-chennai-2\\\"]]\\nthen\\nconfig_endpoint=10.83.47.156\\ntarget_client_id=cfg-api-calvin-ch\\nelif[[$zone==\\\"asia-south1\\\"]]\\nthen\\nconfig_endpoint=api.aso1.cfgsvc-prod.fkcloud.in\\ntarget_client_id=http://api.aso1.cfgsvc-prod.fkcloud.in:80\\nelse\\necho\\\"Invalidzone:$zone\\\"\\nexit1\\nfi\\n\\nrun_command(){\\necho\\\"Operation:${1}\\\"\\ncmd_output=$(eval${3})\\nempty_ok=${4}\\necho\\\"\\\"\\n\\nif[[$empty_ok!=\\\"true\\\"]];then\\niftest-z\\\"$cmd_output\\\"\\nthen\\necho\\\"Failedtodooperation:${2}.Exitting...\\\"\\nexit2\\nfi\\nfi\\n}\\n\\n#Usethiswhentherearehugenumberofentriesinsinglebucket\\nindex=$(($((0x$(sha1sum<<<\\\"$hostn\\\"|cut-c1-2)))%2))\\netc_hosts_key=\\\"etc-hosts-$index\\\"\\n\\netc_hosts_key=\\\"etc-hosts\\\"\\n\\nforVARIABLEin12345\\ndo\\nrun_command\\\"Getsmdmappingbucketdata\\\"\\\"SMDbucketdata\\\"\\\"curl-s-XGET\\\\\\\"http://${config_endpoint}/v1/buckets/${topology_bucket}\\\\\\\"\\\"\\nbucket_data=$cmd_output\\n\\nrun_command\\\"Parseversionfrombucketdata\\\"\\\"parseversion\\\"\\\"echo'$bucket_data'|python3-c\\\\\\\"importsys,json;print(json.dumps(json.load(sys.stdin)['metadata']['version']))\\\\\\\"\\\"\\nexisting_version=$cmd_output\\n\\nrun_command\\\"Parsemappingdatafrombucketdata\\\"\\\"parsemapping\\\"\\\"echo'$bucket_data'|python3-c\\\\\\\"importsys,json;print(json.dumps(json.load(sys.stdin)['keys']['$etc_hosts_key']))\\\\\\\"\\\"\\netc_hosts=$cmd_output\\n\\nif[[\\\"$etc_hosts\\\"==*\\\\\\\"\\\"$hostn\\\"\\\\\\\"*]];then\\nrun_command\\\"Updateexistingmappinginbucketdata\\\"\\\"updatemapping\\\"\\\"echo'$etc_hosts'|python3-c\\\\\\\"importsys,json;value=json.load(sys.stdin);[(item)foriteminvalueif'$hostn'initem][0]['$hostn']='$ip';print(json.dumps(value,sort_keys=True))\\\\\\\"\\\"\\nelse\\nrun_command\\\"Addmappingtobucketdata\\\"\\\"addmapping\\\"\\\"echo'$etc_hosts'|python3-c\\\\\\\"importsys,json;value=json.load(sys.stdin);item={u'$hostn':u'$ip'};value.append(item);print(json.dumps(value,sort_keys=True))\\\\\\\"\\\"\\nfi\\nnew_etc_hosts=$cmd_output\\n\\ndata='{'\\\\\\\"\\\"$etc_hosts_key\\\"\\\\\\\"':'\\\"$new_etc_hosts\\\"'}'\\nif[\\\"$new_etc_hosts\\\"=\\\"$etc_hosts\\\"];then\\necho\\\"NochangesinSMD.Notupdating\\\"\\nexit0\\nelse\\niftest-z\\\"$config_svc_token\\\";then\\nrun_command\\\"Generatingauthntokenfortalkingtoconfigbucket\\\"\\\"TokenConfigsvc\\\"\\\"curl-s-XPOST-F\\\\\\\"client_id=${authn_client_id}\\\\\\\"-F\\\\\\\"client_secret=${authn_secret}\\\\\\\"-F\\\\\\\"grant_type=client_credentials\\\\\\\"-F\\\\\\\"target_client_id=${target_client_id}\\\\\\\"${authn_endpoint}/oauth/token|sed's/,/\\\\n/g'|grep\\\\\\\"access_token\\\\\\\"|sed's/\\\\\\\"//g'|sed's/.*://g'\\\"\\nconfig_svc_token=$cmd_output\\nfi\\n\\necho\\\"ExistingMapping:$etc_hosts,NewMapping:$new_etc_hostsforexistingversion:$existing_version\\\"\\noutput=$(curl-XPOST-s\\\"http://${config_endpoint}/v1/buckets/${topology_bucket}/keys?message=updated_by_$ip\\\"-H\\\"X-Config-Bucket-Version:${existing_version}\\\"-H\\\"Content-Type:application/json\\\"--data-binary\\\"$data\\\"-H\\\"Authorization:Bearer${config_svc_token}\\\")\\nfinal_result=$?\\nfailed_message=\\\"Updatefailed\\\"\\nif[[$final_result-eq0&&\\\"$output\\\"==*\\\"$topology_bucket\\\"*&&\\\"$output\\\"==*\\\"version\\\"*&&\\\"$output\\\"==*\\\"created\\\"*&&\\\"$output\\\"==*\\\"lastUpdated\\\"*]];then\\necho$output\\nexit0\\nelse\\necho\\\"Failedtoupdateip.exiting..\\\"\\nexit1\\nfi\\nfi\\ndone\\n\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"isBootstrap\":false,\"memoryLimit\":\"256Mi\",\"memoryRequest\":\"256Mi\",\"name\":\"publish-myip\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}}],\"isPodServiceRequired\":true,\"labels\":{\"mcs.discovery.fcp.io/enable\":\"true\"},\"name\":\"test-cluster-jn\",\"podManagementPolicy\":\"Parallel\",\"shareProcessNamespace\":false,\"sidecarContainers\":[{\"cpuLimit\":\"100m\",\"cpuRequest\":\"100m\",\"image\":\"edge.fkinternal.com/indradhanush/fk-yak-filebeat:8.11.3-fk1\",\"memoryLimit\":\"128Mi\",\"memoryRequest\":\"128Mi\",\"name\":\"filebeat\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/var/log/flipkart/yak\",\"name\":\"app-log\",\"readOnly\":false},{\"mountPath\":\"/etc/filebeat\",\"name\":\"filebeat\",\"readOnly\":false}]},{\"args\":[\"/tmp\",\"/grid/1/dfs\",\"test-cluster-bkp-0.test-cluster-bkp.test-namespace.svc.cluster.local\"],\"command\":[\"/opt/scripts/copy_backup\"],\"cpuLimit\":\"0.5\",\"cpuRequest\":\"0.5\",\"image\":\"edge.fkinternal.com/indradhanush/yak-base:2.5.3-08-rc7\",\"memoryLimit\":\"256Mi\",\"memoryRequest\":\"256Mi\",\"name\":\"backup\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/grid/1\",\"name\":\"data\",\"readOnly\":true}]}],\"size\":5,\"terminateGracePeriod\":120,\"volumeClaims\":[{\"name\":\"data\",\"storageClassName\":\"local-extreme-1a-ext4\",\"storageSize\":\"16Gi\"}],\"volumes\":[{\"name\":\"nodeinfo\",\"path\":\"/etc/nodeinfo\",\"volumeSource\":\"HostPath\"},{\"name\":\"app-log\",\"volumeSource\":\"EmptyDir\"},{\"configName\":\"filebeat\",\"name\":\"filebeat\",\"volumeSource\":\"ConfigMap\"}]},\"namenode\":{\"annotations\":{\"fcp.k8s.mtl/cosmos-jmx\":\"enabled\",\"fcp.k8s.mtl/cosmos-statsd\":\"disabled\",\"fcp.k8s.mtl/cosmos-tail\":\"disabled\",\"fcp.k8s.mtl/mtl-config\":\"mtl-config-2\",\"fcp.k8s.mtl/mtl-config-map\":\"mtl-nn\",\"fcp.k8s/webhook-inject-fcp-dns\":\"No\"},\"containers\":[{\"args\":[\"/var/log/flipkart/yak/hadoop\",\"/etc/hadoop\",\"/opt/hadoop\"],\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m-x\\n\\nexportHADOOP_LOG_DIR=$0\\nexportHADOOP_CONF_DIR=$1\\nexportHADOOP_HOME=$2\\nexportUSER=$(whoami)\\nexportHADOOP_LOG_FILE=$HADOOP_LOG_DIR/hadoop-$USER-namenode-$(hostname).log\\n\\nmkdir-p$HADOOP_LOG_DIR\\ntouch$HADOOP_LOG_FILE\\n\\nfunctionshutdown(){\\necho\\\"StoppingNamenode\\\"\\nis_active=$($HADOOP_HOME/bin/hdfshaadmin-getAllServiceState|grep\\\"$(hostname-f)\\\"|grep\\\"active\\\"|wc-l)\\n\\nif[[$is_active==1]];then\\nforiin$(echo$NNS|tr\\\",\\\"\\\"\\\\n\\\");do\\nif[[$($HADOOP_HOME/bin/hdfshaadmin-getServiceState$i|grep\\\"standby\\\"|wc-l)==1]];then\\nSTANDBY_SERVICE=$i\\nbreak\\nfi\\ndone\\n\\necho\\\"IsActive.Transitioningtostandby\\\"\\nif[[-n\\\"$MY_SERVICE\\\"&&-n\\\"$STANDBY_SERVICE\\\"&&$MY_SERVICE!=$STANDBY_SERVICE]];then\\necho\\\"Failingoverfrom$MY_SERVICEto$STANDBY_SERVICE\\\"\\n$HADOOP_HOME/bin/hdfshaadmin-failover$MY_SERVICE$STANDBY_SERVICE\\nelse\\necho\\\"$MY_SERVICEor$STANDBY_SERVICEisnotdefinedorsame.Cannotfailover.Exitting...\\\"\\nfi\\nelse\\necho\\\"Isnotactive\\\"\\nfi\\nsleep60\\necho\\\"Completedshutdowncleanup\\\"\\ntouch/lifecycle/nn-terminated\\n$HADOOP_HOME/bin/hdfs--daemonstopnamenode\\n}\\n\\n#Createtempfileforexcludehostsifnotexists.\\nEXCLUDEPATH=$($HADOOP_HOME/bin/hdfsgetconf-confKeydfs.hosts.exclude)\\ntouch$EXCLUDEPATH||true#Ignoreiffailed\\n\\nNAMESERVICES=$($HADOOP_HOME/bin/hdfsgetconf-confKeydfs.nameservices)\\nNNS=$($HADOOP_HOME/bin/hdfsgetconf-confKeydfs.ha.namenodes.$NAMESERVICES)\\nMY_SERVICE=\\\"\\\"\\nHTTP_ADDR=\\\"\\\"\\nforiin$(echo$NNS|tr\\\",\\\"\\\"\\\\n\\\");do\\nif[[$($HADOOP_HOME/bin/hdfsgetconf-confKeydfs.namenode.rpc-address.$NAMESERVICES.$i|sed's/:[0-9]\\\\+$//'|grep$(hostname-f)|wc-l)==1]];then\\nMY_SERVICE=$i\\nHTTP_ADDR=$($HADOOP_HOME/bin/hdfsgetconf-confKeydfs.namenode.http-address.$NAMESERVICES.$i)\\nfi\\ndone\\n\\necho\\\"MyService:$MY_SERVICE\\\"\\n\\ntrapshutdownSIGTERM\\nexec$HADOOP_HOME/bin/hdfsnamenode2>&1|tee-a$HADOOP_LOG_FILE&\\nwait\\n\"],\"cpuLimit\":\"3\",\"cpuRequest\":\"3\",\"livenessProbe\":{\"initialDelay\":180,\"tcpPort\":8020},\"memoryLimit\":\"12Gi\",\"memoryRequest\":\"12Gi\",\"name\":\"namenode\",\"ports\":[{\"name\":\"namenode-0\",\"port\":8020},{\"name\":\"namenode-1\",\"port\":9870},{\"name\":\"namenode-2\",\"port\":50070},{\"name\":\"namenode-3\",\"port\":9000}],\"readinessProbe\":{\"initialDelay\":180,\"tcpPort\":8020},\"securityContext\":{\"addSysPtrace\":false,\"runAsGroup\":1011,\"runAsUser\":1011},\"startupProbe\":{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\n\\nexportHADOOP_LOG_DIR=$0\\nexportHADOOP_CONF_DIR=$1\\nexportHADOOP_HOME=$2\\n\\nif[[$($HADOOP_HOME/bin/hdfsdfsadmin-safemodeget|grep\\\"SafemodeisOFF\\\"|wc-l)==0]];then\\necho\\\"Lookslikethereisnonamenodewithsafemodeoff.Skippingchecks...\\\"\\nexit0\\nelif[[$($HADOOP_HOME/bin/hdfsdfsadmin-safemodeget|grep\\\"$(hostname-f)\\\"|grep\\\"SafemodeisOFF\\\"|wc-l)==1]];then\\necho\\\"Namenodeisoutofsafemode.Exiting...\\\"\\nexit0\\nelse\\necho\\\"Namenodeisstillinsafemode.Failing...\\\"\\nexit1\\nfi\\n\\necho\\\"Somethingunexpectedhappenedatstartupprobe.Failing...\\\"\\nexit1\\n\",\"/var/log/flipkart/yak/hadoop\",\"/etc/hadoop\",\"/opt/hadoop\"],\"failureThreshold\":10,\"initialDelay\":30,\"timeout\":60},\"volumeMounts\":[{\"mountPath\":\"/grid/1\",\"name\":\"data\",\"readOnly\":false},{\"mountPath\":\"/lifecycle\",\"name\":\"lifecycle\",\"readOnly\":false},{\"mountPath\":\"/etc/nodeinfo\",\"name\":\"nodeinfo\",\"readOnly\":true},{\"mountPath\":\"/var/log/flipkart/yak\",\"name\":\"app-log\",\"readOnly\":false}]},{\"args\":[\"/var/log/flipkart/yak/hadoop\",\"/etc/hadoop\",\"/opt/hadoop\"],\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\n\\nexportHADOOP_LOG_DIR=$0\\nexportHADOOP_CONF_DIR=$1\\nexportHADOOP_HOME=$2\\nexportUSER=$(whoami)\\nexportHADOOP_LOG_FILE=$HADOOP_LOG_DIR/hadoop-$USER-zkfc-$(hostname).log\\n\\nmkdir-p$HADOOP_LOG_DIR\\ntouch$HADOOP_LOG_FILE\\n\\nfunctionshutdown(){\\nwhiletrue;do\\nif[[-f\\\"/lifecycle/nn-terminated\\\"]];then\\necho\\\"Stoppingzkfc\\\"\\nsleep10\\n$HADOOP_HOME/bin/hdfs--daemonstopzkfc\\nbreak\\nfi\\necho\\\"Waitingfornamenodetodie\\\"\\nsleep2\\ndone\\n}\\n\\ntrapshutdownSIGTERM\\nexec$HADOOP_HOME/bin/hdfszkfc2>&1|tee-a$HADOOP_LOG_FILE&\\nwait\\n\"],\"cpuLimit\":\"1\",\"cpuRequest\":\"1\",\"livenessProbe\":{\"initialDelay\":60,\"tcpPort\":8019},\"memoryLimit\":\"3Gi\",\"memoryRequest\":\"3Gi\",\"name\":\"zkfc\",\"ports\":[{\"name\":\"zkfc-0\",\"port\":8019}],\"readinessProbe\":{\"initialDelay\":60,\"tcpPort\":8019},\"securityContext\":{\"addSysPtrace\":false,\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/grid/1\",\"name\":\"data\",\"readOnly\":false},{\"mountPath\":\"/lifecycle\",\"name\":\"lifecycle\",\"readOnly\":false},{\"mountPath\":\"/etc/nodeinfo\",\"name\":\"nodeinfo\",\"readOnly\":true},{\"mountPath\":\"/var/log/flipkart/yak\",\"name\":\"app-log\",\"readOnly\":false}]}],\"initContainers\":[{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\n\\ni=0\\nwhiletrue;do\\necho\\\"$iiteration\\\"\\ndig+short$(hostname-f)|grep-v-e'^$'\\nif[$?==0];then\\nsleep30#30secondsdefaultdnscaching\\necho\\\"Breaking...\\\"\\nbreak\\nfi\\ni=$((i+1))\\nsleep1\\ndone\\n\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"isBootstrap\":false,\"memoryLimit\":\"128Mi\",\"memoryRequest\":\"128Mi\",\"name\":\"init-dnslookup\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}},{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m-x\\n\\nexportHADOOP_LOG_DIR=/var/log/flipkart/yak/hadoop\\nexportHADOOP_CONF_DIR=/etc/hadoop\\nexportHADOOP_HOME=/opt/hadoop\\n\\necho\\\"N\\\"|$HADOOP_HOME/bin/hdfsnamenode-format$($HADOOP_HOME/bin/hdfsgetconf-confKeydfs.nameservices)||true\\n\"],\"cpuLimit\":\"3\",\"cpuRequest\":\"3\",\"isBootstrap\":true,\"memoryLimit\":\"12Gi\",\"memoryRequest\":\"12Gi\",\"name\":\"init-namenode\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/grid/1\",\"name\":\"data\"}]},{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m-x\\n\\nexportHADOOP_LOG_DIR=/var/log/flipkart/yak/hadoop\\nexportHADOOP_CONF_DIR=/etc/hadoop\\nexportHADOOP_HOME=/opt/hadoop\\n\\necho\\\"N\\\"|$HADOOP_HOME/bin/hdfszkfc-formatZK||true\\n\"],\"cpuLimit\":\"1\",\"cpuRequest\":\"1\",\"isBootstrap\":true,\"memoryLimit\":\"3Gi\",\"memoryRequest\":\"3Gi\",\"name\":\"init-zkfc\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}},{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m-x\\n\\nexportHADOOP_LOG_DIR=/var/log/flipkart/yak/hadoop\\nexportHADOOP_CONF_DIR=/etc/hadoop\\nexportHADOOP_HOME=/opt/hadoop\\n\\n$HADOOP_HOME/bin/hdfsnamenode-metadataVersion2>&1;exit_code=$?\\nif[$exit_code-eq1]\\nthen\\necho\\\"Namenodemetadataisnotaccessible,runningbootstrapstandby\\\"\\n$HADOOP_HOME/bin/hdfsnamenode-bootstrapStandby-nonInteractive\\nelse\\necho\\\"Namenodemetadataisaccessible,soskippingbootstrap\\\"\\nfi\\n\"],\"cpuLimit\":\"3\",\"cpuRequest\":\"3\",\"isBootstrap\":false,\"memoryLimit\":\"12Gi\",\"memoryRequest\":\"12Gi\",\"name\":\"init-nn-bootstrap-standby\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/grid/1\",\"name\":\"data\"}]},{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-exmopipefail\\n\\ntopology_bucket=$FCP_APPID-k8s-hosts\\nauthn_client_id=prod-id-yak-ch-confsvc\\nauthn_secret=38XeZZirsXuqcyjxq6hPBW2aCJJ97r7JokK/XNEnV+ub8ZmK\\nip=$(hostname-i)\\n#TODORemove\\n#ip=$(ifconfig|grep\\\"inet\\\"|grep-Fv127.0.0.1|awk'{print$2}'|head-1)\\n\\nhostn=$(hostname-f)\\nzone=$FCP_ZONE\\nvpc=$FCP_VPC\\n\\nauthn_endpoint=https://service.authn-prod.fkcloud.in/\\nif[[$vpc==\\\"Fk-Preprod\\\"]]\\nthen\\nconfig_endpoint=10.24.2.28\\ntarget_client_id=http://10.24.2.28:80\\nelif[[$zone==\\\"in-hyderabad-1\\\"]]\\nthen\\nconfig_endpoint=10.24.0.32\\ntarget_client_id=http://10.24.0.32:80\\nelif[[$zone==\\\"in-chennai-1\\\"]]\\nthen\\nconfig_endpoint=10.47.0.101\\ntarget_client_id=http://10.47.0.101:80\\nelif[[$zone==\\\"in-chennai-2\\\"]]\\nthen\\nconfig_endpoint=10.83.47.156\\ntarget_client_id=cfg-api-calvin-ch\\nelif[[$zone==\\\"asia-south1\\\"]]\\nthen\\nconfig_endpoint=api.aso1.cfgsvc-prod.fkcloud.in\\ntarget_client_id=http://api.aso1.cfgsvc-prod.fkcloud.in:80\\nelse\\necho\\\"Invalidzone:$zone\\\"\\nexit1\\nfi\\n\\nrun_command(){\\necho\\\"Operation:${1}\\\"\\ncmd_output=$(eval${3})\\nempty_ok=${4}\\necho\\\"\\\"\\n\\nif[[$empty_ok!=\\\"true\\\"]];then\\niftest-z\\\"$cmd_output\\\"\\nthen\\necho\\\"Failedtodooperation:${2}.Exitting...\\\"\\nexit2\\nfi\\nfi\\n}\\n\\n#Usethiswhentherearehugenumberofentriesinsinglebucket\\nindex=$(($((0x$(sha1sum<<<\\\"$hostn\\\"|cut-c1-2)))%2))\\netc_hosts_key=\\\"etc-hosts-$index\\\"\\n\\netc_hosts_key=\\\"etc-hosts\\\"\\n\\nforVARIABLEin12345\\ndo\\nrun_command\\\"Getsmdmappingbucketdata\\\"\\\"SMDbucketdata\\\"\\\"curl-s-XGET\\\\\\\"http://${config_endpoint}/v1/buckets/${topology_bucket}\\\\\\\"\\\"\\nbucket_data=$cmd_output\\n\\nrun_command\\\"Parseversionfrombucketdata\\\"\\\"parseversion\\\"\\\"echo'$bucket_data'|python3-c\\\\\\\"importsys,json;print(json.dumps(json.load(sys.stdin)['metadata']['version']))\\\\\\\"\\\"\\nexisting_version=$cmd_output\\n\\nrun_command\\\"Parsemappingdatafrombucketdata\\\"\\\"parsemapping\\\"\\\"echo'$bucket_data'|python3-c\\\\\\\"importsys,json;print(json.dumps(json.load(sys.stdin)['keys']['$etc_hosts_key']))\\\\\\\"\\\"\\netc_hosts=$cmd_output\\n\\nif[[\\\"$etc_hosts\\\"==*\\\\\\\"\\\"$hostn\\\"\\\\\\\"*]];then\\nrun_command\\\"Updateexistingmappinginbucketdata\\\"\\\"updatemapping\\\"\\\"echo'$etc_hosts'|python3-c\\\\\\\"importsys,json;value=json.load(sys.stdin);[(item)foriteminvalueif'$hostn'initem][0]['$hostn']='$ip';print(json.dumps(value,sort_keys=True))\\\\\\\"\\\"\\nelse\\nrun_command\\\"Addmappingtobucketdata\\\"\\\"addmapping\\\"\\\"echo'$etc_hosts'|python3-c\\\\\\\"importsys,json;value=json.load(sys.stdin);item={u'$hostn':u'$ip'};value.append(item);print(json.dumps(value,sort_keys=True))\\\\\\\"\\\"\\nfi\\nnew_etc_hosts=$cmd_output\\n\\ndata='{'\\\\\\\"\\\"$etc_hosts_key\\\"\\\\\\\"':'\\\"$new_etc_hosts\\\"'}'\\nif[\\\"$new_etc_hosts\\\"=\\\"$etc_hosts\\\"];then\\necho\\\"NochangesinSMD.Notupdating\\\"\\nexit0\\nelse\\niftest-z\\\"$config_svc_token\\\";then\\nrun_command\\\"Generatingauthntokenfortalkingtoconfigbucket\\\"\\\"TokenConfigsvc\\\"\\\"curl-s-XPOST-F\\\\\\\"client_id=${authn_client_id}\\\\\\\"-F\\\\\\\"client_secret=${authn_secret}\\\\\\\"-F\\\\\\\"grant_type=client_credentials\\\\\\\"-F\\\\\\\"target_client_id=${target_client_id}\\\\\\\"${authn_endpoint}/oauth/token|sed's/,/\\\\n/g'|grep\\\\\\\"access_token\\\\\\\"|sed's/\\\\\\\"//g'|sed's/.*://g'\\\"\\nconfig_svc_token=$cmd_output\\nfi\\n\\necho\\\"ExistingMapping:$etc_hosts,NewMapping:$new_etc_hostsforexistingversion:$existing_version\\\"\\noutput=$(curl-XPOST-s\\\"http://${config_endpoint}/v1/buckets/${topology_bucket}/keys?message=updated_by_$ip\\\"-H\\\"X-Config-Bucket-Version:${existing_version}\\\"-H\\\"Content-Type:application/json\\\"--data-binary\\\"$data\\\"-H\\\"Authorization:Bearer${config_svc_token}\\\")\\nfinal_result=$?\\nfailed_message=\\\"Updatefailed\\\"\\nif[[$final_result-eq0&&\\\"$output\\\"==*\\\"$topology_bucket\\\"*&&\\\"$output\\\"==*\\\"version\\\"*&&\\\"$output\\\"==*\\\"created\\\"*&&\\\"$output\\\"==*\\\"lastUpdated\\\"*]];then\\necho$output\\nexit0\\nelse\\necho\\\"Failedtoupdateip.exiting..\\\"\\nexit1\\nfi\\nfi\\ndone\\n\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"isBootstrap\":false,\"memoryLimit\":\"256Mi\",\"memoryRequest\":\"256Mi\",\"name\":\"publish-myip\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}}],\"isPodServiceRequired\":false,\"labels\":{\"mcs.discovery.fcp.io/enable\":\"true\"},\"name\":\"test-cluster-nn\",\"podManagementPolicy\":\"OrderedReady\",\"shareProcessNamespace\":false,\"sidecarContainers\":[{\"cpuLimit\":\"100m\",\"cpuRequest\":\"100m\",\"image\":\"edge.fkinternal.com/indradhanush/fk-yak-filebeat:8.11.3-fk1\",\"memoryLimit\":\"128Mi\",\"memoryRequest\":\"128Mi\",\"name\":\"filebeat\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/var/log/flipkart/yak\",\"name\":\"app-log\",\"readOnly\":false},{\"mountPath\":\"/etc/filebeat\",\"name\":\"filebeat\",\"readOnly\":false}]},{\"args\":[\"/tmp\",\"/grid/1/dfs\",\"test-cluster-bkp-0.test-cluster-bkp.test-namespace.svc.cluster.local\"],\"command\":[\"/opt/scripts/copy_backup\"],\"cpuLimit\":\"0.5\",\"cpuRequest\":\"0.5\",\"image\":\"edge.fkinternal.com/indradhanush/yak-base:2.5.3-08-rc7\",\"memoryLimit\":\"256Mi\",\"memoryRequest\":\"256Mi\",\"name\":\"backup\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/grid/1\",\"name\":\"data\",\"readOnly\":true}]}],\"size\":3,\"terminateGracePeriod\":120,\"volumeClaims\":[{\"name\":\"data\",\"storageClassName\":\"local-extreme-1a-ext4\",\"storageSize\":\"32Gi\"}],\"volumes\":[{\"name\":\"lifecycle\",\"volumeSource\":\"EmptyDir\"},{\"name\":\"nodeinfo\",\"path\":\"/etc/nodeinfo\",\"volumeSource\":\"HostPath\"},{\"name\":\"app-log\",\"volumeSource\":\"EmptyDir\"},{\"configName\":\"filebeat\",\"name\":\"filebeat\",\"volumeSource\":\"ConfigMap\"}]},\"zookeeper\":{\"podDisruptionBudget\":{\"maxUnavailable\":1},\"annotations\":{\"fcp.k8s.mtl/cosmos-jmx\":\"enabled\",\"fcp.k8s.mtl/cosmos-statsd\":\"disabled\",\"fcp.k8s.mtl/cosmos-tail\":\"disabled\",\"fcp.k8s.mtl/mtl-config\":\"mtl-config-2\",\"fcp.k8s.mtl/mtl-config-map\":\"mtl-zk\",\"fcp.k8s/webhook-inject-fcp-dns\":\"No\"},\"containers\":[{\"args\":[\"/var/log/flipkart/yak/hbase\",\"/etc/hbase\",\"/opt/hbase\"],\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m-x\\nset-opipefail\\n\\nexportHBASE_LOG_DIR=$0\\nexportHBASE_CONF_DIR=$1\\nexportHBASE_HOME=$2\\nexportUSER=$(whoami)\\n\\nmkdir-p$HBASE_LOG_DIR\\ntouch$HBASE_LOG_DIR/hbase-$USER-zookeeper-$(hostname).log&&tail-F$HBASE_LOG_DIR/hbase-$USER-zookeeper-$(hostname).log&\\ntouch$HBASE_LOG_DIR/hbase-$USER-zookeeper-$(hostname).out&&tail-F$HBASE_LOG_DIR/hbase-$USER-zookeeper-$(hostname).out&\\n\\nfunctionshutdown(){\\nechostat|nclocalhost2181|grep\\\"Mode:follower\\\"\\nexit_status=$?\\necho\\\"StoppingZookeeper\\\"\\n$HBASE_HOME/bin/hbase-daemon.shstopzookeeper\\nif[$exit_status!=0];then\\nif[!-f\\\"$HBASE_CONF_DIR/hbase-site.xml\\\"];then\\necho\\\"$HBASE_CONF_DIR/hbase-site.xmldoesnotexists\\\"\\nsleep120\\nexit1\\nfi\\n\\nif!grep-q\\\"<name>hbase.zookeeper.quorum</name>\\\"\\\"$HBASE_CONF_DIR/hbase-site.xml\\\";then\\necho\\\"Error:$HBASE_CONF_DIR/hbase-site.xmldoesnotcontain<name>hbase.zookeeper.quorum</name>.\\\"\\nsleep120\\nexit1\\nfi\\n\\nquorum=$(grep-A1\\\"<name>hbase.zookeeper.quorum</name>\\\"\\\"$HBASE_CONF_DIR/hbase-site.xml\\\"|grep\\\"<value>\\\"|sed-e's/<value>\\\\(.*\\\\)<\\\\/value>/\\\\1/'|xargs)\\nif[-z\\\"$quorum\\\"];then\\necho\\\"Error:<value>for<name>hbase.zookeeper.quorum</name>in$HBASE_CONF_DIR/hbase-site.xmlisemptyornotproperlyformatted.\\\"\\nsleep120\\nexit1\\nfi\\n\\nif!echo\\\"$quorum\\\"|grep-q\\\",\\\";then\\necho\\\"Error:<value>for<name>hbase.zookeeper.quorum</name>in$HBASE_CONF_DIR/hbase-site.xmldoesnotappeartobeacomma-separatedlistofzookeepers\\\"\\nsleep120\\nexit1\\nfi\\n\\nIFS=','read-raZKs<<<$quorum\\nif[[${#ZKs[@]}-gt0]];then\\nfunctionleaderElection(){\\nforzkin\\\"${ZKs[@]}\\\";do\\nif[[$zk!=$(hostname-f)]];then\\nhost=$(echo$zk2181)\\nmode=$(echo\\\"stat\\\"|timeout1snc$host|grep\\\"Mode:\\\"|sed's/Mode://'|sed-e's/[[:space:]]*$//')\\nif[[$?-eq0&&-n\\\"$mode\\\"]];then\\nif[[$mode==leader]];then\\necho\\\"$zkisa$mode\\\"\\necho\\\"Leaderelectioncompleted\\\"\\nexit0\\nelse\\necho\\\"$zkisa$mode\\\"\\nfi\\nfi\\nfi\\ndone\\n}\\n\\npod_timeout=110\\nendTime=$(($(date+%s)+$pod_timeout))\\nwhile[$(date+%s)-lt$endTime];do\\nleaderElection\\nsleep1\\ndone\\necho\\\"Leaderelectiondidnotcompletebutthiszookeeperpodisshuttingdownaspodtimeoutisbreached\\\"\\nexit1\\nelse\\nsleep120\\nexit1\\nfi\\nfi\\n}\\n\\ntrapshutdownSIGTERM\\nexec$HBASE_HOME/bin/hbase-daemon.shforeground_startzookeeper&\\nwait\\n\"],\"cpuLimit\":\"5\",\"cpuRequest\":\"5\",\"livenessProbe\":{\"initialDelay\":20,\"tcpPort\":2181},\"memoryLimit\":\"5Gi\",\"memoryRequest\":\"5Gi\",\"name\":\"zookeeper\",\"ports\":[{\"name\":\"zookeeper-0\",\"port\":2181},{\"name\":\"zookeeper-1\",\"port\":2888},{\"name\":\"zookeeper-2\",\"port\":3888}],\"readinessProbe\":{\"initialDelay\":20,\"tcpPort\":2181},\"securityContext\":{\"addSysPtrace\":false,\"runAsGroup\":1011,\"runAsUser\":1011},\"startupProbe\":{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\n\\nexportHBASE_LOG_DIR=$0\\nexportHBASE_CONF_DIR=$1\\nexportHBASE_HOME=$2\\n\\n#TODO:Findbetteralternative\\nIFS=','read-raZKs<<<$($HBASE_HOME/bin/hbasezkcliquit2>/dev/null|grep\\\"Connectingto\\\"|sed's/Connectingto//')\\nvisited=\\\"\\\"\\nquorum=\\\"\\\"\\nmyhost=\\\"localhost2181\\\"\\nforzkin\\\"${ZKs[@]}\\\";do\\nif[[$(echo$zk|grep$(hostname-f)|wc-l)==1]];then\\nmyhost=$(echo$zk|sed's/://')\\nfi\\n\\nif[[$(echo\\\"stat\\\"|nc$(echo$zk|sed's/://')|grep\\\"Mode:\\\"|wc-l)==1]];then\\nquorum=\\\"present\\\"\\nfi\\nvisited=\\\"true\\\"\\ndone\\n\\nif[[-n$visited&&-z$quorum]];then\\necho\\\"Quorumisabsent,disablingstartupchecks...\\\"\\nsleep5\\nexit0\\nfi\\n\\nif[[$(echo\\\"stat\\\"|nc$myhost|grep\\\"Mode:\\\"|wc-l)==1]];then\\nexit0\\nelse\\necho\\\"zookeeperisnotabletoconnecttoquorum\\\"\\nexit1\\nfi\\n\",\"/var/log/flipkart/yak/hbase\",\"/etc/hbase\",\"/opt/hbase\"],\"failureThreshold\":10,\"initialDelay\":30,\"timeout\":60},\"volumeMounts\":[{\"mountPath\":\"/grid/1\",\"name\":\"data\",\"readOnly\":false},{\"mountPath\":\"/var/log/flipkart/yak\",\"name\":\"app-log\",\"readOnly\":false},{\"mountPath\":\"/etc/nodeinfo\",\"name\":\"nodeinfo\",\"readOnly\":true}]}],\"initContainers\":[{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-m\\n\\ni=0\\nwhiletrue;do\\necho\\\"$iiteration\\\"\\ndig+short$(hostname-f)|grep-v-e'^$'\\nif[$?==0];then\\nsleep30#30secondsdefaultdnscaching\\necho\\\"Breaking...\\\"\\nbreak\\nfi\\ni=$((i+1))\\nsleep1\\ndone\\n\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"isBootstrap\":false,\"memoryLimit\":\"128Mi\",\"memoryRequest\":\"128Mi\",\"name\":\"init-dnslookup\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}},{\"command\":[\"/bin/bash\",\"-c\",\"#!/bin/bash\\nset-exmopipefail\\n\\ntopology_bucket=$FCP_APPID-k8s-hosts\\nauthn_client_id=prod-id-yak-ch-confsvc\\nauthn_secret=38XeZZirsXuqcyjxq6hPBW2aCJJ97r7JokK/XNEnV+ub8ZmK\\nip=$(hostname-i)\\n#TODORemove\\n#ip=$(ifconfig|grep\\\"inet\\\"|grep-Fv127.0.0.1|awk'{print$2}'|head-1)\\n\\nhostn=$(hostname-f)\\nzone=$FCP_ZONE\\nvpc=$FCP_VPC\\n\\nauthn_endpoint=https://service.authn-prod.fkcloud.in/\\nif[[$vpc==\\\"Fk-Preprod\\\"]]\\nthen\\nconfig_endpoint=10.24.2.28\\ntarget_client_id=http://10.24.2.28:80\\nelif[[$zone==\\\"in-hyderabad-1\\\"]]\\nthen\\nconfig_endpoint=10.24.0.32\\ntarget_client_id=http://10.24.0.32:80\\nelif[[$zone==\\\"in-chennai-1\\\"]]\\nthen\\nconfig_endpoint=10.47.0.101\\ntarget_client_id=http://10.47.0.101:80\\nelif[[$zone==\\\"in-chennai-2\\\"]]\\nthen\\nconfig_endpoint=10.83.47.156\\ntarget_client_id=cfg-api-calvin-ch\\nelif[[$zone==\\\"asia-south1\\\"]]\\nthen\\nconfig_endpoint=api.aso1.cfgsvc-prod.fkcloud.in\\ntarget_client_id=http://api.aso1.cfgsvc-prod.fkcloud.in:80\\nelse\\necho\\\"Invalidzone:$zone\\\"\\nexit1\\nfi\\n\\nrun_command(){\\necho\\\"Operation:${1}\\\"\\ncmd_output=$(eval${3})\\nempty_ok=${4}\\necho\\\"\\\"\\n\\nif[[$empty_ok!=\\\"true\\\"]];then\\niftest-z\\\"$cmd_output\\\"\\nthen\\necho\\\"Failedtodooperation:${2}.Exitting...\\\"\\nexit2\\nfi\\nfi\\n}\\n\\n#Usethiswhentherearehugenumberofentriesinsinglebucket\\nindex=$(($((0x$(sha1sum<<<\\\"$hostn\\\"|cut-c1-2)))%2))\\netc_hosts_key=\\\"etc-hosts-$index\\\"\\n\\netc_hosts_key=\\\"etc-hosts\\\"\\n\\nforVARIABLEin12345\\ndo\\nrun_command\\\"Getsmdmappingbucketdata\\\"\\\"SMDbucketdata\\\"\\\"curl-s-XGET\\\\\\\"http://${config_endpoint}/v1/buckets/${topology_bucket}\\\\\\\"\\\"\\nbucket_data=$cmd_output\\n\\nrun_command\\\"Parseversionfrombucketdata\\\"\\\"parseversion\\\"\\\"echo'$bucket_data'|python3-c\\\\\\\"importsys,json;print(json.dumps(json.load(sys.stdin)['metadata']['version']))\\\\\\\"\\\"\\nexisting_version=$cmd_output\\n\\nrun_command\\\"Parsemappingdatafrombucketdata\\\"\\\"parsemapping\\\"\\\"echo'$bucket_data'|python3-c\\\\\\\"importsys,json;print(json.dumps(json.load(sys.stdin)['keys']['$etc_hosts_key']))\\\\\\\"\\\"\\netc_hosts=$cmd_output\\n\\nif[[\\\"$etc_hosts\\\"==*\\\\\\\"\\\"$hostn\\\"\\\\\\\"*]];then\\nrun_command\\\"Updateexistingmappinginbucketdata\\\"\\\"updatemapping\\\"\\\"echo'$etc_hosts'|python3-c\\\\\\\"importsys,json;value=json.load(sys.stdin);[(item)foriteminvalueif'$hostn'initem][0]['$hostn']='$ip';print(json.dumps(value,sort_keys=True))\\\\\\\"\\\"\\nelse\\nrun_command\\\"Addmappingtobucketdata\\\"\\\"addmapping\\\"\\\"echo'$etc_hosts'|python3-c\\\\\\\"importsys,json;value=json.load(sys.stdin);item={u'$hostn':u'$ip'};value.append(item);print(json.dumps(value,sort_keys=True))\\\\\\\"\\\"\\nfi\\nnew_etc_hosts=$cmd_output\\n\\ndata='{'\\\\\\\"\\\"$etc_hosts_key\\\"\\\\\\\"':'\\\"$new_etc_hosts\\\"'}'\\nif[\\\"$new_etc_hosts\\\"=\\\"$etc_hosts\\\"];then\\necho\\\"NochangesinSMD.Notupdating\\\"\\nexit0\\nelse\\niftest-z\\\"$config_svc_token\\\";then\\nrun_command\\\"Generatingauthntokenfortalkingtoconfigbucket\\\"\\\"TokenConfigsvc\\\"\\\"curl-s-XPOST-F\\\\\\\"client_id=${authn_client_id}\\\\\\\"-F\\\\\\\"client_secret=${authn_secret}\\\\\\\"-F\\\\\\\"grant_type=client_credentials\\\\\\\"-F\\\\\\\"target_client_id=${target_client_id}\\\\\\\"${authn_endpoint}/oauth/token|sed's/,/\\\\n/g'|grep\\\\\\\"access_token\\\\\\\"|sed's/\\\\\\\"//g'|sed's/.*://g'\\\"\\nconfig_svc_token=$cmd_output\\nfi\\n\\necho\\\"ExistingMapping:$etc_hosts,NewMapping:$new_etc_hostsforexistingversion:$existing_version\\\"\\noutput=$(curl-XPOST-s\\\"http://${config_endpoint}/v1/buckets/${topology_bucket}/keys?message=updated_by_$ip\\\"-H\\\"X-Config-Bucket-Version:${existing_version}\\\"-H\\\"Content-Type:application/json\\\"--data-binary\\\"$data\\\"-H\\\"Authorization:Bearer${config_svc_token}\\\")\\nfinal_result=$?\\nfailed_message=\\\"Updatefailed\\\"\\nif[[$final_result-eq0&&\\\"$output\\\"==*\\\"$topology_bucket\\\"*&&\\\"$output\\\"==*\\\"version\\\"*&&\\\"$output\\\"==*\\\"created\\\"*&&\\\"$output\\\"==*\\\"lastUpdated\\\"*]];then\\necho$output\\nexit0\\nelse\\necho\\\"Failedtoupdateip.exiting..\\\"\\nexit1\\nfi\\nfi\\ndone\\n\"],\"cpuLimit\":\"0.2\",\"cpuRequest\":\"0.2\",\"isBootstrap\":false,\"memoryLimit\":\"256Mi\",\"memoryRequest\":\"256Mi\",\"name\":\"publish-myip\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011}}],\"isPodServiceRequired\":true,\"labels\":{\"mcs.discovery.fcp.io/enable\":\"true\"},\"name\":\"test-cluster-zk\",\"podManagementPolicy\":\"Parallel\",\"shareProcessNamespace\":false,\"sidecarContainers\":[{\"cpuLimit\":\"100m\",\"cpuRequest\":\"100m\",\"image\":\"edge.fkinternal.com/indradhanush/fk-yak-filebeat:8.11.3-fk1\",\"memoryLimit\":\"128Mi\",\"memoryRequest\":\"128Mi\",\"name\":\"filebeat\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/var/log/flipkart/yak\",\"name\":\"app-log\",\"readOnly\":false},{\"mountPath\":\"/etc/filebeat\",\"name\":\"filebeat\",\"readOnly\":false}]},{\"args\":[\"/tmp\",\"/grid/1/dfs\",\"test-cluster-bkp-0.test-cluster-bkp.test-namespace.svc.cluster.local\"],\"command\":[\"/opt/scripts/copy_backup\"],\"cpuLimit\":\"0.5\",\"cpuRequest\":\"0.5\",\"image\":\"edge.fkinternal.com/indradhanush/yak-base:2.5.3-08-rc7\",\"memoryLimit\":\"256Mi\",\"memoryRequest\":\"256Mi\",\"name\":\"backup\",\"securityContext\":{\"runAsGroup\":1011,\"runAsUser\":1011},\"volumeMounts\":[{\"mountPath\":\"/grid/1\",\"name\":\"data\",\"readOnly\":true}]}],\"size\":5,\"terminateGracePeriod\":120,\"volumeClaims\":[{\"name\":\"data\",\"storageClassName\":\"local-extreme-1a-ext4\",\"storageSize\":\"16Gi\"}],\"volumes\":[{\"name\":\"nodeinfo\",\"path\":\"/etc/nodeinfo\",\"volumeSource\":\"HostPath\"},{\"name\":\"app-log\",\"volumeSource\":\"EmptyDir\"},{\"configName\":\"filebeat\",\"name\":\"filebeat\",\"volumeSource\":\"ConfigMap\"}]}},\"fsgroup\":1011,\"isBootstrap\":false,\"serviceLabels\":{\"hbase-operator.cfg-statefulset-update/enable\":\"config-only\",\"mcs.discovery.fcp.io/enable\":\"true\"},\"tenantNamespaces\":[\"yak-tenant-test-1\",\"yak-tenant-test-2\"]}}"
	err := json.Unmarshal([]byte(hbaseclusterJson), cluster)
	if err != nil {
		fmt.Println(err)
	}
	return cluster
}
