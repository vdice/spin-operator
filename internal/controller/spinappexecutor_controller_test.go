/*
Copyright 2023.

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
	"sync"
	"testing"
	"time"

	spinv1alpha1 "github.com/spinkube/spin-operator/api/v1alpha1"
	"github.com/spinkube/spin-operator/internal/generics"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func setupExecutorController(t *testing.T) (*envTestState, ctrl.Manager, *SpinAppExecutorReconciler) {
	t.Helper()

	envTest := SetupEnvTest(t)

	opts := zap.Options{
		Development: true,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))

	mgr, err := ctrl.NewManager(envTest.cfg, manager.Options{
		Metrics: metricsserver.Options{BindAddress: "0"},
		Scheme:  envTest.scheme,
		// Provide a real logger to controllers - this means that when tests fail we
		// get to see the controller logs that lead to the failure - if we decide this
		// is too noisy then we can gate this behind an env var like SPINKUBE_TEST_LOGS.
		Logger: logger,
	})

	require.NoError(t, err)

	ctrlr := &SpinAppExecutorReconciler{
		Client:   envTest.k8sClient,
		Scheme:   envTest.scheme,
		Recorder: mgr.GetEventRecorderFor("spinappexecutor-reconciler"),
	}

	require.NoError(t, ctrlr.SetupWithManager(mgr))

	return envTest, mgr, ctrlr
}
func TestSpinAppExecutorReconcile_StartupShutdown(t *testing.T) {
	t.Parallel()

	_, mgr, _ := setupExecutorController(t)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	require.NoError(t, mgr.Start(ctx))
}

func TestSpinAppExecutorReconcile_ContainerDShimSpinExecutorCreate(t *testing.T) {
	t.Parallel()

	envTest, mgr, _ := setupExecutorController(t)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		require.NoError(t, mgr.Start(ctx))
		wg.Done()
	}()

	executor := testContainerdShimSpinExecutor()
	list := &spinv1alpha1.SpinAppExecutorList{}
	require.NoError(t, envTest.k8sClient.List(ctx, list))
	require.True(t, len(list.Items) == 0)
	require.NoError(t, envTest.k8sClient.Create(ctx, executor))
	require.NoError(t, envTest.k8sClient.List(ctx, list))
	require.True(t, len(list.Items) == 1)
}

func TestSpinAppExecutorReconcile_ContainerDShimSpinExecutorDelete(t *testing.T) {
	t.Parallel()

	envTest, mgr, _ := setupExecutorController(t)
	executor := testContainerdShimSpinExecutor()

	envTest.k8sClient = fake.NewClientBuilder().WithScheme(envTest.scheme).WithObjects(
		executor,
	).Build()

	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		require.NoError(t, mgr.Start(ctx))
		wg.Done()
	}()

	list := &spinv1alpha1.SpinAppExecutorList{}
	require.NoError(t, envTest.k8sClient.List(ctx, list))
	require.True(t, len(list.Items) == 1)
	require.NoError(t, envTest.k8sClient.Delete(ctx, executor))
	require.NoError(t, envTest.k8sClient.List(ctx, list))
	require.True(t, len(list.Items) == 0)
}

func testContainerdShimSpinExecutor() *spinv1alpha1.SpinAppExecutor {
	return &spinv1alpha1.SpinAppExecutor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-executor",
			Namespace: "default",
		},
		Spec: spinv1alpha1.SpinAppExecutorSpec{
			CreateDeployment: true,
			DeploymentConfig: &spinv1alpha1.ExecutorDeploymentConfig{
				RuntimeClassName: generics.Ptr("test-runtime"),
			},
		},
	}
}
