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

	"github.com/go-logr/logr"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	webappv1 "kuvesz.sch/testoperator/api/v1"
)

// TestOperatorReconciler reconciles a TestOperator object
type TestOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=webapp.kuvesz.sch,resources=testoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webapp.kuvesz.sch,resources=testoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=webapp.kuvesz.sch,resources=testoperators/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;get;patch
//+kubebuilder:rbac:groups=core,resources=services,verbs=list;watch;get;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TestOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx).WithValues("test operator", req.NamespacedName)
	webapp := &webappv1.TestOperator{}

	err := r.Client.Get(ctx, req.NamespacedName, webapp)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Resuource not found, exiting reconcile")
			return ctrl.Result{}, nil
		}

		log.Error(err, "Unable to get resource!")
		return ctrl.Result{}, err
	}

	r.Client.Status().Update(ctx, webapp)

	err = r.reconcileWebapp(ctx, webapp, log)
	if err != nil {
		log.Error(err, "Failed to reconcile Webapp")
		return ctrl.Result{}, err
	}

	err = r.reconcileWebappIngress(ctx, webapp, log)
	if err != nil {
		log.Error(err, "Failed to reconcile Ingress")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *TestOperatorReconciler) reconcileWebapp(ctx context.Context, webapp *webappv1.TestOperator, log logr.Logger) error {
	deployment, err := r.createDeployment(webapp)
	if err != nil {
		return err
	}

	foundDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Deployment")
		err = r.Create(ctx, deployment)

		if err != nil {
			return err
		}

		return nil

	} else if err != nil {
		return err
	}

	log.Info("Skipping reconcile for Deployment as it already exists")
	//I believe replicas are only need to be updated in case we already had the deployment
	if webapp.Spec.Replicas != *foundDeployment.Spec.Replicas {
		log.Info("Reconciling number of replicas")
		foundDeployment.Spec.Replicas = &webapp.Spec.Replicas

		err = r.Update(ctx, foundDeployment)
		if err != nil {
			return err
		}

		return nil
	}

	var service *corev1.Service
	service, err = r.createService(webapp)
	if err != nil {
		return err
	}

	foundService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Service")
		err = r.Create(ctx, service)

		if err != nil {
			return err
		}

		return nil

	} else if err != nil {
		return err
	}

	log.Info("Skipping reconcile for Service as it already exists")

	return nil
}

func (r *TestOperatorReconciler) reconcileWebappIngress(ctx context.Context, webapp *webappv1.TestOperator, log logr.Logger) error {
	err := r.reconcileIngressCert(ctx, webapp, log)
	if err != nil {
		log.Error(err, "Failed to reconcile")
		return err
	}

	var ingress *networkv1.Ingress
	ingress, err = r.createIngress(webapp)
	if err != nil {
		return err
	}

	foundIngress := &networkv1.Ingress{}
	err = r.Get(ctx, types.NamespacedName{Name: ingress.Name, Namespace: ingress.Namespace}, foundIngress)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Ingress")
		err = r.Create(ctx, ingress)

		if err != nil {
			return err
		}

		return nil

	} else if err != nil {
		return err
	}

	log.Info("Skipping reconcile for Ingress as it already exists")

	return nil
}

func (r *TestOperatorReconciler) reconcileIngressCert(ctx context.Context, webapp *webappv1.TestOperator, log logr.Logger) error {
	var issuer *certmanager.ClusterIssuer
	var err error
	issuer, err = r.createIssuer(webapp)
	if err != nil {
		return err
	}

	foundIssuer := &certmanager.ClusterIssuer{}
	err = r.Get(ctx, types.NamespacedName{Name: issuer.Name, Namespace: issuer.Namespace}, foundIssuer)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Issuer")
		err = r.Create(ctx, issuer)

		if err != nil {
			return err
		}

		return nil

	} else if err != nil {
		return err
	}

	log.Info("Skipping reconcile for Issuer as it already exists")

	var cert *certmanager.Certificate
	cert, err = r.createCert(webapp)
	if err != nil {
		return err
	}

	foundCert := &certmanager.Certificate{}
	err = r.Get(ctx, types.NamespacedName{Name: cert.Name, Namespace: cert.Namespace}, foundCert)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Certificate")
		err = r.Create(ctx, cert)

		if err != nil {
			return err
		}

		return nil

	} else if err != nil {
		return err
	}

	log.Info("Skipping reconcile for Certificate as it already exists")

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TestOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.TestOperator{}).
		Complete(r)
}
