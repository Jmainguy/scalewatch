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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1" // Import the meta package
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	scalewatchv1alpha1 "github.com/jmainguy/scalewatch/api/v1alpha1"
)

const (
	ghAPIBase = "https://api.github.com"
)

// RunnerScaleSetReconciler reconciles a RunnerScaleSet object
type RunnerScaleSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=scalewatch.soh.re,resources=runnerscalesets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=scalewatch.soh.re,resources=runnerscalesets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=scalewatch.soh.re,resources=runnerscalesets/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RunnerScaleSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *RunnerScaleSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	var runnerScaleSet scalewatchv1alpha1.RunnerScaleSet
	if err := r.Client.Get(ctx, req.NamespacedName, &runnerScaleSet); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Extract the secret name from the spec
	secretName := runnerScaleSet.Spec.GithubConfigSecret
	githubConfigURL := runnerScaleSet.Spec.GithubConfigURL
	runnerScaleSetName := runnerScaleSet.Spec.Name

	// Fetch the secret from Kubernetes
	var secret corev1.Secret
	if err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: req.Namespace, // Make sure you are in the right namespace
		Name:      secretName,
	}, &secret); err != nil {
		return ctrl.Result{}, err
	}

	// Extract the secret data
	githubToken := string(secret.Data["github_token"])

	// Step 3: Get the runner scale set status
	runnerScaleSetStatus, err := getRunnerScaleSetStatus(githubToken, githubConfigURL, runnerScaleSetName)
	if err != nil {
		log.Error(err, "Error getting runner scale set status")
	}

	// Step 4: Set the runnerScaleSet status

	if runnerScaleSetStatus.Status == "online" {
		if runnerScaleSet.Status.State != "Online" {
			log.Info("Setting status to Online", "runnerScaleSetName", runnerScaleSetName)
			runnerScaleSet.Status.State = "Online"
		}
	} else {
		if runnerScaleSet.Status.State != "Offline" {
			log.Info("Setting status to Offline", "runnerScaleSetName", runnerScaleSetName)
			runnerScaleSet.Status.State = "Offline"
		}
		// Bounce listener pod, its offline
		// Get All pods
		pods, err := GetMatchingPods(ctx, r.Client, req.Namespace, runnerScaleSetName)
		if err != nil {
			log.Error(err, "Error getting listener pods")
		}
		for _, pod := range pods {
			log.Info("Deleting listener pod, in hopes it comes up healthy", "pod", pod.Name)
			err := DeletePod(ctx, r.Client, req.Namespace, pod.Name)
			if err != nil {
				log.Error(err, "Error Deleting pod")
			}
		}
	}

	err = r.Status().Update(ctx, &runnerScaleSet)

	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RunnerScaleSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalewatchv1alpha1.RunnerScaleSet{}).
		Complete(r)
}

// Custom functions from scratch

type RegistrationTokenResponse struct {
	Token string `json:"token"`
}

type PipelineInfo struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

type ScaleSetList struct {
	Count int64            `json:"count"`
	Value []ScaleSetStatus `json:"value"`
}

type ScaleSetStatus struct {
	AcquireJobsURL       string `json:"acquireJobsUrl"`
	CreateSessionURL     string `json:"createSessionUrl"`
	CreatedOn            string `json:"createdOn"`
	Enabled              bool   `json:"enabled"`
	GetAcquirableJobsURL string `json:"getAcquirableJobsUrl"`
	ID                   int64  `json:"id"`
	Labels               []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"labels"`
	Name               string `json:"name"`
	QueueName          string `json:"queueName"`
	RunnerGroupID      int64  `json:"runnerGroupId"`
	RunnerGroupName    string `json:"runnerGroupName"`
	RunnerJitConfigURL string `json:"runnerJitConfigUrl"`
	RunnerSetting      struct {
		DisableUpdate bool        `json:"disableUpdate"`
		Ephemeral     bool        `json:"ephemeral"`
		ID            int64       `json:"id"`
		Name          interface{} `json:"name"`
		RunnerGroupID interface{} `json:"runnerGroupId"`
		Status        int64       `json:"status"`
		Version       interface{} `json:"version"`
	} `json:"runnerSetting"`
	Statistics struct {
		TotalAcquiredJobs      int64 `json:"totalAcquiredJobs"`
		TotalAssignedJobs      int64 `json:"totalAssignedJobs"`
		TotalAvailableJobs     int64 `json:"totalAvailableJobs"`
		TotalBusyRunners       int64 `json:"totalBusyRunners"`
		TotalIdleRunners       int64 `json:"totalIdleRunners"`
		TotalRegisteredRunners int64 `json:"totalRegisteredRunners"`
		TotalRunningJobs       int64 `json:"totalRunningJobs"`
	} `json:"statistics"`
	Status string `json:"status"`
}

func returnToken(url, githubToken string) (string, error) {
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth("token", githubToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("failed to fetch registration token: %s", resp.Status)
	}

	var result RegistrationTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Token, nil
}

func getRegistrationToken(orgOrEnterprise, githubToken string) (string, string, error) {

	// Enterprise
	enterpriseURL := fmt.Sprintf("%s/enterprises/%s/actions/runners/registration-token", ghAPIBase, orgOrEnterprise)

	// Org
	orgURL := fmt.Sprintf("%s/orgs/%s/actions/runners/registration-token", ghAPIBase, orgOrEnterprise)

	token, err := returnToken(enterpriseURL, githubToken)

	if err != nil {
		token, err := returnToken(orgURL, githubToken)
		if err != nil {
			return "", "", err
		}
		return token, "org", nil
	}

	return token, "enterprise", nil
}

func getPipelineInfo(registrationToken, runnerRegistrationURL, runnerKind string) (*PipelineInfo, error) {
	url := fmt.Sprintf("%s/actions/runner-registration", ghAPIBase)
	payloadURL := runnerRegistrationURL

	if runnerKind == "enterprise" {
		payloadURL = strings.ReplaceAll(runnerRegistrationURL, "https://github.com/", "https://github.com/enterprises/")
	}

	payload := map[string]string{
		"url":         payloadURL,
		"runnerEvent": "register",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "RemoteAuth "+registrationToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch pipeline info: %s", resp.Status)
	}

	var result PipelineInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func getRunnerScaleSetStatus(githubToken, githubConfigURL, runnerScaleSetName string) (rss ScaleSetStatus, err error) {
	orgOrEnterprise := strings.TrimPrefix(githubConfigURL, "https://github.com/")

	// Step 1: Get the registration token
	registrationToken, runnerKind, err := getRegistrationToken(orgOrEnterprise, githubToken)
	if err != nil {
		return rss, err
	}

	// Step 2: Get pipeline information
	pipelineInfo, err := getPipelineInfo(registrationToken, githubConfigURL, runnerKind)
	if err != nil {
		return rss, err
	}
	url := fmt.Sprintf("%s/_apis/runtime/runnerscalesets?api-version=%s", pipelineInfo.URL, "6.0-preview")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return rss, err
	}
	req.Header.Set("Authorization", "Bearer "+pipelineInfo.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return rss, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return rss, fmt.Errorf("failed to fetch runner scale set status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)

	var runnerScaleSetList ScaleSetList

	err = json.Unmarshal(body, &runnerScaleSetList)
	if err != nil {
		return rss, err
	}

	var match bool

	for _, scaleSet := range runnerScaleSetList.Value {
		if scaleSet.Name == runnerScaleSetName {
			rss = scaleSet
			match = true
		}
	}

	if !match {
		err := fmt.Errorf("unable to find runnerScaleSet")
		return rss, err
	}

	return rss, nil
}

func GetMatchingPods(ctx context.Context, cli client.Client, namespace, runnerScaleSetName string) ([]corev1.Pod, error) {
	var matchingPods []corev1.Pod

	// Define the label selector
	labelSelector := labels.SelectorFromSet(labels.Set{
		"actions.github.com/scale-set-name": runnerScaleSetName,
	})

	// List options with label selector
	listOptions := client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labelSelector,
	}

	// Create a Pod list
	podList := &corev1.PodList{}

	// List the pods in the namespace with the specified label
	if err := cli.List(ctx, podList, &listOptions); err != nil {
		return nil, err
	}

	// Iterate over the pods and check their owner references
	for _, pod := range podList.Items {
		for _, ownerRef := range pod.OwnerReferences {
			if ownerRef.Kind == "AutoscalingListener" && ownerRef.UID == types.UID(pod.OwnerReferences[0].UID) {
				matchingPods = append(matchingPods, pod)
				break
			}
		}
	}

	return matchingPods, nil
}

// DeletePod deletes a pod by name in the specified namespace.
func DeletePod(ctx context.Context, cli client.Client, namespace string, podName string) error {
	// Create a key for the pod
	podKey := types.NamespacedName{
		Namespace: namespace,
		Name:      podName,
	}

	// Create a Pod object to hold the pod's data
	pod := &corev1.Pod{}

	// Fetch the pod to confirm it exists
	if err := cli.Get(ctx, podKey, pod); err != nil {
		return err // Return other errors
	}

	// Delete the pod
	if err := cli.Delete(ctx, pod); err != nil {
		return err // Return any errors encountered during deletion
	}

	return nil // Pod deleted successfully
}
