# calc_micro:
HTTP REST Api Distributed Calculator in Go

# How to start:
You can simply run it by
```
go run cmd/orchestrator/main.go
go run cmd/agent/main.go
```
or build
```
go build -o OUTPUT_ORCHESTRATOR_BINARY cmd/orchestrator/main.go
go build -o OUTPUT_AGENT_BINARY cmd/agent/main.go
```

By default orchestrator listens on localhost:8080
You can set listening port&address by using startup flags
```
go run cmd/orchestrator/main.go -ip='*' -port='1234'
./orchestrator -ip='*' -port='1234'
```

For agent you can set base-url (by default: http://localhost:8080)
```
go run cmd/agent/main.go -base-url="http://1.2.3.4:12345"
agent -base-url="http://1.2.3.4:12345"
```

and after that - run as standalone application

# Examples:
   ## api/v1/calculate
   ### Wrong HTTP Method.
   Expect code 500 and {"error": "Invalid request"}
   ```http
   GET http://localhost/api/v1/calculate
   ```
   ### OK Expression.
   Expect code 201 and {"id": "0A2DDEF9-F67C-6899-5F72-25639EEBD08F"}
   ```http
   POST http://localhost/api/v1/calculate
   Content-Type: application/json

   {
     "expression": "2+2*2"
   }
   ```
   ### Empty or Incorrect expression.
   Expect code 422 and {"error": "Invalid request body"}
   ```http
   POST http://localhost/api/v1/calculate
   Content-Type: application/json

   {
     "expression": ""
   }
   ```

## api/v1/expressions
### OK Expression.
    Expect code 200 and response
    {
        "expressions": [
            {
            "id": "0A2DDEF9-F67C-6899-5F72-25639EEBD08F",
            "result": 0,
            "status": "pending"
            }
        ]
    }

    ```http
    GET http://localhost/api/v1/expressions
    ```
You can do a simple test with curl like
```
curl --location 'localhost/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
  "expression": "2+2*2"
}'
```