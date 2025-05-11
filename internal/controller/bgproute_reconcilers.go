package controller

import (
	"context"
	"fmt"
	rbacv1 "k8s.io/api/rbac/v1"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *BGPRouteReconciler) reconcileAgentConfigMap(ctx context.Context, desiredCMap *corev1.ConfigMap) error {
	if err := r.genericReconciliationWithDiff(ctx, desiredCMap, func(existing, desired client.Object) bool {
		existingCM := existing.(*corev1.ConfigMap)
		desiredCM := desired.(*corev1.ConfigMap)

		// Compare the Data field to determine if an update is needed
		if !reflect.DeepEqual(existingCM.Data, desiredCM.Data) {
			existingCM.Data = desiredCM.Data
			return true // Signal that an update should happen
		}
		return false // No update needed
	}); err != nil {
		return err
	}

	return nil
}

func (r *BGPRouteReconciler) reconcileAgentServiceAccount(ctx context.Context, desiredSAccount *corev1.ServiceAccount) error {
	if _, err := r.genericReconciliation(ctx, desiredSAccount); err != nil {
		return err
	}

	return nil
}

func (r *BGPRouteReconciler) reconcileAgentClusterRoles(ctx context.Context, desiredCRole *rbacv1.ClusterRole, desiredCRBinding *rbacv1.ClusterRoleBinding) error {
	if _, err := r.genericReconciliation(ctx, desiredCRole); err != nil {
		return err
	}

	if _, err := r.genericReconciliation(ctx, desiredCRBinding); err != nil {
		return err
	}

	return nil
}

func (r *BGPRouteReconciler) reconcileAgentDaemonSet(ctx context.Context, desiredDSet *appsv1.DaemonSet) error {
	// todo() use genericReconciliationWithDiff so that DaemonSet is updated if spec changes
	if _, err := r.genericReconciliation(ctx, desiredDSet); err != nil {
		return err
	}
	return nil
}

// genericReconciliation is a generic reconciliation function that handles that ensures the desired resource state.
func (r *BGPRouteReconciler) genericReconciliation(
	ctx context.Context,
	desired client.Object,
) (client.Object, error) {
	logger := log.FromContext(ctx)

	existing := desired.DeepCopyObject().(client.Object)
	err := r.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	// Create the object if it doesn't exist
	if apierrors.IsNotFound(err) {
		if errCreate := r.Create(ctx, desired); errCreate != nil {
			logger.Error(errCreate, "Failed to create", "Kind", desired.GetObjectKind().GroupVersionKind().Kind, "Name", desired.GetName())
			return nil, errCreate
		}
		logger.Info("Created", "Kind", desired.GetObjectKind().GroupVersionKind().Kind, "Name", desired.GetName())

		// Fetch the freshly created object to ensure correct state
		if errFetch := r.Get(ctx, client.ObjectKeyFromObject(desired), existing); errFetch != nil {
			logger.Error(errFetch, "Failed to fetch after creation", "Kind", desired.GetObjectKind().GroupVersionKind().Kind, "Name", desired.GetName())
			return nil, err
		}

		return existing, nil
	} else if err != nil {
		logger.Error(err, "Failed to get", "Kind", desired.GetObjectKind().GroupVersionKind().Kind, "Name", desired.GetName())
		return nil, err
	}

	// If the object already exists, just return it without any changes

	return existing, nil
}

// genericReconciliationWithDiff is a generic reconciliation function that ensures the desired resource state.
// It updates the resource if a difference is detected by the provided diffCheck function.
func (r *BGPRouteReconciler) genericReconciliationWithDiff(
	ctx context.Context,
	desired client.Object,
	diffCheck func(existing, desired client.Object) bool,
) error {
	// Run the generic reconciliation to get the existing resource
	existing, err := r.genericReconciliation(ctx, desired)
	if err != nil || existing == nil {
		return err
	}

	if diffCheck == nil {
		return fmt.Errorf("diffCheck function cannot be nil")
	}

	// Check if the existing resource differs from the desired state. In case of being true, update the resource
	if diffCheck(existing, desired) {
		logger := log.FromContext(ctx)
		if errUpdate := r.Update(ctx, existing); errUpdate != nil {
			logger.Error(errUpdate, "Failed to update", "Kind", desired.GetObjectKind().GroupVersionKind().Kind, "Name", desired.GetName())
			return errUpdate
		}
		logger.Info("Updated", "Kind", desired.GetObjectKind().GroupVersionKind().Kind, "Name", desired.GetName())
	}

	return nil
}
