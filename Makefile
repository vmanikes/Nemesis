# TODO Add lambda builder
test:
	cd lambda && mkdir -p cover && CGO_ENABLED=0 go test -v $(go list ./... | grep -v vendor/) -coverprofile=cover/cover.out ./...

build:
	cd lambda && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o main main.go && zip kinesis_scaling.zip main && rm main