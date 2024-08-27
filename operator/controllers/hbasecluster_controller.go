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
	sha256 "crypto/sha256"
	fmt "fmt"
	strconv "strconv"
	time "time"

	appsv1 "k8s.io/api/apps/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	runtime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"

	logr "github.com/go-logr/logr"

	kvstorev1 "github.com/flipkart-incubator/hbase-k8s-operator/api/v1"
)

// HbaseClusterReconciler reconciles a HbaseCluster object
type HbaseClusterReconciler struct {
	Client client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func asSha256(o interface{}) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", o)))

	return fmt.Sprintf("%x", h.Sum(nil))
}

var hashStore = make(map[string]string)

//+kubebuilder:rbac:groups=kvstore.flipkart.com,resources=hbaseclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kvstore.flipkart.com,resources=hbaseclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kvstore.flipkart.com,resources=hbaseclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;
//+kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *HbaseClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("hbasecluster", req.NamespacedName).WithValues("requestid", time.Now().Unix())
	log.Info("Received request to reconcile")

	hbasecluster := &kvstorev1.HbaseCluster{}
	err := r.Client.Get(ctx, req.NamespacedName, hbasecluster)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			log.Info("HbaseCluster resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get HbaseCluster")
		return ctrl.Result{RequeueAfter: time.Second * 5}, err
	}

	// Check if the configmap reconciliation is enabled from cluster, this is controlled from serviceLabels
	// If the desired service label is set to true, then we will reconcile the configmaps
	value, exists := hbasecluster.Spec.ServiceLabels[RECONCILE_CONFIG_LABEL]

	deployments := []kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Journalnode, hbasecluster.Spec.Deployments.Namenode, hbasecluster.Spec.Deployments.Datanode, hbasecluster.Spec.Deployments.Hmaster}
	if hbasecluster.Spec.Deployments.Zookeeper.Size != 0 {
		deployments = append([]kvstorev1.HbaseClusterDeployment{hbasecluster.Spec.Deployments.Zookeeper}, deployments...)
	}

	svc := buildService(hbasecluster.Name, hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.ServiceLabels, hbasecluster.Spec.ServiceSelectorLabels, deployments, true)
	ctrl.SetControllerReference(hbasecluster, svc, r.Scheme)
	result, err := reconcileService(ctx, log, hbasecluster.Namespace, svc, r.Client)
	if (ctrl.Result{}) != result || err != nil {
		return result, err
	}

	result, err = validateConfiguration(ctx, log, hbasecluster.Namespace, hbasecluster.Spec.Configuration, r.Client)
	if err != nil {
		publishEvent(ctx, log, hbasecluster.Namespace, "ConfigValidateFailed", err.Error(), "Warning", "ConfigMap", r.Client)
		log.Error(err, "Failed to validate configuration")
		return result, err
	}

	namespaces := hbasecluster.Spec.TenantNamespaces
	namespaces = append(namespaces, hbasecluster.Namespace)
	// if namespaces are not specified under tenantNamespaces of HbaseClusterDeploymentSpec, then use the namespace of the HbaseCluster only
	// make sure "hbase.operator.tenant-config/enable" is set to true in HbaseTenant deployment, under service-Labels
	for _, namespace := range namespaces {
		cfg := buildConfigMap(hbasecluster.Spec.Configuration.HbaseConfigName, hbasecluster.Name, namespace, hbasecluster.Spec.Configuration.HbaseConfig, hbasecluster.Spec.Configuration.HbaseTenantConfig, log)
		ctrl.SetControllerReference(hbasecluster, cfg, r.Scheme)
		result, err = reconcileConfigMap(ctx, log, namespace, cfg, r.Client)
		if (ctrl.Result{}) != result || err != nil {
			return result, err
		}

		cfg = buildConfigMap(hbasecluster.Spec.Configuration.HadoopConfigName, hbasecluster.Name, namespace, hbasecluster.Spec.Configuration.HadoopConfig, hbasecluster.Spec.Configuration.HadoopTenantConfig, log)
		ctrl.SetControllerReference(hbasecluster, cfg, r.Scheme)
		result, err = reconcileConfigMap(ctx, log, namespace, cfg, r.Client)
		if (ctrl.Result{}) != result || err != nil {
			return result, err
		}
	}

	// gets the resource version of the configmap if it has create-time annotation - else returns nil
	// this is to make deployment backward compatible with v1 - else upon new operator deployment, entire cluster will
	// be restarted at the sametime - which is not desirable.
	resourceVersionOfHbaseConfigMap := ""
	// If label exists and set to true, then validate the config format and reconcile afterwards
	if exists && (value == "true" || value == "yes") {
		// Get the resource version of the configmap, if it is v2 then we will use the resource version
		log.Info("Configmap recon enabled, new Resource Version will be used")
		resourceVersionOfHbaseConfigMap = getCfgResourceVersionIfV2OrNil(log, r.Client, ctx,
			hbasecluster.Spec.Configuration.HbaseConfigName, hbasecluster.Namespace)
	} else {
		// if Restart of statefulSet is turned off, use existing Annotation as reference
		log.Info("Configmap recon not enabled, getting existing resource version")
		resourceVersionOfHbaseConfigMap = getExistingAnnotationOfClusterStatefulSet(log, r.Client, ctx, hbasecluster)
	}

	for _, d := range deployments {
		//TODO: Error handling
		if d.IsPodServiceRequired {
			var name string
			var index int32 = 0
			for index < d.Size {
				name = d.Name + "-" + strconv.Itoa(int(index))
				svc = buildService(name, hbasecluster.Name, hbasecluster.Namespace, nil, nil, []kvstorev1.HbaseClusterDeployment{d}, false)
				ctrl.SetControllerReference(hbasecluster, svc, r.Scheme)
				result, err = reconcileService(ctx, log, hbasecluster.Namespace, svc, r.Client)
				if (ctrl.Result{}) != result || err != nil {
					return result, err
				}
				index += 1
			}
		}

		newSS := buildStatefulSet(hbasecluster.Name, hbasecluster.Namespace, hbasecluster.Spec.BaseImage,
			hbasecluster.Spec.IsBootstrap, hbasecluster.Spec.Configuration, resourceVersionOfHbaseConfigMap,
			hbasecluster.Spec.FSGroup, d, log, true)
		ctrl.SetControllerReference(hbasecluster, newSS, r.Scheme)
		result, err := reconcileStatefulSet(ctx, log, hbasecluster.Namespace, newSS, d, r.Client)
		if (ctrl.Result{}) != result || err != nil {
			return result, err
		}

		log.Info("starting pdb reconciliation")
		pdb := buildPodDisruptionBudget(hbasecluster.Name, hbasecluster.Namespace, d, log)
		if pdb != nil {
			ctrl.SetControllerReference(hbasecluster, pdb, r.Scheme)
			result, err = reconcilePodDisruptionBudget(ctx, log, pdb, d, r.Client)
			if (ctrl.Result{}) != result || err != nil {
				return result, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HbaseClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kvstorev1.HbaseCluster{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
