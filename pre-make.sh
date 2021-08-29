#!/bin/bash

UNAME=$(uname)
echo $UNAME
if [ "$UNAME" == "Linux" ] ; then
	sudo apt-get install make &&
	sudo apt-get install -y protobuf-compiler
elif [ "$UNAME" == "Darwin" ] ; then
	brew install make
	brew install protobuf
fi

go get -u google.golang.org/grpc && \
go get -u google.golang.org/protobuf/cmd/protoc-gen-go && \
go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc && \
go get -u github.com/golang/protobuf/proto && \
go get -u github.com/gin-gonic/gin && \
go get -u golang.org/x/crypto/bcrypt && \
go get -u github.com/golang-jwt/jwt/v4
