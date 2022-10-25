FROM golang:1.17

ENV GO111MODULE=on

RUN go install github.com/golang/protobuf/protoc-gen-go@v1.5.2 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

RUN apt-get update && apt-get install -y zip && apt install -y protobuf-compiler

RUN mkdir -p /go/src/loggy

WORKDIR /go/src/loggy

COPY . /go/src/loggy

RUN make

CMD ./loggy.exe
