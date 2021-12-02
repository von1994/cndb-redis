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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RedisStandaloneSpec defines the desired state of RedisStandalone
// +k8s:openapi-gen=true
type RedisStandaloneSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Size               int32                         `json:"size,omitempty"`
	Resources          corev1.ResourceRequirements   `json:"resources,omitempty"`
	Image              string                        `json:"image,omitempty"`
	ImagePullPolicy    corev1.PullPolicy             `json:"imagePullPolicy,omitempty"`
	ImagePullSecrets   []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	Command            []string                      `json:"command,omitempty"`
	ShutdownConfigMap  string                        `json:"shutdownConfigMap,omitempty"`
	Storage            RedisStorage                  `json:"storage,omitempty"`
	Password           string                        `json:"password,omitempty"`
	Exporter           RedisExporter                 `json:"exporter,omitempty"`
	Affinity           *corev1.Affinity              `json:"affinity,omitempty"`
	SecurityContext    *corev1.PodSecurityContext    `json:"securityContext,omitempty"`
	ToleRations        []corev1.Toleration           `json:"toleRations,omitempty"`
	NodeSelector       map[string]string             `json:"nodeSelector,omitempty"`
	Config             map[string]string             `json:"config,omitempty"`
	Annotations        map[string]string             `json:"annotations,omitempty"`
	DisablePersistence bool                          `json:"disablePersistence,omitempty"`
}

// RedisStandalone is the Schema for the redisstandalones API
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="SIZE",type=integer,JSONPath=`.spec.size`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="HEALTHY",type=string,JSONPath=`.status.conditions[?(@.type=="Healthy")].status`
// +kubebuilder:printcolumn:name="MASTER-IP",type=string,JSONPath=".status.masterIP",priority=10
// +kubebuilder:printcolumn:name="SENTINEL-IP",type=string,JSONPath=".status.sentinelIP",priority=10
// +kubebuilder:resource:scope=Namespaced,shortName={redisstandalone},singular=redisstandalone
// +kubebuilder:subresource:status
type RedisStandalone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisStandaloneSpec   `json:"spec,omitempty"`
	Status RedisStandaloneStatus `json:"status,omitempty"`
}

// RedisStandaloneList contains a list of RedisStandalone
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
type RedisStandaloneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisStandalone `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisStandalone{}, &RedisStandaloneList{})
}
