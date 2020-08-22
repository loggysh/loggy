all:
	protoc --go_out=. -I loggy --go-grpc_out=requireUnimplementedServers=false:. loggy/loggy.proto 
	mv github.com/tuxcanfly/loggy/loggy/loggy_grpc.pb.go loggy/
	mv github.com/tuxcanfly/loggy/loggy/loggy.pb.go loggy/
	go build -o client.exe ./client
	go build -o server.exe ./server
	go build -o logger.exe ./logger
	rm -rf github.com

clean:
	rm -rf github.com loggy/loggy.pb.go loggy/loggy_grpc.pb.go *.exe test.db
