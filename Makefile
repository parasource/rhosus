build:
	CGO_ENABLED=0 go build -a -installsuffix cgo -o ./bin ./cmd/rhosus
	CGO_ENABLED=0 go build -a -installsuffix cgo -o ./bin ./cmd/rhosusd