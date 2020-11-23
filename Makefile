.PHONY: clean

encoder: cmd/encoder/main.go
	go build -o encoder cmd/encoder/main.go

decoder: cmd/decoder/main.go
	go build -o decoder cmd/decoder/main.go
clean:
	rm -f encoder decoder	
