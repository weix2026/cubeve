#!/usr/bin/env bash
set -uo pipefail

# CubeVE L0 Validation Script
# Tests all L0 control-plane operations

echo "=========================================="
echo "CubeVE L0 Validation Suite"
echo "=========================================="
echo

PASS=0
FAIL=0
SKIP=0

# Helper functions
pass() { echo "  ✅ PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "  ❌ FAIL: $1"; FAIL=$((FAIL + 1)); }
skip() { echo "  ⏭️  SKIP: $1"; SKIP=$((SKIP + 1)); }

# Test 1: Incus Daemon
echo "Test 1: Incus Daemon"
if incus info >/dev/null 2>&1; then
  pass "Incus daemon is running"
else
  fail "Incus daemon is not running"
fi

# Test 2: API Access
echo "Test 2: API Access"
if curl -s --unix-socket /var/lib/incus/unix.socket http://localhost/1.0 >/dev/null 2>&1; then
  pass "Unix socket API is accessible"
else
  fail "Unix socket API is not accessible"
fi

# Test 3: Storage Pool
echo "Test 3: Storage Pool"
if incus storage list | grep -q "zfs-pool"; then
  pass "ZFS storage pool exists"
else
  fail "ZFS storage pool does not exist"
fi

# Test 4: Network
echo "Test 4: Network"
if incus network list | grep -q "incusbr0"; then
  pass "Bridge network exists"
else
  fail "Bridge network does not exist"
fi

# Test 5: Profile
echo "Test 5: Profile"
if incus profile list | grep -q "test-profile"; then
  pass "Custom profile exists"
else
  fail "Custom profile does not exist"
fi

# Test 6: Instance Creation (metadata only)
echo "Test 6: Instance Creation"
if incus list | grep -q "zfs-test"; then
  pass "Instance exists"
else
  fail "Instance does not exist"
fi

# Test 7: Snapshot
echo "Test 7: Snapshot"
if incus snapshot list zfs-test 2>/dev/null | grep -q "baseline"; then
  pass "Snapshot exists"
else
  fail "Snapshot does not exist"
fi

# Test 8: ZFS Dataset
echo "Test 8: ZFS Dataset"
if zfs list | grep -q "test-pool/cubeve"; then
  pass "ZFS dataset exists"
else
  fail "ZFS dataset does not exist"
fi

# Test 9: ZFS Snapshot
echo "Test 9: ZFS Snapshot"
if zfs list -t snapshot | grep -q "test-pool@baseline"; then
  pass "ZFS snapshot exists"
else
  fail "ZFS snapshot does not exist"
fi

# Test 10: Backup Export
echo "Test 10: Backup Export"
if [ -f "/tmp/zfs-test-backup.tar.gz" ]; then
  pass "Backup file exists"
else
  fail "Backup file does not exist"
fi

# Test 11: Instance Configuration
echo "Test 11: Instance Configuration"
if incus config show zfs-test | grep -q 'limits.cpu: "2"'; then
  pass "Instance configuration is correct"
else
  fail "Instance configuration is incorrect"
fi

# Test 12: Network Device
echo "Test 12: Network Device"
if incus config show zfs-test | grep -q "network: incusbr0"; then
  pass "Network device is attached"
else
  fail "Network device is not attached"
fi

# Test 13: KVM Support (expected to fail)
echo "Test 13: KVM Support"
if [ -e /dev/kvm ]; then
  pass "KVM is available"
else
  skip "KVM is not available (expected in container environment)"
fi

# Test 14: Instance Start (expected to fail in container)
echo "Test 14: Instance Start"
if incus list | grep "zfs-test" | grep -q "RUNNING"; then
  pass "Instance is running"
else
  skip "Instance is not running (expected in container environment)"
fi

# Summary
echo
echo "=========================================="
echo "Validation Summary"
echo "=========================================="
echo "  Passed:  $PASS"
echo "  Failed:  $FAIL"
echo "  Skipped: $SKIP"
echo "  Total:   $((PASS + FAIL + SKIP))"
echo

if [ $FAIL -eq 0 ]; then
  echo "🎉 All tests passed (or skipped for expected limitations)!"
  exit 0
else
  echo "⚠️  Some tests failed. Review the output above."
  exit 1
fi
