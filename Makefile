.PHONY: clean coverage


execs: encoder decoder

encoder: cmd/encoder/main.go
	go build -o encoder cmd/encoder/main.go

decoder: cmd/decoder/main.go
	go build -o decoder cmd/decoder/main.go

coverage:
	go test -cover -coverprofile testcoverage.out -coverpkg=alexhalogen/rsfileprotect/internal/... ./test/...
	go tool cover -html=testcoverage.out -o coverage_report.html
clean:
	rm -f encoder decoder	testcoverage.out coverage_report.html
