# Regression Test Report

## Date: 2026-06-08

## Build Status: ✅ ALL PASS

| Component | Status | Binary Size |
|-----------|--------|-------------|
| api-gateway | ✅ | 15MB |
| cubeconsole | ✅ | 7.5MB |
| version-tool | ✅ | 1.9MB |
| cubesandbox-operator | ✅ | 11MB |
| instance-controller | ✅ | 8MB |

## Test Results: ✅ ALL PASS

| Test Suite | Count | Status |
|-----------|-------|--------|
| Script validation | 8/8 | ✅ |
| L0 API integration | 14/14 | ✅ |
| Binary verification | 5/5 | ✅ |

## Key Fixes Applied
- Fixed api-gateway createInstance to pass full request body (not just Config)
- Fixed instance-controller stub compilation (no K8s deps)
- Fixed cubeconsole output format (fmt.Sprintf for mixed types)
- Fixed L1 AppArmor heredoc syntax

## Environment
- OS: Ubuntu 24.04 LTS
- Go: 1.22.2
- Incus: Running (unix socket)
- Network: Restricted (offline build mode)

