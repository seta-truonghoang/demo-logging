# ☸️ Master Kubectl CLI: The Comprehensive Tutorial & Cheat Sheet

`kubectl` is the command-line interface (CLI) tool used to communicate with a Kubernetes cluster's **API Server**. This guide will take you from absolute basics to intermediate-advanced troubleshooting commands.

---

## 📌 1. Anatomy of a Kubectl Command

Every `kubectl` command follows a common, predictable pattern:

```bash
kubectl [action] [resource_type] [resource_name] [flags]
```

*   **`action`**: What you want to do (e.g., `get`, `describe`, `apply`, `delete`, `logs`, `exec`).
*   **`resource_type`**: The kind of Kubernetes resource (e.g., `pod` / `po`, `deployment` / `deploy`, `service` / `svc`, `configmap` / `cm`). *Both singular, plural, and short names are accepted.*
*   **`resource_name`**: The specific name of the resource (optional; if omitted, the action applies to all resources of that type).
*   **`flags`**: Optional modifiers (e.g., `-n kube-system` to specify namespace, `-o yaml` to format output).

---

## 🧭 2. Cluster Information & Context Management

Before running commands, you must know *which* cluster and *which* namespace you are targeting.

| Command | Explanation |
| :--- | :--- |
| `kubectl cluster-info` | Displays addresses of the control plane and core cluster services. |
| `kubectl config get-contexts` | Lists all available cluster connections configured in your `~/.kube/config`. |
| `kubectl config current-context` | Shows the active cluster connection you are currently targeting. |
| `kubectl config use-context <name>` | Switches your active connection to another cluster context. |
| `kubectl get nodes` | Lists all physical/virtual worker nodes making up your cluster. |

---

## 🛠️ 3. Creating & Running Resources

How to deploy resources to your cluster.

| Command | Explanation |
| :--- | :--- |
| **`kubectl apply -f <filename.yaml>`** | **The Production Standard (Declarative)**: Creates or updates resources defined in a YAML file. |
| `kubectl create -f <filename.yaml>` | **Imperative Creation**: Attempts to create resources. Fails if the resource already exists. |
| `kubectl run my-pod --image=nginx` | Starts a single isolated Pod running an Nginx container (great for quick testing). |
| `kubectl create deployment my-dep --image=nginx` | Creates a Deployment wrapper running Nginx. |

---

## 🔍 4. Inspecting & Viewing Resources

Once resources are applied, you need to verify their status, health, and configurations.

| Command | Explanation |
| :--- | :--- |
| `kubectl get pods` | Lists all active Pods in the `default` namespace. Add `-n <namespace>` for others. |
| `kubectl get all` | Displays a high-level summary of all Pods, Services, Deployments, and ReplicaSets. |
| `kubectl describe pod <pod-name>` | **Deep Inspection**: Shows detailed metadata, lifecycle events, and container statuses of a Pod. |
| `kubectl get pod <pod-name> -o yaml` | Exports the full runtime configuration of a Pod in clean YAML format. |
| `kubectl explain deployment` | Inline documentation! Describes schema fields of any resource definition. |

---

## 📈 5. Scaling, Modifying & Rollouts

How to scale workloads and deploy updates.

| Command | Explanation |
| :--- | :--- |
| `kubectl scale deployment/<name> --replicas=5` | Instantly adjusts the desired number of Pods for a deployment. |
| `kubectl edit deployment/<name>` | Opens the resource schema directly in your default terminal editor (Vim/Nano) to update configs live. |
| `kubectl rollout status deployment/<name>` | Monitors the progress of a rolling update rollout. |
| `kubectl rollout history deployment/<name>` | Reviews revision histories of a deployment. |
| `kubectl rollout undo deployment/<name>` | Rolls back the deployment to the previous stable revision instantly. |

---

## 🩺 6. Debugging, Troubleshooting & Logs

These commands are your main toolkit when things go wrong in your applications.

### A. Reading Logs
```bash
# View stdout/stderr logs of a Pod
kubectl logs <pod-name>

# Stream logs in real-time (like tail -f)
kubectl logs -f <pod-name>

# View logs of a specific container inside a multi-container Pod
kubectl logs <pod-name> -c <container-name>

# View logs from the previously crashed container instance
kubectl logs <pod-name> --previous
```

### B. Interactive Container Access
```bash
# Start an interactive shell inside a running container (like docker exec)
kubectl exec -it <pod-name> -- sh

# Run a quick one-off command inside a container without opening shell
kubectl exec <pod-name> -- env
```

### C. Network Diagnostics (Port Forwarding)
```bash
# Forward traffic from your host port 8080 to the Pod's port 80 internally
kubectl port-forward <pod-name> 8080:80

# Forward traffic from host port 9000 to a Service's port 80
kubectl port-forward svc/<service-name> 9000:80
```

---

## ⚡ 7. Power-User Pro Tips

### 1. The Dynamic Watch Flag (`-w`)
Append `-w` to any `get` command to stream state updates live instead of polling continuously:
```bash
kubectl get pods -w
```

### 2. Output Formatting (`-o`)
Extract customized fields or export definitions easily:
```bash
# List pods with more detailed information (IP, Node name)
kubectl get pods -o wide

# Get only the names of running pods (useful for bash scripts)
kubectl get pods -o name

# Get specific JSON elements (e.g. print container image names)
kubectl get pods -o jsonpath='{.items[*].spec.containers[*].image}'
```

### 3. Safe Dry-Runs (`--dry-run=client`)
Test if your syntax or command is valid without making any changes to the cluster:
```bash
kubectl create deployment test-dep --image=nginx --dry-run=client -o yaml
```
*(This will print the generated YAML file without actually creating the deployment inside K8s!)*

---

## 🎮 8. Hands-On Practice Session

Using the K8s manifests we just created in the `k8s-demo` folder, run these commands sequentially to practice!

1.  **Deploy everything:**
    ```bash
    kubectl apply -f configmap.yaml
    kubectl apply -f deployment.yaml
    kubectl apply -f service.yaml
    ```
2.  **Verify what you created:**
    ```bash
    kubectl get cm,deploy,svc
    ```
3.  **Inspect our application Deployment specifications:**
    ```bash
    kubectl describe deployment/golang-app-deployment
    ```
4.  **Find a Pod name and check its environment variables:**
    ```bash
    # Get the pod name first
    kubectl get pods
    
    # Run a command inside that pod to verify CUSTOM_MESSAGE is present
    kubectl exec <pod-name> -- env | grep CUSTOM_MESSAGE
    ```
5.  **Scale it up:**
    ```bash
    kubectl scale deployment/golang-app-deployment --replicas=5
    kubectl get pods -o wide
    ```
6.  **Tear it all down:**
    ```bash
    kubectl delete -f .
    ```
