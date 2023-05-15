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

package v1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HbaseClusterContainerPort struct {
	Port int32  `json:"port"`
	Name string `json:"name"`
}

type HbaseClusterVolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	//+optional
	ReadOnly bool `json:"readOnly"`
}

type HbaseClusterProbe struct {
	//+optional
	Command []string `json:"command"`
	//+optional
	Port int `json:"tcpPort"`
	//+optional
	InitialDelaySeconds int32 `json:"initialDelay"`
	//+optional
	SuccessThreshold int32 `json:"successThreshold"`
	//+optional
	TimeoutSeconds int32 `json:"timeout"`
	//+optional
	PeriodSeconds int32 `json:"period"`
	//+optional
	FailureThreshold int32 `json:"failureThreshold"`
}

type HbaseClusterLifecycle struct {
	//+optional
	PostStart []string `json:"postStart"`
	//+optional
	PreStop []string `json:"preStop"`
}

type HbaseClusterSideCarContainer struct {
	Name  string `json:"name"`
	Image string `json:"image"`
	//+optional
	Command []string `json:"command"`
	//+optional
	Args            []string             `json:"args"`
	CpuLimit        string               `json:"cpuLimit"`
	CpuRequest      string               `json:"cpuRequest"`
	MemoryLimit     string               `json:"memoryLimit"`
	MemoryRequest   string               `json:"memoryRequest"`
	SecurityContext HbaseClusterSecurity `json:"securityContext"`
	//+optional
	VolumeMounts []HbaseClusterVolumeMount `json:"volumeMounts"`
}

type HbaseClusterContainer struct {
	Name    string   `json:"name"`
	Command []string `json:"command"`
	//+optional
	Args          []string                    `json:"args"`
	CpuLimit      string                      `json:"cpuLimit"`
	CpuRequest    string                      `json:"cpuRequest"`
	MemoryLimit   string                      `json:"memoryLimit"`
	MemoryRequest string                      `json:"memoryRequest"`
	Ports         []HbaseClusterContainerPort `json:"ports"`
	//+optional
	VolumeMounts    []HbaseClusterVolumeMount `json:"volumeMounts"`
	SecurityContext HbaseClusterSecurity      `json:"securityContext"`
	LivenessProbe   HbaseClusterProbe         `json:"livenessProbe"`
	// +optional
	ReadinessProbe HbaseClusterProbe `json:"readinessProbe"`
	// +optional
	StartupProbe HbaseClusterProbe `json:"startupProbe"`
	//+optional
	Lifecycle HbaseClusterLifecycle `json:"lifecycle"`
}

type HbaseClusterInitContainer struct {
	Name    string   `json:"name"`
	Command []string `json:"command"`
	//+optional
	Args          []string `json:"args"`
	CpuLimit      string   `json:"cpuLimit"`
	CpuRequest    string   `json:"cpuRequest"`
	MemoryLimit   string   `json:"memoryLimit"`
	MemoryRequest string   `json:"memoryRequest"`
	//+optional
	VolumeMounts    []HbaseClusterVolumeMount `json:"volumeMounts"`
	SecurityContext HbaseClusterSecurity      `json:"securityContext"`
	//+optional
	IsBootstrap bool `json:"isBootstrap"`
}

type HbaseClusterVolumeClaim struct {
	Name        string `json:"name"`
	StorageSize string `json:"storageSize"`
	//+optional
	StorageClassName string `json:"storageClassName"`
}

type HbaseClusterVolume struct {
	Name string `json:"name"`
	//+kubebuilder:validation:Enum:=ConfigMap;EmptyDir;HostPath;
	VolumeSource string `json:"volumeSource"`
	//+optional
	ConfigName string `json:"configName"`
	//+optional
	Path string `json:"path"`
}

type HbaseClusterDeployment struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Minimum:=1
	Size int32 `json:"size"`
	// +optional
	Labels map[string]string `json:"labels"`
	// +optional
	Annotations map[string]string       `json:"annotations"`
	Containers  []HbaseClusterContainer `json:"containers"`
	// +optional
	SideCarContainers []HbaseClusterSideCarContainer `json:"sidecarContainers"`
	// +optional
	InitContainers []HbaseClusterInitContainer `json:"initContainers"`
	// +optional
	VolumeClaims []HbaseClusterVolumeClaim `json:"volumeClaims"`
	// +optional
	Volumes []HbaseClusterVolume `json:"volumes"`
	// +kubebuilder:validation:Minimum:=10
	TerminationGracePeriodSeconds int64 `json:"terminateGracePeriod"`
	// +optional
	// +kubebuilder:default:=false
	ShareProcessNamespace bool `json:"shareProcessNamespace"`
	// +kubebuilder:default:=false
	IsPodServiceRequired bool `json:"isPodServiceRequired"`
	// +optional
	// +kubebuilder:default:=Parallel
	// +kubebuilder:validation:Enum:=Parallel;OrderedReady;
	PodManagementPolicy appsv1.PodManagementPolicyType `json:"podManagementPolicy"`
	// +optional
	Hostname string `json:"hostname,omitempty"`
	// +optional
	Subdomain string `json:"subdomain,omitempty"`
	// +optional
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`
	// +optional
	DNSConfig *corev1.PodDNSConfig `json:"dnsConfig,omitempty"`
	// +optional
	HostAliases []corev1.HostAlias `json:"hostAliases,omitempty" patchStrategy:"merge" patchMergeKey:"ip"`
}

type HbaseClusterConfiguration struct {
	HbaseConfigName       string            `json:"hbaseConfigName"`
	HbaseConfigMountPath  string            `json:"hbaseConfigMountPath"`
	HbaseConfig           map[string]string `json:"hbaseConfig"`
	HadoopConfigName      string            `json:"hadoopConfigName"`
	HadoopConfigMountPath string            `json:"hadoopConfigMountPath"`
	HadoopConfig          map[string]string `json:"hadoopConfig"`
	// +optional
	HbaseTenantConfig []map[string]string `json:"hbaseTenantConfig"`
	// +optional
	HadoopTenantConfig []map[string]string `json:"hadoopTenantConfig"`
}

type HbaseClusterSecurity struct {
	RunAsUser  int64 `json:"runAsUser"`
	RunAsGroup int64 `json:"runAsGroup"`
	// +optional
	AddSysPtrace bool `json:"addSysPtrace"`
}

type HbaseClusterDeployments struct {
	//+optional
	Zookeeper   HbaseClusterDeployment `json:"zookeeper"`
	Journalnode HbaseClusterDeployment `json:"journalnode"`
	Namenode    HbaseClusterDeployment `json:"namenode"`
	Datanode    HbaseClusterDeployment `json:"datanode"`
	Hmaster     HbaseClusterDeployment `json:"hmaster"`
}

// HbaseClusterSpec defines the desired state of HbaseCluster
type HbaseClusterSpec struct {
	Deployments   HbaseClusterDeployments   `json:"deployments"`
	Configuration HbaseClusterConfiguration `json:"configuration"`
	FSGroup       int64                     `json:"fsgroup"`
	IsBootstrap   bool                      `json:"isBootstrap"`
	BaseImage     string                    `json:"baseImage"`
	// +optional
	TenantNamespaces []string `json:"tenantNamespaces"`
	// +optional
	ServiceLabels map[string]string `json:"serviceLabels"`
	// +optional
	ServiceSelectorLabels map[string]string `json:"serviceSelectorLabels"`
}

// HbaseClusterStatus defines the observed state of HbaseCluster
type HbaseClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	//TODO
	Nodes      []string           `json:"nodes"`
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// HbaseCluster is the Schema for the hbaseclusters API
type HbaseCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HbaseClusterSpec   `json:"spec,omitempty"`
	Status HbaseClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HbaseClusterList contains a list of HbaseCluster
type HbaseClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HbaseCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HbaseCluster{}, &HbaseClusterList{})
}
