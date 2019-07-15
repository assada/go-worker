APP_NAME = worker

SHELL ?= /bin/bash
ARGS = $(filter-out $@,$(MAKECMDGOALS))

APP_PATH = $(pwd)

IMAGE_TAG = latest
IMAGE_NAME = assada/worker

BUILD_ID ?= $(shell /bin/date "+%Y%m%d-%H%M%S")

.SILENT: ;               # no need for @
.ONESHELL: ;             # recipes execute in same shell
.NOTPARALLEL: ;          # wait for this target to finish
.EXPORT_ALL_VARIABLES: ; # send all vars to shell
Makefile: ;              # skip prerequisite discovery

# Run make help by default
.DEFAULT_GOAL = help

ifneq ("$(wildcard ./VERSION)","")
VERSION ?= $(shell cat ./VERSION | head -n 1)
else
VERSION ?= 0.1.0
endif

BUILD_FILE = build/${APP_NAME}_build_${VERSION}_${BUILD_ID}

%.env:
	cp $@.dist $@

# Public targets
.PHONY: .title
.title:
	printf "\n                \033[95m%s: v%s\033[0m\n" "$(APP_NAME)" $(VERSION)

.PHONY: docker-build
docker-build: ## Build docker image with application
	docker build \
    		--build-arg VERSION=$(VERSION) \
    		--build-arg APP_NAME=$(APP_NAME) \
    		--build-arg BUILD_ID=$(BUILD_ID) \
    		--build-arg APP_PATH=$(make build) \
    		-t $(IMAGE_NAME):$(IMAGE_TAG) \
    		--no-cache \
    		--force-rm .

.PHONY: build
build: ## Bild application
	go build -o ${BUILD_FILE}
	chmod a+x ${BUILD_FILE}
	echo ${BUILD_FILE}

.PHONY: help
help: .title ## Show this help and exit (default target)
	echo ''
	printf "                %s: \033[94m%s\033[0m \033[90m[%s] [%s]\033[0m\n" "Usage" "make" "target" "ENV_VARIABLE=ENV_VALUE ..."
	echo ''
	echo '                Available targets:'
	# Print all commands, which have '##' comments right of it's name.
	# Commands gives from all Makefiles included in project.
	# Sorted in alphabetical order.
	echo '                ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━'
	grep -hE '^[a-zA-Z. 0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		 awk 'BEGIN {FS = ":.*?## " }; {printf "\033[36m%+15s\033[0m: %s\n", $$1, $$2}'
	echo '                ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━'
	echo ''

%:
	@: