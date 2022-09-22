FROM golang:latest

ENV GO111MODULE=on

RUN go get -u google.golang.org/grpc && \
    go get -u google.golang.org/protobuf/proto && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc

RUN apt-get update && apt-get install -y zip && apt install -y protobuf-compiler

RUN mkdir -p /go/src/loggy

WORKDIR /go/src/loggy

COPY . /go/src/loggy

RUN make

CMD ./loggy.exe
