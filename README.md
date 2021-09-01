loggy
=====

loggy implements a simple streaming grpc service.


install
=======
```
make
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

user
====

Sign up
```
curl -X POST -H "Content-Type: application/json" -d '{ "Email":"a@a.com", "Password":"a"}'  http://localhost:8080/api/public/signup
```

Response
```
{"created_at":"2021-09-01T18:59:09.10998+05:30","updated_at":"2021-09-01T18:59:09.10998+05:30","deleted_at":null,"ID":"2eba827d0fcb43c491c376f8e6ea95c5","name":"","email":"a@a.com","password":"$2a$14$WpD40lIz973owdfENHPUseNfnO9lA1/dsLpkeeb7rtGWvntv7o/yS","api_key":"c32951dc491c4f05872747d1cc8a18cc"}%
```

Login
```
curl -X POST -H "Content-Type: application/json" -d '{ "Email":"a@a.com", "Password":"a"}'  http://localhost:8080/api/public/login
```

Response
```
{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6ImFAYS5jb20iLCJleHAiOjE2MzA1ODkzNTMsImlzcyI6IkF1dGhTZXJ2aWNlIn0.rKmAL8LqNEirIUrjlHQuTaFu7uqkvbIFyFibqqN-95s","user_id":"2eba827d0fcb43c491c376f8e6ea95c5"}
```

Test With API key
```
go run scripts/apikey/main.go -apikey=c32951dc491c4f05872747d1cc8a18cc
```


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

