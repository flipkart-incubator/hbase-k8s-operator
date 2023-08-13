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

// HbaseTenantSpec defines the desired state of HbaseTenant
type HbaseTenantSpec struct {
	// Important: Run "make" to regenerate code after modifying this file
	Datanode      HbaseClusterDeployment    `json:"datanode"`
	Configuration HbaseClusterConfiguration `json:"configuration"`
	FSGroup       int64                     `json:"fsgroup"`
	BaseImage     string                    `json:"baseImage"`
	// +optional
	ServiceLabels map[string]string `json:"serviceLabels"`
	// +optional
	ServiceSelectorLabels map[string]string `json:"serviceSelectorLabels"`
}

// HbaseTenantStatus defines the observed state of HbaseTenant
type HbaseTenantStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	//TODO
	Nodes      []string           `json:"nodes"`
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// HbaseTenant is the Schema for the hbasetenants API
type HbaseTenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HbaseTenantSpec   `json:"spec,omitempty"`
	Status HbaseTenantStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HbaseTenantList contains a list of HbaseTenant
type HbaseTenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HbaseTenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HbaseTenant{}, &HbaseTenantList{})
}
