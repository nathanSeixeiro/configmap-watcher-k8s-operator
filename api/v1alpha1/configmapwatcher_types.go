/*
Copyright 2026.

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
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ConfigMapWatcherSpec defines the desired state of ConfigMapWatcher
type ConfigMapWatcherSpec struct {
	ConfigMapName      string `json:"configMapName"`
	ConfigMapNamespace string `json:"configMapNamespace"`
	EventEndpoint      string `json:"eventEndpoint"`
}

// ConfigMapWatcherStatus defines the observed state of ConfigMapWatcher.
type ConfigMapWatcherStatus struct {
	LastConfigMapVersion string             `json:"lastConfigMapVersion,omitempty"`
	LastEventSent        metav1.Time        `json:"lastEventSent,omitempty"`
	Conditions           []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ConfigMapWatcher is the Schema for the configmapwatchers API
type ConfigMapWatcher struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of ConfigMapWatcher
	// +required
	Spec ConfigMapWatcherSpec `json:"spec"`

	// status defines the observed state of ConfigMapWatcher
	// +optional
	Status ConfigMapWatcherStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ConfigMapWatcherList contains a list of ConfigMapWatcher
type ConfigMapWatcherList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ConfigMapWatcher `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &ConfigMapWatcher{}, &ConfigMapWatcherList{})
		return nil
	})
}
