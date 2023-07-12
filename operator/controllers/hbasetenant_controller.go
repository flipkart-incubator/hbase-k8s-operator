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
	corev1 "k8s.io/api/core/v1"
	time "time"

	appsv1 "k8s.io/api/apps/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	runtime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"

	logr "github.com/go-logr/logr"

	kvstorev1 "github.com/flipkart-incubator/hbase-k8s-operator/api/v1"
)

// HbaseTenantReconciler reconciles a HbaseTenant object
type HbaseTenantReconciler struct {
	Client client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kvstore.flipkart.com,resources=hbasetenants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kvstore.flipkart.com,resources=hbasetenants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kvstore.flipkart.com,resources=hbasetenants/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;
//+kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HbaseTenant object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *HbaseTenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("hbasetenant", req.NamespacedName)
	log.Info("Received request to reconcile")

	// Fetch the HbaseTenant instance
	hbasetenant := &kvstorev1.HbaseTenant{}
	err := r.Client.Get(ctx, req.NamespacedName, hbasetenant)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("HbaseTenant resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get HbaseTenant")
		return ctrl.Result{RequeueAfter: time.Second * 5}, err
	}

	configuration := &corev1.ConfigMap{}
	configuration.Name = "hbase-config"
	configuration.Namespace = req.Namespace
	err = r.Client.Get(ctx, req.NamespacedName, configuration)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("HbaseTenant resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get HbaseTenant")
		return ctrl.Result{RequeueAfter: time.Second * 5}, err
	}
	log.Info("ConfigMap version: ", configuration.ResourceVersion)

	svc := buildService(hbasetenant.Name, hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.ServiceLabels, hbasetenant.Spec.ServiceSelectorLabels, []kvstorev1.HbaseClusterDeployment{hbasetenant.Spec.Datanode}, true)
	ctrl.SetControllerReference(hbasetenant, svc, r.Scheme)
	result, err := reconcileService(ctx, log, hbasetenant.Namespace, svc, r.Client)
	if (ctrl.Result{}) != result || err != nil {
		return result, err
	}

	newSS := buildStatefulSet(hbasetenant.Name, hbasetenant.Namespace, hbasetenant.Spec.BaseImage, false, hbasetenant.Spec.Configuration, hbasetenant.Spec.FSGroup, hbasetenant.Spec.Datanode)
	ctrl.SetControllerReference(hbasetenant, newSS, r.Scheme)
	result, err = reconcileStatefulSet(ctx, log, hbasetenant.Namespace, newSS, hbasetenant.Spec.Datanode, r.Client)
	if (ctrl.Result{}) != result || err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HbaseTenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kvstorev1.HbaseTenant{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
