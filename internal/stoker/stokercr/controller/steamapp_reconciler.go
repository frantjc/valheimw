package controller

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/appinfoutil"
	"github.com/frantjc/sindri/internal/stoker/stokercr"
	"github.com/frantjc/sindri/internal/stoker/stokercr/api/v1alpha1"
	"github.com/frantjc/sindri/steamapp"
	xio "github.com/frantjc/x/io"
	xslice "github.com/frantjc/x/slice"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
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

	if len(sa.Status.Conditions) > 0 && xslice.Every(sa.Status.Conditions, func(condition metav1.Condition, _ int) bool {
		return condition.Status == metav1.ConditionTrue && condition.ObservedGeneration == sa.Generation
	}) {
		return ctrl.Result{}, nil
	}

	sa.Status.Phase = v1alpha1.PhasePending
	sa.Status.Conditions = []metav1.Condition{}

	if err := r.Client.Status().Update(ctx, sa); err != nil {
		return ctrl.Result{}, err
	}

	appInfo, err := appinfoutil.GetAppInfo(ctx, sa.Spec.AppID)
	if err != nil {
		r.Eventf(sa, corev1.EventTypeWarning, "AppInfoPrintFailed", "Could not get app info: %v", err)
		sa.Status.Conditions = append(sa.Status.Conditions, metav1.Condition{
			Type:               "AppInfoPrint",
			Status:             metav1.ConditionFalse,
			ObservedGeneration: sa.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             "AppInfoPrintFailed",
			Message:            err.Error(),
		})
		sa.Status.Phase = v1alpha1.PhaseFailed
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	sa.Status.Conditions = append(sa.Status.Conditions, metav1.Condition{
		Type:               "AppInfoPrint",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: sa.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             "AppInfoPrintSucceeded",
	})
	sa.Status.Name = appInfo.Common.Name

	u, err := url.Parse("https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps")
	if err != nil {
		return ctrl.Result{}, err
	}

	sa.Status.IconURL = u.JoinPath(fmt.Sprint(sa.Spec.AppID), fmt.Sprintf("%s.jpg", appInfo.Common.Icon)).String()

	if err := r.Client.Status().Update(ctx, sa); err != nil {
		return ctrl.Result{}, err
	}

	branch, ok := appInfo.Depots.Branches[sa.Spec.Branch]
	if !ok {
		r.Eventf(sa, corev1.EventTypeWarning, "BranchMissing", "Branch %s not found", sa.Spec.Branch)
		sa.Status.Conditions = append(sa.Status.Conditions, metav1.Condition{
			Type:               "Branch",
			Status:             metav1.ConditionFalse,
			ObservedGeneration: sa.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             "BranchMissing",
			Message:            fmt.Sprintf("Branch %s not found", sa.Spec.Branch),
		})
		sa.Status.Phase = v1alpha1.PhaseFailed
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	betaPwd := sa.Spec.BetaPassword

	if branch.PwdRequired && betaPwd == "" {
		r.Eventf(sa, corev1.EventTypeWarning, "BetaPwdMissing", "Branch %s requires a password", sa.Spec.Branch)
		sa.Status.Conditions = append(sa.Status.Conditions, metav1.Condition{
			Type:               "BetaPwd",
			Status:             metav1.ConditionFalse,
			ObservedGeneration: sa.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             "BetaPwdMissing",
			Message:            fmt.Sprintf("Branch %s requires a password, but none was given", sa.Spec.Branch),
		})
		sa.Status.Phase = v1alpha1.PhaseFailed
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	} else if !branch.PwdRequired && betaPwd != "" {
		r.Eventf(sa, corev1.EventTypeWarning, "UnexpectedBetaPwd", "Branch %s does not require a password, refusing to use given: %s", sa.Spec.Branch, sa.Spec.BetaPassword)
		sa.Status.Conditions = append(sa.Status.Conditions, metav1.Condition{
			Type:               "BetaPwd",
			Status:             metav1.ConditionFalse,
			ObservedGeneration: sa.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             "UnexpectedBetaPwd",
			Message:            fmt.Sprintf("Branch %s does not require a password, refusing to use given: %s", sa.Spec.Branch, sa.Spec.BetaPassword),
		})
		betaPwd = ""
	}

	sa.Status.Conditions = append(sa.Status.Conditions, metav1.Condition{
		Type:               "BetaPwd",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: sa.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             "BetaPwdValid",
	})
	awaitingApproval := true

	if sa.Annotations != nil {
		if approved, _ := strconv.ParseBool(sa.Annotations[stokercr.AnnotationApproved]); approved {
			awaitingApproval = false
		}
	}

	if awaitingApproval {
		r.Event(sa, corev1.EventTypeNormal, "AwaitingApproval", "Steamapp requires approval to build")
		sa.Status.Conditions = append(sa.Status.Conditions, metav1.Condition{
			Type:               "Approved",
			Status:             metav1.ConditionFalse,
			ObservedGeneration: sa.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             "PendingApproval",
			Message:            fmt.Sprintf("Approval not given via annotation %s", stokercr.AnnotationApproved),
		})
		sa.Status.Phase = v1alpha1.PhasePaused
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	r.Eventf(sa, corev1.EventTypeNormal, "Building", "Attempting image build with approval: %s", sa.Annotations[stokercr.AnnotationApproved])
	sa.Status.Conditions = append(sa.Status.Conditions, metav1.Condition{
		Type:               "Approved",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: sa.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             "ApprovalReceived",
	})

	if err := r.BuildImage(ctx, sa.Spec.AppID, xio.WriterCloser{Writer: io.Discard, Closer: xio.CloserFunc(func() error { return nil })}, &steamapp.GettableBuildImageOpts{
		BaseImageRef: sa.Spec.SteamappSpecImageOpts.BaseImageRef,
		AptPkgs:      sa.Spec.SteamappSpecImageOpts.AptPkgs,
		BetaPassword: betaPwd,
		LaunchType:   sa.Spec.SteamappSpecImageOpts.LaunchType,
		PlatformType: steamcmd.PlatformType(sa.Spec.SteamappSpecImageOpts.PlatformType),
		Execs:        sa.Spec.SteamappSpecImageOpts.Execs,
		Entrypoint:   sa.Spec.SteamappSpecImageOpts.Entrypoint,
		Cmd:          sa.Spec.SteamappSpecImageOpts.Cmd,
	}); err != nil {
		r.Eventf(sa, corev1.EventTypeWarning, "DidNotBuild", "Image did not build successfully: %v", err)
		sa.Status.Conditions = append(sa.Status.Conditions, metav1.Condition{
			Type:               "Built",
			Status:             metav1.ConditionFalse,
			ObservedGeneration: sa.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             "BuildFailed",
			Message:            err.Error(),
		})
		sa.Status.Phase = v1alpha1.PhaseFailed
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	r.Event(sa, corev1.EventTypeNormal, "Built", "Image successfully built")
	sa.Status.Conditions = append(sa.Status.Conditions, metav1.Condition{
		Type:               "Built",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: sa.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             "BuildSucceeded",
	})
	sa.Status.Phase = v1alpha1.PhaseReady

	return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SteamappReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.EventRecorder = mgr.GetEventRecorderFor("sindri")

	if err := ctrl.NewControllerManagedBy(mgr).
		Named("boiler").
		For(&v1alpha1.Steamapp{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				if annotations := e.ObjectNew.GetAnnotations(); annotations != nil {
					if approved, _ := strconv.ParseBool(annotations[stokercr.AnnotationApproved]); approved {
						if sa, ok := e.ObjectNew.(*v1alpha1.Steamapp); ok {
							return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() || sa.Status.Phase == v1alpha1.PhasePaused
						}
					}
				}

				return false
			},
			CreateFunc: func(_ event.CreateEvent) bool {
				return true
			},
			DeleteFunc: func(_ event.DeleteEvent) bool {
				return false
			},
			GenericFunc: func(_ event.GenericEvent) bool {
				return false
			},
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

	if sa.Spec.SteamappSpecImageOpts.LaunchType == "" {
		sa.Spec.SteamappSpecImageOpts.LaunchType = steamapp.DefaultLaunchType
	}

	if sa.Spec.SteamappSpecImageOpts.PlatformType == "" {
		sa.Spec.SteamappSpecImageOpts.PlatformType = steamcmd.PlatformTypeLinux.String()
	}

	if sa.Spec.SteamappSpecImageOpts.BaseImageRef == "" {
		sa.Spec.SteamappSpecImageOpts.BaseImageRef = steamapp.DefaultBaseImageRef
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
		steamcmd.PlatformType(sa.Spec.SteamappSpecImageOpts.PlatformType),
	) {
		return nil, fmt.Errorf("unsupported platform type %s", sa.Spec.SteamappSpecImageOpts.PlatformType)
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
		steamcmd.PlatformType(sa.Spec.SteamappSpecImageOpts.PlatformType),
	) {
		return nil, fmt.Errorf("unsupported platform type %s", sa.Spec.SteamappSpecImageOpts.PlatformType)
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
