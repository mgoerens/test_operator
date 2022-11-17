/*
Copyright 2021.

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

package controllers

import (
	"context"

	"github.com/imdario/mergo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	examplev1alpha1 "example.com/m/v2/api/v1alpha1"
)

// ExampleResourceReconciler reconciles a ExampleResource object
type ExampleResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=example.example.com,resources=exampleresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=example.example.com,resources=exampleresources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=example.example.com,resources=exampleresources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ExampleResource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *ExampleResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var er examplev1alpha1.ExampleResource

	// Load the er by name
	if err := r.Get(ctx, req.NamespacedName, &er); err != nil {
		if k8serrors.IsNotFound(err) {
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			log.Error(
				err,
				"Cannot find er - has it been deleted ?",
				"er Name", er.Name,
				"er Namespace", er.Namespace,
			)
			return ctrl.Result{}, nil
		}
		log.Error(
			err,
			"Error fetching er",
			"er Name", er.Name,
			"er Namespace", er.Namespace,
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	replicas := int32(1)
	ls := map[string]string{"app": "er", "er_cr": er.Name}

	desired := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      er.Name,
			Namespace: er.Namespace,
			// Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:    "busybox",
						Image:   "busybox",
						Command: []string{"sleep", "3600"},
					}},
				},
			},
		},
	}

	// Set er instance as the owner and controller
	if err := ctrl.SetControllerReference(&er, desired, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	current := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: er.Name, Namespace: er.Namespace}, current); err != nil {
		// Not exist - Create
		r.Create(ctx, desired)
	} else {
		patchDiff := client.MergeFrom(current)
		if err = mergo.Merge(current, desired, mergo.WithOverride); err != nil {
			return ctrl.Result{}, err
		}
		if err = r.Patch(ctx, desired, patchDiff); err != nil {
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExampleResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplev1alpha1.ExampleResource{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
