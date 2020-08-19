loggy
=====

loggy implements a simple streaming grpc service.


install
=======

    go get -u google.golang.org/grpc

    go get -u github.com/golang/protobuf/{proto,protoc-gen-go}

    go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc

    install protobuf-dev

demo
====

make builds all the files, then run server.exe and client.exe in different terminals:

    make

    ./server.exe
    2020/08/19 00:43:36 Listening on tcp://localhost:50111
    2020/08/19 00:43:43 Started stream
    2020/08/19 00:43:44 Received value
    2020/08/19 00:43:44 Got 2020-08-19T00:43:44.474707353+05:30
    2020/08/19 00:43:45 Received value


    ./client.exe
    2020/08/19 00:43:43 Sleeping...
    2020/08/19 00:43:44 msg: "2020-08-19T00:43:44.474707353+05:30"
    2020/08/19 00:43:44 Sleeping...
    2020/08/19 00:43:45 msg: "2020-08-19T00:43:45.47600339+05:30"
    2020/08/19 00:43:45 Sleeping...



android client
===============

client needs to generate a protobuf from `simple/simple.proto`

Create a Client from `SimpleService`, generate a stream from `SimpleRPC`, and
send a `SimpleData` on the stream.
