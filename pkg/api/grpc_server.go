package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"go.uber.org/zap"
)

// GRPCServer implements the CubeAPI gRPC server
type GRPCServer struct {
	listener   net.Listener
	server     *grpc.Server
	logger     *zap.Logger
	incus      *IncusClient
	instanceID string
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(addr string, logger *zap.Logger, incus *IncusClient) (*GRPCServer, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()

	s := &GRPCServer{
		listener: lis,
		server:   grpcServer,
		logger:   logger,
		incus:    incus,
	}

	// Register services
	RegisterCubeAPIServer(grpcServer, s)

	return s, nil
}

// Run starts the gRPC server
func (s *GRPCServer) Run() error {
	s.logger.Info("Starting gRPC server", zap.String("addr", s.listener.Addr().String()))
	return s.server.Serve(s.listener)
}

// Stop stops the gRPC server
func (s *GRPCServer) Stop() {
	s.server.GracefulStop()
}

// CubeAPIServer implementation

func (s *GRPCServer) ListInstances(ctx context.Context, req *ListInstancesRequest) (*ListInstancesResponse, error) {
	s.logger.Debug("ListInstances", zap.String("runtime", req.Runtime))

	instances, err := s.incus.ListInstances(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var result []*Instance
	for _, name := range instances {
		inst, err := s.incus.GetInstance(ctx, name)
		if err != nil {
			continue
		}
		result = append(result, convertInstance(inst))
	}

	return &ListInstancesResponse{
		Instances: result,
		Count:     int32(len(result)),
	}, nil
}

func (s *GRPCServer) CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*Instance, error) {
	s.logger.Info("CreateInstance",
		zap.String("name", req.Name),
		zap.String("runtime", req.Runtime),
		zap.String("image", req.Image))

	config := map[string]interface{}{
		"name":         req.Name,
		"architecture": "x86_64",
		"source": map[string]string{
			"type":  "image",
			"alias": req.Image,
		},
	}

	if req.Runtime == "vm" || req.Runtime == "incus-vm" {
		config["type"] = "virtual-machine"
	} else {
		config["type"] = "container"
	}

	if req.Cpu != "" {
		config["config"] = map[string]string{
			"limits.cpu": req.Cpu,
		}
	}
	if req.Memory != "" {
		if config["config"] == nil {
			config["config"] = map[string]string{}
		}
		config["config"].(map[string]string)["limits.memory"] = req.Memory
	}

	if err := s.incus.CreateInstance(ctx, config); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &Instance{
		Id:     req.Name,
		Status: "creating",
		Runtime: req.Runtime,
	}, nil
}

func (s *GRPCServer) GetInstance(ctx context.Context, req *GetInstanceRequest) (*Instance, error) {
	inst, err := s.incus.GetInstance(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return convertInstance(inst), nil
}

func (s *GRPCServer) DeleteInstance(ctx context.Context, req *DeleteInstanceRequest) (*emptypb.Empty, error) {
	if err := s.incus.DeleteInstance(ctx, req.Id); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) StartInstance(ctx context.Context, req *StartInstanceRequest) (*Instance, error) {
	if err := s.incus.StartInstance(ctx, req.Id); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &Instance{Id: req.Id, Status: "starting"}, nil
}

func (s *GRPCServer) StopInstance(ctx context.Context, req *StopInstanceRequest) (*Instance, error) {
	if err := s.incus.StopInstance(ctx, req.Id); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &Instance{Id: req.Id, Status: "stopping"}, nil
}

func (s *GRPCServer) RestartInstance(ctx context.Context, req *RestartInstanceRequest) (*Instance, error) {
	if err := s.incus.RestartInstance(ctx, req.Id); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &Instance{Id: req.Id, Status: "restarting"}, nil
}

func (s *GRPCServer) CreateSnapshot(ctx context.Context, req *CreateSnapshotRequest) (*Snapshot, error) {
	if err := s.incus.CreateSnapshot(ctx, req.InstanceId, req.Name); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &Snapshot{
		Id:       req.Name,
		Instance: req.InstanceId,
		Status:   "created",
	}, nil
}

func (s *GRPCServer) ListSnapshots(ctx context.Context, req *ListSnapshotsRequest) (*ListSnapshotsResponse, error) {
	snaps, err := s.incus.ListSnapshots(ctx, req.InstanceId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var result []*Snapshot
	for _, name := range snaps {
		result = append(result, &Snapshot{Id: name, Instance: req.InstanceId})
	}

	return &ListSnapshotsResponse{Snapshots: result}, nil
}

func (s *GRPCServer) RestoreSnapshot(ctx context.Context, req *RestoreSnapshotRequest) (*Instance, error) {
	if err := s.incus.RestoreSnapshot(ctx, req.InstanceId, req.SnapshotId); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &Instance{Id: req.InstanceId, Status: "restored"}, nil
}

func (s *GRPCServer) DeleteSnapshot(ctx context.Context, req *DeleteSnapshotRequest) (*emptypb.Empty, error) {
	if err := s.incus.DeleteSnapshot(ctx, req.InstanceId, req.SnapshotId); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) MigrateInstance(ctx context.Context, req *MigrateInstanceRequest) (*Operation, error) {
	// In real implementation, this would initiate migration
	return &Operation{
		Id:     fmt.Sprintf("migrate-%d", time.Now().Unix()),
		Status: "running",
		Type:   "migrate",
	}, nil
}

func (s *GRPCServer) GetConsole(ctx context.Context, req *GetConsoleRequest) (*Console, error) {
	return &Console{
		Instance: req.InstanceId,
		Url:      fmt.Sprintf("wss://%s/1.0/instances/%s/console", s.instanceID, req.InstanceId),
	}, nil
}

func (s *GRPCServer) ExecInstance(ctx context.Context, req *ExecInstanceRequest) (*ExecResult, error) {
	return &ExecResult{
		Instance: req.InstanceId,
		Output:   "",
		ExitCode: 0,
	}, nil
}

func (s *GRPCServer) ListStoragePools(ctx context.Context, req *ListStoragePoolsRequest) (*ListStoragePoolsResponse, error) {
	pools, err := s.incus.ListStoragePools(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var result []*StoragePool
	for _, name := range pools {
		pool, err := s.incus.GetStoragePool(ctx, name)
		if err != nil {
			continue
		}
		result = append(result, convertStoragePool(pool))
	}

	return &ListStoragePoolsResponse{Pools: result}, nil
}

func (s *GRPCServer) CreateStoragePool(ctx context.Context, req *CreateStoragePoolRequest) (*StoragePool, error) {
	config := map[string]interface{}{
		"name":   req.Name,
		"driver": req.Driver,
		"config": req.Config,
	}
	if err := s.incus.CreateStoragePool(ctx, config); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &StoragePool{Name: req.Name, Driver: req.Driver, Status: "created"}, nil
}

func (s *GRPCServer) GetStoragePool(ctx context.Context, req *GetStoragePoolRequest) (*StoragePool, error) {
	pool, err := s.incus.GetStoragePool(ctx, req.Name)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return convertStoragePool(pool), nil
}

func (s *GRPCServer) DeleteStoragePool(ctx context.Context, req *DeleteStoragePoolRequest) (*emptypb.Empty, error) {
	if err := s.incus.DeleteStoragePool(ctx, req.Name); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) ListNetworks(ctx context.Context, req *ListNetworksRequest) (*ListNetworksResponse, error) {
	networks, err := s.incus.ListNetworks(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var result []*Network
	for _, name := range networks {
		net, err := s.incus.GetNetwork(ctx, name)
		if err != nil {
			continue
		}
		result = append(result, convertNetwork(net))
	}

	return &ListNetworksResponse{Networks: result}, nil
}

func (s *GRPCServer) CreateNetwork(ctx context.Context, req *CreateNetworkRequest) (*Network, error) {
	config := map[string]interface{}{
		"name":   req.Name,
		"type":   req.Type,
		"config": req.Config,
	}
	if err := s.incus.CreateNetwork(ctx, config); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &Network{Name: req.Name, Type: req.Type, Status: "created"}, nil
}

func (s *GRPCServer) GetNetwork(ctx context.Context, req *GetNetworkRequest) (*Network, error) {
	net, err := s.incus.GetNetwork(ctx, req.Name)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return convertNetwork(net), nil
}

func (s *GRPCServer) DeleteNetwork(ctx context.Context, req *DeleteNetworkRequest) (*emptypb.Empty, error) {
	if err := s.incus.DeleteNetwork(ctx, req.Name); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) ListProfiles(ctx context.Context, req *ListProfilesRequest) (*ListProfilesResponse, error) {
	profiles, err := s.incus.ListProfiles(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var result []*Profile
	for _, name := range profiles {
		prof, err := s.incus.GetProfile(ctx, name)
		if err != nil {
			continue
		}
		result = append(result, convertProfile(prof))
	}

	return &ListProfilesResponse{Profiles: result}, nil
}

func (s *GRPCServer) CreateProfile(ctx context.Context, req *CreateProfileRequest) (*Profile, error) {
	config := map[string]interface{}{
		"name":    req.Name,
		"config":  req.Config,
		"devices": req.Devices,
	}
	if err := s.incus.CreateProfile(ctx, config); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &Profile{Name: req.Name, Status: "created"}, nil
}

func (s *GRPCServer) GetProfile(ctx context.Context, req *GetProfileRequest) (*Profile, error) {
	prof, err := s.incus.GetProfile(ctx, req.Name)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return convertProfile(prof), nil
}

func (s *GRPCServer) DeleteProfile(ctx context.Context, req *DeleteProfileRequest) (*emptypb.Empty, error) {
	if err := s.incus.DeleteProfile(ctx, req.Name); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) ListClusterMembers(ctx context.Context, req *ListClusterMembersRequest) (*ListClusterMembersResponse, error) {
	members, err := s.incus.GetClusterMembers(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var result []*ClusterMember
	for _, name := range members {
		result = append(result, &ClusterMember{Name: name})
	}

	return &ListClusterMembersResponse{Members: result}, nil
}

func (s *GRPCServer) AddClusterMember(ctx context.Context, req *AddClusterMemberRequest) (*Operation, error) {
	return &Operation{
		Id:     fmt.Sprintf("join-%d", time.Now().Unix()),
		Status: "running",
		Type:   "cluster_join",
	}, nil
}

func (s *GRPCServer) RemoveClusterMember(ctx context.Context, req *RemoveClusterMemberRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) EvacuateMember(ctx context.Context, req *EvacuateMemberRequest) (*Operation, error) {
	return &Operation{
		Id:     fmt.Sprintf("evacuate-%d", time.Now().Unix()),
		Status: "running",
		Type:   "evacuate",
	}, nil
}

func (s *GRPCServer) RestoreMember(ctx context.Context, req *RestoreMemberRequest) (*Operation, error) {
	return &Operation{
		Id:     fmt.Sprintf("restore-%d", time.Now().Unix()),
		Status: "running",
		Type:   "restore",
	}, nil
}

func (s *GRPCServer) ListOperations(ctx context.Context, req *ListOperationsRequest) (*ListOperationsResponse, error) {
	return &ListOperationsResponse{Operations: []*Operation{}}, nil
}

func (s *GRPCServer) GetOperation(ctx context.Context, req *GetOperationRequest) (*Operation, error) {
	return &Operation{Id: req.Id, Status: "running"}, nil
}

func (s *GRPCServer) CancelOperation(ctx context.Context, req *CancelOperationRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// Helper functions

func convertInstance(inst *Instance) *Instance {
	if inst == nil {
		return nil
	}
	return &Instance{
		Id:      inst.Name,
		Name:    inst.Name,
		Status:  inst.Status,
		Type:    inst.Type,
		Runtime: "incus",
	}
}

func convertStoragePool(pool *StoragePool) *StoragePool {
	if pool == nil {
		return nil
	}
	return &StoragePool{
		Name:   pool.Name,
		Driver: pool.Driver,
		Status: "created",
	}
}

func convertNetwork(net *Network) *Network {
	if net == nil {
		return nil
	}
	return &Network{
		Name: net.Name,
		Type: net.Type,
	}
}

func convertProfile(prof *Profile) *Profile {
	if prof == nil {
		return nil
	}
	return &Profile{
		Name: prof.Name,
	}
}

// Protobuf types (placeholder - would be generated from proto file)

// ListInstancesRequest ...