# Test Plan: Machine API Operator Component Credential Integration

**Story:** #20 - Machine API Operator Component Credential Integration  
**Epic:** #14 - vSphere multi-account credential management  
**Dependency:** #19 - CCO Detects and Provisions Component Credentials

## Overview

This test plan validates that the Machine API Operator correctly integrates with component-specific credentials provisioned by the Cloud Credential Operator (CCO). The operator must read credentials from the openshift-machine-api namespace, support multi-vCenter deployments, validate vSphere privileges, and handle credential rotation gracefully.

## Test Scope

### In Scope
- Reading vsphere-machine-api-creds from openshift-machine-api namespace
- FQDN-based credential lookup for multi-vCenter support
- Validation of 35 required vSphere privileges
- Error reporting to cluster operator status
- Machine lifecycle operations with component credentials
- Graceful credential rotation without downtime
- Multi-vCenter credential isolation

### Out of Scope
- CCO credential provisioning logic (covered by Story #19)
- Installer credential setup (covered by other stories)
- Storage operator integration
- Cloud controller manager integration

## Test Categories

### 1. Credential Reading
**File:** credential_reader_test.go  
**Objective:** Verify the operator reads credentials from the correct namespace and secret.

### 2. vCenter Lookup
**File:** vcenter_lookup_test.go  
**Objective:** Verify correct credential selection based on vCenter FQDN.

### 3. Privilege Validation
**File:** privilege_validator_test.go  
**Objective:** Validate all 35 required vSphere privileges before machine operations.

### 4. Status Reporting
**File:** status_reporter_test.go  
**Objective:** Verify validation errors appear in cluster operator status with clear messaging.

### 5. Machine Lifecycle
**File:** machine_lifecycle_test.go  
**Objective:** Verify machine operations succeed using component credentials.

### 6. Credential Rotation
**File:** credential_rotation_test.go  
**Objective:** Verify graceful credential rotation without downtime.

### 7. Multi-vCenter Isolation
**File:** multi_vcenter_isolation_test.go  
**Objective:** Verify credentials cannot cross vCenter boundaries.

## Success Criteria

- ✅ All acceptance criteria have corresponding test cases
- ✅ Edge cases covered (missing credentials, invalid privileges, rotation failures)
- ✅ Tests are deterministic and reproducible
- ✅ Tests run in CI pipeline
- ✅ Unit tests: >80% code coverage
- ✅ All tests pass consistently

## Test Execution

```bash
# Run all tests
cd projects/machine-api-operator
go test ./tests/... -v

# Run specific test category
go test ./tests/credential_reader_test.go -v
go test ./tests/privilege_validator_test.go -v
```

## Dependencies

- Story #19 (CCO credential provisioning) must be complete
- vSphere test environment with multiple vCenter instances
- Test accounts with varying privilege levels
- OpenShift cluster for integration/e2e tests
