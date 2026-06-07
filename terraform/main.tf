# CubeVE Infrastructure as Code (Terraform)

terraform {
  required_providers {
    incus = {
      source = "lxc/incus"
      version = "1.0.0"
    }
  }
}

# Configure Incus provider
provider "incus" {
  generate_client_certificates = true
  accept_remote_certificate    = true

  remote {
    name     = "local"
    scheme   = "unix"
    address  = "/var/lib/incus/unix.socket"
    default  = true
  }
}

# Storage Pool
resource "incus_storage_pool" "zfs" {
  name   = "cubeve-zfs"
  driver = "zfs"
  config = {
    source = "test-pool"
  }
}

# Network
resource "incus_network" "cubeve" {
  name = "cubeve-net"
  type = "bridge"
  
  config = {
    "ipv4.address" = "10.200.0.1/24"
    "ipv4.nat"     = "true"
    "ipv6.address" = "fd42:cube::1/64"
    "ipv6.nat"     = "true"
  }
}

# Profile
resource "incus_profile" "default" {
  name = "cubeve-default"

  config = {
    "limits.cpu"    = "2"
    "limits.memory" = "4GB"
  }

  device {
    name = "root"
    type = "disk"
    properties = {
      pool = incus_storage_pool.zfs.name
      path = "/"
      size = "20GB"
    }
  }

  device {
    name = "eth0"
    type = "nic"
    properties = {
      network = incus_network.cubeve.name
      name    = "eth0"
    }
  }
}

# Container Instance
resource "incus_instance" "web" {
  count    = 2
  name     = "web-${count.index + 1}"
  image    = "ubuntu/24.04"
  type     = "container"
  profiles = [incus_profile.default.name]

  config = {
    "boot.autostart" = "true"
  }
}

# VM Instance
resource "incus_instance" "database" {
  count    = 1
  name     = "db-1"
  image    = "ubuntu/24.04"
  type     = "virtual-machine"
  profiles = [incus_profile.default.name]

  config = {
    "boot.autostart" = "true"
    "limits.cpu"     = "4"
    "limits.memory"  = "8GB"
  }
}

# Outputs
output "web_instances" {
  value = incus_instance.web[*].ipv4_address
}

output "db_instance" {
  value = incus_instance.database[0].ipv4_address
}
