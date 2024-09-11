PROTOC:=protoc
PROTO_PATH:=$(HOME)/go/src/github.com/koopa0/auth/
GOOGLE_PROTOBUF_PATH:=$(GOPATH)/pkg/mod/github.com/google/protobuf@v5.27.3+incompatible/src/
GO_OUT:=proto/pb
PROTO_FILES:=proto/*.proto

.PHONY: all run test db-up db-down redis-up redis-down nsq-up nsq-down build docker-build docker-push k8s-deploy k8s-delete clean proto

all: run

run:
	go run cmd/api/main.go

test:
	go test ./...

db-up:
	docker-compose up -d db

db-down:
	docker-compose down

redis-up:
	docker-compose up -d redis

redis-down:
	docker-compose down

nsq-up:
	docker-compose up -d nsq

nsq-down:
	docker-compose down

build:
	go build -o neomart ./cmd/api

docker-build:
	docker build -t your-org/neomart:latest .

docker-push:
	docker push your-org/neomart:latest

k8s-deploy:
	kubectl apply -f k8s/

k8s-delete:
	kubectl delete -f k8s/

clean:
	go clean
	docker-compose down
	kubectl delete -f k8s/

sqlc-generate:
	sqlc generate

migrate-up:
	migrate -database ${POSTGRESQL_URL} -path ./migrations up

migrate-down:
	migrate -database ${POSTGRESQL_URL} -path ./migrations down

gcp-auth:
	gcloud auth login

gcp-unset-token:
	gcloud config unset auth/access_token_file

gcp-auth-application:
	gcloud auth application-default login

proto:
	$(PROTOC) -I $(PROTO_PATH) \
	-I $(GOOGLE_PROTOBUF_PATH) \
	--proto_path=proto --go_out=$(GO_OUT) --go_opt=paths=source_relative \
	--go-grpc_out=$(GO_OUT) --go-grpc_opt=paths=source_relative \
	$(PROTO_FILES)