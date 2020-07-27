# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:crdVersions=v1"
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
	kubectl apply -f deployments/k8s/manifests/projects.vmware.com_projects.yaml
	kubectl apply -f deployments/k8s/manifests/projects.vmware.com_projectaccesses.yaml

generate: generate-deepcopy generate-rbac generate-crd
	go generate ./...

generate-deepcopy: controller-gen
	$(CONTROLLER_GEN) \
		object:headerFile=./hack/boilerplate.go.txt \
		paths=./api/...

generate-crd: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) \
		output:crd:artifacts:config=deployments/k8s/manifests \
		paths=./...

generate-rbac: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) \
		rbac:roleName=projects-manager-role \
		output:rbac:stdout \
		paths=./controllers/... > deployments/k8s/manifests/manager-role.yaml
	$(CONTROLLER_GEN) $(CRD_OPTIONS) \
		rbac:roleName=projectaccesses-manager-role \
		output:rbac:stdout \
		paths=./pkg/... > deployments/k8s/manifests/projectaccess-role.yaml
	$(CONTROLLER_GEN) $(CRD_OPTIONS) \
		rbac:roleName=projects-leader-election-role \
		output:rbac:stdout \
		paths=./cmd/manager/... > deployments/k8s/manifests/leader-election-role.yaml
	./scripts/yttify-yaml

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

#################### k14s ####################

kapp-deploy:
	./scripts/kapp-deploy

kapp-local-deploy:
	./scripts/kapp-deploy --local

kapp-delete:
	./scripts/kapp-delete
