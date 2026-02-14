build:
	go build -o server ./cmd/server

run:
	go run ./cmd/server

test:
	go test -v ./...

clean:
	rm -f server ssh_host_key ssh_host_key.pub id_ed25519 id_ed25519.pub
