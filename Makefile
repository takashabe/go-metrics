SUBPACKAGES := $(shell go list ./...)

test:
	$(SHOW_ENV)
	go test -v $(SUBPACKAGES)

vet:
	go vet $(SUBPACKAGES)

lint:
	golint $(SUBPACKAGES)
