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
