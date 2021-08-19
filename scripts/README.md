
- Get authorization and user id from web or database
- Use url i.e localhost:8123, staging.loggy.sh:8123

### Run client sample
```go run scripts/client/main.go -authorization=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6ImFAYS5jb20iLCJleHAiOjE2Mjg3NzI2NzQsImlzcyI6IkF1dGhTZXJ2aWNlIn0.uadOcylJb6EKy6LLLq1PK53EnelpnJy57FoNPRYD6JA -userid=49eaeef551c94deaa22eaf003f76c5bd -url=localhost:8123```

### Run live sample
```go run scripts/live/main.go -authorization=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6ImFAYS5jb20iLCJleHAiOjE2Mjg3NzI2NzQsImlzcyI6IkF1dGhTZXJ2aWNlIn0.uadOcylJb6EKy6LLLq1PK53EnelpnJy57FoNPRYD6JA -userid=49eaeef551c94deaa22eaf003f76c5bd -url=localhost:8123 -sessionid=1```
