v1.3.0
------
- Breaking change: root object moved to functional options parametrization
- Added support for graphql-ws (graphql-transport-ws subprotocol)
- Ensured only pre-execution operation errors are returned as `error` type per apollows spec
- Fixed incorrect OnConnect/OnOperation callback sequence

v1.2.3
------
- Added OnDisconnect handler without respnsibility to handle error, callback sequence diagram

v1.2.2
------
- Correct termination request handling

v1.2.1
------
- Fixes, clarifications for websocket request teardown sequence

- Added CHANGELOG.md

- Added READMEs to examples

- Updated LICENSE year

v1.0.0-v1.2.0
------
Major refactor, cleaned up implementation
Complete test coverage, versioned package scheme

v0.0.1-v0.5.0
---
Initial implemnetation
