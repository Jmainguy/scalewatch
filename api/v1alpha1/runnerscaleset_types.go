/*
Copyright 2024.

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

// RunnerScaleSetSpec defines the desired state of RunnerScaleSet
type RunnerScaleSetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	GithubConfigSecret string `json:"githubConfigSecret,omitempty"`
	GithubConfigURL    string `json:"githubConfigURL,omitempty"`
	Name               string `json:"name,omitempty"`
}

// RunnerScaleSetStatus defines the observed state of RunnerScaleSet
type RunnerScaleSetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State string `json:"state,omitempty"` // Add the state field here
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RunnerScaleSet is the Schema for the runnerscalesets API
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="Current state of the Runner Scale Set"
// +kubebuilder:printcolumn:name="GithubConfigURL",type="string",JSONPath=".spec.githubConfigURL",description="Github Config URL for Runner"
// +kubebuilder:printcolumn:name="RunnerScaleSetName",type="string",JSONPath=".spec.name",description="RunnerScaleSet Name"
type RunnerScaleSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunnerScaleSetSpec   `json:"spec,omitempty"`
	Status RunnerScaleSetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RunnerScaleSetList contains a list of RunnerScaleSet
type RunnerScaleSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RunnerScaleSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RunnerScaleSet{}, &RunnerScaleSetList{})
}
