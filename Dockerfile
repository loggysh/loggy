FROM golang:latest

ENV GO111MODULE=on

RUN go get -u google.golang.org/grpc && \
    go get -u github.com/golang/protobuf/proto && \
    go get -u github.com/golang/protobuf/protoc-gen-go && \
    go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc


RUN apt-get update && apt-get install -y zip && \
    mkdir /opt/protoc && cd /opt/protoc && wget https://github.com/protocolbuffers/protobuf/releases/download/v3.7.0/protoc-3.7.0-linux-x86_64.zip && \
    unzip protoc-3.7.0-linux-x86_64.zip


ENV PATH=$PATH:$GOPATH/bin:/opt/protoc/bin

RUN mkdir -p /go/src/loggy

COPY . /go/src/loggy

RUN cd /go/src/loggy && make

ENTRYPOINT cd /go/src/loggy && ./loggy.exe





