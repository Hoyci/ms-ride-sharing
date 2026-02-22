PROTO_DIR = proto/v1
PROTO_OUT = shared/proto/v1

.PHONY: dev-up dev-down dev-reset minikube-start minikube-stop generate_types

minikube-start:
	minikube start

minikube-stop:
	minikube stop

dev-up:
	tilt up

dev-down:
	tilt down

dev-reset:
	tilt down
	tilt up

generate-types:
	@echo "Limpando tipagens antigas..."
	rm -rf $(PROTO_OUT)
	mkdir -p $(PROTO_OUT)

	@echo "Gerando arquivos de tipagem do Golang"
	protoc \
		-I=$(PROTO_DIR) \
		--go_out=$(PROTO_OUT) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT) \
		--go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=$(PROTO_OUT) \
		--grpc-gateway_opt=paths=source_relative \
		--validate_out="lang=go,paths=source_relative:$(PROTO_OUT)" \
		$(PROTO_DIR)/user/*.proto

	@echo "Sucesso! Tipagens geradas em $(PROTO_OUT)."