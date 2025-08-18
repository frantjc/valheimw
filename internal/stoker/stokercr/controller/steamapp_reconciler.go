package controller

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strconv"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/appinfoutil"
	"github.com/frantjc/sindri/internal/stoker/stokercr"
	"github.com/frantjc/sindri/internal/stoker/stokercr/api/v1alpha1"
	"github.com/frantjc/sindri/steamapp"
	xio "github.com/frantjc/x/io"
	xslices "github.com/frantjc/x/slices"
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

	if len(sa.Status.Conditions) > 0 && xslices.Every(sa.Status.Conditions, func(condition metav1.Condition, _ int) bool {
		return condition.Status == metav1.ConditionTrue && condition.ObservedGeneration == sa.Generation
	}) {
		return ctrl.Result{}, nil
	}

	sa.Status.Phase = v1alpha1.PhasePending

	if err := r.Client.Status().Update(ctx, sa); err != nil {
		return ctrl.Result{}, err
	}

	appInfo, err := appinfoutil.GetAppInfo(ctx, sa.Spec.AppID)
	if err != nil {
		r.Eventf(sa, corev1.EventTypeWarning, "AppInfoPrintFailed", "Could not get app info: %v", err)
		SetCondition(sa, metav1.Condition{
			Type:    "AppInfoPrint",
			Status:  metav1.ConditionFalse,
			Reason:  "AppInfoPrintFailed",
			Message: err.Error(),
		})
		sa.Status.Phase = v1alpha1.PhaseFailed
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	SetCondition(sa, metav1.Condition{
		Type:   "AppInfoPrint",
		Status: metav1.ConditionTrue,
		Reason: "AppInfoPrintSucceeded",
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
		SetCondition(sa, metav1.Condition{
			Type:    "Branch",
			Status:  metav1.ConditionFalse,
			Reason:  "BranchMissing",
			Message: fmt.Sprintf("Branch %s not found", sa.Spec.Branch),
		})
		sa.Status.Phase = v1alpha1.PhaseFailed
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	betaPwd := sa.Spec.BetaPassword

	if branch.PwdRequired && betaPwd == "" {
		r.Eventf(sa, corev1.EventTypeWarning, "BetaPwdMissing", "Branch %s requires a password", sa.Spec.Branch)
		SetCondition(sa, metav1.Condition{
			Type:    "BetaPwd",
			Status:  metav1.ConditionFalse,
			Reason:  "BetaPwdMissing",
			Message: fmt.Sprintf("Branch %s requires a password, but none was given", sa.Spec.Branch),
		})
		sa.Status.Phase = v1alpha1.PhaseFailed
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	} else if !branch.PwdRequired && betaPwd != "" {
		r.Eventf(sa, corev1.EventTypeWarning, "UnexpectedBetaPwd", "Branch %s does not require a password, refusing to use given: %s", sa.Spec.Branch, sa.Spec.BetaPassword)
		SetCondition(sa, metav1.Condition{
			Type:    "BetaPwd",
			Status:  metav1.ConditionFalse,
			Reason:  "UnexpectedBetaPwd",
			Message: fmt.Sprintf("Branch %s does not require a password, refusing to use given: %s", sa.Spec.Branch, sa.Spec.BetaPassword),
		})
		betaPwd = ""
	}

	SetCondition(sa, metav1.Condition{
		Type:   "BetaPwd",
		Status: metav1.ConditionTrue,
		Reason: "BetaPwdValid",
	})
	awaitingApproval := true

	if sa.Annotations != nil {
		if approved, _ := strconv.ParseBool(sa.Annotations[stokercr.AnnotationApproved]); approved {
			awaitingApproval = false
		}
	}

	opts := &steamapp.BuildImageOpts{
		BaseImageRef: sa.Spec.BaseImageRef,
		AptPkgs:      sa.Spec.AptPkgs,
		BetaPassword: betaPwd,
		LaunchType:   sa.Spec.LaunchType,
		PlatformType: steamcmd.PlatformType(sa.Spec.PlatformType),
		Execs:        sa.Spec.Execs,
		Entrypoint:   sa.Spec.Entrypoint,
		Cmd:          sa.Spec.Cmd,
	}

	imageConfig, err := r.GetImageConfig(ctx, sa.Spec.AppID, opts)
	if err != nil {
		sa.Status.Phase = v1alpha1.PhaseFailed
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	// Propagate the default values back to the Steamapp's spec.
	sa.Spec.Cmd = imageConfig.Cmd
	sa.Spec.Entrypoint = imageConfig.Entrypoint

	if err := r.Update(ctx, sa); err != nil {
		return ctrl.Result{}, err
	}

	if awaitingApproval {
		r.Event(sa, corev1.EventTypeNormal, "AwaitingApproval", "Steamapp requires approval to build")
		SetCondition(sa, metav1.Condition{
			Type:    "Approved",
			Status:  metav1.ConditionFalse,
			Reason:  "PendingApproval",
			Message: fmt.Sprintf("Approval not given via annotation %s", stokercr.AnnotationApproved),
		})
		sa.Status.Phase = v1alpha1.PhasePaused
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	r.Eventf(sa, corev1.EventTypeNormal, "Building", "Attempting image build with approval: %s", sa.Annotations[stokercr.AnnotationApproved])
	SetCondition(sa, metav1.Condition{
		Type:   "Approved",
		Status: metav1.ConditionTrue,
		Reason: "ApprovalReceived",
	})

	if err := r.BuildImage(ctx, sa.Spec.AppID, xio.WriterCloser{Writer: io.Discard, Closer: xio.CloserFunc(func() error { return nil })}, opts); err != nil {
		r.Eventf(sa, corev1.EventTypeWarning, "DidNotBuild", "Image did not build successfully: %v", err)
		SetCondition(sa, metav1.Condition{
			Type:    "Built",
			Status:  metav1.ConditionFalse,
			Reason:  "BuildFailed",
			Message: err.Error(),
		})
		sa.Status.Phase = v1alpha1.PhaseFailed
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	r.Event(sa, corev1.EventTypeNormal, "Built", "Image successfully built")
	SetCondition(sa, metav1.Condition{
		Type:   "Built",
		Status: metav1.ConditionTrue,
		Reason: "BuildSucceeded",
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

	if sa.Spec.LaunchType == "" {
		sa.Spec.LaunchType = steamapp.DefaultLaunchType
	}

	if sa.Spec.PlatformType == "" {
		sa.Spec.PlatformType = steamcmd.PlatformTypeLinux.String()
	}

	if sa.Spec.BaseImageRef == "" {
		sa.Spec.BaseImageRef = steamapp.DefaultBaseImageRef
	}

	return nil
}

func (r *SteamappReconciler) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	sa, ok := obj.(*v1alpha1.Steamapp)
	if !ok {
		return nil, fmt.Errorf("expected a Steamapp object but got %T", obj)
	}

	if !slices.Contains(
		[]steamcmd.PlatformType{
			steamcmd.PlatformTypeLinux,
			steamcmd.PlatformTypeWindows,
			steamcmd.PlatformTypeMacOS,
		},
		steamcmd.PlatformType(sa.Spec.PlatformType),
	) {
		return nil, fmt.Errorf("unsupported platform type %s", sa.Spec.PlatformType)
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

	if !slices.Contains(
		[]steamcmd.PlatformType{
			steamcmd.PlatformTypeLinux,
			steamcmd.PlatformTypeWindows,
			steamcmd.PlatformTypeMacOS,
		},
		steamcmd.PlatformType(sa.Spec.PlatformType),
	) {
		return nil, fmt.Errorf("unsupported platform type %s", sa.Spec.PlatformType)
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

type ConditionsAware interface {
	GetGeneration() int64
	GetConditions() []metav1.Condition
	SetConditions(conditions []metav1.Condition)
}

func SetCondition(conditionsAware ConditionsAware, condition metav1.Condition) {
	conditions := conditionsAware.GetConditions()
	if conditions == nil {
		conditions = []metav1.Condition{}
	}

	for i, c := range conditions {
		if c.Type == condition.Type {
			if c.Message != condition.Message || c.Reason != condition.Reason || c.Status != condition.Status {
				condition.LastTransitionTime = metav1.Now()
				condition.ObservedGeneration = conditionsAware.GetGeneration()
				conditions[i] = condition
				conditionsAware.SetConditions(conditions)
			}
			return
		}
	}

	condition.LastTransitionTime = metav1.Now()
	condition.ObservedGeneration = conditionsAware.GetGeneration()
	conditions = append(conditions, condition)
	conditionsAware.SetConditions(conditions)
}
