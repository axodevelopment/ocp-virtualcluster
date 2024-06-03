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

	organizationv1 "github.com/axodevelopment/ocp-virtualcluster/controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// VirtualClusterReconciler reconciles a VirtualCluster object
type NodeReconciler struct {
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
//	:	+ When a Node is added we want to iterate over the vc's to add their labels to the node
func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconcile: ")
	logger.Info(req.String())

	//keyNameString := "organization/virtualcluster.name"
	//defaultNamespace := "operator-virtualcluster"

	node := &corev1.Node{}

	if err := r.Get(ctx, req.NamespacedName, node); err != nil {
		logger.Error(err, "Unable to find Node")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	vcl := &organizationv1.VirtualClusterList{}

	if err := r.List(ctx, vcl); err != nil {
		logger.Error(err, "Unable to list VirtualClusters")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if len(vcl.Items) > 0 {
		logger.Info("No vcl's discovered so no labels to apply")
		return ctrl.Result{}, nil
	}

	var updateErrors []error

	for k := range vcl.Items {
		vc := &vcl.Items[k]

		if node.ObjectMeta.Labels == nil {
			node.ObjectMeta.Labels = make(map[string]string)
		}

		labelKey, labelValue := GetAppliedSelectorLabelKeyValue(ctx, vc)

		if node.ObjectMeta.Labels[labelKey] == labelValue {
			logger.Info("Label already exists we can skip updating")
			continue
		}

		node.ObjectMeta.Labels[labelKey] = labelValue

		if err := r.Update(ctx, node); err != nil {
			logger.Error(err, "Unable to add label to node", "node", node.Name)
			updateErrors = append(updateErrors, err)
		} else {
			logger.Info("Added Label to Node: " + node.Name)
		}
	}

	if len(updateErrors) > 0 {
		return ctrl.Result{}, fmt.Errorf("failed to update some nodes: %v", updateErrors)
	}

	//UPDATE + CREATE success fall through to here FYI
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Complete(r)
}
