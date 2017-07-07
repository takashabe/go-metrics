SUBPACKAGES := $(shell go list ./... | grep -v /vendor/)

test:
	$(SHOW_ENV)
	go test -v $(SUBPACKAGES)

vet:
	go vet $(SUBPACKAGES)
