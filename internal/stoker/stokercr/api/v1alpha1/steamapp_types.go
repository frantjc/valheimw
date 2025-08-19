package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Steamapp) GetConditions() []metav1.Condition {
	return s.Status.Conditions
}

func (s *Steamapp) SetConditions(conditions []metav1.Condition) {
	s.Status.Conditions = conditions
}

type SteamappSpecImageOpts struct {
	// +kubebuilder:default="docker.io/library/debian@sha256:8810492a2dd16b7f59239c1e0cc1e56c1a1a5957d11f639776bd6798e795608b"
	BaseImageRef string `json:"baseImage,omitempty"`
	// +kubebuilder:validation:Optional
	AptPkgs []string `json:"aptPackages,omitempty"`
	// +kubebuilder:default="public"
	Branch string `json:"branch,omitempty"`
	// +kubebuilder:validation:Optional
	BetaPassword string `json:"betaPassword,omitempty"`
	// +kubebuilder:default="default"
	LaunchType string `json:"launchType,omitempty"`
	// +kubebuilder:default="linux"
	// +kubebuilder:validation:Enum=linux;windows;macos
	PlatformType string `json:"platformType,omitempty"`
	// +kubebuilder:validation:Optional
	Execs []string `json:"execs,omitempty"`
	// +kubebuilder:validation:Optional
	Entrypoint []string `json:"entrypoint,omitempty"`
	// +kubebuilder:validation:Optional
	Cmd []string `json:"cmd,omitempty"`
}

// SteamappSpec defines the desired state of Steamapp.
type SteamappSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:MultipleOf=10
	AppID int `json:"appID"`
	// +kubebuilder:validation:Optional
	Ports []SteamappPort `json:"ports,omitempty"`
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceList `json:"resources,omitempty"`
	// +kubebuilder:validation:Optional
	Volumes []SteamappVolume `json:"volumes,omitempty"`
	// +kubebuilder:validation:Optional
	SteamappSpecImageOpts `json:",inline"`
}

type SteamappPort struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65536
	Port int32 `json:"port"`
	// Protocols for port.
	// +kubebuilder:default={"UDP"}
	Protocols []corev1.Protocol `json:"protocols,omitempty"`
}

type SteamappVolume struct {
	// +kubebuilder:validation:Required
	Path string `json:"path"`
}

const (
	PhasePending = "Pending"
	PhaseReady   = "Ready"
	PhaseFailed  = "Failed"
	PhasePaused  = "Paused"
)

// SteamappStatus defines the observed state of Steamapp.
type SteamappStatus struct {
	// +kubebuilder:default="Pending"
	// +kubebuilder:validation:Enum=Pending;Ready;Failed;Paused
	Phase string `json:"phase"`
	// +kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Optional
	IconURL string `json:"icon,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AppID",type=string,JSONPath=`.spec.appID`
// +kubebuilder:printcolumn:name="Branch",type=string,JSONPath=`.spec.branch`
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.status.name`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// Steamapp is the Schema for the steamapps API.
type Steamapp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SteamappSpec   `json:"spec,omitempty"`
	Status SteamappStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SteamappList contains a list of Steamapp.
type SteamappList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Steamapp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Steamapp{}, &SteamappList{})
}
