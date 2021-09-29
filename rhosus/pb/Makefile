compile:
	protoc --gofast_out=plugins=grpc:node --gofast_opt=paths=source_relative node.proto
	protoc --gofast_out=plugins=grpc:file_manager --gofast_opt=paths=source_relative file_manager.proto
	protoc --gofast_out=plugins=grpc:registry --gofast_opt=paths=source_relative registry.proto