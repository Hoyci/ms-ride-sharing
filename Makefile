.PHONY: dev-up dev-down dev-reset minikube-start minikube-stop

minikube-start:
minikube start --memory=8192 --cpus=4

minikube-stop:
minikube stop

dev-up:
tilt up

dev-down:
tilt down

dev-reset:
tilt down
kubectl delete namespace ride-sharing
kubectl create namespace ride-sharing
tilt up