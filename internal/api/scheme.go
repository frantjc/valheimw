package api

import (
	"github.com/frantjc/sindri/internal/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{SchemeBuilder: runtime.NewSchemeBuilder(v1alpha1.AddToScheme, corev1.AddToScheme)}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func NewScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	return scheme, AddToScheme(scheme)
}
