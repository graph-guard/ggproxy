# Tests

Directory `tests` must contain all declarative test setups where each setup directory must have the prefix `setup_` followed by the name of the setup.
The setup directory must contain the full server configuration and test declarations which must have the prefix `test_` followed by the name of the test and the `.yaml` file extension.

A test defines the clients inputs and expectations:
- `client.input.method`
- `client.input.endpoint`
- `client.input.body`
- `client.expect-response.status`
- `client.expect-response.headers`
- `client.expect-response.body`

...and optionally, the destination server's outputs and expectations:
- `destination.expect-forwarded.headers`
- `destination.expect-forwarded.body`
- `destination.response.status`
- `destination.response.headers`
- `destination.response.body`

An example test structure:

```
tests
├─ setup_A
│   ├─ config.yaml
│   ├─ services_enabled
│   │   └─ service_1
│   │       ├─ config.yaml
│   │       └─ templates_enabled
│   │           └─ template_1.gqt
│   └─ test_1.yaml
└─ setup_B
    ├─ config.yaml
    ├─ services_enabled
    │   └─ service_1
    │       ├─ config.yaml
    │       └─ templates_enabled
    │           └─ template_1.gqt
    └─ test_1.yaml
```

An example test file:
```yaml
client:
  input:
    method: POST
    endpoint: "/service_a"
    body: >
      this will be sent to ggproxy
  expect-response:
    status: 200
    headers:
      Content-Length: ^74$
      Content-Type: ^text/plain; charset=utf-8$
      Server: "fasthttp"
      Date: .
      X-Custom-Header: value
    body: >
      this is what we expect to get from ggproxy

destination:
  expect-forwarded:
    headers:
      Host: ^http://localhost:8081/service_a$
      Content-Length: ^103$
      Content-Type: ^application/json$
      User-Agent: "fasthttp"
      Date: .
    body: >
      this is what we expect to receive
      on the destination server
  response:
    status: 200
    headers:
      X-Custom-Header: value
    body: >
      this is what the destination server
      will respond with
```