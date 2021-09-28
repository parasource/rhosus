compile:
	protoc --go_out=node --go-grpc_out=node --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative node.proto
	protoc --go_out=file_manager --go-grpc_out=file_manager --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative file_manager.proto