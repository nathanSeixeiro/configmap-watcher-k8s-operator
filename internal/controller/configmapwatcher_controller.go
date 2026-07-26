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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1alpha1 "github.com/nathanSeixeiro/configmap-watcher-k8s-operator/api/v1alpha1"
)

// ConfigMapWatcherReconciler reconciles a ConfigMapWatcher object
type ConfigMapWatcherReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=apps.devops,resources=configmapwatchers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.devops,resources=configmapwatchers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.devops,resources=configmapwatchers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ConfigMapWatcher object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.24.1/pkg/reconcile
func (r *ConfigMapWatcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	var watcher appsv1alpha1.ConfigMapWatcher

	err := r.Get(ctx, req.NamespacedName, &watcher)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "ConfigMapWatcher resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get ConfigMapWatcher ")
		return ctrl.Result{}, err
	}

	configMap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: watcher.Spec.ConfigMapNamespace,
		Name:      watcher.Spec.ConfigMapName,
	}, configMap)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "ConfigMap resource not found. Ignoring since object must be deleted")
			return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
		}
		logger.Error(err, "Failed to get ConfigMap ")
		return ctrl.Result{}, err
	}

	version := configMap.ResourceVersion
	if version == watcher.Status.LastConfigMapVersion {
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	eventData := map[string]any{
		"configMapName":      watcher.Spec.ConfigMapName,
		"configMapNamespace": watcher.Spec.ConfigMapNamespace,
		"version":            watcher.Status.LastConfigMapVersion,
		"data":               configMap.Data,
		"timestamp":          time.Now().Format(time.RFC3339),
		"binaryData":         configMap.BinaryData,
	}

	err = SendEventToEndpoint(eventData, watcher.Spec.EventEndpoint)

	statusErr := r.StatusHandler(ctx, &watcher, version, err)
	if statusErr != nil {
		logger.Error(statusErr, "Failed to update status")
		r.Recorder.Eventf(&watcher, corev1.EventTypeWarning, "SendFailed",
			"failed to update status for %s: %v", watcher.Spec.EventEndpoint, err)
		return ctrl.Result{}, statusErr
	}

	if err != nil {
		logger.Error(err, "Failed to send event to endpoint")
		r.Recorder.Eventf(&watcher, corev1.EventTypeWarning, "SendFailed",
			"failed to send event to %s: %v", watcher.Spec.EventEndpoint, err)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	r.Recorder.Eventf(&watcher, corev1.EventTypeNormal, "SendSucceeded",
		"event sent successfully to %s", watcher.Spec.EventEndpoint)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigMapWatcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.ConfigMapWatcher{}).
		// Watch for ConfigMap events and enqueue requests for ConfigMapWatcher objects that are watching the ConfigMap
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(r.findObjectsConfigMap)).
		Named("configmapwatcher").
		Complete(r)
}

func (r *ConfigMapWatcherReconciler) findObjectsConfigMap(ctx context.Context, obj client.Object) []reconcile.Request {
	configMap := obj.(*corev1.ConfigMap)
	requests := []reconcile.Request{}

	// List all ConfigMapWatcher objects
	var watchers appsv1alpha1.ConfigMapWatcherList
	err := r.List(ctx, &watchers)
	if err != nil {
		// return request because request = nil
		return requests
	}

	// Iterate through the ConfigMapWatcher objects and check if they are watching the ConfigMap
	for _, watcher := range watchers.Items {
		if configMap.Name == watcher.Name && configMap.Namespace == watcher.Namespace { // check if name and namespace
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Name:      watcher.Name,
					Namespace: watcher.Namespace,
				},
			})
		}
	}
	return requests
}

func SendEventToEndpoint(eventData map[string]any, endpoint string) error {
	logger := logf.Log.WithName("SendEventToEndpoint")
	var httpClient = &http.Client{}
	jsonData, err := json.Marshal(eventData)
	if err != nil {
		return err
	}

	res, err := httpClient.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error(err, "Failed to send event to endpoint")
		return err
	}
	defer func() {
		if cerr := res.Body.Close(); cerr != nil {
			logger.Error(cerr, "failed to close response body")
		}
	}()

	if res.StatusCode != http.StatusOK {
		logger.Error(nil, "Failed to send event to endpoint, status code not OK", "statusCode", res.StatusCode)
		return err
	}

	return nil
}

func (r *ConfigMapWatcherReconciler) StatusHandler(ctx context.Context, w *appsv1alpha1.ConfigMapWatcher, configMapVersion string, sendErr error) error {
	key := types.NamespacedName{Name: w.Name, Namespace: w.Namespace}

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		latest := &appsv1alpha1.ConfigMapWatcher{}
		err := r.Get(ctx, key, latest)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}

		condition := metav1.Condition{
			Type:               "Synced",
			ObservedGeneration: latest.Generation,
			LastTransitionTime: metav1.Now(),
		}

		if sendErr != nil {
			latest.Status.LastEventStatus = "Failed"
			condition.Status = metav1.ConditionFalse
			condition.Reason = "SendFailed"
			condition.Message = sendErr.Error()
		} else {
			latest.Status.LastEventStatus = "Success"
			latest.Status.LastSyncTime = metav1.Now()
			latest.Status.LastConfigMapVersion = configMapVersion
			latest.Status.LastEventSent = metav1.Now()
			condition.Status = metav1.ConditionTrue
			condition.Reason = "SendSuccess"
			condition.Message = "Successfully synced"
		}

		latest.Status.ObservedGeneration = latest.Generation
		meta.SetStatusCondition(&latest.Status.Conditions, condition)
		return r.Status().Update(ctx, latest)
	})
}
