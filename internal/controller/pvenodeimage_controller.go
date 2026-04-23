/*
Copyright 2026.

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

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	karpenterv1alpha1 "github.com/algo7/karpenter-provider-pve/api/v1alpha1"
)

// Condition types reported on PVENodeImage.Status.Conditions.
const (
	// ConditionTypeBuilding is True while a Packer Job is running for the
	// current spec hash, False otherwise.
	ConditionTypeBuilding = "Building"

	// ConditionTypeBuildSucceeded is True when the most recent build for
	// the current spec hash completed successfully.
	ConditionTypeBuildSucceeded = "BuildSucceeded"
)

// Reason values used in condition updates.
const (
	ReasonBuildPending    = "BuildPending"
	ReasonBuildStarted    = "BuildStarted"
	ReasonBuildInProgress = "BuildInProgress"
	ReasonBuildComplete   = "BuildComplete"
	ReasonBuildFailed     = "BuildFailed"
	ReasonSpecChanged     = "SpecChanged"
	ReasonTemplateCurrent = "TemplateCurrent"
)

// Requeue intervals used during various reconcile outcomes.
const (
	// requeueWhileBuilding is the interval between polls while a Job is
	// running. Event-driven watches will usually fire first when the Job
	// transitions, but we requeue as a safety net.
	requeueWhileBuilding = 30 * time.Second
)

// PVENodeImageReconciler reconciles a PVENodeImage object.
type PVENodeImageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=karpenter.algo7.dev,resources=pvenodeimages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=karpenter.algo7.dev,resources=pvenodeimages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=karpenter.algo7.dev,resources=pvenodeimages/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile drives a PVENodeImage toward having a Proxmox template that
// matches its spec. It is deliberately stateless across invocations: each
// call reads current state, takes at most one action, and returns.
func (r *PVENodeImageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("pvenodeimage", req.Name)

	// Fetch the CR. If not found, it was deleted and owned resources are
	// cleaned up via ownerReference GC — nothing to do here.
	image := &karpenterv1alpha1.PVENodeImage{}
	if err := r.Get(ctx, req.NamespacedName, image); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("fetch PVENodeImage: %w", err)
	}

	// Respect deletion. Owned Jobs and ConfigMaps are GC'd via ownerRefs.
	if !image.DeletionTimestamp.IsZero() {
		log.V(1).Info("resource is being deleted, nothing to do")
		return ctrl.Result{}, nil
	}

	// Compute the hash of the current spec. A mismatch with the stored
	// ObservedSpecHash is the signal that a rebuild is needed.
	desiredHash, err := hashSpec(&image.Spec)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("hash spec: %w", err)
	}
	log = log.WithValues("desiredHash", desiredHash[:12])

	// Case 1: the current spec has already been successfully built.
	// Nothing to do. Keep conditions accurate and return.
	if image.Status.ObservedSpecHash == desiredHash && image.Status.TemplateVMID != nil {
		log.V(1).Info("template is current")
		setCondition(image, ConditionTypeBuilding, metav1.ConditionFalse,
			ReasonBuildComplete, "Template is up to date")
		setCondition(image, ConditionTypeBuildSucceeded, metav1.ConditionTrue,
			ReasonTemplateCurrent, "Template matches current spec")
		return ctrl.Result{}, r.updateStatus(ctx, image)
	}

	// Look for a Job owned by this CR, labeled with the desired hash.
	// The label lets us distinguish "Job for current spec" from "stale Job
	// from a previous spec version."
	job, err := r.findBuildJob(ctx, image, desiredHash)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("find build job: %w", err)
	}

	// Case 2: no Job exists yet for this spec. Need to start one.
	if job == nil {
		log.Info("no build job exists, would start one")
		setCondition(image, ConditionTypeBuilding, metav1.ConditionTrue,
			ReasonBuildPending, "Build Job not yet created")
		setCondition(image, ConditionTypeBuildSucceeded, metav1.ConditionFalse,
			ReasonSpecChanged, "Build in progress for new spec")

		// TODO(session B): actually create the ConfigMap and Job here.
		// For now we just update status and requeue.

		if err := r.updateStatus(ctx, image); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: requeueWhileBuilding}, nil
	}

	log = log.WithValues("job", job.Name)

	// Case 3: Job exists, still running. Surface that and wait.
	if !isJobFinished(job) {
		log.V(1).Info("build job in progress")
		setCondition(image, ConditionTypeBuilding, metav1.ConditionTrue,
			ReasonBuildInProgress, "Packer build is running")
		setCondition(image, ConditionTypeBuildSucceeded, metav1.ConditionFalse,
			ReasonBuildInProgress, "Build in progress")

		if err := r.updateStatus(ctx, image); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: requeueWhileBuilding}, nil
	}

	// Case 4: Job finished. Either it succeeded or it failed.
	if jobSucceeded(job) {
		log.Info("build job succeeded")

		// TODO(session B): extract VMID from the Pod's termination log
		// and populate image.Status.TemplateVMID and TemplateNode.

		image.Status.ObservedSpecHash = desiredHash
		now := metav1.Now()
		image.Status.LastBuildTime = &now

		setCondition(image, ConditionTypeBuilding, metav1.ConditionFalse,
			ReasonBuildComplete, "Build completed successfully")
		setCondition(image, ConditionTypeBuildSucceeded, metav1.ConditionTrue,
			ReasonBuildComplete, "Template built and available")

		return ctrl.Result{}, r.updateStatus(ctx, image)
	}

	// Job failed. Record the failure and stop requeueing — we wait for a
	// spec change (which produces a new hash and starts a fresh Job) or
	// for manual intervention.
	log.Info("build job failed")
	setCondition(image, ConditionTypeBuilding, metav1.ConditionFalse,
		ReasonBuildFailed, "Packer build failed; see Job logs")
	setCondition(image, ConditionTypeBuildSucceeded, metav1.ConditionFalse,
		ReasonBuildFailed, "Most recent build failed")

	return ctrl.Result{}, r.updateStatus(ctx, image)
}

// SetupWithManager registers the reconciler with the manager. The Watches
// call on batch/v1 Jobs ensures the reconciler wakes up when a build Job
// changes state, regardless of the polling requeue interval.
func (r *PVENodeImageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&karpenterv1alpha1.PVENodeImage{}).
		Watches(
			&batchv1.Job{},
			handler.EnqueueRequestForOwner(
				mgr.GetScheme(),
				mgr.GetRESTMapper(),
				&karpenterv1alpha1.PVENodeImage{},
				handler.OnlyControllerOwner(),
			),
			builder.WithPredicates(),
		).
		Named("pvenodeimage").
		Complete(r)
}

// updateStatus writes the CR's current in-memory status to the API server.
// Separated into its own method so every reconcile path can use the same
// update logic without duplicating error handling.
func (r *PVENodeImageReconciler) updateStatus(ctx context.Context, image *karpenterv1alpha1.PVENodeImage) error {
	if err := r.Status().Update(ctx, image); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

// findBuildJob returns the Job owned by the given PVENodeImage matching the
// desired spec hash, or nil if none exists.
//
// Session B note: we'll label Jobs with the spec hash at creation time, so
// this filter resolves "is there a Job for this exact spec?" cleanly.
func (r *PVENodeImageReconciler) findBuildJob(
	ctx context.Context,
	image *karpenterv1alpha1.PVENodeImage,
	desiredHash string,
) (*batchv1.Job, error) {
	jobs := &batchv1.JobList{}
	if err := r.List(ctx, jobs,
		client.InNamespace(controllerNamespace()),
		client.MatchingLabels{
			labelSpecHash:    desiredHash,
			labelManagedByCR: image.Name,
		},
	); err != nil {
		return nil, err
	}

	for i := range jobs.Items {
		if isOwnedBy(&jobs.Items[i], image) {
			return &jobs.Items[i], nil
		}
	}
	return nil, nil
}

// Label keys attached to build Jobs and their ConfigMaps.
const (
	labelSpecHash    = "karpenter.algo7.dev/spec-hash"
	labelManagedByCR = "karpenter.algo7.dev/pvenodeimage"
)

// isOwnedBy reports whether obj has an ownerReference pointing to owner.
func isOwnedBy(obj client.Object, owner *karpenterv1alpha1.PVENodeImage) bool {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.UID == owner.UID {
			return true
		}
	}
	return false
}

// isJobFinished reports whether a Job has reached a terminal state.
func isJobFinished(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) &&
			c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// jobSucceeded reports whether a terminal Job succeeded.
func jobSucceeded(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// hashSpec computes a stable SHA256 of the fields in spec that affect the
// built template. YAML marshaling gives a deterministic ordering.
//
// Fields that do NOT affect the built template (e.g., RebuildPolicy.Trigger
// changes don't change the image itself) could be excluded here, but for
// v0.1 we hash the whole spec — simpler, and a trivial unnecessary rebuild
// is better than a silent miss.
func hashSpec(spec *karpenterv1alpha1.PVENodeImageSpec) (string, error) {
	data, err := yaml.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("marshal spec: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// setCondition updates or appends a condition on the image's status. It
// preserves LastTransitionTime semantics (only updated when Status changes).
func setCondition(
	image *karpenterv1alpha1.PVENodeImage,
	condType string,
	status metav1.ConditionStatus,
	reason, message string,
) {
	cond := metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: image.Generation,
		LastTransitionTime: metav1.Now(),
	}

	// Look for an existing condition of this type.
	for i, existing := range image.Status.Conditions {
		if existing.Type == condType {
			// Preserve LastTransitionTime if status didn't change.
			if existing.Status == status {
				cond.LastTransitionTime = existing.LastTransitionTime
			}
			image.Status.Conditions[i] = cond
			return
		}
	}

	// No existing condition of this type; append.
	image.Status.Conditions = append(image.Status.Conditions, cond)
}

// controllerNamespace returns the namespace the controller is running in.
// Build Jobs and their ConfigMaps live in this namespace.
//
// Session B note: this is a stub. The real implementation reads
// POD_NAMESPACE (set via downward API) or /var/run/secrets/.../namespace.
func controllerNamespace() string {
	// TODO(session B): read POD_NAMESPACE env var, fall back to the
	// service account namespace file.
	return "karpenter"
}
