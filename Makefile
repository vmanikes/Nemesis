test:
	cd lambda && mkdir -p cover && CGO_ENABLED=0 go test -v $(go list ./... | grep -v vendor/) -coverprofile=cover/cover.out ./... && go tool cover -html=cover/cover.out -o coverage.html

build:
	cd lambda && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o main main.go && cp main ../terraform/main