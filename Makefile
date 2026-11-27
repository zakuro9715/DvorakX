prepare:
	npx skill install
build.win:
	GOOS=windows make build
build:
	go build ./cmd/dvorakz
run:
	./dvorakz
run.win:
	./dvorakz.exe
update:
	go get -u ./... && go mod tidy
lint:
	golangci-lint run
test.cov:
	go test ./... --coverprofile=coverage.out
test:
	go test ./...
test.win:
	GOOS=windows go test ./...
test.win.cov:
	GOOS=windows go test ./... --coverprofile=coverage.out
