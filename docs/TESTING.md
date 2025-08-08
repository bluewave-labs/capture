# Testing Process for Capture

This document provides an overview of the testing strategies used in the Capture, including unit and integration testing.

## Architecture Testing

Architecture testing ensures that the system's architecture meets the specified requirements and is robust against potential issues. This includes verifying the design patterns, module interactions, and overall system structure.

You can see the `test/arch_test.go` file for the architecture tests. It's powered by the [go-arctest](https://github.com/mstrYoda/go-arctest) package, which provides tools for testing the architecture of Go applications.

We don't have specific command for architecture tests, but `just unit-test` command runs all unit tests in the codebase including architecture tests.

### Rules

1. CMD should not depend on handlers.

## Unit Testing

Unit tests are not os/arch specific, they can be run on any platform where the codebase is runnable.

You can search for `test/*_test.go` files in the `test` directory to find unit tests.

`just unit-test` command runs all unit tests in the codebase.

## Integration Testing

Integration tests are designed to test the interactions between different components of the system. They ensure that the components work together as expected and can handle real-world scenarios.

You can see the `test/integration/*_test.go` files for integration tests.

`just integration-test` command runs all integration tests in the codebase.

## OpenAPI Contract Testing

OpenAPI contract testing ensures that the API adheres to the defined OpenAPI specifications. This is crucial for maintaining consistency and reliability in API interactions.

You can see the `schemathesis.toml` file for OpenAPI contract tests.

`just openapi-contract-test` command runs OpenAPI contract tests using [Schemathesis](https://schemathesis.readthedocs.io/).
