
.PHONY: test
test: 
	mkdir -p build
	go test ./... -v -race -coverprofile=build/coverage.out

.PHONY:
test-update-snapshot:
	mkdir -p build
	go test ./... -v -race -coverprofile=build/coverage.out -update