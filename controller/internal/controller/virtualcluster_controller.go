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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
//	: : virtualcluster already has vm.name in it				ignore never err here to block vm deployment
//	: : virtualcluster doesn't have a vm.name in it			append never err here to block vm deployment
//	: we dont' find matching cluster
func (r *VirtualMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconcile: ")
	logger.Info(req.String())

	keyNameString := "organization/virtualcluster.name"
	keyNamespaceString := "organization/virtualcluster.namespace"

	//TODO: need to fix how to handle where VirtualClusters live
	defaultNamespace := "operator-virtualcluster"

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
		//TODO: going to be hard to handle delete because I don't know how to get the labels of the deleted resource, i may need to create a lookup map
		//  like a vcmap which i can use to get vm -> vc, granted this woudl be easier if i just used a db.,,

		if errors.IsNotFound(err) {
			return r.handleVMDeletion(ctx, req.NamespacedName)
		}

		logger.Error(err, "Unable to r.Get VirtualMachine")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//Just want to collect ready status of kubevirt.  It is likely we will need to wait on some changs based
	//  upon the status of where KubeVirt is at in its on state management.
	kubeVirtReady := false

	for _, c := range vm.Status.Conditions {
		if c.Type == "KubeVirtRead" && c.Status == "True" {
			kubeVirtReady = true
			break
		}
	}

	fmt.Println("KubeVirtReady: ", kubeVirtReady)

	//Keeping this commented until i need this, I am sure i do but feature inc.
	//  I may even need this to supprot the vm node selectors we will see how that internally resolves.
	//  in theory a new event should be requeued once we apply the change but i'll test this later
	/*
		if !kubeVirtReady {
			logger.Info("KubeVirt controller has not completed its work yet, requeuing...")
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}
	*/

	for k, v := range vm.Labels {
		logger.Info(fmt.Sprintf("Label: [%s][%s]", k, v))
	}

	keyNameValue, found := vm.Labels[keyNameString]
	keyNamespaceValue, nsfound := vm.Labels[keyNamespaceString]

	vc := &organizationv1.VirtualCluster{}

	//TODO: for now we won't do anything if there is no label
	//maybe in the future there will be a 'catch-all' vc?
	if !found {
		return ctrl.Result{}, nil
	}

	if !nsfound {
		keyNamespaceValue = defaultNamespace
	}

	if err := r.Get(ctx, types.NamespacedName{Name: keyNameValue, Namespace: keyNamespaceValue}, vc); err != nil {
		if errors.IsNotFound(err) {
			//TODO: not found so we need to create... for now
			vc = &organizationv1.VirtualCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keyNameValue,
					Namespace: keyNamespaceValue,
				},
				Spec: organizationv1.VirtualClusterSpec{
					VirtualMachines: []organizationv1.VirtualMachineRef{
						{
							Name:      vm.Name,
							Namespace: vm.Namespace,
						},
					},
				},
			}

			if err := r.Create(ctx, vc); err != nil {
				logger.Error(err, "Failed to create new VirtualCluster")
				return ctrl.Result{}, err //for now nil
			}

		} else {
			//hmm
			logger.Error(err, "Unable to get VirtualCluster")
			return ctrl.Result{}, err
		}
	} else {
		//TODO: found vcluster now need to see if vm is 'attached' or not, if not append
		b := false

		for _, kvm := range vc.Spec.VirtualMachines {
			if kvm.Name == vm.Name && kvm.Namespace == vm.Namespace {
				b = true
				break
			}
		}

		//add the vm to the vc
		if !b {
			vc.Spec.VirtualMachines = append(vc.Spec.VirtualMachines, organizationv1.VirtualMachineRef{
				Name:      vm.Name,
				Namespace: vm.Namespace,
			})

			if err := r.Update(ctx, vc); err != nil {
				logger.Error(err, "Failed to update the VirtualCluster")
				//TODO: for now don't error return nil otherwise we could block the vm deployment
				//return ctrl.Result{}, err
				return ctrl.Result{}, nil
			}

			fmt.Println("AC:")
			fmt.Println(vc)
			if vc.Spec.NodeSelector.Labels != nil {

				for i := 0; i < 3; i++ {

					if vm.Spec.Template.Spec.NodeSelector == nil {
						vm.Spec.Template.Spec.NodeSelector = make(map[string]string)
					}

					for k, v := range vc.Spec.NodeSelector.Labels {
						vm.Spec.Template.Spec.NodeSelector[k] = v
					}

					if err := r.Update(ctx, vm); err != nil {
						if errors.IsConflict(err) {
							logger.Error(err, "Failed to update NodeSelector for vm")
							time.Sleep(100 * time.Millisecond)
							continue
						}

						logger.Error(err, "Failed to update NodeSelector for vm")
						return ctrl.Result{RequeueAfter: time.Second * 10}, err
					}

					break
				}
			}
		}
	}

	//UPDATE + CREATE success fall through to here FYI
	return ctrl.Result{}, nil
}

/*
quick fix to remove vm's that have been deleted from the vc though its not full proof.  Need to test this a bit more
*/
func (r *VirtualMachineReconciler) handleVMDeletion(ctx context.Context, name types.NamespacedName) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	vcList := &organizationv1.VirtualClusterList{}

	if err := r.List(ctx, vcList, client.InNamespace(name.Namespace)); err != nil {
		logger.Error(err, "Unable to list VirtualClusters")
		return ctrl.Result{}, err
	}

	//I probably need to think of adding a partitioned map or something,
	//  ...if we have 100000 vms, while not individually slow can maybe present some scaling issue
	for _, vc := range vcList.Items {
		updated := false

		for i, existingVM := range vc.Spec.VirtualMachines {
			if existingVM.Name == name.Name && existingVM.Namespace == name.Namespace {
				//append to slice up until i, everything after i
				vc.Spec.VirtualMachines = append(vc.Spec.VirtualMachines[:i], vc.Spec.VirtualMachines[i+1:]...)
				updated = true
				break
			}
		}

		if updated {
			if err := r.Update(ctx, &vc); err != nil {
				logger.Error(err, "Failed to update the VirtualCluster after VM deletion")
				return ctrl.Result{}, err
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
