#!/bin/bash
export GO111MODULE=on && \
export PATH="$PATH:$(go env GOPATH)/bin" && \

UNAME=$(uname)
echo $UNAME
if [ "$UNAME" == "Linux" ] ; then
	sudo apt-get install make &&
	sudo apt-get install -y protobuf-compiler &&
	sudo apt  install golang-goprotobuf-dev
elif [ "$UNAME" == "Darwin" ] ; then
	brew install make
	brew install protobuf
fi


go get -u google.golang.org/grpc &&
go get -u github.com/golang/protobuf/{proto,protoc-gen-go} &&
go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc &&
go get -u github.com/gin-gonic/gin &&
go get -u golang.org/x/crypto/bcrypt &&
go get -u github.com/golang-jwt/jwt@v3.2.0
