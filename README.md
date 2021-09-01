loggy
=====

loggy implements a simple streaming grpc service.


install
=======
```
$ ./pre-make.sh
```
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

client needs to generate a protobuf from `loggy/loggy.proto`


Web
- Login - Access Token
- Rest API - Using Gin
- db as users

GRPC - Web
- Access Token and User ID in the metadata

GRPC Android
- Client ID to validate (constant)
- Associate client id with app id (web integration)

Server
- Access Token review
- Secret to generate access token is hard coded

- Client ID generation (short string)
- Should be able the user id

