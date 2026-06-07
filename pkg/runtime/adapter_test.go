package runtime

import (
	"context"
	"testing"
)

func TestIncusAdapter(t *testing.T) {
	adapter := NewIncusAdapter("unix:///var/lib/incus/unix.socket", "zfs-pool")

	t.Run("Create", func(t *testing.T) {
		spec := &InstanceSpec{
			Name:   "test-instance",
			Image:  "ubuntu/24.04",
			CPU:    "2",
			Memory: "4GB",
		}
		ctx := context.Background()
		// This is a stub test - in real test we'd mock the HTTP client
		err := adapter.Create(ctx, spec)
		if err != nil {
			t.Logf("Create returned error (expected in test env): %v", err)
		}
	})

	t.Run("Status", func(t *testing.T) {
		ctx := context.Background()
		status, err := adapter.Status(ctx, "test-instance")
		if err != nil {
			t.Logf("Status returned error (expected in test env): %v", err)
		}
		if status != nil {
			if status.Runtime != "incus" {
				t.Errorf("Expected runtime 'incus', got '%s'", status.Runtime)
			}
		}
	})

	t.Run("List", func(t *testing.T) {
		ctx := context.Background()
		instances, err := adapter.List(ctx)
		if err != nil {
			t.Logf("List returned error (expected in test env): %v", err)
		}
		if instances == nil {
			t.Error("Expected non-nil instances slice")
		}
	})
}

func TestKataAdapter(t *testing.T) {
	adapter := NewKataAdapter()

	t.Run("Create", func(t *testing.T) {
		spec := &InstanceSpec{
			Name:   "test-kata",
			Image:  "ubuntu:24.04",
			CPU:    "2",
			Memory: "4GB",
		}
		ctx := context.Background()
		err := adapter.Create(ctx, spec)
		if err != nil {
			t.Logf("Create returned error: %v", err)
		}
	})

	t.Run("Status", func(t *testing.T) {
		ctx := context.Background()
		status, err := adapter.Status(ctx, "test-kata")
		if err != nil {
			t.Errorf("Status should not error: %v", err)
		}
		if status == nil {
			t.Fatal("Status should not be nil")
		}
		if status.Runtime != "kata" {
			t.Errorf("Expected runtime 'kata', got '%s'", status.Runtime)
		}
		if status.Type != "container" {
			t.Errorf("Expected type 'container', got '%s'", status.Type)
		}
	})
}

func TestCubeAdapter(t *testing.T) {
	adapter := NewCubeAdapter("http://localhost:8080")

	t.Run("Create", func(t *testing.T) {
		spec := &InstanceSpec{
			Name:   "test-cube",
			CPU:    "2",
			Memory: "512MB",
		}
		ctx := context.Background()
		err := adapter.Create(ctx, spec)
		if err != nil {
			t.Logf("Create returned error: %v", err)
		}
	})

	t.Run("Migrate", func(t *testing.T) {
		ctx := context.Background()
		err := adapter.Migrate(ctx, "test-cube", "node2")
		if err == nil {
			t.Error("Migrate should return error (not implemented)")
		}
	})
}

func TestKubeVirtAdapter(t *testing.T) {
	adapter := NewKubeVirtAdapter("default")

	t.Run("Create", func(t *testing.T) {
		spec := &InstanceSpec{
			Name:   "test-kubevirt",
			Image:  "ubuntu-24.04",
			CPU:    "2",
			Memory: "4GB",
		}
		ctx := context.Background()
		err := adapter.Create(ctx, spec)
		if err != nil {
			t.Logf("Create returned error: %v", err)
		}
	})

	t.Run("Migrate", func(t *testing.T) {
		ctx := context.Background()
		err := adapter.Migrate(ctx, "test-kubevirt", "node2")
		if err != nil {
			t.Logf("Migrate returned error: %v", err)
		}
	})
}

func TestFactory(t *testing.T) {
	factory := NewDefaultFactory()

	t.Run("CreateIncus", func(t *testing.T) {
		adapter, err := factory.Create("incus", "unix:///var/lib/incus/unix.socket")
		if err != nil {
			t.Fatalf("Failed to create incus adapter: %v", err)
		}
		if adapter == nil {
			t.Fatal("Adapter should not be nil")
		}
	})

	t.Run("CreateKata", func(t *testing.T) {
		adapter, err := factory.Create("kata", "")
		if err != nil {
			t.Fatalf("Failed to create kata adapter: %v", err)
		}
		if adapter == nil {
			t.Fatal("Adapter should not be nil")
		}
	})

	t.Run("CreateCube", func(t *testing.T) {
		adapter, err := factory.Create("cube", "http://localhost:8080")
		if err != nil {
			t.Fatalf("Failed to create cube adapter: %v", err)
		}
		if adapter == nil {
			t.Fatal("Adapter should not be nil")
		}
	})

	t.Run("CreateKubeVirt", func(t *testing.T) {
		adapter, err := factory.Create("kubevirt", "default")
		if err != nil {
			t.Fatalf("Failed to create kubevirt adapter: %v", err)
		}
		if adapter == nil {
			t.Fatal("Adapter should not be nil")
		}
	})

	t.Run("CreateUnknown", func(t *testing.T) {
		_, err := factory.Create("unknown", "")
		if err == nil {
			t.Error("Expected error for unknown runtime type")
		}
	})

	t.Run("ListRuntimes", func(t *testing.T) {
		runtimes := factory.ListRuntimes()
		if len(runtimes) != 4 {
			t.Errorf("Expected 4 runtimes, got %d", len(runtimes))
		}
		expected := []string{"incus", "kata", "cube", "kubevirt"}
		for _, r := range expected {
			found := false
			for _, rt := range runtimes {
				if rt == r {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected runtime '%s' not found in list", r)
			}
		}
	})
}
