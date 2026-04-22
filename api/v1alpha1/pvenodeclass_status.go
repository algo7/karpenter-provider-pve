package v1alpha1

import (
	"github.com/awslabs/operatorpkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition types set on PVENodeClass status.
const (
	// ConditionTypeTemplateResolved indicates spec.nodeImageRef resolves
	// to an existing PVENodeImage in the cluster.
	ConditionTypeTemplateResolved = "TemplateResolved"

	// ConditionTypeNodeImageReady indicates the referenced PVENodeImage
	// has successfully built its template and reports Ready=True.
	ConditionTypeNodeImageReady = "NodeImageReady"
)

// GetConditions returns the status conditions as operatorpkg's Condition type.
// Required by the status.Object interface.
func (in *PVENodeClass) GetConditions() []status.Condition {
	out := make([]status.Condition, len(in.Status.Conditions))
	for i, c := range in.Status.Conditions {
		out[i] = status.Condition(c)
	}
	return out
}

// SetConditions replaces the status conditions, converting from operatorpkg's
// Condition type to metav1.Condition for storage.
// Required by the status.Object interface.
func (in *PVENodeClass) SetConditions(conditions []status.Condition) {
	out := make([]metav1.Condition, len(conditions))
	for i, c := range conditions {
		out[i] = metav1.Condition(c)
	}
	in.Status.Conditions = out
}

// StatusConditions returns a ConditionSet that aggregates the subconditions
// into a root Ready condition.
// Required by the status.Object interface.
func (in *PVENodeClass) StatusConditions() status.ConditionSet {
	return status.NewReadyConditions(
		ConditionTypeTemplateResolved,
		ConditionTypeNodeImageReady,
	).For(in)
}
