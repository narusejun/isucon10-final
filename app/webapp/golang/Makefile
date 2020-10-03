PREFIX=$(PWD)
BINDIR=$(PREFIX)/bin

COMPILER=go

GOFILES=$(wildcard ./*.go ./**/*.go)

.PHONY: all
all: clean build ## Cleanup and Build

.PHONY: build
build: $(GOFILES) ## Build executable files
	@GOBIN=$(BINDIR) $(COMPILER) install ./cmd/...  


.PHONY: clean
clean: ## Cleanup files
	@$(RM) -r $(BINDIR)

.PHONY: help
help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: restart	
restart:
	sudo systemctl restart xsuportal-api-golang.service
	sudo systemctl restart xsuportal-web-golang.service
<<<<<<< HEAD
	sudo bash -c ":> /var/log/envoy/access.log"
=======
>>>>>>> 1b7ca7d... makefile
