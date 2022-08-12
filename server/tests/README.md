# Tests

Directory `tests` must contain all declarative test setups where each setup directory must have the prefix `setup_` followed by the name of the setup.
The setup directory must contain the full server configuration and test directories which must have the prefix `test_` followed by the name of the test.

Each test directory must contain the following files:

- `input.txt` describes the request body and headers that are sent to ggproxy.
- `expect_forwarded.txt` describes the request body and headers that are expected to be received by the destination server.
- `response.txt` describes the destination server's response status, headers and body.
- `expect_response.txt` describes the expected final response from ggproxy including status, headers and body.

>NOTE: If `response.txt` and `expect_forwarded.txt` are both missing then the request is expected to not be forwarded, otherwise both must be present.

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
│   └─ test_1
│       ├─ input.txt
│       ├─ expect_forwarded.txt
│       ├─ response.txt
│       └─ expect_response.txt
└─ setup_B
    ├─ config.yaml
    ├─ services_enabled
    │   └─ service_1
    │       ├─ config.yaml
    │       └─ templates_enabled
    │           └─ template_1.gqt
    └─ test_1
        ├─ input.txt
        └─ expect_response.txt
```