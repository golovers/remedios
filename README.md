# remedios

Remedios is a mock-live HTTP server with less than 200 lines of code - which can be used for mocking/making fake http services for testing purposes.

## Run
 
1. Update mock config in `remedios.json`
2. Run
```
cd ~/go/src/github.com/golovers/remedios
go run main.go
```

**Note**: Remedios will automatically reload configuration file whenever there is a changed in the file, hence you don't need to restart the server to make new configurations effect.

## Config

Below is an example of the configuration file:
```json
{
    "endpoints": [
        {
            "path":"/api/v1/users",
            "method": "GET",
            "cases" :[
                {
                    "response":{
                        "status": 200,
                        "body":[
                            {
                                "name": "jack"
                            },
                            {
                                "name": "jack sparrow"
                            }
                        ]
                    }
                }
            ]
        },
        {
            "path": "/api/v1/users",
            "method": "POST",
            "cases":[
                {
                    "request": {
                        "body": {
                            "name":"jack sparrow"
                        }
                    },
                    "response":{
                        "status": 201,
                        "body": {
                            "id": "jack-sparrow"
                        }
                    }
                },
                {
                    "request": {
                        "body": {
                            "name":"jack"
                        }
                    },
                    "response":{
                        "status": 500
                    }
                },
                {
                    "response":{
                        "status": 401,
                        "body": {
                           "error":"Unauthorized"
                        }
                    }
                }
            ]
        }
    ]
}
```

## License

This project is under MIT license.