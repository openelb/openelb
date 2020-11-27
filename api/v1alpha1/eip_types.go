/*

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EipSpec defines the desired state of EIP
type EipSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}((\/([0-9]|[1-2][0-9]|3[0-2]))|(\-([0-9]{1,3}\.){3}[0-9]{1,3}))?$`
	Address string `json:"address,required"`
	// +kubebuilder:validation:Enum=bgp;layer2
	Protocol      string `json:"protocol,omitempty"`
	Disable       bool   `json:"disable,omitempty"`
	UsingKnownIPs bool   `json:"usingKnownIPs,omitempty"`
}

// EipStatus defines the observed state of EIP
type EipStatus struct {
	Occupied bool `json:"occupied,omitempty"`
	Usage    int  `json:"usage,omitempty"`
	PoolSize int  `json:"poolSize,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="cidr",type=string,JSONPath=`.spec.address`
// +kubebuilder:printcolumn:name="usage",type=integer,JSONPath=`.status.usage`
// +kubebuilder:printcolumn:name="total",type=integer,JSONPath=`.status.poolSize`
// +kubebuilder:resource:scope=Cluster,categories=networking

// Eip is the Schema for the eips API
type Eip struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EipSpec   `json:"spec,omitempty"`
	Status EipStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EipList contains a list of Eip
type EipList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Eip `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Eip{}, &EipList{})
}
