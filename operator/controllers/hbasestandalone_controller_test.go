package controllers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	standaloneTestName      = "test-standalone"
	standaloneTestNamespace = "test-standalone-ns"
)

// getMockClientAndStandaloneReconciler creates a mock K8s client and an HbaseStandaloneReconciler wired together for unit testing.
func getMockClientAndStandaloneReconciler() (*K8sMockClient, *HbaseStandaloneReconciler) {
	mockClient := new(K8sMockClient)
	scheme := runtime.NewScheme()
	_ = kvstorev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	reconciler := &HbaseStandaloneReconciler{
		Client: mockClient,
		Log:    ctrl.Log.WithName("controllers").WithName("HbaseStandalone"),
		Scheme: scheme,
	}
	return mockClient, reconciler
}

// doStandaloneTestSetup initialises the mock client, reconciler, context, and a standard reconcile request for standalone tests.
func doStandaloneTestSetup() (*K8sMockClient, *HbaseStandaloneReconciler, context.Context, ctrl.Request) {
	k8sMockClient, reconciler := getMockClientAndStandaloneReconciler()
	ctx := context.TODO()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      standaloneTestName,
			Namespace: standaloneTestNamespace,
		},
	}
	return k8sMockClient, reconciler, ctx, req
}

// populateStandaloneHashStore pre-populates the global hashStore with expected hashes for all standalone child resources,
// enabling "rest flow" tests to verify that no unnecessary updates are triggered when nothing has changed.
func populateStandaloneHashStore(standalone *kvstorev1.HbaseStandalone, reconciler *HbaseStandaloneReconciler) {
	log := reconciler.Log

	svc := buildService(standalone.Name, standalone.Name, standalone.Namespace, standalone.Spec.ServiceLabels, standalone.Spec.ServiceSelectorLabels, []kvstorev1.HbaseClusterDeployment{standalone.Spec.Standalone}, true)
	ctrl.SetControllerReference(standalone, svc, reconciler.Scheme)
	svcMarshal, _ := json.Marshal(svc.Spec)
	hashStore["svc-"+svc.Name] = asSha256(svcMarshal)

	cfg := buildConfigMap(standalone.Spec.Configuration.HbaseConfigName, standalone.Name, standalone.Namespace, standalone.Spec.Configuration.HbaseConfig, standalone.Spec.Configuration.HbaseTenantConfig, log)
	ctrl.SetControllerReference(standalone, cfg, reconciler.Scheme)
	cfgMarshal, _ := json.Marshal(cfg.Data)
	hashStore["cfg-"+cfg.Name+cfg.Namespace] = asSha256(cfgMarshal)

	cfg2 := buildConfigMap(standalone.Spec.Configuration.HadoopConfigName, standalone.Name, standalone.Namespace, standalone.Spec.Configuration.HadoopConfig, standalone.Spec.Configuration.HadoopTenantConfig, log)
	ctrl.SetControllerReference(standalone, cfg2, reconciler.Scheme)
	cfg2Marshal, _ := json.Marshal(cfg2.Data)
	hashStore["cfg-"+cfg2.Name+cfg2.Namespace] = asSha256(cfg2Marshal)

	newSS := buildStatefulSet(standalone.Name, standalone.Namespace, standalone.Spec.BaseImage,
		false, standalone.Spec.Configuration, cfg.ResourceVersion, standalone.Spec.FSGroup,
		standalone.Spec.Standalone, log, true)
	ctrl.SetControllerReference(standalone, newSS, reconciler.Scheme)
	ssMarshal, _ := json.Marshal(newSS)
	hashStore["ss-"+newSS.Name] = asSha256(ssMarshal)
}

// TestHbaseStandaloneReconciler_ResNotFound verifies that the reconciler returns success with no requeue when the HbaseStandalone CR is not found (e.g., deleted).
func TestHbaseStandaloneReconciler_ResNotFound(t *testing.T) {
	resetHashStore()
	k8sMockClient, reconciler, ctx, req := doStandaloneTestSetup()
	k8sMockClient.On("Get", ctx, req.NamespacedName, mock.Anything).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	k8sMockClient.AssertExpectations(t)
}

// TestHbaseStandaloneReconciler_ErrorGettingRes verifies that a non-NotFound Get error returns an error and schedules a requeue after 5 seconds.
func TestHbaseStandaloneReconciler_ErrorGettingRes(t *testing.T) {
	resetHashStore()
	k8sMockClient, reconciler, ctx, req := doStandaloneTestSetup()
	k8sMockClient.On("Get", ctx, req.NamespacedName, mock.Anything).Return(assert.AnError)

	result, err := reconciler.Reconcile(ctx, req)
	assert.Error(t, err)
	assert.Equal(t, ctrl.Result{RequeueAfter: time.Second * 5}, result)

	k8sMockClient.AssertExpectations(t)
}

// TestHbaseStandaloneReconciler_SuccessfulReconciliation_ObjectsNotFound verifies that all child resources
// (Service, HBase ConfigMap, Hadoop ConfigMap, StatefulSet) are created when none exist yet.
func TestHbaseStandaloneReconciler_SuccessfulReconciliation_ObjectsNotFound(t *testing.T) {
	resetHashStore()
	standalone := getMockHbaseStandalone()

	k8sMockClient, reconciler, ctx, req := doStandaloneTestSetup()

	k8sMockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseStandalone{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseStandalone)
			*arg = *standalone
		}).
		Return(nil)

	mockSvc := buildService(standalone.Name, standalone.Name, standalone.Namespace, standalone.Spec.ServiceLabels, standalone.Spec.ServiceSelectorLabels, []kvstorev1.HbaseClusterDeployment{standalone.Spec.Standalone}, true)
	ctrl.SetControllerReference(standalone, mockSvc, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockSvc.Name, Namespace: standalone.Namespace}, &corev1.Service{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mockSvc, []client.CreateOption(nil)).Return(nil)

	mockCfgHb := buildConfigMap(standalone.Spec.Configuration.HbaseConfigName, standalone.Name, standalone.Namespace, standalone.Spec.Configuration.HbaseConfig, standalone.Spec.Configuration.HbaseTenantConfig, reconciler.Log)
	ctrl.SetControllerReference(standalone, mockCfgHb, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHb.Name, Namespace: mockCfgHb.Namespace}, &corev1.ConfigMap{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mockCfgHb, []client.CreateOption(nil)).Return(nil)

	mockCfgHd := buildConfigMap(standalone.Spec.Configuration.HadoopConfigName, standalone.Name, standalone.Namespace, standalone.Spec.Configuration.HadoopConfig, standalone.Spec.Configuration.HadoopTenantConfig, reconciler.Log)
	ctrl.SetControllerReference(standalone, mockCfgHd, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHd.Name, Namespace: mockCfgHd.Namespace}, &corev1.ConfigMap{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mockCfgHd, []client.CreateOption(nil)).Return(nil)

	mockSts := buildStatefulSet(standalone.Name, standalone.Namespace, standalone.Spec.BaseImage,
		false, standalone.Spec.Configuration, mockCfgHd.ResourceVersion, standalone.Spec.FSGroup,
		standalone.Spec.Standalone, reconciler.Log, true)
	ctrl.SetControllerReference(standalone, mockSts, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: standalone.Spec.Standalone.Name, Namespace: standalone.Namespace}, &appsv1.StatefulSet{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mockSts, []client.CreateOption(nil)).Return(nil)

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, result)

	k8sMockClient.AssertExpectations(t)
}

// TestHbaseStandaloneReconciler_SuccessfulReconciliation_AllObjectsFoundRestFlow verifies the "rest flow":
// when all resources exist and their hashes match the cached values, no updates are issued; only the PDB is created if missing.
func TestHbaseStandaloneReconciler_SuccessfulReconciliation_AllObjectsFoundRestFlow(t *testing.T) {
	resetHashStore()
	standalone := getMockHbaseStandalone()

	k8sMockClient, reconciler, ctx, req := doStandaloneTestSetup()

	populateStandaloneHashStore(standalone, reconciler)

	k8sMockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseStandalone{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseStandalone)
			*arg = *standalone
		}).
		Return(nil)

	mockSvc := buildService(standalone.Name, standalone.Name, standalone.Namespace, standalone.Spec.ServiceLabels, standalone.Spec.ServiceSelectorLabels, []kvstorev1.HbaseClusterDeployment{standalone.Spec.Standalone}, true)
	ctrl.SetControllerReference(standalone, mockSvc, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockSvc.Name, Namespace: standalone.Namespace}, &corev1.Service{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.Service)
			*arg = *mockSvc
		}).
		Return(nil)

	mockCfgHb := buildConfigMap(standalone.Spec.Configuration.HbaseConfigName, standalone.Name, standalone.Namespace, standalone.Spec.Configuration.HbaseConfig, standalone.Spec.Configuration.HbaseTenantConfig, reconciler.Log)
	ctrl.SetControllerReference(standalone, mockCfgHb, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHb.Name, Namespace: mockCfgHb.Namespace}, &corev1.ConfigMap{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.ConfigMap)
			*arg = *mockCfgHb
		}).
		Return(nil)

	mockCfgHd := buildConfigMap(standalone.Spec.Configuration.HadoopConfigName, standalone.Name, standalone.Namespace, standalone.Spec.Configuration.HadoopConfig, standalone.Spec.Configuration.HadoopTenantConfig, reconciler.Log)
	ctrl.SetControllerReference(standalone, mockCfgHd, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHd.Name, Namespace: mockCfgHd.Namespace}, &corev1.ConfigMap{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.ConfigMap)
			*arg = *mockCfgHd
		}).
		Return(nil)

	mockSts := buildStatefulSet(standalone.Name, standalone.Namespace, standalone.Spec.BaseImage,
		false, standalone.Spec.Configuration, mockCfgHd.ResourceVersion, standalone.Spec.FSGroup,
		standalone.Spec.Standalone, reconciler.Log, true)
	ctrl.SetControllerReference(standalone, mockSts, reconciler.Scheme)
	mockSts.Status.ReadyReplicas = standalone.Spec.Standalone.Size
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: standalone.Spec.Standalone.Name, Namespace: standalone.Namespace}, &appsv1.StatefulSet{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*appsv1.StatefulSet)
			*arg = *mockSts
		}).
		Return(nil)

	mockPdb := buildPodDisruptionBudget(standalone.Name, standalone.Namespace, standalone.Spec.Standalone, reconciler.Log)
	ctrl.SetControllerReference(standalone, mockPdb, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockPdb.Name, Namespace: mockPdb.Namespace}, &policyv1.PodDisruptionBudget{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mockPdb, []client.CreateOption(nil)).Return(nil)

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, result)

	k8sMockClient.AssertExpectations(t)
}

// TestHbaseStandaloneReconciler_InvalidConfig_EventPublish verifies that invalid XML in HBase config
// causes the reconciler to publish a ConfigValidateFailed event and return an error.
func TestHbaseStandaloneReconciler_InvalidConfig_EventPublish(t *testing.T) {
	resetHashStore()
	standalone := getMockHbaseStandalone()
	standalone.Spec.Configuration.HbaseConfig["hbase-site.xml"] = "not-valid-xml<><>"

	k8sMockClient, reconciler, ctx, req := doStandaloneTestSetup()

	k8sMockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseStandalone{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseStandalone)
			*arg = *standalone
		}).
		Return(nil)

	mockSvc := buildService(standalone.Name, standalone.Name, standalone.Namespace, standalone.Spec.ServiceLabels, standalone.Spec.ServiceSelectorLabels, []kvstorev1.HbaseClusterDeployment{standalone.Spec.Standalone}, true)
	ctrl.SetControllerReference(standalone, mockSvc, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockSvc.Name, Namespace: standalone.Namespace}, &corev1.Service{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mockSvc, []client.CreateOption(nil)).Return(nil)

	k8sMockClient.On("Get", ctx, types.NamespacedName{Namespace: standalone.Namespace, Name: "ConfigValidateFailed"}, &corev1.Event{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mock.Anything, []client.CreateOption(nil)).Return(nil)

	result, err := reconciler.Reconcile(ctx, req)
	assert.Error(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	k8sMockClient.AssertExpectations(t)
}

// TestHbaseStandaloneReconciler_ServiceCreateError verifies that a Service creation failure returns an error and triggers a requeue.
func TestHbaseStandaloneReconciler_ServiceCreateError(t *testing.T) {
	resetHashStore()
	standalone := getMockHbaseStandalone()

	k8sMockClient, reconciler, ctx, req := doStandaloneTestSetup()

	k8sMockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseStandalone{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseStandalone)
			*arg = *standalone
		}).
		Return(nil)

	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: standalone.Name, Namespace: standalone.Namespace}, &corev1.Service{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))
	k8sMockClient.On("Create", ctx, mock.Anything, []client.CreateOption(nil)).Return(assert.AnError)

	k8sMockClient.On("Get", ctx, mock.Anything, &corev1.Event{}).Return(errors.NewNotFound(schema.GroupResource{}, req.Name))

	result, err := reconciler.Reconcile(ctx, req)
	assert.Error(t, err)
	assert.Equal(t, ctrl.Result{RequeueAfter: time.Second * 5}, result)
}

// TestHbaseStandaloneReconciler_NoPDB verifies that reconciliation completes successfully when PodDisruptionBudget is nil,
// skipping PDB creation entirely.
func TestHbaseStandaloneReconciler_NoPDB(t *testing.T) {
	resetHashStore()
	standalone := getMockHbaseStandalone()
	standalone.Spec.Standalone.PodDisruptionBudget = nil

	k8sMockClient, reconciler, ctx, req := doStandaloneTestSetup()

	populateStandaloneHashStore(standalone, reconciler)

	k8sMockClient.On("Get", ctx, req.NamespacedName, &kvstorev1.HbaseStandalone{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*kvstorev1.HbaseStandalone)
			*arg = *standalone
		}).
		Return(nil)

	mockSvc := buildService(standalone.Name, standalone.Name, standalone.Namespace, standalone.Spec.ServiceLabels, standalone.Spec.ServiceSelectorLabels, []kvstorev1.HbaseClusterDeployment{standalone.Spec.Standalone}, true)
	ctrl.SetControllerReference(standalone, mockSvc, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockSvc.Name, Namespace: standalone.Namespace}, &corev1.Service{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.Service)
			*arg = *mockSvc
		}).
		Return(nil)

	mockCfgHb := buildConfigMap(standalone.Spec.Configuration.HbaseConfigName, standalone.Name, standalone.Namespace, standalone.Spec.Configuration.HbaseConfig, standalone.Spec.Configuration.HbaseTenantConfig, reconciler.Log)
	ctrl.SetControllerReference(standalone, mockCfgHb, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHb.Name, Namespace: mockCfgHb.Namespace}, &corev1.ConfigMap{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.ConfigMap)
			*arg = *mockCfgHb
		}).
		Return(nil)

	mockCfgHd := buildConfigMap(standalone.Spec.Configuration.HadoopConfigName, standalone.Name, standalone.Namespace, standalone.Spec.Configuration.HadoopConfig, standalone.Spec.Configuration.HadoopTenantConfig, reconciler.Log)
	ctrl.SetControllerReference(standalone, mockCfgHd, reconciler.Scheme)
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: mockCfgHd.Name, Namespace: mockCfgHd.Namespace}, &corev1.ConfigMap{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.ConfigMap)
			*arg = *mockCfgHd
		}).
		Return(nil)

	mockSts := buildStatefulSet(standalone.Name, standalone.Namespace, standalone.Spec.BaseImage,
		false, standalone.Spec.Configuration, mockCfgHd.ResourceVersion, standalone.Spec.FSGroup,
		standalone.Spec.Standalone, reconciler.Log, true)
	ctrl.SetControllerReference(standalone, mockSts, reconciler.Scheme)
	mockSts.Status.ReadyReplicas = standalone.Spec.Standalone.Size
	k8sMockClient.On("Get", ctx, types.NamespacedName{Name: standalone.Spec.Standalone.Name, Namespace: standalone.Namespace}, &appsv1.StatefulSet{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*appsv1.StatefulSet)
			*arg = *mockSts
		}).
		Return(nil)

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	k8sMockClient.AssertExpectations(t)
}