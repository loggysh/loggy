loggy
=====

loggy implements a simple streaming grpc service.


client
======

client needs to generate a protobuf from `simple/simple.proto`

Create a Client from `SimpleService`, generate a stream from `SimpleRPC`, and
send a `SimpleData` on the stream.
