/*
Copyright 2019 The Kubesphere Authors.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BgpConfStatus defines the observed state of BgpConf
type BgpConfStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster

// BgpConf is the Schema for the bgpconfs API
type BgpConf struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BgpConfSpec   `json:"spec,omitempty"`
	Status BgpConfStatus `json:"status,omitempty"`
}

// struct for container bgp:config.
// Configuration parameters relating to the global BGP router.
type BgpConfSpec struct {
	// original -> bgp:as
	// bgp:as's original type is inet:as-number.
	// Local autonomous system number of the router.  Uses
	// the 32-bit as-number type from the model in RFC 6991.
	As uint32 `mapstructure:"as" json:"as,required"`
	// original -> bgp:router-id
	// bgp:router-id's original type is inet:ipv4-address.
	// Router id of the router, expressed as an
	// 32-bit value, IPv4 address.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}$`
	RouterId string `mapstructure:"router-id" json:"routerID,required"`
	// original -> gobgp:port
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `mapstructure:"port" json:"port,required"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BgpConfList contains a list of BgpConf
type BgpConfList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BgpConf `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BgpConf{}, &BgpConfList{})
}
