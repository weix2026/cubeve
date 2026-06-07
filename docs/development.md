# CubeVE Development Guide

## Quick Start

### Prerequisites
- Ubuntu 22.04+ or 24.04+
- Go 1.22+
- Docker (optional)
- Incus 7.1+

### L0: Control Plane Setup
```bash
# Install dependencies
sudo bash scripts/l0-install.sh

# Verify installation
bash tests/l0-validation.sh
```

### L1: Cluster Management
```bash
# Initialize cluster
sudo bash scripts/l1-install.sh

# Add member nodes
incus cluster add <node-name>
```

### L2: Kubernetes Integration
```bash
# Deploy manifests
kubectl apply -f manifests/
```

### L3: Full Production
```bash
# Deploy with Helm
helm install cubeve ./helm/cubeve
```

## Project Structure

```
cubeve/
├── api/                    # API definitions
│   └── proto/              # Protobuf schemas
├── cmd/                    # CLI tools
│   ├── api-gateway/        # REST API server
│   ├── instance-controller/# K8s controller
│   ├── cubesandbox-operator/ # CubeSandbox operator
│   └── cubeconsole/        # CLI client
├── config/                 # Configuration files
├── docs/                   # Documentation
├── helm/                   # Helm charts
├── manifests/              # K8s manifests
├── pkg/                    # Go packages
│   ├── api/                # API implementations
│   ├── controller/         # K8s controllers
│   └── runtime/            # Runtime adapters
├── scripts/                # Deployment scripts
│   ├── l0-install.sh
│   ├── l1-install.sh
│   ├── l2-install.sh
│   └── l3-install.sh
├── terraform/              # IaC configurations
├── tests/                  # Test suites
│   ├── l0-validation.sh
│   └── integration-test.sh
├── Dockerfile
├── go.mod
├── Makefile
└── README.md
```

## API Usage

### REST API
```bash
# List instances
curl http://localhost:8080/api/v1/instances

# Create instance
curl -X POST http://localhost:8080/api/v1/instances \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-vm",
    "runtime": "incus",
    "image": "ubuntu/24.04",
    "cpu": "2",
    "memory": "4GB"
  }'
```

### gRPC API
```bash
# Using grpcurl
grpcurl -plaintext localhost:50051 cubeve.api.v1.CubeAPI/ListInstances
```

## CLI Usage

```bash
# Build cubeconsole
go build -o cubeconsole ./cmd/cubeconsole

# List instances
./cubeconsole list

# Create instance
./cubeconsole create --name test-vm --image ubuntu/24.04

# Delete instance
./cubeconsole delete test-vm
```

## Development

### Building
```bash
make all
```

### Testing
```bash
make test
# or
bash tests/integration-test.sh
```

### Docker
```bash
docker build -t cubeve:latest .
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes
4. Run tests
5. Submit a pull request

## License

MIT License
