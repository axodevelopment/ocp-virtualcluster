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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	organizationv1 "github.com/axodevelopment/ocp-virtualcluster/controller/api/v1"
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
// LOGIC:
// We get a valid VM name / namespace 									!not we return nil //ignore
// We find keyNameStringlabel
//
//	: we find matching cluster
//	: : virtualcluster already has vm.name in it				update
//	: : virtualcluster doesn't have a vm.name in it
//	: we dont' find matching cluster
func (r *VirtualMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconcile: ")
	logger.Info(req.String())

	keyNameString := "organization/virtualcluster/name"
	keyNamespaceString := "organization/virtualcluster/namespace"

	//TODO: need to fix how to handle where VirtualClusters live
	fixLaterNamespace := "operator-virtualcluster"

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

	keyNameValue, found := vm.Labels[keyNameString]
	keyNamespaceValue, nsfound := vm.Labels[keyNamespaceString]

	vc := &organizationv1.VirtualCluster{}

	if found {
		if !nsfound {
			keyNamespaceValue = fixLaterNamespace
		}

		if err := r.Get(ctx, types.NamespacedName{Name: keyNameValue, Namespace: keyNamespaceValue}, vc); err != nil {
			if errors.IsNotFound(err) {
				//TODO: not found so we need to create
			}
		} else {
			//TODO: found cluster now need to see if vm is 'attached' or not, if not append
			b := false

			for _, kvm := range vc.Spec.VirtualMachines {
				if kvm == vm.Name {
					b = true
					break
				}
			}

			if !b {
				vc.Spec.VirtualMachines = append(vc.Spec.VirtualMachines, vm.Name)

				if err := r.Update(ctx, vc); err != nil {
					logger.Error(err, "Failed to update the VirtualCluster")
					//TODO: for now don't error return nil otherwise we could block the vm deployment
					//return ctrl.Result{}, err
					return ctrl.Result{}, nil
				}
			}

		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubevirtv1.VirtualMachine{}).
		Complete(r)
}
