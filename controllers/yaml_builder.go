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
	acme "github.com/jetstack/cert-manager/pkg/apis/acme/v1"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	meta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	webappv1 "kuvesz.sch/testoperator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *TestOperatorReconciler) createDeployment(webapp *webappv1.TestOperator) (*appsv1.Deployment, error) {
	/*
		apiVersion: apps/v1
		kind: Deployment
		metadata:
		  labels:
		    webapp: [name]
		  name: [name]
		  namespace: [namespace]
		spec:
		  replicas: [1-]
		  selector:
		    matchLabels:
		      webapp: [name]
		  template:
		    metadata:
		      labels:
		        webapp: [name]
		      name: [name]
		    spec:
		      containers:
		      - image: [image]
			    name: webapp-operator-test
		        ports:
				- name: http
		          containerPort: 80
		          protocol: TCP
				- name: https
		          containerPort: 443
		          protocol: TCP
	*/
	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      webapp.Name,
			Namespace: webapp.Namespace,
			Labels:    map[string]string{"webapp": webapp.Name},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &webapp.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"webapp": webapp.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   webapp.Name,
					Labels: map[string]string{"webapp": webapp.Name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "webapp-operator-test",
							Image: webapp.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 80,
									Protocol:      corev1.ProtocolTCP,
								}, {
									Name:          "https",
									ContainerPort: 443,
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
	}

	err := ctrl.SetControllerReference(webapp, &deployment, r.Scheme)
	if err != nil {
		return &deployment, err
	}

	return &deployment, nil
}

func (r *TestOperatorReconciler) createService(webapp *webappv1.TestOperator) (*corev1.Service, error) {
	/*
		apiVersion: v1
		kind: Service
		metadata:
		  name: [name]
		  namespace: [namespace]
		spec:
		  selector:
		    webapp: [name]
		  ports:
		  - name: https
		    port: 443
			protocol: TCP
		    targetPort: 443
		  - name: http
		    port: 80
			protocol: TCP
		    targetPort: 80
	*/
	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      webapp.Name,
			Namespace: webapp.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"webapp": webapp.Name},
			Ports: []corev1.ServicePort{
				{
					Name:       "https",
					Port:       443,
					Protocol:   "TCP",
					TargetPort: intstr.FromString("https"),
				}, {
					Name:       "http",
					Port:       80,
					Protocol:   "TCP",
					TargetPort: intstr.FromString("http"),
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(webapp, &service, r.Scheme); err != nil {
		return &service, err
	}

	return &service, nil
}

func (r *TestOperatorReconciler) createIngress(webapp *webappv1.TestOperator) (*networkv1.Ingress, error) {
	/*
		apiVersion: extensions/v1
		kind: Ingress
		metadata:
		  name: [name]-ingress
		  annotations:
		    cert-manager.io/cluster-issuer: letsencrypt-prod
		    kubernetes.io/ingress.class: nginx
		spec:
		  TLS:
		  - hosts:
		    - [host]
			secretName: letsencrypt-prod
		  rules:
		  - host: [host]
		    http:
		      paths:
		        - path: /
		          backend:
		            serviceName: [name]
		            servicePort: 80
	*/
	ingress := networkv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: networkv1.SchemeGroupVersion.String(),
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      webapp.Name + "-ingress",
			Namespace: webapp.Namespace,
			Annotations: map[string]string{
				"cert-manager.io/cluster-issuer": "letsencrypt-prod",
				"kubernetes.io/ingress.class":    "nginx",
			},
		},
		Spec: networkv1.IngressSpec{
			TLS: []networkv1.IngressTLS{
				{
					Hosts:      []string{webapp.Spec.Host},
					SecretName: "letsencrypt-prod",
				},
			},
			Rules: []networkv1.IngressRule{
				{
					Host: webapp.Spec.Host,
					IngressRuleValue: networkv1.IngressRuleValue{
						HTTP: &networkv1.HTTPIngressRuleValue{
							Paths: []networkv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: func(pt networkv1.PathType) *networkv1.PathType { return &pt }(networkv1.PathTypePrefix),
									Backend: networkv1.IngressBackend{
										Service: &networkv1.IngressServiceBackend{
											Name: webapp.Name,
											Port: networkv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// always set the controller reference so that we know which object owns this.
	if err := ctrl.SetControllerReference(webapp, &ingress, r.Scheme); err != nil {
		return &ingress, err
	}

	return &ingress, nil
}

func (r *TestOperatorReconciler) createIssuer(webapp *webappv1.TestOperator) (*certmanager.ClusterIssuer, error) {
	/*
		apiVersion: cert-manager.io/v1alpha2
		kind: ClusterIssuer
		metadata:
		  name: letsencrypt-prod
		spec:
		  acme:
		    email: kuveszkuvesz@gmail.com
		    server: https://acme-v02.api.letsencrypt.org/directory
		    privateKeySecretRef:
		      name: letsencrypt-prod
		    solvers:
		    - http01:
		        ingress:
		          class: nginx
	*/
	ingressClass := "nginx"
	issuer := certmanager.ClusterIssuer{
		TypeMeta: metav1.TypeMeta{
			APIVersion: certmanager.SchemeGroupVersion.String(),
			Kind:       "ClusterIssuer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "letsencrypt-prod",
			Namespace: webapp.Namespace,
		},
		Spec: certmanager.IssuerSpec{
			IssuerConfig: certmanager.IssuerConfig{
				ACME: &acme.ACMEIssuer{
					Server: "https://acme-v02.api.letsencrypt.org/directory",
					Email:  "kuveszkuvesz@gmail.com",
					PrivateKey: meta.SecretKeySelector{
						LocalObjectReference: meta.LocalObjectReference{
							Name: "letsencrypt-prod",
						},
					},
					Solvers: []acme.ACMEChallengeSolver{
						{
							HTTP01: &acme.ACMEChallengeSolverHTTP01{
								Ingress: &acme.ACMEChallengeSolverHTTP01Ingress{
									Class: &ingressClass,
								},
							},
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(webapp, &issuer, r.Scheme); err != nil {
		return &issuer, err
	}

	return &issuer, nil
}

func (r *TestOperatorReconciler) createCert(webapp *webappv1.TestOperator) (*certmanager.Certificate, error) {
	/*
		apiVersion: cert-manager.io/v1
		kind: Certificate
		metadata:
		  name: [name]
		  namespace: [namespace]
		spec:
		  secretName: [host]-tls
		  dnsNames:
		    - [host]
		  issuerRef:
		    kind: ClusterIssuer
		    name: letsencrypt-prod
	*/
	certificate := certmanager.Certificate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: certmanager.SchemeGroupVersion.String(),
			Kind:       "Certificate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      webapp.Spec.Host,
			Namespace: webapp.Namespace,
		},
		Spec: certmanager.CertificateSpec{
			SecretName: webapp.Spec.Host + "-tls",
			DNSNames:   []string{webapp.Spec.Host},
			IssuerRef: meta.ObjectReference{
				Kind: "ClusterIssuer",
				Name: "letsencrypt-prod",
			},
		},
	}

	if err := ctrl.SetControllerReference(webapp, &certificate, r.Scheme); err != nil {
		return &certificate, err
	}

	return &certificate, nil
}
