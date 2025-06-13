package stokercr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/httputil"
	"github.com/frantjc/sindri/internal/stoker"
	"github.com/frantjc/sindri/internal/stoker/stokercr/api/v1alpha1"
	"github.com/frantjc/sindri/steamapp"
	xslices "github.com/frantjc/x/slices"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func NewScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	schemeBuilder := runtime.NewSchemeBuilder(v1alpha1.AddToScheme, clientgoscheme.AddToScheme)
	return scheme, schemeBuilder.AddToScheme(scheme)
}

type databaseURLOpener struct{}

const (
	DefaultNamespace = "sindri-system"
)

// OpenDatabase implements steamapp.DatabaseURLOpener.
func (o *databaseURLOpener) OpenDatabase(_ context.Context, u *url.URL) (steamapp.Database, error) {
	cfgFlags := genericclioptions.NewConfigFlags(true)

	namespace := u.Query().Get("namespace")
	if namespace == "" {
		namespace = DefaultNamespace
	}
	cfgFlags.Namespace = &namespace

	context := u.Query().Get("context")
	if context != "" {
		cfgFlags.Context = &context
	}

	restCfg, err := cfgFlags.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	scheme, err := NewScheme()
	if err != nil {
		return nil, err
	}

	cli, err := client.New(restCfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}

	return &Database{
		Namespace: namespace,
		Client:    cli,
	}, nil
}

const Scheme = "stokercr"

func init() {
	steamapp.RegisterDatabase(
		new(databaseURLOpener),
		Scheme,
	)
}

type Database struct {
	Namespace string
	Client    client.Client
	APIReader client.Reader
}

// GetBuildImageOpts implements steamapp.Database.
func (d *Database) GetBuildImageOpts(ctx context.Context, appID int, branch string) (*steamapp.GettableBuildImageOpts, error) {
	var (
		o  = newGetOpts(&stoker.GetOpts{Branch: branch})
		sa = &v1alpha1.Steamapp{}
	)

	if err := d.Client.Get(ctx, client.ObjectKey{Namespace: d.Namespace, Name: fmt.Sprintf("%d-%s", appID, o.Branch)}, sa); err != nil {
		return nil, err
	}

	switch sa.Status.Phase {
	case v1alpha1.PhaseFailed:
		return nil, httputil.NewHTTPStatusCodeError(fmt.Errorf("%s has failed validation", sa.Name), http.StatusPreconditionFailed)
	case v1alpha1.PhasePending:
		return nil, httputil.NewHTTPStatusCodeError(fmt.Errorf("%s has not finished validation", sa.Name), http.StatusPreconditionRequired)
	}

	if sa.Labels != nil {
		if v, ok := sa.Labels[LabelValidated]; ok {
			if validated, _ := strconv.ParseBool(v); !validated {
				return nil, httputil.NewHTTPStatusCodeError(fmt.Errorf("%s failed validation", sa.Name), http.StatusPreconditionFailed)
			}
		} else {
			return nil, httputil.NewHTTPStatusCodeError(fmt.Errorf("%s has not finished validation", sa.Name), http.StatusPreconditionRequired)
		}
	} else {
		return nil, httputil.NewHTTPStatusCodeError(fmt.Errorf("%s has not finished validation", sa.Name), http.StatusPreconditionRequired)
	}

	return &steamapp.GettableBuildImageOpts{
		BaseImageRef: sa.Spec.BaseImageRef,
		AptPkgs:      sa.Spec.AptPkgs,
		BetaPassword: sa.Spec.BetaPassword,
		LaunchType:   sa.Spec.LaunchType,
		PlatformType: steamcmd.PlatformType(sa.Spec.PlatformType),
		Execs:        sa.Spec.Execs,
		Entrypoint:   sa.Spec.Entrypoint,
		Cmd:          sa.Spec.Cmd,
	}, nil
}

// +kubebuilder:rbac:groups=sindri.frantj.cc,resources=steamapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sindri.frantj.cc,resources=steamapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sindri.frantj.cc,resources=steamapps/finalizers,verbs=update

const (
	AnnotationApproved = "sindri.frantj.cc/approved"
	LabelValidated     = "sindri.frantj.cc/validated"
	AnnotationLocked   = "sindri.frantj.cc/locked"
)

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (d *Database) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var (
		log = log.FromContext(ctx)
		sa  = &v1alpha1.Steamapp{}
	)

	log.Info("reconciling")

	if err := d.Client.Get(ctx, req.NamespacedName, sa); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	if sa.Labels == nil {
		sa.Labels = map[string]string{}
	}

	if sa.Status.Phase == v1alpha1.PhaseReady {
		if validated, _ := strconv.ParseBool(sa.Labels[LabelValidated]); !validated {
			sa.Labels[LabelValidated] = fmt.Sprint(true)

			if err := d.Client.Update(ctx, sa); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if validated, _ := strconv.ParseBool(sa.Labels[LabelValidated]); validated {
			delete(sa.Labels, LabelValidated)

			if err := d.Client.Update(ctx, sa); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (d *Database) SetupWithManager(mgr ctrl.Manager) error {
	d.Client = mgr.GetClient()
	d.APIReader = mgr.GetAPIReader()

	if err := ctrl.NewControllerManagedBy(mgr).
		Named("stoker").
		For(&v1alpha1.Steamapp{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			if sa, ok := obj.(*v1alpha1.Steamapp); ok {
				return sa.Status.Phase == v1alpha1.PhaseReady
			}
			return false
		}))).
		Complete(d); err != nil {
		return err
	}

	return nil
	// ctrl.NewWebhookManagedBy(mgr).
	// 	For(&v1alpha1.Steamapp{}).
	// 	WithValidator(d).
	// 	Complete()
}

func (d *Database) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	_, ok := obj.(*v1alpha1.Steamapp)
	if !ok {
		return nil, fmt.Errorf("expected a Steamapp object but got %T", obj)
	}

	return nil, nil
}

func (d *Database) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	_, ok := newObj.(*v1alpha1.Steamapp)
	if !ok {
		return nil, fmt.Errorf("expected a Steamapp object but got %T", newObj)
	}

	osa, ok := oldObj.(*v1alpha1.Steamapp)
	if !ok {
		return nil, fmt.Errorf("expected a Steamapp object but got %T", oldObj)
	}

	if osa.Annotations != nil {
		if locked, _ := strconv.ParseBool(osa.Annotations[AnnotationLocked]); locked {
			return nil, fmt.Errorf("cannot update locked Steamapp")
		}
	}

	return nil, nil
}

func (d *Database) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	_, ok := obj.(*v1alpha1.Steamapp)
	if !ok {
		return nil, fmt.Errorf("expected a Steamapp object but got %T", obj)
	}

	return nil, nil
}

var _ stoker.Database = &Database{}

func newGetOpts(opts ...stoker.GetOpt) *stoker.GetOpts {
	o := &stoker.GetOpts{
		Branch: steamapp.DefaultBranchName,
	}

	for _, opt := range opts {
		opt.ApplyToGet(o)
	}

	return o
}

// Get implements stoker.Database.
func (d *Database) Get(ctx context.Context, steamappID int, opts ...stoker.GetOpt) (*stoker.Steamapp, error) {
	var (
		o  = newGetOpts(opts...)
		sa = &v1alpha1.Steamapp{}
	)

	if err := d.Client.Get(ctx, client.ObjectKey{Namespace: d.Namespace, Name: fmt.Sprintf("%d-%s", steamappID, sanitizeBranchName(o.Branch))}, sa); err != nil {
		return nil, err
	}

	locked := false
	if sa.Annotations != nil {
		locked, _ = strconv.ParseBool(sa.Annotations[AnnotationLocked])
	}

	if sa.Labels != nil {
		if v, ok := sa.Labels[LabelValidated]; ok {
			if validated, _ := strconv.ParseBool(v); !validated {
				return nil, httputil.NewHTTPStatusCodeError(fmt.Errorf("%s failed validation", sa.Name), http.StatusPreconditionFailed)
			}
		} else {
			return nil, httputil.NewHTTPStatusCodeError(fmt.Errorf("%s has not finished validation", sa.Name), http.StatusPreconditionRequired)
		}
	} else {
		return nil, httputil.NewHTTPStatusCodeError(fmt.Errorf("%s has not finished validation", sa.Name), http.StatusPreconditionRequired)
	}

	return &stoker.Steamapp{
		SteamappDetail: stoker.SteamappDetail{
			Ports: xslices.Map(sa.Spec.Ports, func(port v1alpha1.SteamappPort, _ int) stoker.SteamappPort {
				return stoker.SteamappPort{
					Port: port.Port,
					Protocols: xslices.Map(port.Protocols, func(protocol corev1.Protocol, _ int) string {
						return string(protocol)
					}),
				}
			}),
			Resources: stoker.SteamappResources{
				CPU:    sa.Spec.Resources.Cpu().String(),
				Memory: sa.Spec.Resources.Memory().String(),
			},
			Volumes: xslices.Map(sa.Spec.Volumes, func(volume v1alpha1.SteamappVolume, _ int) stoker.SteamappVolume {
				return stoker.SteamappVolume(volume)
			}),
			SteamappImageOpts: stoker.SteamappImageOpts{
				BetaPassword: sa.Spec.BetaPassword,
				BaseImageRef: sa.Spec.BaseImageRef,
				AptPkgs:      sa.Spec.AptPkgs,
				LaunchType:   sa.Spec.LaunchType,
				PlatformType: sa.Spec.PlatformType,
				Execs:        sa.Spec.Execs,
				Entrypoint:   sa.Spec.Entrypoint,
				Cmd:          sa.Spec.Cmd,
			},
		},
		SteamappSummary: stoker.SteamappSummary{
			AppID:   steamappID,
			Name:    sa.Status.Name,
			Branch:  sa.Spec.Beta,
			IconURL: sa.Status.IconURL,
			Created: sa.CreationTimestamp.Time,
			Locked:  locked,
		},
	}, nil
}

func newListOpts(opts ...stoker.ListOpt) *stoker.ListOpts {
	o := &stoker.ListOpts{
		Limit: 10,
	}

	for _, opt := range opts {
		opt.ApplyToList(o)
	}

	return o
}

// List implements stoker.Database.
func (d *Database) List(ctx context.Context, opts ...stoker.ListOpt) ([]stoker.SteamappSummary, string, error) {
	var (
		steamapps = &v1alpha1.SteamappList{}
		o         = newListOpts(opts...)
	)

	if err := d.APIReader.List(ctx, steamapps, &client.ListOptions{
		Namespace: d.Namespace,
		Continue:  o.Continue,
		Limit:     o.Limit,
		LabelSelector: labels.SelectorFromSet(labels.Set{
			LabelValidated: fmt.Sprint(true),
		}),
	}); err != nil {
		return nil, "", err
	}

	return xslices.Map(steamapps.Items, func(sa v1alpha1.Steamapp, _ int) stoker.SteamappSummary {
		locked := false
		if sa.Annotations != nil {
			locked, _ = strconv.ParseBool(sa.Annotations[AnnotationLocked])
		}

		return stoker.SteamappSummary{
			AppID:   sa.Spec.AppID,
			Name:    sa.Status.Name,
			Branch:  sa.Spec.Beta,
			IconURL: sa.Status.IconURL,
			Created: sa.CreationTimestamp.Time,
			Locked:  locked,
		}
	}), steamapps.Continue, nil
}

func newUpsertOpts(opts ...stoker.UpsertOpt) *stoker.UpsertOpts {
	o := &stoker.UpsertOpts{
		Branch: steamapp.DefaultBranchName,
	}

	for _, opt := range opts {
		opt.ApplyToUpsert(o)
	}

	return o
}

func sanitizeBranchName(branch string) string {
	return strings.ReplaceAll(branch, "_", "-")
}

// Upsert implements stoker.Database.
func (d *Database) Upsert(ctx context.Context, appID int, detail *stoker.SteamappDetail, opts ...stoker.UpsertOpt) error {
	var (
		o    = newUpsertOpts(opts...)
		spec = v1alpha1.SteamappSpec{
			Ports: xslices.Map(detail.Ports, func(port stoker.SteamappPort, _ int) v1alpha1.SteamappPort {
				return v1alpha1.SteamappPort{
					Port: port.Port,
					Protocols: xslices.Map(port.Protocols, func(protocol string, _ int) corev1.Protocol {
						return corev1.Protocol(protocol)
					}),
				}
			}),
			Resources: corev1.ResourceList{},
			Volumes: xslices.Map(detail.Volumes, func(volume stoker.SteamappVolume, _ int) v1alpha1.SteamappVolume {
				return v1alpha1.SteamappVolume(volume)
			}),
			AppID: appID,
			SteamappSpecImageOpts: v1alpha1.SteamappSpecImageOpts{
				BaseImageRef: detail.BaseImageRef,
				AptPkgs:      detail.AptPkgs,
				Beta:         o.Branch,
				BetaPassword: detail.BetaPassword,
				LaunchType:   detail.LaunchType,
				PlatformType: detail.PlatformType,
				Execs:        detail.Execs,
				Entrypoint:   detail.Entrypoint,
				Cmd:          detail.Cmd,
			},
		}
	)

	cpu, err := resource.ParseQuantity(detail.Resources.CPU)
	if err != nil {
		return err
	}

	spec.Resources["cpu"] = cpu

	memory, err := resource.ParseQuantity(detail.Resources.Memory)
	if err != nil {
		return err
	}

	spec.Resources["cpu"] = memory

	var (
		sa = &v1alpha1.Steamapp{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: d.Namespace,
				Name:      fmt.Sprintf("%d-%s", appID, sanitizeBranchName(o.Branch)),
			},
			Spec: spec,
		}
	)

	if _, err := controllerutil.CreateOrUpdate(ctx, d.Client, sa, func() error {
		sa.Spec = spec
		return nil
	}); err != nil {
		return err
	}

	return nil
}
