
<a name="0.4.0"></a>
## [0.4.0](https://github.com/graph-guard/ggproxy/compare/0.3.0...0.4.0)

### Chore

* Add 0.3.0 changelog
* **changelog:** Update changelog
* **changelog:** Update changelog
* **changelog:** Update changelog

### Refactor

* **engine:** Engine tests enhancement ([#16](https://github.com/graph-guard/ggproxy/issues/16))


<a name="0.3.0"></a>
## [0.3.0](https://github.com/graph-guard/ggproxy/compare/0.2.0...0.3.0)

### Chore

* Add changelog generated with git-chglog

### Feat

* Ease int usage, change uint16 to int
* Add combinations ([#11](https://github.com/graph-guard/ggproxy/issues/11))

### Fix

* Fix comparison of byte slices
* Fix lvs failure on empty license
* Reset segmented array index counter
* Accept enum values
* Fix broken pipe handling


<a name="0.2.0"></a>
## [0.2.0](https://github.com/graph-guard/ggproxy/compare/0.1.0...0.2.0)

### Feat

* Add custom panic message on missing public key
* Add inline fragments suppot for rmap engine ([#8](https://github.com/graph-guard/ggproxy/issues/8))

### Fix

* Fix collision of fragments and selection fields with same name

### Refactor

* Change rmap engine structure ([#9](https://github.com/graph-guard/ggproxy/issues/9))

### Test

* Engine test/bench suites improvement ([#7](https://github.com/graph-guard/ggproxy/issues/7))


<a name="0.1.0"></a>
## 0.1.0

### Chore

* Fix pquery naming in tests
* Change repository name
* Delete obsolete code
* Remove obsolete engine implementation
* Add common ignored files
* Migrate gguard to this repository

### Docs

* Complete the tests README guide

### Feat

* Add 'null' support
* Add processing of enums
* Prepare to Beta release
* Use LVS validation and rename 'licence' to 'license'
* Add licence key environment variable
* Add support for basic auth in API server
* Add service and template statistics
* Add ggproxy GraphQL API
* Make maximum request body size configurable
* Add support for option forward_reduced
* Implement ggproxy stop CLI command
* Add debug and command servers
* Add CLI
* Structure config parser errors
* Add server configuration file
* Add initial server ([#2](https://github.com/graph-guard/ggproxy/issues/2))
* Add config parser

### Fix

* Fix go.mod
* Remove superfluous log
* Remove unused config option
* Fix lambda capturing and template lists
* Normalize CLI message format
* Add missing line breaks to CLI messages
* Ignore EOF while reading command socket
* Use buffered channel for signal notifier
* Remove unnecessary imports from go mod
* Validate service id

### Refactor

* Refactor LVS ([#4](https://github.com/graph-guard/ggproxy/issues/4))
* Get rid of the matcher interface
* Simplify code

### Sec

* Variable Bomb Security Patch ([#6](https://github.com/graph-guard/ggproxy/issues/6))

### Test

* Add declarative server tests

### BREAKING CHANGE


Rename "Server" to "Ingress(Server)" and "ServerDebug" to "API(Server)".
Change `config.yaml` structure.
Provide sublogger to the fasthttp server.

