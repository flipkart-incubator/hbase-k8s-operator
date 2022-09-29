package test

import (
	"testing"
	"fmt"
	"strings"

	json "encoding/json"

	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	rbac "k8s.io/api/rbac/v1"

	"github.com/gruntwork-io/terratest/modules/helm"
)

var additionalNamespaceKey = "ADDITIONAL_WATCH_NAMESPACES"
var serviceAccountName = "hbase-operator-controller-manager"
var namespaceName = "hbase-operator-ns"
var deploymentName = "operator-hbase"
var operatorConfigMapName = "hbase-operator-config"
var replicaCount = "1"
var managerRoleName = "hbase-operator-manager-role"
var managerRoleBindingName = "hbase-operator-manager-rolebinding"
var leaderElectionRoleName = "hbase-operator-leader-election-role"
var leaderElectionRoleBindingName = "hbase-operator-leader-election-rolebinding"
var kubeRbacProxyImageName = "gcr.io/kubebuilder/kube-rbac-proxy"
var kubeRbacProxyImageVersion = "v0.8.0"
var operatorImageName = "hbase-operator"
var operatorImageVersion = "v1.0.0"
var namespacesEscaped = "\"hbase-standalone-ns\\,hbase-cluster-ns\\,hbase-tenant-ns\""
var namespaces = "hbase-standalone-ns,hbase-cluster-ns,hbase-tenant-ns"

func TestOperatorServiceAccountHappyCase(t *testing.T) {
	helmChartPath := "../operator-chart"
	_ = "--values " + helmChartPath + "/empty_values.yaml"

	options := &helm.Options{
		SetValues: map[string]string{
			"image.kube_rbac_proxy.image_name": kubeRbacProxyImageName,
			"image.kube_rbac_proxy.tag": kubeRbacProxyImageVersion,
			"image.hbase_operator.image_name": operatorImageName,
			"image.hbase_operator.tag": operatorImageVersion,
			"name": deploymentName,
			"namespace": namespaceName,
			"replicaCount": replicaCount,
			"managerRoleName": managerRoleName,
			"managerRoleBindingName": managerRoleBindingName,
			"leaderElectionRoleName": leaderElectionRoleName,
			"serviceAccountName": serviceAccountName,
			"namespaces": namespacesEscaped,
		},
	}

	output := helm.RenderTemplate(t, options, helmChartPath, "operator-example", []string{})

	k8sObjects := strings.Split(output, "---")

	// Service Account
	var sa corev1.ServiceAccount
	helm.UnmarshalK8SYaml(t, k8sObjects[1], &sa)

	if sa.Name != serviceAccountName {
		t.Fatalf("Rendered service account name (%s) is not expected (%s)", sa.Name, serviceAccountName)
	}

	if sa.Namespace != namespaceName {
		t.Fatalf("Rendered namespace (%s) is not expected (%s)", sa.Namespace, namespaceName)
	}

	// ConfigMap for operator
	var configmap corev1.ConfigMap
	helm.UnmarshalK8SYaml(t, k8sObjects[3], &configmap)

	if configmap.Name != operatorConfigMapName {
		t.Fatalf("Rendered configmap name (%s) is not expected (%s)", configmap.Name, operatorConfigMapName)
	}

	if configmap.Namespace != namespaceName {
		t.Fatalf("Rendered configmap namespace (%s) is not expected (%s)", configmap.Namespace, namespaceName)
	}

	if val, ok := configmap.Data[additionalNamespaceKey]; ok {
		if val != namespaces {
			t.Fatalf("Rendered configmap name (%s)  key (%s) value (%s) is not expected (%s)", configmap.Name, additionalNamespaceKey, val, namespaces)
		}
	} else {
		t.Fatalf("Rendered configmap name (%s) does not contain key (%s)", configmap.Name, additionalNamespaceKey)
	}

	// Leader role
	var leaderRole rbac.Role
	helm.UnmarshalK8SYaml(t, k8sObjects[4], &leaderRole)

	if leaderRole.Name != leaderElectionRoleName {
		t.Fatalf("Rendered leaderrole name (%s) is not expected (%s)", leaderRole.Name, leaderElectionRoleName)
	}

	if leaderRole.Namespace != namespaceName {
		t.Fatalf("Rendered leaderrole namespace (%s) is not expected (%s)", leaderRole.Namespace, namespaceName)
	}

	// Manager role
	var managerRole rbac.Role
	helm.UnmarshalK8SYaml(t, k8sObjects[5], &managerRole)

	if managerRole.Name != managerRoleName {
		t.Fatalf("Rendered managerrole name (%s) is not expected (%s)", managerRole.Name, managerRoleName)
	}

	if managerRole.Namespace != namespaceName {
		t.Fatalf("Rendered managerrole namespace (%s) is not expected (%s)", managerRole.Namespace, namespaceName)
	}


	// Role binding leader role
	var leaderRoleBinding rbac.RoleBinding
	helm.UnmarshalK8SYaml(t, k8sObjects[6], &leaderRoleBinding)
	if leaderRoleBinding.Name != leaderElectionRoleBindingName {
		t.Fatalf("Rendered leaderrolebinding name (%s) is not expected (%s)", leaderRoleBinding.Name, leaderElectionRoleBindingName)
	}

	if leaderRoleBinding.Namespace != namespaceName {
		t.Fatalf("Rendered leaderrolebinding namespace (%s) is not expected (%s)", leaderRoleBinding.Namespace, namespaceName)
	}

	if leaderRoleBinding.RoleRef.Kind != "Role" {
		t.Fatalf("Rendered leaderrolebinding roleref kind (%s) is not expected (%s)", leaderRoleBinding.RoleRef.Kind, "Role")
	}

	if leaderRoleBinding.RoleRef.Name != leaderElectionRoleName {
		t.Fatalf("Rendered leaderrolebinding roleref name (%s) is not expected (%s)", leaderRoleBinding.RoleRef.Name, leaderElectionRoleName)
	}

	if leaderRoleBinding.Subjects[0].Kind != "ServiceAccount" {
		t.Fatalf("Rendered leaderrolebinding subject kind (%s) is not expected (%s)", leaderRoleBinding.Subjects[0].Kind, "ServiceAccount")
	}

	if leaderRoleBinding.Subjects[0].Name != serviceAccountName {
		t.Fatalf("Rendered leaderrolebinding subject name (%s) is not expected (%s)", leaderRoleBinding.Subjects[0].Name, serviceAccountName)
	}

	if leaderRoleBinding.Subjects[0].Namespace != namespaceName {
		t.Fatalf("Rendered leaderrolebinding subject namespace (%s) is not expected (%s)", leaderRoleBinding.Subjects[0].Namespace, namespaceName)
	}


	// Role binding manager role
	var managerRoleBinding rbac.RoleBinding
	helm.UnmarshalK8SYaml(t, k8sObjects[7], &managerRoleBinding)
	if managerRoleBinding.Name != managerRoleBindingName {
		t.Fatalf("Rendered managerrolebinding name (%s) is not expected (%s)", managerRoleBinding.Name, managerRoleBindingName)
	}

	if managerRoleBinding.Namespace != namespaceName {
		t.Fatalf("Rendered managerrolebinding namespace (%s) is not expected (%s)", managerRoleBinding.Namespace, namespaceName)
	}

	if managerRoleBinding.RoleRef.Kind != "Role" {
		t.Fatalf("Rendered managerrolebinding roleref kind (%s) is not expected (%s)", managerRoleBinding.RoleRef.Kind, "Role")
	}

	if managerRoleBinding.RoleRef.Name != managerRoleName {
		t.Fatalf("Rendered managerrolebinding roleref name (%s) is not expected (%s)", managerRoleBinding.RoleRef.Name, managerRoleName)
	}

	if managerRoleBinding.Subjects[0].Kind != "ServiceAccount" {
		t.Fatalf("Rendered managerrolebinding subject kind (%s) is not expected (%s)", managerRoleBinding.Subjects[0].Kind, "ServiceAccount")
	}

	if managerRoleBinding.Subjects[0].Name != serviceAccountName {
		t.Fatalf("Rendered managerrolebinding subject name (%s) is not expected (%s)", managerRoleBinding.Subjects[0].Name, serviceAccountName)
	}

	if managerRoleBinding.Subjects[0].Namespace != namespaceName {
		t.Fatalf("Rendered managerrolebinding subject namespace (%s) is not expected (%s)", managerRoleBinding.Subjects[0].Namespace, namespaceName)
	}

	sss, _ := json.MarshalIndent(managerRoleBinding, "", "\t")
	fmt.Print(string(sss))

	// Deployment of operator
	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, k8sObjects[8], &deployment)

	if deployment.Name != deploymentName {
		t.Fatalf("Rendered deployment name (%s) is not expected (%s)", deployment.Name, deploymentName)
	}

	if deployment.Namespace != namespaceName {
		t.Fatalf("Rendered deployment namespace (%s) is not expected (%s)", deployment.Namespace, namespaceName)
	}

	actualReplicaCount := fmt.Sprint(*deployment.Spec.Replicas)
	if actualReplicaCount != replicaCount {
		t.Fatalf("Rendered deployment replicas (%s) is not expected (%s)", actualReplicaCount, replicaCount)
	}

	//fmt.Printf("%s", "-----------")
	//fmt.Printf("%s", k8sObjects[8])
	//fmt.Printf("%s", "-----------")

	//sss, _ := json.MarshalIndent(deployment, "", "\t")
	//fmt.Print(string(sss))
}


/*func TestOperatorServiceAccountMissingCase(t *testing.T) {
	helmChartPath := "../operator-chart"

	options := &helm.Options{
		SetValues: map[string]string{
			"image.kube_rbac_proxy.image_name": kubeRbacProxyImageName,
			"image.kube_rbac_proxy.tag": kubeRbacProxyImageVersion,
			"image.hbase_operator.image_name": operatorImageName,
			"image.hbase_operator.tag": operatorImageVersion,
			"name": deploymentName,
			"namespace": "",
			"replicaCount": replicaCount,
			"managerRoleName": managerRoleName,
			"managerRoleBindingName": managerRoleBindingName,
			"leaderElectionRoleName": leaderElectionRoleName,
			"serviceAccountName": "",
		},
	}

	output := helm.RenderTemplate(t, options, helmChartPath, "operator-example", []string{})
	k8sObjects := strings.Split(output, "---")

	var sa corev1.ServiceAccount
	helm.UnmarshalK8SYaml(t, k8sObjects[1], &sa)

	if sa.Name != "" {
		t.Fatalf("Rendered service account name (%s) is not expected (%s)", sa.Name, serviceAccountName)
	}

	if sa.Namespace != "" {
		t.Fatalf("Rendered namespace (%s) is not expected (%s)", sa.Namespace, namespaceName)
	}
}*/
