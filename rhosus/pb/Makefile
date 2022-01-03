compile:
	protoc --gofast_out=plugins=grpc:fs --gofast_opt=paths=source_relative fs.proto
	protoc --gofast_out=plugins=grpc:journal --gofast_opt=paths=source_relative journal.proto
	protoc --gofast_out=plugins=grpc:transmission --gofast_opt=paths=source_relative transmission.proto
	protoc --gofast_out=plugins=grpc:file_manager --gofast_opt=paths=source_relative file_manager.proto
	protoc --gofast_out=plugins=grpc:registry --gofast_opt=paths=source_relative registry.proto