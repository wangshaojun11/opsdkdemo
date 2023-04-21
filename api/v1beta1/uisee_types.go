/*
Copyright 2023 wsj.

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

package v1beta1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// UiseeSpec defines the desired state of Uisee
type UiseeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Size      *int32                      `json:"size"`
	Image     string                      `json:"image"`
	Ports     []corev1.ServicePort        `json:"ports"`               // service自带数组的内容
	Resources corev1.ResourceRequirements `json:"resources,omitempty"` // 资源申请与限制requests/limits。 omitempty 表示可以不写
	Envs      []corev1.EnvVar             `json:"envs,omitempty"`
}

// UiseeStatus defines the observed state of Uisee
type UiseeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	appsv1.DeploymentStatus `json:",inline"` // Deployment 很多状态用inline
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Uisee is the Schema for the uisees API
type Uisee struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UiseeSpec   `json:"spec,omitempty"`
	Status UiseeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UiseeList contains a list of Uisee
type UiseeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Uisee `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Uisee{}, &UiseeList{})
}
