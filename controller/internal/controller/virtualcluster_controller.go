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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	organizationv1 "github.com/axodevelopment/ocp-virtualcluster/controller/api/v1"
)

// VirtualClusterReconciler reconciles a VirtualCluster object
type VirtualClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=organization.prototypes.com,resources=virtualclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=organization.prototypes.com,resources=virtualclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=organization.prototypes.com,resources=virtualclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VirtualCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// LOGIC:
//
//	: Create
//	:	+ Add label to each node
//	: + Delete remove label from each node
func (r *VirtualClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconcile: ")
	logger.Info(req.String())

	//labelKey := "organization/virtualcluster.name"

	vc := &organizationv1.VirtualCluster{}

	if err := r.Get(ctx, req.NamespacedName, vc); err != nil {
		logger.Error(err, "Unable to find VirtualCluster ")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	labelKey, labelValue := GetAppliedSelectorLabelKeyValue(ctx, vc)

	if vc.Namespace != DefaultNamespace {
		logger.Info("VirtualCluster should be in: " + DefaultNamespace + " not located in: " + vc.Namespace + " ignoring")
		return ctrl.Result{}, nil
	}

	nodeList := &corev1.NodeList{}

	if err := r.List(ctx, nodeList); err != nil {
		logger.Error(err, "Somehow no nodes...")
		return ctrl.Result{}, err
	}

	var updateErrors []error

	for k := range nodeList.Items {
		n := &nodeList.Items[k]

		if n.ObjectMeta.Labels == nil {
			n.ObjectMeta.Labels = make(map[string]string)
		}

		n.ObjectMeta.Labels[labelKey] = labelValue

		logger.Info("Attempting to label node: " + n.Name)

		if err := r.Update(ctx, n); err != nil {
			logger.Error(err, "Unable to add label to node", "node", n.Name)
			updateErrors = append(updateErrors, err)
		} else {
			logger.Info("Added Label to Node: " + n.Name)
		}

		if vc.Spec.Nodes == nil {
			vc.Spec.Nodes = make([]organizationv1.NodeRef, 0)
		}

		nodeExists := false

		for _, ref := range vc.Spec.Nodes {
			if ref.Name == n.Name {
				nodeExists = true
				break
			}
		}

		if !nodeExists {
			vc.Spec.Nodes = append(vc.Spec.Nodes, organizationv1.NodeRef{Name: n.Name})

			if err := r.Update(ctx, vc); err != nil {
				logger.Info(vc.Name)
				logger.Error(err, "Unable to add node name to vc", "node", n.Name)
				updateErrors = append(updateErrors, err)
			} else {
				logger.Info("Added node to Node: " + n.Name)
			}
		}
	}

	if len(updateErrors) > 0 {
		return ctrl.Result{}, fmt.Errorf("failed to update some nodes: %v", updateErrors)
	}

	//UPDATE + CREATE success fall through to here FYI
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&organizationv1.VirtualCluster{}).
		Complete(r)
}
