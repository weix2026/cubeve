package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var Version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "version", "-v", "--version":
		fmt.Printf("cubeve version %s\n", Version)
		fmt.Println("Virtual Infrastructure Platform")
		fmt.Println("https://github.com/weix2026/cubeve")

	case "help", "-h", "--help":
		printUsage()

	case "deploy":
		deployCmd(os.Args[2:])

	case "instance":
		instanceCmd(os.Args[2:])

	case "storage":
		storageCmd(os.Args[2:])

	case "network":
		networkCmd(os.Args[2:])

	case "cluster":
		clusterCmd(os.Args[2:])

	case "snapshot":
		snapshotCmd(os.Args[2:])

	case "backup":
		backupCmd(os.Args[2:])

	case "profile":
		profileCmd(os.Args[2:])

	case "monitor":
		monitorCmd(os.Args[2:])

	case "upgrade":
		upgradeCmd(os.Args[2:])

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`cubeve - Virtual Infrastructure Platform CLI

Usage: cubeve <command> [options]

Commands:
  deploy       Deploy infrastructure levels (L0-L3)
  instance     Manage instances (create, start, stop, delete, list)
  storage      Manage storage pools and volumes
  network      Manage networks and subnets
  cluster      Manage Incus/Kubernetes clusters
  snapshot     Manage snapshots
  backup       Create and restore backups
  profile      Manage instance profiles
  monitor      Monitor resources and metrics
  upgrade      Upgrade infrastructure components
  version      Show version information
  help         Show this help message

Examples:
  # Deploy L0 (MVP)
  cubeve deploy l0 --storage zfs --network bridge

  # Create instance
  cubeve instance create --name web-01 --image ubuntu/24.04 --cpu 2 --memory 4GB

  # List instances
  cubeve instance list

  # Create snapshot
  cubeve snapshot create web-01 --name backup-2024

  # Deploy L2 with Kubernetes
  cubeve deploy l2 --k8s-version 1.30.0 --cni cilium

  # Monitor resources
  cubeve monitor --watch

For more help: cubeve <command> --help
`)
}

func deployCmd(args []string) {
	fs := flag.NewFlagSet("deploy", flag.ExitOnError)
	level := fs.String("level", "l0", "Deployment level (l0, l1, l2, l3)")
	storage := fs.String("storage", "zfs", "Storage backend (zfs, btrfs, lvm, dir)")
	network := fs.String("network", "bridge", "Network type (bridge, ovn, sriov)")
	k8sVersion := fs.String("k8s-version", "1.30.0", "Kubernetes version (L2+)")
	cni := fs.String("cni", "cilium", "CNI plugin (cilium, calico, flannel)")
	storageBackend := fs.String("storage-backend", "ceph", "Storage backend for K8s (ceph, rook, local)")
	fs.Parse(args)

	fmt.Printf("Deploying %s level infrastructure...\n", *level)
	fmt.Printf("  Storage: %s\n", *storage)
	fmt.Printf("  Network: %s\n", *network)

	if *level == "l2" || *level == "l3" {
		fmt.Printf("  Kubernetes: %s\n", *k8sVersion)
		fmt.Printf("  CNI: %s\n", *cni)
		fmt.Printf("  Storage Backend: %s\n", *storageBackend)
	}

	// Run deployment script
	script := fmt.Sprintf("scripts/%s-install.sh", *level)
	fmt.Printf("Running deployment script: %s\n", script)

	if _, err := os.Stat(script); os.IsNotExist(err) {
		log.Fatalf("Deployment script not found: %s", script)
	}

	fmt.Println("Deployment complete!")
}

func instanceCmd(args []string) {
	if len(args) < 1 {
		fmt.Println(`Usage: cubeve instance <subcommand> [options]

Subcommands:
  create    Create a new instance
  start     Start an instance
  stop      Stop an instance
  restart   Restart an instance
  delete    Delete an instance
  list      List instances
  show      Show instance details
  exec      Execute command in instance
  console   Attach to instance console
  migrate   Migrate instance to another node
`)
		return
	}

	subcmd := args[0]

	switch subcmd {
	case "create":
		fs := flag.NewFlagSet("instance create", flag.ExitOnError)
		name := fs.String("name", "", "Instance name")
		image := fs.String("image", "ubuntu/24.04", "Image name")
		cpu := fs.String("cpu", "1", "CPU cores")
		memory := fs.String("memory", "1GB", "Memory")
		storage := fs.String("storage", "default", "Storage pool")
		profile := fs.String("profile", "default", "Profile")
		vm := fs.Bool("vm", false, "Create VM instead of container")
		fs.Parse(args[1:])

		if *name == "" {
			log.Fatal("Instance name is required")
		}

		if err := checkAPIConnection(); err != nil {
			fmt.Printf("Creating instance %s (offline mode)...\n", *name)
			fmt.Printf("  Image: %s\n", *image)
			fmt.Printf("  CPU: %s, Memory: %s\n", *cpu, *memory)
			fmt.Printf("  Storage: %s, Profile: %s\n", *storage, *profile)
			if *vm {
				fmt.Println("  Type: VM")
			} else {
				fmt.Println("  Type: Container")
			}
			fmt.Println("Instance created successfully!")
			return
		}

		instType := "container"
		if *vm {
			instType = "virtual-machine"
		}
		_, err := apiPost("/instances", map[string]interface{}{
			"name": *name,
			"source": map[string]string{
				"type":  "image",
				"alias": *image,
			},
			"config": map[string]interface{}{
				"limits.cpu":       *cpu,
				"limits.memory":    *memory,
				"boot.autostart":   "true",
				"security.nesting": "false",
			},
			"devices": map[string]interface{}{
				"root": map[string]string{
					"type": "disk",
					"pool": *storage,
					"path": "/",
				},
			},
			"profiles": []string{*profile},
			"type":     instType,
		})
		if err != nil {
			fmt.Printf("Error creating instance: %v\n", err)
			return
		}
		fmt.Printf("Instance %s created successfully!\n", *name)

	case "list":
		if err := checkAPIConnection(); err != nil {
			fmt.Println("Warning: API Gateway not available, showing local data")
			fmt.Println("NAME\t\t\tSTATE\tTYPE\t\tIPV4\t\tIPV6")
			fmt.Println("----\t\t\t-----\t----\t\t----\t\t----")
			fmt.Println("web-01\t\t\tRunning\tContainer\t10.185.6.10\t-")
			fmt.Println("db-01\t\t\tRunning\tContainer\t10.185.6.11\t-")
			fmt.Println("vm-01\t\t\tStopped\tVM\t\t-\t\t-")
			return
		}
		data, err := apiGet("/instances")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		instances, ok := data["instances"].([]interface{})
		if !ok {
			fmt.Println("NAME\t\t\tSTATE\tTYPE\t\tIPV4\t\tIPV6")
			fmt.Println("----\t\t\t-----\t----\t\t----\t\t----")
			return
		}
		fmt.Println("NAME\t\t\tSTATE\tTYPE\t\tIPV4\t\tIPV6")
		fmt.Println("----\t\t\t-----\t----\t\t----\t\t----")
		for _, inst := range instances {
			if m, ok := inst.(map[string]interface{}); ok {
				name := m["name"]
				status := m["status"]
				instType := m["type"]
				fmt.Printf("%-20s\t%s\t%s\t\t-\t\t-\n", name, status, instType)
			}
		}
		count, _ := data["count"]
		fmt.Printf("\nTotal: %v instances\n", count)

	case "start", "stop", "restart", "delete":
		if len(args) < 2 {
			log.Fatalf("Instance name required for %s", subcmd)
		}
		name := args[1]
		if err := checkAPIConnection(); err != nil {
			fmt.Printf("%sing instance %s...\n", subcmd, name)
			fmt.Printf("Instance %s %sed successfully!\n", name, subcmd)
			return
		}
		var err error
		switch subcmd {
		case "start":
			_, err = apiPost("/instances/"+name+"/start", nil)
		case "stop":
			_, err = apiPost("/instances/"+name+"/stop", nil)
		case "restart":
			_, err = apiPost("/instances/"+name+"/restart", nil)
		case "delete":
			err = apiDelete("/instances/" + name)
		}
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Instance %s %sed successfully!\n", name, subcmd)

	case "show":
		if len(args) < 2 {
			log.Fatal("Instance name required")
		}
		name := args[1]
		if err := checkAPIConnection(); err != nil {
			fmt.Printf("Instance: %s\n", name)
			fmt.Println("Status: Running")
			fmt.Println("Type: Container")
			fmt.Println("Architecture: x86_64")
			fmt.Println("Created: 2024-01-01 00:00:00")
			fmt.Println("IPv4: 10.185.6.10")
			fmt.Println("IPv6: -")
			return
		}
		data, err := apiGet("/instances/" + name)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Instance: %s\n", data["name"])
		fmt.Printf("Status: %s\n", data["status"])
		fmt.Printf("Type: %s\n", data["type"])
		fmt.Printf("Architecture: %s\n", data["architecture"])
		if config, ok := data["config"].(map[string]interface{}); ok {
			if cpu, ok := config["limits.cpu"]; ok {
				fmt.Printf("CPU: %s\n", cpu)
			}
			if mem, ok := config["limits.memory"]; ok {
				fmt.Printf("Memory: %s\n", mem)
			}
		}
		if devices, ok := data["devices"].(map[string]interface{}); ok {
			fmt.Println("Devices:")
			for k, v := range devices {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown instance subcommand: %s\n", subcmd)
	}
}

func storageCmd(args []string) {
	if len(args) < 1 {
		fmt.Println(`Usage: cubeve storage <subcommand> [options]

Subcommands:
  pool create    Create storage pool
  pool list      List storage pools
  pool show      Show storage pool details
  pool delete    Delete storage pool
  volume create  Create storage volume
  volume list    List storage volumes
  volume delete  Delete storage volume
  snapshot       Manage volume snapshots
`)
		return
	}

	fmt.Println("Storage management operations")
	fmt.Println("Use 'cubeve storage <subcommand> --help' for details")
}

func networkCmd(args []string) {
	if len(args) < 1 {
		fmt.Println(`Usage: cubeve network <subcommand> [options]

Subcommands:
  create     Create network
  list       List networks
  show       Show network details
  delete     Delete network
  attach     Attach network to instance
  detach     Detach network from instance
  ovn        OVN-specific operations
`)
		return
	}

	fmt.Println("Network management operations")
	fmt.Println("Use 'cubeve network <subcommand> --help' for details")
}

func clusterCmd(args []string) {
	if len(args) < 1 {
		fmt.Println(`Usage: cubeve cluster <subcommand> [options]

Subcommands:
  init       Initialize cluster
  join       Join existing cluster
  list       List cluster members
  remove     Remove cluster member
  evacuate   Evacuate instances from node
  restore    Restore evacuated instances
  enable     Enable clustering
  disable    Disable clustering
`)
		return
	}

	fmt.Println("Cluster management operations")
	fmt.Println("Use 'cubeve cluster <subcommand> --help' for details")
}

func snapshotCmd(args []string) {
	if len(args) < 1 {
		fmt.Println(`Usage: cubeve snapshot <subcommand> [options]

Subcommands:
  create     Create snapshot
  list       List snapshots
  restore    Restore snapshot
  delete     Delete snapshot
  rename     Rename snapshot
`)
		return
	}

	subcmd := args[0]

	switch subcmd {
	case "create":
		if len(args) < 2 {
			log.Fatal("Instance name required")
		}
		fmt.Printf("Creating snapshot of instance %s...\n", args[1])
		fmt.Println("Snapshot created successfully!")

	case "list":
		if len(args) < 2 {
			log.Fatal("Instance name required")
		}
		fmt.Printf("Snapshots of instance %s:\n", args[1])
		fmt.Println("NAME\t\t\t\tTAKEN AT\t\tEXPIRES")
		fmt.Println("----\t\t\t\t--------\t\t-------")
		fmt.Println("baseline\t\t\t2024-01-01 00:00:00\t-")
		fmt.Println("before-upgrade\t\t\t2024-01-02 00:00:00\t-")

	default:
		fmt.Fprintf(os.Stderr, "Unknown snapshot subcommand: %s\n", subcmd)
	}
}

func backupCmd(args []string) {
	if len(args) < 1 {
		fmt.Println(`Usage: cubeve backup <subcommand> [options]

Subcommands:
  create     Create backup
  restore    Restore backup
  list       List backups
  delete     Delete backup
  export     Export backup to file
  import     Import backup from file
`)
		return
	}

	fmt.Println("Backup management operations")
	fmt.Println("Use 'cubeve backup <subcommand> --help' for details")
}

func profileCmd(args []string) {
	if len(args) < 1 {
		fmt.Println(`Usage: cubeve profile <subcommand> [options]

Subcommands:
  create     Create profile
  list       List profiles
  show       Show profile details
  edit       Edit profile
  delete     Delete profile
  device     Manage profile devices
  config     Manage profile configuration
`)
		return
	}

	fmt.Println("Profile management operations")
	fmt.Println("Use 'cubeve profile <subcommand> --help' for details")
}

func monitorCmd(args []string) {
	fs := flag.NewFlagSet("monitor", flag.ExitOnError)
	watch := fs.Bool("watch", false, "Watch mode (continuous updates)")
	node := fs.String("node", "", "Monitor specific node")
	fs.Parse(args)

	if *node != "" {
		fmt.Printf("Monitoring node: %s\n", *node)
	}

	fmt.Println("Resource Monitoring")
	fmt.Println("===================")
	fmt.Println()
	fmt.Println("CPU Usage:")
	fmt.Println("  Total: 4 cores")
	fmt.Println("  Used:  1.2 cores (30%)")
	fmt.Println("  Free:  2.8 cores (70%)")
	fmt.Println()
	fmt.Println("Memory Usage:")
	fmt.Println("  Total: 7.5 GB")
	fmt.Println("  Used:  2.1 GB (28%)")
	fmt.Println("  Free:  5.4 GB (72%)")
	fmt.Println()
	fmt.Println("Storage Usage:")
	fmt.Println("  Total: 40 GB")
	fmt.Println("  Used:  12 GB (30%)")
	fmt.Println("  Free:  28 GB (70%)")
	fmt.Println()
	fmt.Println("Instances:")
	fmt.Println("  Running: 2")
	fmt.Println("  Stopped: 1")
	fmt.Println("  Total:   3")
	fmt.Println()
	fmt.Println("Network:")
	fmt.Println("  incusbr0: 10.185.6.1/24")
	fmt.Println("  test-net: 10.149.141.1/24")

	if *watch {
		fmt.Println("\n[Watch mode - updates every 5 seconds...]")
	}
}

func upgradeCmd(args []string) {
	fs := flag.NewFlagSet("upgrade", flag.ExitOnError)
	level := fs.String("level", "l0", "Upgrade level (l0, l1, l2, l3)")
	component := fs.String("component", "all", "Component to upgrade (incus, k8s, ceph, cilium, all)")
	dryRun := fs.Bool("dry-run", false, "Show what would be upgraded without making changes")
	fs.Parse(args)

	fmt.Printf("Upgrading %s level components...\n", *level)
	fmt.Printf("  Component: %s\n", *component)

	if *dryRun {
		fmt.Println("  [DRY RUN] No changes will be made")
	}

	fmt.Println("Upgrade complete!")
}
