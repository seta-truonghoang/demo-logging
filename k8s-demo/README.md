# ☸️ Kubernetes Demo Guide: Step-by-Step Practical Scenarios

This folder contains a complete, highly visual demonstration of the core capabilities of **Kubernetes (K8s)**. Using the optimized **Go Web Application** we created in the previous step, you can show your team or audience how Kubernetes handles **High Availability**, **Self-Healing**, **Zero-Downtime Rolling Updates**, and **Elastic Scaling** in real-time.

---

## 🏗️ Preparation & Setup

To run this demo locally, you can use any local Kubernetes cluster such as **Minikube**, **Kind**, or **Docker Desktop (Kubernetes enabled)**.

### Step 1: Build & Load your Docker Image
Since we are using our custom Go application, we need to make sure the Kubernetes cluster can find the `go-docker-demo:latest` image.

*   **If you are using Minikube:**
    Configure your terminal to point to Minikube's Docker daemon and build the image directly inside it:
    ```bash
    # Point terminal to Minikube's Docker registry
    eval $(minikube docker-env)

    # Move to the Go directory and build the image
    cd ../docker-demo/golang
    docker build -t go-docker-demo:latest .
    cd ../../k8s-demo
    ```

*   **If you are using Kind:**
    Build the image locally and then load it into your Kind cluster:
    ```bash
    cd ../docker-demo/golang
    docker build -t go-docker-demo:latest .
    cd ../../k8s-demo
    kind load docker-image go-docker-demo:latest
    ```

*   **If you are using Docker Desktop K8s:**
    Build the image locally. Docker Desktop shares the host Docker engine with its K8s cluster, so it is immediately available:
    ```bash
    cd ../docker-demo/golang
    docker build -t go-docker-demo:latest .
    cd ../../k8s-demo
    ```

### Step 2: Deploy the manifests to Kubernetes
Apply all the files in this directory to spin up the ConfigMap, Deployment (3 replicas), and NodePort Service:
```bash
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

Verify that all 3 Pods are running:
```bash
kubectl get pods -o wide
```

---

## 🎬 Demo Scenario 1: Load Balancing & High Availability

### 💡 The Goal
Demonstrate how a Kubernetes `Service` acts as an intelligent, built-in load balancer distributing traffic among active Pods.

### 🛠️ Execution
1.  **Expose/Port-forward the Service** (if not using Minikube or NodePort directly):
    *   *Docker Desktop / NodePort*: The app is already accessible at `http://localhost:30080`.
    *   *Minikube*: If you cannot access port 30080 directly, run:
        ```bash
        minikube service golang-app-service
        ```
        (This will output the local URL, or you can run `kubectl port-forward svc/golang-app-service 8080:8080` in a separate terminal and use port `8080`).

2.  **Run a continuous curl loop in Terminal 1**:
    ```bash
    # Change port to 8080 if using port-forwarding
    while true; do curl -s http://localhost:30080; echo ""; sleep 0.8; done
    ```

### 📺 What your audience will see:
```json
{"language":"Go","message":"Xin chào từ Docker Container tối ưu cho Go!","timestamp":"...","hostname":"golang-app-deployment-55d8f6f5dc-abc12","version":"1.0.0"}
{"language":"Go","message":"Xin chào từ Docker Container tối ưu cho Go!","timestamp":"...","hostname":"golang-app-deployment-55d8f6f5dc-xyz34","version":"1.0.0"}
{"language":"Go","message":"Xin chào từ Docker Container tối ưu cho Go!","timestamp":"...","hostname":"golang-app-deployment-55d8f6f5dc-qwe56","version":"1.0.0"}
```
Notice how the `hostname` field (which corresponds to the unique name of the Pod) changes on every request! This proves Kubernetes is round-robin load-balancing traffic across the three running containers transparently.

---

## 🎬 Demo Scenario 2: Self-Healing (Tự Chữa Lành)

### 💡 The Goal
Demonstrate how Kubernetes automatically detects container death or Pod failure and instantly recreates them to maintain the "desired state" (Replicas = 3).

### 🛠️ Execution
1.  Keep the **continuous curl loop** running in **Terminal 1**.
2.  Open **Terminal 2** and watch the Pods in real-time:
    ```bash
    kubectl get pods -w
    ```
3.  Open **Terminal 3** and randomly delete one of the running Pods:
    ```bash
    # Get pod names
    kubectl get pods
    
    # Delete one specific pod
    kubectl delete pod <pod-name-from-previous-command>
```

### 📺 What your audience will see:
*   In **Terminal 2 (Watch)**: You will see the deleted Pod instantly transition to `Terminating`. Simultaneously, a brand-new Pod with a different ID is created in a fraction of a second and transitions to `Pending` -> `ContainerCreating` -> `Running`.
*   In **Terminal 1 (Curl loop)**: There is **zero downtime!** The requests continue to resolve successfully because the K8s Service immediately removes the terminating Pod from the load-balancer endpoints and redirects traffic only to the remaining healthy Pods.

---

## 🎬 Demo Scenario 3: Zero-Downtime Rolling Update & Rollback

### 💡 The Goal
Show how we can rollout changes (like code updates or environment configs) to production step-by-step with zero client-side service interruptions.

### 🛠️ Execution
1.  Keep the **curl loop** running in **Terminal 1**.
2.  Let's modify the application's configuration by editing the ConfigMap. Update `configmap.yaml` to change the message:
    ```yaml
    CUSTOM_MESSAGE: "Kubernetes is absolutely brilliant and fully updated!"
    ```
3.  Apply the updated ConfigMap:
    ```bash
    kubectl apply -f configmap.yaml
    ```
4.  Trigger a Rolling Update rollout of our deployment so it picks up the configuration changes:
    ```bash
    kubectl rollout restart deployment/golang-app-deployment
    ```
5.  Watch the rollout progress in Terminal 2:
    ```bash
    kubectl rollout status deployment/golang-app-deployment
    ```

### 📺 What your audience will see:
In **Terminal 1 (Curl loop)**, you will see a graceful transition without a single failed request:
```json
... old message ... (hostname: pod-abc)
... old message ... (hostname: pod-xyz)
... Kubernetes is absolutely brilliant ... (hostname: new-pod-123)
... old message ... (hostname: pod-abc)
... Kubernetes is absolutely brilliant ... (hostname: new-pod-456)
... Kubernetes is absolutely brilliant ... (hostname: new-pod-789)
```
Kubernetes does this by spinning up one new Pod at a time, waiting for its `readinessProbe` to pass, routing traffic to it, and only then shutting down an old Pod.

#### ↩️ Instant Rollback Demo:
Imagine you accidentally rolled out a broken configuration. You can rollback to the previous stable state instantly with one simple command:
```bash
kubectl rollout undo deployment/golang-app-deployment
```
The cluster immediately rolls back to the previous deployment version gracefully!

---

## 🎬 Demo Scenario 4: Elastic Scaling (Co Giãn Linh Hoạt)

### 💡 The Goal
Demonstrate how easily Kubernetes handles traffic surges by dynamically scaling containers up or down on demand.

### 🛠️ Execution
1.  Keep the **curl loop** running in **Terminal 1**.
2.  Open **Terminal 2** and watch the Pod instances:
    ```bash
    kubectl get pods -w
    ```
3.  In **Terminal 3**, scale up the deployment from 3 replicas to **6 replicas**:
    ```bash
    kubectl scale deployment/golang-app-deployment --replicas=6
    ```
4.  Watch the Pod list in Terminal 2 grow instantly as 3 new containers are provisioned.
5.  Check the **curl loop** in Terminal 1. Within seconds, you will see traffic being distributed across all 6 distinct Pods!
6.  Once the surge is over, scale back down to save resources:
    ```bash
    kubectl scale deployment/golang-app-deployment --replicas=2
    ```
    Kubernetes gracefully terminates 4 Pods, keeping the service perfectly balanced.

---

## 🧹 Cleanup
When you are done with the demo, you can tear down all resources easily:
```bash
kubectl delete -f .
```
This removes the Service, Deployment, and ConfigMap cleanly, leaving your cluster spotless!
