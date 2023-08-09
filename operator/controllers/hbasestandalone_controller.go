/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	context "context"
	time "time"

	logr "github.com/go-logr/logr"
	errors "k8s.io/apimachinery/pkg/api/errors"
	runtime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"

	kvstorev1 "github.com/flipkart-incubator/hbase-k8s-operator/api/v1"
)

// HbaseStandaloneReconciler reconciles a HbaseStandalone object
type HbaseStandaloneReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kvstore.flipkart.com,resources=hbasestandalones,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kvstore.flipkart.com,resources=hbasestandalones/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kvstore.flipkart.com,resources=hbasestandalones/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;
//+kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *HbaseStandaloneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("hbasestandalone", req.NamespacedName).WithValues("requestid", time.Now().Unix())
	log.Info("Received request to reconcile")

	hbasestandalone := &kvstorev1.HbaseStandalone{}
	err := r.Client.Get(ctx, req.NamespacedName, hbasestandalone)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			log.Info("HbaseStandalone resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get HbaseStandalone")
		return ctrl.Result{RequeueAfter: time.Second * 5}, err
	}

	svc := buildService(hbasestandalone.Name, hbasestandalone.Name, hbasestandalone.Namespace, hbasestandalone.Spec.ServiceLabels, hbasestandalone.Spec.ServiceSelectorLabels, []kvstorev1.HbaseClusterDeployment{hbasestandalone.Spec.Standalone}, true)
	ctrl.SetControllerReference(hbasestandalone, svc, r.Scheme)

	result, err := reconcileService(ctx, log, hbasestandalone.Namespace, svc, r.Client)
	if (ctrl.Result{}) != result || err != nil {
		return result, err
	}

	result, err = validateConfiguration(ctx, log, hbasestandalone.Namespace, hbasestandalone.Spec.Configuration, r.Client)
	if err != nil {
		publishEvent(ctx, log, hbasestandalone.Namespace, "ConfigValidateFailed", err.Error(), "Warning", "ConfigMap", r.Client)
		log.Error(err, "Failed to validate configuration")
		return result, err
	}

	cfg := buildConfigMap(hbasestandalone.Spec.Configuration.HbaseConfigName, hbasestandalone.Name, hbasestandalone.Namespace, hbasestandalone.Spec.Configuration.HbaseConfig, hbasestandalone.Spec.Configuration.HbaseTenantConfig, log)
	ctrl.SetControllerReference(hbasestandalone, cfg, r.Scheme)
	result, err = reconcileConfigMap(ctx, log, hbasestandalone.Namespace, cfg, r.Client)
	if (ctrl.Result{}) != result || err != nil {
		return result, err
	}

	cfg = buildConfigMap(hbasestandalone.Spec.Configuration.HadoopConfigName, hbasestandalone.Name, hbasestandalone.Namespace, hbasestandalone.Spec.Configuration.HadoopConfig, hbasestandalone.Spec.Configuration.HadoopTenantConfig, log)
	ctrl.SetControllerReference(hbasestandalone, cfg, r.Scheme)
	result, err = reconcileConfigMap(ctx, log, hbasestandalone.Namespace, cfg, r.Client)
	if (ctrl.Result{}) != result || err != nil {
		return result, err
	}

	newSS := buildStatefulSet(hbasestandalone.Name, hbasestandalone.Namespace, hbasestandalone.Spec.BaseImage,
		false, hbasestandalone.Spec.Configuration, cfg.ResourceVersion, hbasestandalone.Spec.FSGroup,
		hbasestandalone.Spec.Standalone, log)
	ctrl.SetControllerReference(hbasestandalone, newSS, r.Scheme)
	result, err = reconcileStatefulSet(ctx, log, hbasestandalone.Namespace, newSS, hbasestandalone.Spec.Standalone, r.Client)
	if (ctrl.Result{}) != result || err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HbaseStandaloneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kvstorev1.HbaseStandalone{}).
		Complete(r)
}
