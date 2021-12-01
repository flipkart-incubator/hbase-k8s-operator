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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HbaseStandaloneConfiguration struct {
	HbaseConfigName      string            `json:"hbaseConfigName"`
	HbaseConfigMountPath string            `json:"hbaseConfigMountPath"`
	HbaseConfig          map[string]string `json:"hbaseConfig"`
}

// HbaseStandaloneSpec defines the desired state of HbaseStandalone
type HbaseStandaloneSpec struct {
	// Important: Run "make" to regenerate code after modifying this file
	Standalone HbaseClusterDeployment `json:"standalone"`
	// TODO: Move away from Cluster Configuration to HbaseStandaloneConfiguration
	Configuration HbaseClusterConfiguration `json:"configuration"`
	FSGroup       int64                     `json:"fsgroup"`
	BaseImage     string                    `json:"baseImage"`
}

// HbaseStandaloneStatus defines the observed state of HbaseStandalone
type HbaseStandaloneStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	//TODO
	Nodes      []string           `json:"nodes"`
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// HbaseStandalone is the Schema for the hbasestandalones API
type HbaseStandalone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HbaseStandaloneSpec   `json:"spec,omitempty"`
	Status HbaseStandaloneStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HbaseStandaloneList contains a list of HbaseStandalone
type HbaseStandaloneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HbaseStandalone `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HbaseStandalone{}, &HbaseStandaloneList{})
}
