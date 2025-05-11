package controller

import (
	"context"
	rbacv1 "k8s.io/api/rbac/v1"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *BGPRouteReconciler) reconcileAgentConfigMap(ctx context.Context, desiredCMap *corev1.ConfigMap) error {
	logger := log.FromContext(ctx)

	var existingCM corev1.ConfigMap
	err := r.Get(ctx, client.ObjectKeyFromObject(desiredCMap), &existingCM)
	if apierrors.IsNotFound(err) {
		if errCreate := r.Create(ctx, desiredCMap); errCreate != nil {
			logger.Error(errCreate, "Failed to create ConfigMap", "ConfigMap.Name", desiredCMap.Name)
			return errCreate
		}
		logger.Info("Created ConfigMap", "ConfigMap.Name", desiredCMap.Name)
		return nil
	} else if err != nil {
		logger.Error(err, "Failed to get ConfigMap", "ConfigMap.Name", desiredCMap.Name)
		return err
	}

	if !reflect.DeepEqual(existingCM.Data, desiredCMap.Data) {
		existingCM.Data = desiredCMap.Data
		if errUpdate := r.Update(ctx, &existingCM); errUpdate != nil {
			logger.Error(errUpdate, "Failed to update ConfigMap", "ConfigMap.Name", desiredCMap.Name)
			return errUpdate
		}
		logger.Info("Updated ConfigMap", "ConfigMap.Name", desiredCMap.Name)
	}

	return nil
}

func (r *BGPRouteReconciler) reconcileAgentServiceAccount(ctx context.Context, desiredSAccount *corev1.ServiceAccount) error {
	logger := log.FromContext(ctx)

	var currentSAccount corev1.ServiceAccount
	err := r.Get(ctx, client.ObjectKey{Name: desiredSAccount.Name, Namespace: desiredSAccount.Namespace}, &currentSAccount)
	if apierrors.IsNotFound(err) {
		if errCreate := r.Create(ctx, desiredSAccount); errCreate != nil {
			logger.Error(errCreate, "Failed to create ServiceAccount", "ServiceAccount.Name", desiredSAccount.Name)
			return errCreate
		}
		logger.Info("Created ServiceAccount", "ServiceAccount.Name", desiredSAccount.Name)
	} else if err != nil {
		logger.Error(err, "Failed to get ServiceAccount", "ServiceAccount.Name", desiredSAccount.Name)
		return err
	}

	// TODO: Optionally reconcile/update ServiceAccount fields here if needed.

	return nil
}

func (r *BGPRouteReconciler) reconcileAgentClusterRoles(ctx context.Context, desiredCRole *rbacv1.ClusterRole, desiredCRBinding *rbacv1.ClusterRoleBinding) error {
	logger := log.FromContext(ctx)

	var currentCRole rbacv1.ClusterRole
	err := r.Get(ctx, client.ObjectKey{Name: desiredCRole.Name}, &currentCRole)
	if apierrors.IsNotFound(err) {
		if errCreate := r.Create(ctx, desiredCRole); errCreate != nil {
			logger.Error(errCreate, "Failed to create ClusterRole", "ClusterRole.Name", desiredCRole.Name)
			return errCreate
		}
		logger.Info("Created ClusterRole", "ClusterRole.Name", desiredCRole.Name)
	} else if err != nil {
		logger.Error(err, "Failed to get ClusterRole", "ClusterRole.Name", desiredCRole.Name)
		return err
	}

	var currentCRBinding rbacv1.ClusterRoleBinding
	err = r.Get(ctx, client.ObjectKey{Name: desiredCRBinding.Name}, &currentCRBinding)
	if apierrors.IsNotFound(err) {
		if errCreate := r.Create(ctx, desiredCRBinding); errCreate != nil {
			logger.Error(errCreate, "Failed to create ClusterRoleBinding", "ClusterRole.Name", desiredCRBinding.Name)
			return errCreate
		}
		logger.Info("Created ClusterRoleBinding", "ClusterRole.Name", desiredCRBinding.Name)
	} else if err != nil {
		logger.Error(err, "Failed to get ClusterRoleBinding", "ClusterRole.Name", desiredCRBinding.Name)
		return err
	}

	return nil
}

func (r *BGPRouteReconciler) reconcileAgentDaemonSet(ctx context.Context, desiredDSet *appsv1.DaemonSet) error {
	logger := log.FromContext(ctx)

	var currentDSet appsv1.DaemonSet
	err := r.Get(ctx, client.ObjectKey{Name: desiredDSet.Name, Namespace: desiredDSet.Namespace}, &currentDSet)
	if apierrors.IsNotFound(err) {
		if errCreate := r.Create(ctx, desiredDSet); errCreate != nil {
			logger.Error(errCreate, "Failed to create DaemonSet", "DaemonSet.Name", desiredDSet.Name)
			return errCreate
		}
		logger.Info("Created DaemonSet", "DaemonSet.Name", desiredDSet.Name)
	} else if err != nil {
		logger.Error(err, "Failed to get DaemonSet", "DaemonSet.Name", desiredDSet.Name)
		return err
	}

	// TODO: Add update logic here to compare and update DaemonSet if specs differ.

	return nil
}
