.PHONY: image deploy-local

image:
	docker build -t my-server:local .

deploy-local: image
	kind load docker-image my-server:local
	kubectl apply -f deployment.yaml -f service.yaml
	kubectl rollout restart deployment/my-server