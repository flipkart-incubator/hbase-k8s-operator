package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
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
	tenantNamespace1 = "yak-tenant-test-1-ns"
	tenantNamespace2 = "yak-tenant-test-2-ns"
)

type K8sMockClient struct {
	mock.Mock
	client.Client
}

func (m *K8sMockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	args := m.Called(ctx, key, obj)
	return args.Error(0)
}

func (m *K8sMockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *K8sMockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func TestHbaseClusterReconciler_ResNotFound(t *testing.T) {
	k8sMockClient, reconciler, ctx, req := doClusterTestSetup()
	k8sMockClient.On("Get", ctx, req.NamespacedName, mock.Anything).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	k8sMockClient.AssertExpectations(t)
}

func TestHbaseClusterReconciler_ErrorGettingRes(t *testing.T) {
	k8sMockClient, reconciler, ctx, req := doClusterTestSetup()
	k8sMockClient.On("Get", ctx, req.NamespacedName, mock.Anything).Return(assert.AnError)

	result, err := reconciler.Reconcile(ctx, req)
	assert.Error(t, err)
	assert.Equal(t, ctrl.Result{RequeueAfter: time.Second * 5}, result)

	k8sMockClient.AssertExpectations(t)

}

func TestHbaseClusterReconciler_SuccessfulReconciliation_ObjectsNotFound(t *testing.T) {
	//mock hbase cluster object
	hbasecluster := getMockHbaseCluster()

	k8sMockClient, reconciler, ctx, req := doClusterTestSetup()

	deployments := []kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Journalnode, hbasecluster.Spec.Deployments.Namenode, hbasecluster.Spec.Deployments.Datanode, hbasecluster.Spec.Deployments.Hmaster}
	if hbasecluster.Spec.Deployments.Zookeeper.Size != 0 {
		deployments = append([]kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Zookeeper}, deployments...)
	}

	k8sMockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseCluster{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseCluster)
			*arg = *hbasecluster
		}).
		Return(nil)

	mockSvc := buildService(hbasecluster.Name, hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.ServiceLabels, hbasecluster.Spec.ServiceSelectorLabels, deployments, true)
	assert.Equal(t, testCluster, mockSvc.Name)
	assert.Equal(t, testCluster, mockSvc.Spec.Selector["hbasecluster_cr"])

	k8sMockClient.On("Get", ctx, req.NamespacedName, &corev1.Service{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	ctrl.SetControllerReference(hbasecluster, mockSvc, reconciler.Scheme)
	k8sMockClient.On("Create", ctx, mockSvc, []client.CreateOption(nil)).Return(nil)

	for _, namespace := range []string{tenantNamespace1, tenantNamespace2, testNamespace} {
		mockCfgHb := buildConfigMap(hbasecluster.Spec.Configuration.HbaseConfigName, hbasecluster.Name, namespace, hbasecluster.Spec.Configuration.HbaseConfig, hbasecluster.Spec.Configuration.HbaseTenantConfig, reconciler.Log)
		ctrl.SetControllerReference(hbasecluster, mockCfgHb, reconciler.Scheme)
		k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHb.Name, Namespace: mockCfgHb.Namespace}, &corev1.ConfigMap{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
		k8sMockClient.On("Create", ctx, mockCfgHb, []client.CreateOption(nil)).Return(nil)

		mockCfgHd := buildConfigMap(hbasecluster.Spec.Configuration.HadoopConfigName, hbasecluster.Name, namespace, hbasecluster.Spec.Configuration.HadoopConfig, hbasecluster.Spec.Configuration.HadoopTenantConfig, reconciler.Log)
		ctrl.SetControllerReference(hbasecluster, mockCfgHd, reconciler.Scheme)
		k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHd.Name, Namespace: mockCfgHd.Namespace}, &corev1.ConfigMap{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
		k8sMockClient.On("Create", ctx, mockCfgHd, []client.CreateOption(nil)).Return(nil)
	}

	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: hbasecluster.Spec.Deployments.Datanode.Name, Namespace: hbasecluster.Namespace}, &appsv1.StatefulSet{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))

	// only one component zk is mocked here as sts reconcile method requeues the call after sts create
	// other component's reconcile does not happen unless for previous one it ensures to have ready replica same as desired.
	if hbasecluster.Spec.Deployments.Zookeeper.IsPodServiceRequired {
		var name string
		var index int32 = 0
		for index < hbasecluster.Spec.Deployments.Zookeeper.Size {
			name = hbasecluster.Spec.Deployments.Zookeeper.Name + "-" + strconv.Itoa(int(index))
			mockJNPodSvc := buildService(name, hbasecluster.Name, hbasecluster.Namespace, nil, nil, []kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Zookeeper}, false)
			ctrl.SetControllerReference(hbasecluster, mockJNPodSvc, reconciler.Scheme)
			k8sMockClient.On("Get", ctx, types.NamespacedName{Name: name, Namespace: hbasecluster.Namespace}, &corev1.Service{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
			k8sMockClient.On("Create", ctx, mockJNPodSvc, []client.CreateOption(nil)).Return(nil)
			index += 1
		}
	}

	mockStsZK := buildStatefulSet(hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.BaseImage,
		hbasecluster.Spec.IsBootstrap, hbasecluster.Spec.Configuration, "",
		hbasecluster.Spec.FSGroup, hbasecluster.Spec.Deployments.Zookeeper, reconciler.Log, true)
	ctrl.SetControllerReference(hbasecluster, mockStsZK, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: hbasecluster.Spec.Deployments.Zookeeper.Name, Namespace: hbasecluster.Namespace}, &appsv1.StatefulSet{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mockStsZK, []client.CreateOption(nil)).Return(nil)

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, result)

	// AssertExpectations asserts that everything specified with On and Return was in fact called as expected.
	k8sMockClient.AssertExpectations(t)
}

func TestHbaseClusterReconciler_SuccessfulReconciliation_AllObjectsFound(t *testing.T) {
	//mock hbase cluster object
	hbasecluster := getMockHbaseCluster()

	k8sMockClient, reconciler, ctx, req := doClusterTestSetup()

	deployments := []kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Journalnode, hbasecluster.Spec.Deployments.Namenode, hbasecluster.Spec.Deployments.Datanode, hbasecluster.Spec.Deployments.Hmaster}
	if hbasecluster.Spec.Deployments.Zookeeper.Size != 0 {
		deployments = append([]kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Zookeeper}, deployments...)
	}

	k8sMockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseCluster{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseCluster)
			*arg = *hbasecluster
		}).
		Return(nil)

	mockSvc := buildService(hbasecluster.Name, hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.ServiceLabels, hbasecluster.Spec.ServiceSelectorLabels, deployments, true)
	assert.Equal(t, testCluster, mockSvc.Name)
	assert.Equal(t, testCluster, mockSvc.Spec.Selector["hbasecluster_cr"])

	ctrl.SetControllerReference(hbasecluster, mockSvc, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, req.NamespacedName, &corev1.Service{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.Service)
			*arg = *mockSvc
		}).
		Return(nil)
	k8sMockClient.On("Update", ctx, mockSvc, []client.UpdateOption(nil)).Return(nil)

	for _, namespace := range []string{tenantNamespace1, tenantNamespace2, testNamespace} {
		mockCfgHb := buildConfigMap(hbasecluster.Spec.Configuration.HbaseConfigName, hbasecluster.Name, namespace, hbasecluster.Spec.Configuration.HbaseConfig, hbasecluster.Spec.Configuration.HbaseTenantConfig, reconciler.Log)
		ctrl.SetControllerReference(hbasecluster, mockCfgHb, reconciler.Scheme)
		k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHb.Name, Namespace: mockCfgHb.Namespace}, &corev1.ConfigMap{}).
			Run(func(args mock.Arguments) {
				arg := args.Get(2).(*corev1.ConfigMap)
				*arg = *mockCfgHb
			}).
			Return(nil)
		k8sMockClient.On("Update", ctx, mockCfgHb, []client.UpdateOption(nil)).Return(nil)

		mockCfgHd := buildConfigMap(hbasecluster.Spec.Configuration.HadoopConfigName, hbasecluster.Name, namespace, hbasecluster.Spec.Configuration.HadoopConfig, hbasecluster.Spec.Configuration.HadoopTenantConfig, reconciler.Log)
		ctrl.SetControllerReference(hbasecluster, mockCfgHd, reconciler.Scheme)
		k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHd.Name, Namespace: mockCfgHd.Namespace}, &corev1.ConfigMap{}).
			Run(func(args mock.Arguments) {
				arg := args.Get(2).(*corev1.ConfigMap)
				*arg = *mockCfgHd
			}).
			Return(nil)
		k8sMockClient.On("Update", ctx, mockCfgHd, []client.UpdateOption(nil)).Return(nil)
	}

	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: hbasecluster.Spec.Deployments.Datanode.Name, Namespace: hbasecluster.Namespace}, &appsv1.StatefulSet{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))

	// only one component zk is mocked here as sts reconcile method requeues the call after sts create
	// other component's reconcile does not happen unless for previous one it ensures to have ready replica same as desired.
	if hbasecluster.Spec.Deployments.Zookeeper.IsPodServiceRequired {
		var name string
		var index int32 = 0
		for index < hbasecluster.Spec.Deployments.Zookeeper.Size {
			name = hbasecluster.Spec.Deployments.Zookeeper.Name + "-" + strconv.Itoa(int(index))
			mockZKPodSvc := buildService(name, hbasecluster.Name, hbasecluster.Namespace, nil, nil, []kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Zookeeper}, false)
			ctrl.SetControllerReference(hbasecluster, mockZKPodSvc, reconciler.Scheme)
			k8sMockClient.On("Get", ctx, types.NamespacedName{Name: name, Namespace: hbasecluster.Namespace}, &corev1.Service{}).
				Run(func(args mock.Arguments) {
					arg := args.Get(2).(*corev1.Service)
					*arg = *mockZKPodSvc
				}).
				Return(nil)
			k8sMockClient.On("Update", ctx, mockZKPodSvc, []client.UpdateOption(nil)).Return(nil)
			index += 1
		}
	}

	mockStsZK := buildStatefulSet(hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.BaseImage,
		hbasecluster.Spec.IsBootstrap, hbasecluster.Spec.Configuration, "",
		hbasecluster.Spec.FSGroup, hbasecluster.Spec.Deployments.Zookeeper, reconciler.Log, true)
	ctrl.SetControllerReference(hbasecluster, mockStsZK, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: hbasecluster.Spec.Deployments.Zookeeper.Name, Namespace: hbasecluster.Namespace}, &appsv1.StatefulSet{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*appsv1.StatefulSet)
			*arg = *mockStsZK
		}).
		Return(nil)
	k8sMockClient.On("Update", ctx, mockStsZK, []client.UpdateOption(nil)).Return(nil)

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: time.Second * 20}, result)

	// AssertExpectations asserts that everything specified with On and Return was in fact called as expected.
	k8sMockClient.AssertExpectations(t)
}

// This test will fail if ran as individual as it depends on hashstore impl.
// when run along with other tests it will pass as hashstore will have values filled and update method will not be called.
func TestHbaseClusterReconciler_SuccessfulReconciliation_AllObjectsFoundRestFlow(t *testing.T) {
	//mock hbase cluster object
	hbasecluster := getMockHbaseCluster()

	k8sMockClient, reconciler, ctx, req := doClusterTestSetup()

	deployments := []kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Journalnode, hbasecluster.Spec.Deployments.Namenode, hbasecluster.Spec.Deployments.Datanode, hbasecluster.Spec.Deployments.Hmaster}
	if hbasecluster.Spec.Deployments.Zookeeper.Size != 0 {
		deployments = append([]kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Zookeeper}, deployments...)
	}

	k8sMockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseCluster{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseCluster)
			*arg = *hbasecluster
		}).
		Return(nil)

	mockSvc := buildService(hbasecluster.Name, hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.ServiceLabels, hbasecluster.Spec.ServiceSelectorLabels, deployments, true)
	assert.Equal(t, testCluster, mockSvc.Name)
	assert.Equal(t, testCluster, mockSvc.Spec.Selector["hbasecluster_cr"])

	ctrl.SetControllerReference(hbasecluster, mockSvc, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, req.NamespacedName, &corev1.Service{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.Service)
			*arg = *mockSvc
		}).
		Return(nil)

	for _, namespace := range []string{tenantNamespace1, tenantNamespace2, testNamespace} {
		mockCfgHb := buildConfigMap(hbasecluster.Spec.Configuration.HbaseConfigName, hbasecluster.Name, namespace, hbasecluster.Spec.Configuration.HbaseConfig, hbasecluster.Spec.Configuration.HbaseTenantConfig, reconciler.Log)
		ctrl.SetControllerReference(hbasecluster, mockCfgHb, reconciler.Scheme)
		k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHb.Name, Namespace: mockCfgHb.Namespace}, &corev1.ConfigMap{}).
			Run(func(args mock.Arguments) {
				arg := args.Get(2).(*corev1.ConfigMap)
				*arg = *mockCfgHb
			}).
			Return(nil)

		mockCfgHd := buildConfigMap(hbasecluster.Spec.Configuration.HadoopConfigName, hbasecluster.Name, namespace, hbasecluster.Spec.Configuration.HadoopConfig, hbasecluster.Spec.Configuration.HadoopTenantConfig, reconciler.Log)
		ctrl.SetControllerReference(hbasecluster, mockCfgHd, reconciler.Scheme)
		k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHd.Name, Namespace: mockCfgHd.Namespace}, &corev1.ConfigMap{}).
			Run(func(args mock.Arguments) {
				arg := args.Get(2).(*corev1.ConfigMap)
				*arg = *mockCfgHd
			}).
			Return(nil)
	}

	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: hbasecluster.Spec.Deployments.Datanode.Name, Namespace: hbasecluster.Namespace}, &appsv1.StatefulSet{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))

	// only one component zk is mocked here as sts reconcile method requeues the call after sts create
	// other component's reconcile does not happen unless for previous one it ensures to have ready replica same as desired.
	if hbasecluster.Spec.Deployments.Zookeeper.IsPodServiceRequired {
		var name string
		var index int32 = 0
		for index < hbasecluster.Spec.Deployments.Zookeeper.Size {
			name = hbasecluster.Spec.Deployments.Zookeeper.Name + "-" + strconv.Itoa(int(index))
			mockZKPodSvc := buildService(name, hbasecluster.Name, hbasecluster.Namespace, nil, nil, []kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Zookeeper}, false)
			ctrl.SetControllerReference(hbasecluster, mockZKPodSvc, reconciler.Scheme)
			k8sMockClient.On("Get", ctx, types.NamespacedName{Name: name, Namespace: hbasecluster.Namespace}, &corev1.Service{}).
				Run(func(args mock.Arguments) {
					arg := args.Get(2).(*corev1.Service)
					*arg = *mockZKPodSvc
				}).
				Return(nil)
			index += 1
		}
	}

	mockStsZK := buildStatefulSet(hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.BaseImage,
		hbasecluster.Spec.IsBootstrap, hbasecluster.Spec.Configuration, "",
		hbasecluster.Spec.FSGroup, hbasecluster.Spec.Deployments.Zookeeper, reconciler.Log, true)
	ctrl.SetControllerReference(hbasecluster, mockStsZK, reconciler.Scheme)
	mockStsZK.Status.ReadyReplicas = hbasecluster.Spec.Deployments.Zookeeper.Size
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: hbasecluster.Spec.Deployments.Zookeeper.Name, Namespace: hbasecluster.Namespace}, &appsv1.StatefulSet{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*appsv1.StatefulSet)
			*arg = *mockStsZK
		}).
		Return(nil)

	mockPdbZk := buildPodDisruptionBudget(hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.Deployments.Zookeeper, reconciler.Log)
	ctrl.SetControllerReference(hbasecluster, mockPdbZk, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockPdbZk.Name, Namespace: mockPdbZk.Namespace}, &policyv1.PodDisruptionBudget{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mockPdbZk, []client.CreateOption(nil)).Return(nil)

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, result)

	// AssertExpectations asserts that everything specified with On and Return was in fact called as expected.
	k8sMockClient.AssertExpectations(t)
}

func getMockClientAndReconciler() (*K8sMockClient, *HbaseClusterReconciler) {
	k8sMockClient := new(K8sMockClient)
	scheme := runtime.NewScheme()
	_ = kvstorev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	reconciler := &HbaseClusterReconciler{
		Client: k8sMockClient,
		Log:    ctrl.Log.WithName("controllers").WithName("HbaseCluster"),
		Scheme: scheme,
	}
	return k8sMockClient, reconciler
}

func doClusterTestSetup() (*K8sMockClient, *HbaseClusterReconciler, context.Context, ctrl.Request) {
	k8sMockClient, reconciler := getMockClientAndReconciler()
	ctx := context.TODO()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testCluster,
			Namespace: testNamespace,
		},
	}
	return k8sMockClient, reconciler, ctx, req
}

func getMockHbaseCluster() *kvstorev1.HbaseCluster {
	out, err := os.ReadFile("../testdata/testhbasecluster.json")
	if err != nil {
		fmt.Println(err)
	}
	cluster := &kvstorev1.HbaseCluster{}
	unmarshalErr := json.Unmarshal(out, cluster)
	if unmarshalErr != nil {
		fmt.Println(unmarshalErr)
	}
	return cluster
}
