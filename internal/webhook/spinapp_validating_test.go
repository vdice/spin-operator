package webhook

import (
	"errors"
	"testing"

	spinv1alpha1 "github.com/spinframework/spin-operator/api/v1alpha1"
	"github.com/spinframework/spin-operator/internal/constants"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestValidateExecutor(t *testing.T) {
	t.Parallel()

	_, fldErr := validateExecutor(spinv1alpha1.SpinAppSpec{}, func(string) (*spinv1alpha1.SpinAppExecutor, error) { return nil, nil })
	require.EqualError(t, fldErr, "spec.executor: Invalid value: \"\": executor must be set, likely no default executor was set because you have no executors installed")

	_, fldErr = validateExecutor(
		spinv1alpha1.SpinAppSpec{Executor: constants.CyclotronExecutor},
		func(string) (*spinv1alpha1.SpinAppExecutor, error) { return nil, errors.New("executor not found?") })
	require.EqualError(t, fldErr, "spec.executor: Invalid value: \"cyclotron\": executor does not exist in namespace")

	_, fldErr = validateExecutor(spinv1alpha1.SpinAppSpec{Executor: constants.ContainerDShimSpinExecutor}, func(string) (*spinv1alpha1.SpinAppExecutor, error) { return nil, nil })
	require.Nil(t, fldErr)
}

func TestValidateReplicas(t *testing.T) {
	t.Parallel()

	fldErr := validateReplicas(spinv1alpha1.SpinAppSpec{})
	require.EqualError(t, fldErr, "spec.replicas: Invalid value: 0: replicas must be > 0")

	fldErr = validateReplicas(spinv1alpha1.SpinAppSpec{Replicas: 1})
	require.Nil(t, fldErr)
}

func TestValidateAnnotations(t *testing.T) {
	t.Parallel()

	deploymentlessExecutor := &spinv1alpha1.SpinAppExecutor{
		Spec: spinv1alpha1.SpinAppExecutorSpec{
			CreateDeployment: false,
		},
	}
	deploymentfullExecutor := &spinv1alpha1.SpinAppExecutor{
		Spec: spinv1alpha1.SpinAppExecutorSpec{
			CreateDeployment: true,
		},
	}

	fldErr := validateAnnotations(spinv1alpha1.SpinAppSpec{
		Executor:              "an-executor",
		DeploymentAnnotations: map[string]string{"key": "asdf"},
	}, deploymentlessExecutor)
	require.EqualError(t, fldErr,
		`spec.deploymentAnnotations: Invalid value: {"key":"asdf"}: `+
			`deploymentAnnotations can't be set when the executor does not use operator deployments`)

	fldErr = validateAnnotations(spinv1alpha1.SpinAppSpec{
		Executor:       "an-executor",
		PodAnnotations: map[string]string{"key": "asdf"},
	}, deploymentlessExecutor)
	require.EqualError(t, fldErr,
		`spec.podAnnotations: Invalid value: {"key":"asdf"}: `+
			`podAnnotations can't be set when the executor does not use operator deployments`)

	fldErr = validateAnnotations(spinv1alpha1.SpinAppSpec{
		Executor:              "an-executor",
		DeploymentAnnotations: map[string]string{"key": "asdf"},
	}, deploymentfullExecutor)
	require.Nil(t, fldErr)

	fldErr = validateAnnotations(spinv1alpha1.SpinAppSpec{
		Executor: "an-executor",
	}, deploymentlessExecutor)
	require.Nil(t, fldErr)
}

func TestValidateInvocationLimits(t *testing.T) {
	t.Parallel()
	spec := spinv1alpha1.SpinAppSpec{Replicas: 1}
	fldErr := validateInvocationLimits(spec)
	require.Nil(t, fldErr)

	// Only invocation limit set
	spec.InvocationLimits = map[string]string{
		"memory": "50Mi",
	}
	fldErr = validateInvocationLimits(spec)
	require.Nil(t, fldErr)

	// Invocation limit > resource request
	spec.Resources = spinv1alpha1.Resources{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("40Mi")},
	}
	fldErr = validateInvocationLimits(spec)
	require.EqualError(t, fldErr, "spec.invocationLimits[memory]: Invalid value: \"50Mi\": invocation limit quantity cannot be greater than the memory request (40Mi)")

	// Invocation limit > resource request < resource limit
	spec.Resources = spinv1alpha1.Resources{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("40Mi")},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("100Mi")},
	}
	fldErr = validateInvocationLimits(spec)
	require.Nil(t, fldErr)

	// Invocation limit < resource limit
	spec.Resources = spinv1alpha1.Resources{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("100Mi")},
	}
	fldErr = validateInvocationLimits(spec)
	require.Nil(t, fldErr)

	// Invocation limit > resource limit
	spec.Resources = spinv1alpha1.Resources{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("40Mi")},
	}
	fldErr = validateInvocationLimits(spec)
	require.EqualError(t, fldErr, "spec.invocationLimits[memory]: Invalid value: \"50Mi\": invocation limit quantity cannot be greater than the memory limit (40Mi)")
}
