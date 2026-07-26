# Testing Guide

## Test Types

### Unit and envtest suite

Command:

make test

Includes:

- API and controller package tests
- envtest-based controller tests (without full cluster)
- Coverage output in cover.out

Does not include:

- e2e suite in test/e2e

### End-to-end suite

Command:

make test-e2e

Includes:

- Kind cluster setup
- Controller image build and load
- CRD install and controller deploy
- Runtime validations against a real cluster

## Recommended Local Workflow

1. Run unit and envtest:

make test

2. Run e2e when changing controller logic, manifests, or deployment behavior:

make test-e2e

## Existing Controller Scenarios

Current controller tests cover:

- Successful reconcile path
- ConfigMapWatcher not found
- Referenced ConfigMap not found
- Endpoint unreachable
- Status updates after successful reconcile

## e2e Scope

Current e2e tests validate:

- Controller manager starts
- Metrics endpoint is reachable and returns 200
- ConfigMapWatcher custom resource can be created

## Troubleshooting

### make test fails with envtest assets issues

Run:

make setup-envtest

Then run tests again.

### make test-e2e fails while creating kind cluster

If host uses cgroup v1, keep default KIND_NODE_IMAGE from Makefile.
If host uses cgroup v2, you can override image:

make test-e2e KIND_NODE_IMAGE=kindest/node:v1.36.1

### e2e fails due to missing image

Ensure Docker is running and image build succeeds:

make docker-build IMG=example.com/configmapwatcher:v0.0.1
