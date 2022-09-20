## Builder Image
FROM golang:latest as builder

ENV PATH=$PATH:$GOPATH/bin

RUN apt-get update && apt-get install -y zip && \
    apt-get install -y make && apt-get install -y protobuf-compiler

RUN mkdir -p /go/src/loggy
WORKDIR /go/src/loggy
COPY ./go.mod /go/src/loggy
COPY ./go.sum /go/src/loggy

RUN go get -u google.golang.org/grpc
RUN go get -u github.com/golang/protobuf/proto
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

RUN mv go.mod go.mod.bak
RUN mv go.sum go.sum.bak

COPY . /go/src/loggy

RUN mv go.mod.bak go.mod
RUN mv go.sum.bak go.sum

RUN protoc --go_out=. -I loggy --go-grpc_out=require_unimplemented_servers=false:. loggy/loggy.proto

RUN go build -o loggy.exe ./cmd/loggy
RUN	go build -o user.exe ./cmd/user

## Loggy server image
FROM alpine:latest as loggy

RUN mkdir -p /go/src/loggy/loggy
WORKDIR /go/src/loggy

COPY --from=builder /go/src/loggy/github.com/loggysh/loggy/loggy/loggy_grpc.pb.go loggy/
COPY --from=builder /go/src/loggy/github.com/loggysh/loggy/loggy/loggy.pb.go loggy/

COPY --from=builder /go/src/loggy/loggy.exe .
CMD ./loggy.exe
