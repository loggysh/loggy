export GO111MODULE=on
export PATH := $(PATH):$(HOME)/go/bin

all:
	echo "Fetching and installing dependencies..."
	./pre-make.sh
	echo "Done fetching and installing dependencies."
	echo "Building proto files"
	protoc --go_out=. -I loggy --go-grpc_out=require_unimplemented_servers=false:. loggy/loggy.proto
	mv github.com/tuxcanfly/loggy/loggy/loggy_grpc.pb.go loggy/
	mv github.com/tuxcanfly/loggy/loggy/loggy.pb.go loggy/
	echo "Done building proto files for grpc"
	go build -o loggy.exe ./cmd/loggy
	go build -o user.exe ./cmd/user
	rm -rf github.com
clean:
	rm -rf github.com loggy/loggy.pb.go loggy/loggy_grpc.pb.go *.exe test.db logs loggy.index
