compile:
	protoc --gofast_out=plugins=grpc:fs_pb --gofast_opt=paths=source_relative fs.proto
	protoc --gofast_out=plugins=grpc:transport --gofast_opt=paths=source_relative transport.proto
	protoc --gofast_out=plugins=grpc:registry --gofast_opt=paths=source_relative registry.proto
	protoc --gofast_out=plugins=grpc:control --gofast_opt=paths=source_relative control.proto