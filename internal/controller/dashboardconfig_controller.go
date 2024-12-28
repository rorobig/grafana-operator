/*
Copyright 2024.

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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	grafanav1alpha1 "github.com/rorobig/grafana-operator/api/v1alpha1"
)

// DashboardConfigReconciler reconciles a DashboardConfig object
type DashboardConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=grafana.cyberwizzard.com,resources=dashboardconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=grafana.cyberwizzard.com,resources=dashboardconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=grafana.cyberwizzard.com,resources=dashboardconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DashboardConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *DashboardConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the DashboardConfig instance
	var dashboardConfig grafanav1alpha1.DashboardConfig
	if err := r.Get(ctx, req.NamespacedName, &dashboardConfig); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to fetch DashboardConfig")
		}
		// Resource not found, could have been deleted
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Define the desired ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dashboardConfig.Name + "-dashboard",
			Namespace: req.Namespace,
			Labels: map[string]string{
				"grafana_dashboard": "true",
			},
		},
		Data: map[string]string{
			"dashboard.json": dashboardConfig.Spec.Json,
		},
	}

	// Set the controller reference for garbage collection
	if err := ctrl.SetControllerReference(&dashboardConfig, configMap, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference")
		return ctrl.Result{}, err
	}

	// Check if the ConfigMap already exists
	var existingConfigMap corev1.ConfigMap
	err := r.Get(ctx, client.ObjectKey{Name: configMap.Name, Namespace: configMap.Namespace}, &existingConfigMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// ConfigMap doesn't exist, create it
			if err := r.Create(ctx, configMap); err != nil {
				log.Error(err, "Failed to create ConfigMap")
				return ctrl.Result{}, err
			}
			log.Info("ConfigMap created successfully", "ConfigMap", configMap.Name)
		} else {
			log.Error(err, "Failed to fetch ConfigMap")
			return ctrl.Result{}, err
		}
	} else {
		// ConfigMap exists, update it if necessary
		existingConfigMap.Data = configMap.Data
		if err := r.Update(ctx, &existingConfigMap); err != nil {
			log.Error(err, "Failed to update ConfigMap")
			return ctrl.Result{}, err
		}
		log.Info("ConfigMap updated successfully", "ConfigMap", existingConfigMap.Name)
	}

	// Update the status of the DashboardConfig resource
	dashboardConfig.Status.Applied = true
	if err := r.Status().Update(ctx, &dashboardConfig); err != nil {
		log.Error(err, "Failed to update DashboardConfig status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DashboardConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&grafanav1alpha1.DashboardConfig{}).
		Complete(r)
}
