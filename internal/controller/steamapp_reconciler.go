package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/api/v1alpha1"
	"github.com/frantjc/sindri/internal/appinfoutil"
	"github.com/frantjc/sindri/internal/logutil"
	"github.com/frantjc/sindri/steamapp"
	xio "github.com/frantjc/x/io"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// SteamappReconciler reconciles a Steamapp object.
type SteamappReconciler struct {
	client.Client
	record.EventRecorder
	*steamapp.ImageBuilder
	ImageScanner
}

// +kubebuilder:rbac:groups=sindri.frantj.cc,resources=steamapps,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=sindri.frantj.cc,resources=steamapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sindri.frantj.cc,resources=steamapps/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;patch

const LogKey = "buildkitd.log"

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SteamappReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var (
		log = logutil.SloggerFrom(ctx)
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

	r.Eventf(sa, corev1.EventTypeNormal, "BranchFound", "Branch %s found", sa.Spec.Branch)
	SetCondition(sa, metav1.Condition{
		Type:   "Branch",
		Status: metav1.ConditionTrue,
		Reason: "BranchFound",
	})

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
		if approved, _ := strconv.ParseBool(sa.Annotations[AnnotationApproved]); approved {
			awaitingApproval = false
		}
	}

	var (
		buf  = new(bytes.Buffer)
		opts = &steamapp.BuildImageOpts{
			BaseImageRef: sa.Spec.BaseImageRef,
			AptPkgs:      sa.Spec.AptPkgs,
			BetaPassword: betaPwd,
			LaunchType:   sa.Spec.LaunchType,
			PlatformType: steamcmd.PlatformType(sa.Spec.PlatformType),
			Execs:        sa.Spec.Execs,
			Entrypoint:   sa.Spec.Entrypoint,
			Cmd:          sa.Spec.Cmd,
			Log:          buf,
		}
	)

	imageConfig, err := r.GetImageConfig(ctx, sa.Spec.AppID, opts)
	if err != nil {
		r.Eventf(sa, corev1.EventTypeWarning, "ImageConfigFailed", "Could not get image config: %v", err)
		SetCondition(sa, metav1.Condition{
			Type:    "ImageConfig",
			Status:  metav1.ConditionFalse,
			Reason:  "ImageConfigFailed",
			Message: err.Error(),
		})
		sa.Status.Phase = v1alpha1.PhaseFailed
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	r.Eventf(sa, corev1.EventTypeNormal, "ImageConfigSucceeded", "Got image config")
	SetCondition(sa, metav1.Condition{
		Type:   "ImageConfig",
		Status: metav1.ConditionTrue,
		Reason: "ImageConfigSucceeded",
	})

	// Propagate the default values back to the Steamapp's spec.
	sa.Spec.Cmd = imageConfig.Cmd
	sa.Spec.Entrypoint = imageConfig.Entrypoint

	if err := r.Update(ctx, sa); err != nil {
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	if awaitingApproval {
		r.Event(sa, corev1.EventTypeNormal, "AwaitingApproval", "Steamapp requires approval to build")
		SetCondition(sa, metav1.Condition{
			Type:    "Approved",
			Status:  metav1.ConditionFalse,
			Reason:  "PendingApproval",
			Message: fmt.Sprintf("Approval not given via annotation %s", AnnotationApproved),
		})
		sa.Status.Phase = v1alpha1.PhasePaused
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	r.Eventf(sa, corev1.EventTypeNormal, "Building", "Attempting image build with approval: %s", sa.Annotations[AnnotationApproved])
	SetCondition(sa, metav1.Condition{
		Type:   "Approved",
		Status: metav1.ConditionTrue,
		Reason: "ApprovalReceived",
	})

	var (
		pr io.Reader
		pw io.WriteCloser = &xio.WriterCloser{Writer: io.Discard, Closer: xio.CloserFunc(func() error { return nil })}
	)
	if r.ImageScanner != nil {
		// Build and scan image in parallel.
		pr, pw = io.Pipe()
	}

	eg, egctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		var errs error

		if err := r.BuildImage(
			egctx,
			sa.Spec.AppID,
			xio.WriterCloser{Writer: pw, Closer: xio.CloserFunc(func() error { return nil })},
			opts,
		); err != nil {
			r.Eventf(sa, corev1.EventTypeWarning, "DidNotBuild", "Image did not build successfully: %v", err)
			SetCondition(sa, metav1.Condition{
				Type:    "Built",
				Status:  metav1.ConditionFalse,
				Reason:  "BuildFailed",
				Message: err.Error(),
			})

			errs = errors.Join(errs, err)
		}

		// Close this so that the scan can begin ASAP.
		errs = errors.Join(errs, pw.Close())

		r.Event(sa, corev1.EventTypeNormal, "Built", "Image successfully built")
		SetCondition(sa, metav1.Condition{
			Type:   "Built",
			Status: metav1.ConditionTrue,
			Reason: "BuildSucceeded",
		})

		var (
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sa.Name,
					Namespace: sa.Namespace,
				},
				Data: map[string]string{
					LogKey: buf.String(),
				},
			}
		)

		if err := controllerutil.SetControllerReference(sa, cm, r.Scheme()); err != nil {
			return errors.Join(errs, err)
		}

		if _, err := controllerutil.CreateOrPatch(ctx, r, cm, func() error {
			cm.Data = map[string]string{
				LogKey: buf.String(),
			}
			return controllerutil.SetControllerReference(sa, cm, r.Scheme())
		}); err != nil {
			return errors.Join(errs, err)
		}

		return errs
	})

	if r.ImageScanner != nil {
		eg.Go(func() error {
			vulns, err := r.Scan(egctx, pr)
			if err != nil {
				r.Eventf(sa, corev1.EventTypeWarning, "ScanFailed", "Image scan failed: %v", err)
				SetCondition(sa, metav1.Condition{
					Type:    "Scanned",
					Status:  metav1.ConditionFalse,
					Reason:  "ScanFailed",
					Message: err.Error(),
				})

				return err
			}

			r.Event(sa, corev1.EventTypeNormal, "ScanFinished", "Image scan finished")
			SetCondition(sa, metav1.Condition{
				Type:   "Scanned",
				Status: metav1.ConditionTrue,
				Reason: "ScanFinished",
			})

			sa.Status.Vulnerabilities = vulns

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		sa.Status.Phase = v1alpha1.PhaseFailed
		return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
	}

	sa.Status.Phase = v1alpha1.PhaseReady

	return ctrl.Result{}, r.Client.Status().Update(ctx, sa)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SteamappReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.EventRecorder = mgr.GetEventRecorderFor("sindri")

	if err := ctrl.NewControllerManagedBy(mgr).
		Named("boiler").
		For(&v1alpha1.Steamapp{}, builder.WithPredicates(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}))).
		Complete(r); err != nil {
		return err
	}

	return nil
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

	condition.ObservedGeneration = conditionsAware.GetGeneration()

	for i, c := range conditions {
		if c.Type == condition.Type {
			if c.Message != condition.Message || c.Reason != condition.Reason || c.Status != condition.Status {
				condition.LastTransitionTime = metav1.Now()
			} else {
				condition.LastTransitionTime = c.LastTransitionTime
			}
			conditions[i] = condition
			conditionsAware.SetConditions(conditions)
			return
		}
	}

	condition.LastTransitionTime = metav1.Now()
	conditions = append(conditions, condition)
	conditionsAware.SetConditions(conditions)
}
