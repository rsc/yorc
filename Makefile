#!/usr/bin/env bash
# Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GOTOOLS = github.com/kardianos/govendor github.com/jteeuwen/go-bindata/... github.com/abice/go-enum github.com/google/addlicense

VETARGS?=-all -asmdecl -atomic -bool -buildtags -copylocks -methods \
         -nilfunc -printf -rangeloops -shift -structtags -unsafeptr

VERSION=$(shell grep "yorc_version" versions.yaml | awk '{print $$2}')
COMMIT_HASH=$(shell git rev-parse HEAD)
TF_CONSUL_PLUGIN_VERSION=$(shell grep "tf_consul_plugin_version" versions.yaml | awk '{print $$2}')
TF_AWS_PLUGIN_VERSION=$(shell grep "tf_aws_plugin_version" versions.yaml | awk '{print $$2}')
TF_OPENSTACK_PLUGIN_VERSION=$(shell grep "tf_openstack_plugin_version" versions.yaml | awk '{print $$2}')
TF_GOOGLE_PLUGIN_VERSION=$(shell grep "tf_google_plugin_version" versions.yaml | awk '{print $$2}')

build: test
	@echo "--> Running go build"
	@CGO_ENABLED=0 go build $(BUILD_ARGS) -ldflags "-X github.com/ystia/yorc/commands.version=v$(VERSION) -X github.com/ystia/yorc/commands.gitCommit=$(COMMIT_HASH) \
	 -X github.com/ystia/yorc/commands.tfConsulPluginVersion=$(TF_CONSUL_PLUGIN_VERSION) \
	 -X github.com/ystia/yorc/commands.tfAWSPluginVersion=$(TF_AWS_PLUGIN_VERSION) \
	 -X github.com/ystia/yorc/commands.tfOpenStackPluginVersion=$(TF_OPENSTACK_PLUGIN_VERSION) \
	 -X github.com/ystia/yorc/commands.tfGooglePluginVersion=$(TF_GOOGLE_PLUGIN_VERSION)"

generate: checks
	@go generate ./...

checks:
	@./build/checks.sh $(GOTOOLS)

header:
	@echo "--> Adding licensing headers if necessary"
	@./build/header.sh

dist: build
	@rm -rf ./dist && mkdir -p ./dist
	@echo "--> Creating an archive"
	@tar czvf yorc-$(VERSION).tgz yorc && echo "TODO: clean this part after CI update" &&  cp yorc yorc-$(VERSION).tgz dist/
	@cd doc && make html latexpdf && cd _build && cp -r html latex/Yorc.pdf ../../dist
	@cd ./dist && zip -r yorc-server-$(VERSION)-documentation.zip html Yorc.pdf && zip yorc-server-$(VERSION)-distrib.zip yorc yorc-server-$(VERSION)-documentation.zip

test: generate header format
ifndef SKIP_TESTS
	@echo "--> Running go test"
	@export PATH=$$PWD/build:$$PATH; go test $(TESTARGS) -p 1 ./...
endif


cover:
	@go test -p 1 -cover $(COVERARGS) ./...  

format:
	@echo "--> Running go fmt"
	@go fmt ./...

vet:
	@echo "--> Running go tool vet $(VETARGS) ."
	@go list ./... \
		| cut -d '/' -f 4- \
		| xargs -n1 \
			go tool vet $(VETARGS) ;\
	if [ $$? -ne 0 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for reviewal."; \
	fi

tools:
	@./build/tools.sh $(GOTOOLS)

savedeps: checks
	@godep save -v ./...

restoredeps: checks
	@godep restore -v

.PHONY: build cov checks test cover format vet tools dist
