#!/usr/bin/bash
go get github.com/gin-gonic/gin && \
go get -u google.golang.org/grpc && \
go get -u github.com/golang/protobuf/{proto,protoc-gen-go} && \
go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc && \
go get "golang.org/x/crypto/bcrypt" && \
go get github.com/dgrijalva/jwt-go
