all:
	protoc --go_out=. -I simple --go-grpc_out=requireUnimplementedServers=false:. simple/simple.proto 
	mv github.com/tuxcanfly/loggy/simple/simple_grpc.pb.go simple/
	mv github.com/tuxcanfly/loggy/simple/simple.pb.go simple/
	go build -o client.exe ./client
	go build -o server.exe ./server
	rm -rf github.com

clean:
	rm -rf github.com simple/simple.pb.go simple/simple_grpc.pb.go 
