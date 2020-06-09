# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
GINKGO_ARGS = -r -p -randomizeSuites -randomizeAllSpecs

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: generate format manager webhook

test: lint unit-tests acceptance-tests

unit-tests:
	ginkgo ${GINKGO_ARGS} -skipPackage acceptance

acceptance-tests:
	ginkgo -r acceptance

manager:
	go build -o bin/manager cmd/manager/main.go

webhook:
	go build -o bin/webhook cmd/webhook/main.go

run: generate format
	go run ./cmd/manager/main.go

install: generate
	kubectl apply -f helm/projects-operator/crds

generate: generate-deepcopy generate-rbac generate-crd
	go generate ./...

generate-deepcopy: controller-gen
	$(CONTROLLER_GEN) \
		object:headerFile=./hack/boilerplate.go.txt \
		paths=./api/...

generate-crd: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) \
		output:crd:artifacts:config=helm/projects-operator/crds \
		paths=./...

generate-rbac: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) \
		rbac:roleName=projects-manager-role \
		output:rbac:stdout \
		paths=./controllers/... > helm/projects-operator/templates/manager-role.yaml
	$(CONTROLLER_GEN) $(CRD_OPTIONS) \
		rbac:roleName=projectaccesses-manager-role \
		output:rbac:stdout \
		paths=./pkg/... > helm/projects-operator/templates/projectaccess-role.yaml
	$(CONTROLLER_GEN) $(CRD_OPTIONS) \
		rbac:roleName=projects-leader-election-role \
		output:rbac:stdout \
		paths=./cmd/manager/... > helm/projects-operator/templates/leader-election-role.yaml
	./scripts/helmify-yaml

controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.4
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

#### Custom tasks ####
clean-crs:
	kubectl delete projects --all

lint:
	golangci-lint run --timeout 2m30s --verbose

format:
	golangci-lint run --fix --timeout 2m30s --verbose

dev:
	kubectl create namespace "$$NAMESPACE" || true
	kubectl create secret docker-registry "$$REGISTRY_SECRET_NAME" \
		--namespace "$$NAMESPACE" \
		--docker-server="$$REGISTRY_URL" \
		--docker-username="$$REGISTRY_USERNAME" \
		--docker-password="$$REGISTRY_PASSWORD" \
		--docker-email="$$REGISTRY_EMAIL" || true
	IMAGE_TAG=$(shell hostname) skaffold dev --force=false

#################### HELM ####################

helm-install:
	./scripts/helm-install

helm-local-install:
	CLUSTER_ROLE_REF=acceptance-test-clusterrole ./scripts/helm-install --local

helm-uninstall:
	./scripts/helm-uninstall
