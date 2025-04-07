package stokercr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/stoker"
	"github.com/frantjc/sindri/internal/stoker/stokercr/api/v1alpha1"
	"github.com/frantjc/sindri/steamapp"
	xslice "github.com/frantjc/x/slice"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
	namespace := u.Query().Get("namespace")
	if namespace == "" {
		namespace = DefaultNamespace
	}

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}

	scheme, err := NewScheme()
	if err != nil {
		return nil, err
	}

	cli, err := client.New(cfg, client.Options{
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
		return nil, stoker.NewHTTPStatusCodeError(fmt.Errorf("%s has failed validation", sa.Name), http.StatusPreconditionFailed)
	case v1alpha1.PhasePending:
		return nil, stoker.NewHTTPStatusCodeError(fmt.Errorf("%s has not finished validation", sa.Name), http.StatusPreconditionRequired)
	}

	if sa.Labels != nil {
		if v, ok := sa.Labels[LabelValidated]; ok {
			if validated, _ := strconv.ParseBool(v); !validated {
				return nil, stoker.NewHTTPStatusCodeError(fmt.Errorf("%s failed validation", sa.Name), http.StatusPreconditionFailed)
			}
		} else {
			return nil, stoker.NewHTTPStatusCodeError(fmt.Errorf("%s has not finished validation", sa.Name), http.StatusPreconditionRequired)
		}
	} else {
		return nil, stoker.NewHTTPStatusCodeError(fmt.Errorf("%s has not finished validation", sa.Name), http.StatusPreconditionRequired)
	}

	return &steamapp.GettableBuildImageOpts{
		BaseImageRef: sa.Spec.ImageOpts.BaseImageRef,
		AptPkgs:      sa.Spec.ImageOpts.AptPkgs,
		BetaPassword: sa.Spec.BetaPassword,
		LaunchType:   sa.Spec.ImageOpts.LaunchType,
		PlatformType: steamcmd.PlatformType(sa.Spec.ImageOpts.PlatformType),
		Execs:        sa.Spec.ImageOpts.Execs,
		Entrypoint:   sa.Spec.ImageOpts.Entrypoint,
		Cmd:          sa.Spec.ImageOpts.Cmd,
	}, nil
}

// +kubebuilder:rbac:groups=sindri.frantj.cc,resources=steamapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sindri.frantj.cc,resources=steamapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sindri.frantj.cc,resources=steamapps/finalizers,verbs=update

const (
	LabelValidated = "sindri.frantj.cc/validated"
	LabelLocked    = "sindri.frantj.cc/locked"
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

	// if sa.Labels != nil {
	// 	sa.Labels = map[string]string{}
	// }

	// delete(sa.Labels, LabelValidated)

	// if err := d.Client.Update(ctx, sa); err != nil {
	// 	return ctrl.Result{}, err
	// }

	// if sa.Status.Phase == v1alpha1.PhaseReady {
	// 	if validated, _ := strconv.ParseBool(sa.Labels[LabelValidated]); !validated {
	// 		sa.Labels[LabelValidated] = fmt.Sprint(true)

	// 		if err := d.Client.Update(ctx, sa); err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 	}
	// } else {
	// 	if validated, _ := strconv.ParseBool(sa.Labels[LabelValidated]); validated {
	// 		delete(sa.Labels, LabelValidated)

	// 		if err := d.Client.Update(ctx, sa); err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 	}
	// }

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (d *Database) SetupWithManager(mgr ctrl.Manager) error {
	d.Client = mgr.GetClient()

	if err := ctrl.NewControllerManagedBy(mgr).
		Named("stoker").
		For(&v1alpha1.Steamapp{}).
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
	nsa, ok := newObj.(*v1alpha1.Steamapp)
	if !ok {
		return nil, fmt.Errorf("expected a Steamapp object but got %T", newObj)
	}

	osa, ok := oldObj.(*v1alpha1.Steamapp)
	if !ok {
		return nil, fmt.Errorf("expected a Steamapp object but got %T", oldObj)
	}

	if osa.Labels != nil {
		if locked, _ := strconv.ParseBool(osa.Labels[LabelLocked]); locked {
			return nil, fmt.Errorf("cannot update locked Steamapp")
		}
	}

	if osa.Spec.AppID != nsa.Spec.AppID {
		return nil, fmt.Errorf(".spec.appID is immutable")
	}

	if osa.Spec.Branch != nsa.Spec.Branch {
		return nil, fmt.Errorf(".spec.branch is immutable")
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

	switch sa.Status.Phase {
	case v1alpha1.PhaseFailed:
		return nil, stoker.NewHTTPStatusCodeError(fmt.Errorf("%s has failed validation", sa.Name), http.StatusPreconditionFailed)
	case v1alpha1.PhasePending:
		return nil, stoker.NewHTTPStatusCodeError(fmt.Errorf("%s has not finished validation", sa.Name), http.StatusPreconditionRequired)
	}

	locked := false
	if sa.Labels != nil {
		if v, ok := sa.Labels[LabelValidated]; ok {
			if validated, _ := strconv.ParseBool(v); !validated {
				return nil, stoker.NewHTTPStatusCodeError(fmt.Errorf("%s failed validation", sa.Name), http.StatusPreconditionFailed)
			}
		} else {
			return nil, stoker.NewHTTPStatusCodeError(fmt.Errorf("%s has not finished validation", sa.Name), http.StatusPreconditionRequired)
		}

		locked, _ = strconv.ParseBool(sa.Labels[LabelLocked])
	} else {
		return nil, stoker.NewHTTPStatusCodeError(fmt.Errorf("%s has not finished validation", sa.Name), http.StatusPreconditionRequired)
	}

	return &stoker.Steamapp{
		SteamappDetail: stoker.SteamappDetail(sa.Spec.ImageOpts),
		SteamappSummary: stoker.SteamappSummary{
			AppID:   steamappID,
			Name:    sa.Status.Name,
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

	if err := d.Client.List(ctx, steamapps, &client.ListOptions{
		Namespace: d.Namespace,
		Continue:  o.Continue,
		Limit:     o.Limit,
		LabelSelector: labels.SelectorFromSet(labels.Set{
			LabelValidated: fmt.Sprint(true),
		}),
	}); err != nil {
		return nil, "", err
	}

	return xslice.Map(steamapps.Items, func(sa v1alpha1.Steamapp, _ int) stoker.SteamappSummary {
		locked := false
		if sa.Labels != nil {
			locked, _ = strconv.ParseBool(sa.Labels[LabelLocked])
		}

		return stoker.SteamappSummary{
			AppID:   sa.Spec.AppID,
			Name:    sa.Status.Name,
			Branch:  sa.Spec.Branch,
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
func (d *Database) Upsert(ctx context.Context, steamappID int, detail *stoker.SteamappDetail, opts ...stoker.UpsertOpt) error {
	var (
		o        = newUpsertOpts(opts...)
		steamapp = &v1alpha1.Steamapp{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: d.Namespace,
				Name:      fmt.Sprintf("%d-%s", steamappID, sanitizeBranchName(o.Branch)),
			},
			Spec: v1alpha1.SteamappSpec{
				AppID:        steamappID,
				Branch:       o.Branch,
				BetaPassword: o.BetaPassword,
				ImageOpts:    v1alpha1.SteamappSpecImageOpts(*detail),
			},
		}
	)

	if _, err := controllerutil.CreateOrUpdate(ctx, d.Client, steamapp, func() error {
		steamapp.Spec.AppID = steamappID
		steamapp.Spec.Branch = o.Branch
		steamapp.Spec.BetaPassword = o.BetaPassword
		steamapp.Spec.ImageOpts = v1alpha1.SteamappSpecImageOpts(*detail)
		return nil
	}); err != nil {
		return err
	}

	return nil
}
