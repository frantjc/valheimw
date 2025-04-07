package controller

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/appinfoutil"
	"github.com/frantjc/sindri/internal/stoker/stokercr"
	"github.com/frantjc/sindri/internal/stoker/stokercr/api/v1alpha1"
	"github.com/frantjc/sindri/steamapp"
	xslice "github.com/frantjc/x/slice"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SteamappReconciler reconciles a Steamapp object.
type SteamappReconciler struct {
	client.Client
	record.EventRecorder
	*steamapp.ImageBuilder
}

// +kubebuilder:rbac:groups=sindri.frantj.cc,resources=steamapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sindri.frantj.cc,resources=steamapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sindri.frantj.cc,resources=steamapps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SteamappReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var (
		log = log.FromContext(ctx)
		sa  = &v1alpha1.Steamapp{}
	)

	log.Info("reconciling")

	if err := r.Get(ctx, req.NamespacedName, sa); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	sa.Status.Phase = v1alpha1.PhasePending

	if err := r.Client.Status().Update(ctx, sa); err != nil {
		return ctrl.Result{}, err
	}

	appInfo, err := appinfoutil.GetAppInfo(ctx, sa.Spec.AppID)
	if err != nil {
		r.Eventf(sa, corev1.EventTypeWarning, "AppInfoPrintFailed", "Could not get app info: %v", err)
		sa.Status.Phase = v1alpha1.PhaseFailed
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	sa.Status.Name = appInfo.Common.Name

	u, err := url.Parse("https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps")
	if err != nil {
		return ctrl.Result{}, err
	}

	sa.Status.IconURL = u.JoinPath(fmt.Sprint(sa.Spec.AppID), fmt.Sprintf("%s.jpg", appInfo.Common.Icon)).String()

	if err := r.Client.Status().Update(ctx, sa); err != nil {
		return ctrl.Result{}, err
	}

	if appInfo.Depots.Branches[sa.Spec.Branch].PwdRequired && sa.Spec.BetaPassword == "" {
		r.Eventf(sa, corev1.EventTypeWarning, "BetaPwdMissing", "Branch %s requires a password", sa.Spec.Branch)
		sa.Status.Phase = v1alpha1.PhaseFailed
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	if err := r.BuildImage(ctx, sa.Spec.AppID, &steamapp.GettableBuildImageOpts{
		BaseImageRef: sa.Spec.ImageOpts.BaseImageRef,
		AptPkgs:      sa.Spec.ImageOpts.AptPkgs,
		BetaPassword: sa.Spec.BetaPassword,
		LaunchType:   sa.Spec.ImageOpts.LaunchType,
		PlatformType: steamcmd.PlatformType(sa.Spec.ImageOpts.PlatformType),
		Execs:        sa.Spec.ImageOpts.Execs,
		Entrypoint:   sa.Spec.ImageOpts.Entrypoint,
		Cmd:          sa.Spec.ImageOpts.Cmd,
	}); err != nil {
		r.Eventf(sa, corev1.EventTypeWarning, "DidNotBuild", "Image did not build: %v", err)
		sa.Status.Phase = v1alpha1.PhaseFailed
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	sa.Status.Phase = v1alpha1.PhaseReady

	if err := r.Client.Status().Update(ctx, sa); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SteamappReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.EventRecorder = mgr.GetEventRecorderFor("sindri")

	if err := ctrl.NewControllerManagedBy(mgr).
		Named("boiler").
		For(&v1alpha1.Steamapp{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		WithEventFilter(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			if annotations := obj.GetAnnotations(); annotations != nil {
				approved, _ := strconv.ParseBool(annotations[stokercr.AnnotationApproved])
				return approved
			}
			return false
		})).
		Complete(r); err != nil {
		return err
	}

	return nil
	// ctrl.NewWebhookManagedBy(mgr).
	// 	For(&v1alpha1.Steamapp{}).
	// 	WithDefaulter(r).
	// 	WithValidator(r).
	// 	Complete()
}

func (r *SteamappReconciler) Default(_ context.Context, obj runtime.Object) error {
	sa, ok := obj.(*v1alpha1.Steamapp)
	if !ok {
		return fmt.Errorf("expected a Steamapp object but got %T", obj)
	}

	if sa.Status.Phase == "" {
		sa.Status.Phase = v1alpha1.PhasePending
	}

	if sa.Spec.Branch == "" {
		sa.Spec.Branch = steamapp.DefaultBranchName
	}

	if sa.Spec.ImageOpts.LaunchType == "" {
		sa.Spec.ImageOpts.LaunchType = steamapp.DefaultLaunchType
	}

	if sa.Spec.ImageOpts.PlatformType == "" {
		sa.Spec.ImageOpts.PlatformType = steamcmd.PlatformTypeLinux.String()
	}

	if sa.Spec.ImageOpts.BaseImageRef == "" {
		sa.Spec.ImageOpts.BaseImageRef = steamapp.DefaultBaseImageRef
	}

	return nil
}

func (r *SteamappReconciler) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	sa, ok := obj.(*v1alpha1.Steamapp)
	if !ok {
		return nil, fmt.Errorf("expected a Steamapp object but got %T", obj)
	}

	if !xslice.Includes(
		[]steamcmd.PlatformType{
			steamcmd.PlatformTypeLinux,
			steamcmd.PlatformTypeWindows,
			steamcmd.PlatformTypeMacOS,
		},
		steamcmd.PlatformType(sa.Spec.ImageOpts.PlatformType),
	) {
		return nil, fmt.Errorf("unsupported platform type %s", sa.Spec.ImageOpts.PlatformType)
	}

	return nil, nil
}

func (r *SteamappReconciler) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	_, ok := oldObj.(*v1alpha1.Steamapp)
	if !ok {
		return nil, fmt.Errorf("expected a Steamapp object but got %T", oldObj)
	}

	sa, ok := newObj.(*v1alpha1.Steamapp)
	if !ok {
		return nil, fmt.Errorf("expected a Steamapp object but got %T", newObj)
	}

	if !xslice.Includes(
		[]steamcmd.PlatformType{
			steamcmd.PlatformTypeLinux,
			steamcmd.PlatformTypeWindows,
			steamcmd.PlatformTypeMacOS,
		},
		steamcmd.PlatformType(sa.Spec.ImageOpts.PlatformType),
	) {
		return nil, fmt.Errorf("unsupported platform type %s", sa.Spec.ImageOpts.PlatformType)
	}

	return nil, nil
}

func (r *SteamappReconciler) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	_, ok := obj.(*v1alpha1.Steamapp)
	if !ok {
		return nil, fmt.Errorf("expected a Steamapp object but got %T", obj)
	}

	return nil, nil
}
