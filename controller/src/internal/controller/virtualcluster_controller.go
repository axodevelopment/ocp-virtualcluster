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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	//organizationv1 "github.com/axodevelopment/ocp-virtualcluster/controller/api/v1"
	kubevirtv1 "kubevirt.io/api/core/v1"
)

// VirtualClusterReconciler reconciles a VirtualCluster object
type VirtualMachineReconciler struct {
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
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *VirtualMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconcile: ")
	logger.Info(req.String())

	vm := &kubevirtv1.VirtualMachine{}

	if err := r.Get(ctx, req.NamespacedName, vm); err != nil {
		//TODO: this may not be true we should check if this is linked on the vc
		/*
			We need to handle things like
			delete
			scale up
			scale down
			live migration
			etc
		*/
		logger.Error(err, "Unable to r.Get VirtualMachine")
		return ctrl.Result{}, err
	}

	for k, v := range vm.Labels {
		logger.Info(fmt.Sprintf("Label: [%s][%s]", k, v))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubevirtv1.VirtualMachine{}).
		Complete(r)
}
