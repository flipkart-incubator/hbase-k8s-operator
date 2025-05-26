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
	out, err := os.ReadFile("../testdata/test_hbase_tenant.json")
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
