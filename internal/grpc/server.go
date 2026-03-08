package grpc

import (
	"context"
	"fmt"

	"collaborative-markdown-editor/internal/document"
	"collaborative-markdown-editor/internal/grpc/internalpb"

	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"

	log "github.com/rs/zerolog/log"
)

type Server struct {
	internalpb.UnimplementedInternalServiceServer

	documentService document.Service
	internalSecret  string
}

func NewServer(docService document.Service, internalSecret string) *Server {
	return &Server{
		documentService: docService,
		internalSecret:  internalSecret,
	}
}

// loggingInterceptor emits an info-level message for every incoming gRPC request
func (s *Server) loggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	log.Info().Str("method", info.FullMethod).Msg("gRPC request received")
	return handler(ctx, req)
}

// Unary interceptor verifies that every request contains correct internal
// secret metadata.  This is equivalent to the HTTP InternalAuthMiddleware used
// by the gin router.
func (s *Server) authInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}

	vals := md.Get("x-internal-secret")
	if len(vals) == 0 || vals[0] != s.internalSecret {
		return nil, fmt.Errorf("unauthorized")
	}

	return handler(ctx, req)
}

// Start creates and begins serving a gRPC server listening on the given
// address.  It returns the underlying *grpc.Server and net.Listener so the
// caller can later shut the server down gracefully.  The function itself does
// not block; it spawns Serve in a goroutine.
func (s *Server) Start(address string) (*grpc.Server, net.Listener, error) {
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(s.loggingInterceptor, s.authInterceptor),
	}

	grpcServer := grpc.NewServer(opts...)

	internalpb.RegisterInternalServiceServer(grpcServer, s)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, nil, err
	}

	// run serve asynchronously; caller may ignore the returned error or log it
	go func() {
		_ = grpcServer.Serve(listener)
	}()

	return grpcServer, listener, nil
}

// Register starts a dedicated gRPC server listening on the given address.
func (s *Server) Register(address string) error {
	grpcServer, listener, err := s.Start(address)
	if err != nil {
		return err
	}
	return grpcServer.Serve(listener)
}

// --- internalpb.InternalServiceServer methods ---

func (s *Server) GetUserRole(ctx context.Context, req *internalpb.PermissionRequest) (*internalpb.PermissionResponse, error) {
	role, err := s.documentService.FetchUserRole(ctx, req.DocId, req.UserId)
	if err != nil {
		return nil, err
	}
	return &internalpb.PermissionResponse{Role: role}, nil
}

func (s *Server) GetDocumentState(ctx context.Context, req *internalpb.DocumentIDRequest) (*internalpb.DocumentStateResponse, error) {
	state, err := s.documentService.GetDocumentState(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	// translate service response into protobuf message
	pbUpdates := make([]*internalpb.DocumentUpdate, 0, len(state.Updates))
	for _, u := range state.Updates {
		pbUpdates = append(pbUpdates, &internalpb.DocumentUpdate{
			Seq:    u.Seq,
			Binary: u.Binary,
		})
	}

	return &internalpb.DocumentStateResponse{
		Snapshot:    state.Snapshot,
		SnapshotSeq: state.SnapshotSeq,
		Updates:     pbUpdates,
	}, nil
}

func (s *Server) CreateUpdate(ctx context.Context, req *internalpb.UpdateRequest) (*emptypb.Empty, error) {
	if err := s.documentService.CreateDocumentUpdate(ctx, req.DocId, req.UserId, req.Update); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) CreateSnapshot(ctx context.Context, req *internalpb.SnapshotRequest) (*emptypb.Empty, error) {
	if err := s.documentService.CreateDocumentSnapshot(ctx, req.DocId, req.Snapshot); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
