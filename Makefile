all:
	protoc -I simple simple/simple.proto --go_out=plugins=grpc:simple
