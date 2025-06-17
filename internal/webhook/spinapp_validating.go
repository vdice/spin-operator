package webhook

import (
	"context"
	"fmt"

	spinv1alpha1 "github.com/spinkube/spin-operator/api/v1alpha1"
	"github.com/spinkube/spin-operator/internal/logging"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// nolint:lll
//+kubebuilder:webhook:path=/validate-core-spinkube-dev-v1alpha1-spinapp,mutating=false,failurePolicy=fail,sideEffects=None,groups=core.spinkube.dev,resources=spinapps,verbs=create;update,versions=v1alpha1,name=vspinapp.kb.io,admissionReviewVersions=v1

// SpinAppValidator validates SpinApps
type SpinAppValidator struct {
	Client client.Client
}

// ValidateCreate implements webhook.Validator
func (v *SpinAppValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	log := logging.FromContext(ctx)

	spinApp := obj.(*spinv1alpha1.SpinApp)
	log.Info("validate create", "name", spinApp.Name)

	return nil, v.validateSpinApp(ctx, spinApp)
}

// ValidateUpdate implements webhook.Validator
func (v *SpinAppValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	log := logging.FromContext(ctx)

	spinApp := newObj.(*spinv1alpha1.SpinApp)
	log.Info("validate update", "name", spinApp.Name)

	return nil, v.validateSpinApp(ctx, spinApp)
}

// ValidateDelete implements webhook.Validator
func (v *SpinAppValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	log := logging.FromContext(ctx)

	spinApp := obj.(*spinv1alpha1.SpinApp)
	log.Info("validate delete", "name", spinApp.Name)

	return nil, nil
}

func (v *SpinAppValidator) validateSpinApp(ctx context.Context, spinApp *spinv1alpha1.SpinApp) error {
	var allErrs field.ErrorList
	executor, err := validateExecutor(spinApp.Spec, v.fetchExecutor(ctx, spinApp.Namespace))
	if err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateReplicas(spinApp.Spec); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateAnnotations(spinApp.Spec, executor); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateInvocationLimits(spinApp.Spec); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "core.spinkube.dev", Kind: "SpinApp"},
		spinApp.Name, allErrs)
}

// fetchExecutor returns a function that fetches a named executor in the provided namespace.
//
// We assume that the executor must exist in the same namespace as the SpinApp.
func (v *SpinAppValidator) fetchExecutor(ctx context.Context, spinAppNs string) func(name string) (*spinv1alpha1.SpinAppExecutor, error) {
	return func(name string) (*spinv1alpha1.SpinAppExecutor, error) {
		var executor spinv1alpha1.SpinAppExecutor
		if err := v.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: spinAppNs}, &executor); err != nil {
			return nil, err
		}

		return &executor, nil
	}
}

func validateExecutor(spec spinv1alpha1.SpinAppSpec, fetchExecutor func(name string) (*spinv1alpha1.SpinAppExecutor, error)) (*spinv1alpha1.SpinAppExecutor, *field.Error) {
	if spec.Executor == "" {
		return nil, field.Invalid(
			field.NewPath("spec").Child("executor"),
			spec.Executor,
			"executor must be set, likely no default executor was set because you have no executors installed")
	}
	executor, err := fetchExecutor(spec.Executor)
	if err != nil {
		// Handle errors that are not just "Not Found"
		return nil, field.Invalid(field.NewPath("spec").Child("executor"), spec.Executor, "executor does not exist in namespace")
	}

	return executor, nil
}

func validateReplicas(spec spinv1alpha1.SpinAppSpec) *field.Error {
	if spec.EnableAutoscaling && spec.Replicas != 0 {
		return field.Invalid(field.NewPath("spec").Child("replicas"), spec.Replicas, "replicas cannot be set when autoscaling is enabled")
	}
	if !spec.EnableAutoscaling && spec.Replicas < 1 {
		return field.Invalid(field.NewPath("spec").Child("replicas"), spec.Replicas, "replicas must be > 0")
	}

	return nil
}

func validateAnnotations(spec spinv1alpha1.SpinAppSpec, executor *spinv1alpha1.SpinAppExecutor) *field.Error {
	// We can't do any validation if the executor isn't available, but validation
	// will fail because of earlier errors.
	if executor == nil {
		return nil
	}

	if executor.Spec.CreateDeployment {
		return nil
	}
	// TODO: Make these validations opt in for executors? - Some runtimes may want these regardless.
	if len(spec.DeploymentAnnotations) != 0 {
		return field.Invalid(
			field.NewPath("spec").Child("deploymentAnnotations"),
			spec.DeploymentAnnotations,
			"deploymentAnnotations can't be set when the executor does not use operator deployments")
	}
	if len(spec.PodAnnotations) != 0 {
		return field.Invalid(field.NewPath("spec").Child("podAnnotations"), spec.PodAnnotations, "podAnnotations can't be set when the executor does not use operator deployments")
	}

	return nil
}

// Helper function to validate invocation quantity against resource constraints
func validateInvocationQuantity(invocationQuantity resource.Quantity, spec spinv1alpha1.SpinAppSpec, resourceType corev1.ResourceName) *field.Error {
	requestQuantity, hasRequest := spec.Resources.Requests[resourceType]
	limitQuantity, hasLimit := spec.Resources.Limits[resourceType]

	// Determine the constraint to compare against
	var constraintQuantity resource.Quantity
	var constraintType string

	switch {
	case hasRequest && hasLimit:
		// Both exist - use the limit
		constraintQuantity = limitQuantity
		constraintType = "limit"
	case hasRequest:
		// Only request exists
		constraintQuantity = requestQuantity
		constraintType = "request"
	case hasLimit:
		// Only limit exists
		constraintQuantity = limitQuantity
		constraintType = "limit"
	default:
		// Neither exists - no validation needed
		return nil
	}

	// Check if invocation quantity exceeds the constraint
	if invocationQuantity.Cmp(constraintQuantity) > 0 {
		return field.Invalid(
			field.NewPath("spec").Child("invocationLimits").Key(resourceType.String()),
			invocationQuantity.String(),
			fmt.Sprintf("invocation limit quantity cannot be greater than the %s %s (%s)",
				resourceType, constraintType, constraintQuantity.String()))
	}

	return nil
}

// validateInvocationLimits checks if the known invocation limits are valid.
func validateInvocationLimits(spec spinv1alpha1.SpinAppSpec) *field.Error {
	if limit, exists := spec.InvocationLimits["memory"]; exists {
		// The memory limit must be a valid Kubernetes quantity.
		invocationQuantity, err := resource.ParseQuantity(limit)
		if err != nil {
			return field.Invalid(
				field.NewPath("spec").Child("invocationLimits").Key("memory"),
				limit,
				"memory limit must be a valid memory quantity (e.g. 128Mi, 1Gi)")
		}

		return validateInvocationQuantity(invocationQuantity, spec, corev1.ResourceMemory)
	}

	return nil
}
