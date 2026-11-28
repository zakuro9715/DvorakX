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
	make genhtml
test:
	go test ./...
test.win:
	GOOS=windows go test ./...
test.win.cov:
	GOOS=windows go test ./... --coverprofile=coverage.out
	make genhtml
genhtml:
	go run ./cmd/tools/gocov2lcov coverage.out lcov.info
	genhtml lcov.info --o coverage
open-cov:
	open coverage/index.html
