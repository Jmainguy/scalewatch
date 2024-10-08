# Scalewatch: GitHub Runner Scale Set Operator

Scalewatch is a Kubernetes Operator that manages GitHub Runner Scale Sets. The operator interacts with the GitHub API to monitor the state of self-hosted runners and ensure they are running smoothly. If a runner goes offline, scalewatch automatically restarts the Kubernetes Pod running the affected runner, ensuring continuous uptime for your GitHub Actions workflows.

## Features
* Automated Monitoring: Continuously checks the status of GitHub Runner Scale Sets.
* Automatic Recovery: Detects when a runner is offline and automatically restarts the pod.
* CRD Based: Leverages a Custom Resource Definition (CRD) to configure the Runner Scale Sets.
* Namespace-Scoped Pods: Manages pods within the same namespace, ensuring efficient runner scale set monitoring.
* Configurable GitHub Runner Names: Define runner scale set names and GitHub repository configurations through Kubernetes resources.
* Status Reporting: Provides real-time status updates on the GitHub Runner Scale Sets when queried via kubectl.

## Custom Resource Definition (CRD)

The core of the operator is the RunnerScaleSet custom resource. This CRD allows users to define their GitHub Runner Scale Set and configure key parameters such as the GitHub repository and the runner set name.

### Example RunnerScaleSet YAML
```yaml
apiVersion: scalewatch.soh.re/v1alpha1
kind: RunnerScaleSet
metadata:
  name: runnerscaleset-sample
  namespace: arc-system
spec:
  githubConfigSecret: gha-runner-token  # Kubernetes secret for GitHub token
  githubConfigURL: https://github.com/Standouthost  # GitHub Org / Enterprise URL
  name: self-hosted  # Name of the GitHub Runner Scale Set
status:
  state: Online  # Real-time status of the Runner Scale Set
```

## Fields
* githubConfigSecret: Kubernetes secret that stores the GitHub token needed for API access.
* githubConfigURL: The URL of the GitHub Org or Enterprise this scale set is linked to.
* name: The name of the self-hosted runner scale set.
* status.state: Displays the real-time state of the runner (e.g., Online, Offline).

## Prerequisites
* A Kubernetes cluster.
* GitHub Actions configured to use self-hosted runners.
* A Kubernetes secret containing your GitHub token to authenticate with the GitHub API.

## Creating the Secret
Before deploying a RunnerScaleSet, you need to create a secret that contains your GitHub token:

```bash
kubectl create secret generic gha-runner-token --from-literal=token=<YOUR_GITHUB_TOKEN> -n arc-system
```

## Installing scalewatch

Clone the Repository:

```bash
git clone https://github.com/jmainguy/scalewatch.git
cd scalewatch
Deploy the Operator using Kustomize:
```

```bash
kubectl apply -k ./config/default
```

Create a RunnerScaleSet Resource: Create the RunnerScaleSet YAML, or use the example provided above, and apply it:

```bash
kubectl apply -f runnerscaleset-sample.yaml
```

## Checking the Status

You can check the status of your runner scale sets by running:

```/bin/bash
kubectl get runnerscaleset -A
```

This will show the status of the runner, including the state (e.g., Online or Offline), and other details such as the GitHub configuration URL and runner scale set name.

Example output:

```bash
NAMESPACE    NAME                    STATE    GITHUBCONFIGURL                    RUNNERSCALESETNAME
arc-system   runnerscaleset-sample   Online   https://github.com/Standouthost    self-hosted
emu-test     runnerscaleset-emu      Online   https://github.com/enterprise-emu  self-hosted-enterprise
You can also inspect the full resource by running:

```bash
kubectl get runnerscaleset runnerscaleset-sample -n arc-system -o yaml
```
## Troubleshooting

* Pod Not Restarting: Ensure that the operator has the correct permissions to delete and restart pods in the target namespace.
* GitHub API Errors: Check if the GitHub token stored in the secret is valid and has the correct permissions to manage runners.
* CRD Not Applied: Make sure the CRD is correctly installed in your cluster by running kubectl get crd.

## Contributing
Contributions are welcome! Please open an issue or submit a pull request with improvements or new features.
