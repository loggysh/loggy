all:
	protoc --go_out=. -I loggy --go-grpc_out=requireUnimplementedServers=false:. loggy/loggy.proto 
	mv github.com/tuxcanfly/loggy/loggy/loggy_grpc.pb.go loggy/
	mv github.com/tuxcanfly/loggy/loggy/loggy.pb.go loggy/
	go build -o loggy.exe ./cmd/loggy
	rm -rf github.com

clean:
	rm -rf github.com loggy/loggy.pb.go loggy/loggy_grpc.pb.go *.exe test.db logs
