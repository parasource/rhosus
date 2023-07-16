all: build

build: build_r build_d build_c

build_r:
	CGO_ENABLED=0 go build -a -installsuffix cgo -o ./bin/rhosusr ./cmd/rhosusr

build_d:
	CGO_ENABLED=0 go build -a -installsuffix cgo -o ./bin/rhosusd ./cmd/rhosusd

build_c:
	CGO_ENABLED=0 go build -a -installsuffix cgo -o ./bin/rhosus ./cmd/rhosus

test:
	go test -v ./...