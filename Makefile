# Image URL to use all building/pushing image targets
IMG ?= gcr.io/cf-ism-0/projects-operator:latest
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

generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./api/...
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role paths=./... output:crd:artifacts:config=helm/projects-operator/crds

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

#################### PDC HELM ####################

helm-install:
	./scripts/helm-install
