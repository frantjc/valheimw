package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SteamappSpecImageOpts struct {
	// +kubebuilder:default="docker.io/library/debian:stable-slim"
	BaseImageRef string `json:"baseImage,omitempty"`
	// +kubebuilder:validation:Optional
	AptPkgs []string `json:"aptPackages,omitempty"`
	// +kubebuilder:default=server
	LaunchType string `json:"launchType,omitempty"`
	// +kubebuilder:default=linux
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
	// +kubebuilder:default=public
	Branch string `json:"branch,omitempty"`
	// +kubebuilder:validation:Optional
	BetaPassword string `json:"betaPassword,omitempty"`
	// +kubebuilder:validation:Optional
	ImageOpts SteamappSpecImageOpts `json:"imageOpts,omitempty"`
}

const (
	PhasePending = "Pending"
	PhaseReady   = "Ready"
	PhaseFailed  = "Failed"
	PhasePaused  = "Paused"
)

// SteamappStatus defines the observed state of Steamapp.
type SteamappStatus struct {
	// +kubebuilder:default=Pending
	// +kubebuilder:validation:Enum=Pending;Ready;Failed;Paused
	Phase string `json:"phase"`
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
