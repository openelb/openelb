/*
Copyright 2022 The Kubesphere Authors.

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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DefinedSetSpec defines the desired state of DefinedSet
type DefinedSetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Prefixes       []*Prefix        `json:"prefixes,omitempty"`
	Neighbors      []*Neighbor      `json:"neighbours,omitempty"`
	BgpDefinedSets []*BgpDefinedSet `json:"bgpDefinedSets,omitempty"`
}

// DefinedSetStatus defines the observed state of DefinedSet
type DefinedSetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// DefinedSet is the Schema for the definedsets API
type DefinedSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DefinedSetSpec   `json:"spec,omitempty"`
	Status DefinedSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DefinedSetList contains a list of DefinedSet
type DefinedSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DefinedSet `json:"items"`
}

type Prefix struct {
	Name       string     `json:"name,omitempty"`
	PrefixList PrefixList `json:"prefixList,omitempty"`
}

type PrefixList struct {
	IpPrefix        string `json:"ipPrefix,omitempty"`
	MaskLengthRange string `json:"maskLengthRange,omitempty"`
}

type Neighbor struct {
	Name         string   `json:"name,omitempty"`
	NeighborList []string `json:"neighborList,omitempty"`
}

type BgpDefinedSet struct {
	CommunitySets      []*CommunitySet      `json:"communitySets,omitempty"`
	ExtCommunitySets   []*ExtCommunitySet   `json:"extCommunitySets,omitempty"`
	AsPathSets         []*AsPathSet         `json:"asPathSets,omitempty"`
	LargeCommunitySets []*LargeCommunitySet `json:"largeCommunitySets,omitempty"`
}

type CommunitySet struct {
	Name          string   `json:"name,omitempty"`
	CommunityList []string `json:"communityList,omitempty"`
}

type ExtCommunitySet struct {
	Name             string   `json:"name,omitempty"`
	ExtCommunityList []string `json:"extCommunityList,omitempty"`
}

type AsPathSet struct {
	Name       string   `json:"name,omitempty"`
	AsPathList []string `json:"asPathList,omitempty"`
}

type LargeCommunitySet struct {
	Name               string   `json:"name,omitempty"`
	LargeCommunityList []string `json:"largeCommunityList,omitempty"`
}

func init() {
	SchemeBuilder.Register(&DefinedSet{}, &DefinedSetList{})
}
