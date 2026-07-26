/*
Copyright 2026.

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
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1alpha1 "github.com/nathanSeixeiro/configmap-watcher-k8s-operator/api/v1alpha1"
)

var _ = Describe("ConfigMapWatcher Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName      = "test-resource"
			resourceNamespace = "default"
		)

		ctx := context.Background()
		var endpointServer *httptest.Server

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}
		configMapNamespacedName := types.NamespacedName{
			Name:      "test-configmap",
			Namespace: resourceNamespace,
		}
		configmapwatcher := &appsv1alpha1.ConfigMapWatcher{}

		BeforeEach(func() {
			endpointServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			By("creating the referenced ConfigMap")
			configMap := &corev1.ConfigMap{}
			err := k8sClient.Get(ctx, configMapNamespacedName, configMap)
			if err != nil && errors.IsNotFound(err) {
				configMap = &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapNamespacedName.Name,
						Namespace: configMapNamespacedName.Namespace,
					},
					Data: map[string]string{"key": "value"},
				}
				Expect(k8sClient.Create(ctx, configMap)).To(Succeed())
			}

			By("creating the custom resource for the Kind ConfigMapWatcher")
			err = k8sClient.Get(ctx, typeNamespacedName, configmapwatcher)
			if err != nil && errors.IsNotFound(err) {
				resource := &appsv1alpha1.ConfigMapWatcher{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: resourceNamespace,
					},
					Spec: appsv1alpha1.ConfigMapWatcherSpec{
						ConfigMapName:      configMapNamespacedName.Name,
						ConfigMapNamespace: configMapNamespacedName.Namespace,
						EventEndpoint:      endpointServer.URL,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			if endpointServer != nil {
				endpointServer.Close()
			}

			resource := &appsv1alpha1.ConfigMapWatcher{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				By("Cleanup the specific resource instance ConfigMapWatcher")
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			} else {
				Expect(errors.IsNotFound(err)).To(BeTrue())
			}

			configMap := &corev1.ConfigMap{}
			err = k8sClient.Get(ctx, configMapNamespacedName, configMap)
			if err == nil {
				By("Cleanup the referenced ConfigMap")
				Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
			} else {
				Expect(errors.IsNotFound(err)).To(BeTrue())
			}
		})

		It("should ignore reconcile when the ConfigMapWatcher does not exist", func() {
			By("deleting the resource before reconcile")
			resource := &appsv1alpha1.ConfigMapWatcher{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			By("reconciling a missing resource")
			controllerReconciler := &ConfigMapWatcherReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})

		It("should ignore reconcile when referenced ConfigMap does not exist", func() {
			By("deleting the referenced ConfigMap before reconcile")
			configMap := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, configMapNamespacedName, configMap)).To(Succeed())
			Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())

			By("reconciling with a missing ConfigMap")
			controllerReconciler := &ConfigMapWatcherReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})

		It("should return error when endpoint is unreachable", func() {
			By("updating the watcher endpoint to an unreachable URL")
			resource := &appsv1alpha1.ConfigMapWatcher{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
			resource.Spec.EventEndpoint = "http://127.0.0.1:1"
			Expect(k8sClient.Update(ctx, resource)).To(Succeed())

			By("reconciling and expecting an endpoint error")
			controllerReconciler := &ConfigMapWatcherReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).To(HaveOccurred())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ConfigMapWatcherReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the status to be updated as Success")
			Eventually(func(g Gomega) {
				resource := &appsv1alpha1.ConfigMapWatcher{}
				getErr := k8sClient.Get(ctx, typeNamespacedName, resource)
				g.Expect(getErr).NotTo(HaveOccurred())
				g.Expect(resource.Status.LastEventStatus).To(Equal("Success"))
				g.Expect(resource.Status.LastConfigMapVersion).NotTo(BeEmpty())
				g.Expect(resource.Status.ObservedGeneration).To(Equal(resource.Generation))
			}).Should(Succeed())
		})
	})
})
