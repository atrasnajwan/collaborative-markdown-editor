package sync

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"collaborative-markdown-editor/internal/config"
	"collaborative-markdown-editor/internal/sync/syncpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func prepareConfig() {
	config.AppConfig.SyncServerAddress = ""
	config.AppConfig.SyncServerGRPCAddress = ""
	config.AppConfig.SyncServerSecret = "test-secret"
}

func TestSyncClient_HTTPFallback(t *testing.T) {
	prepareConfig()

	// prepare httptest server that mimics the HTTP sync API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// auth check
		if r.Header.Get("X-Internal-Secret") != config.AppConfig.SyncServerSecret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/internal/documents/123/state":
			w.Write([]byte("snapshot-bytes"))
		case r.Method == http.MethodPut && r.URL.Path == "/internal/documents/123/permission":
			var req UpdateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if req.UserID != 55 || req.Role != "editor" {
				http.Error(w, "unexpected payload", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/internal/documents/123":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	config.AppConfig.SyncServerAddress = ts.URL

	client := NewSyncClient()
	defer client.Close()

	// if no gRPC address provided, grpcClient must be nil
	if client.grpcClient != nil {
		t.Fatal("expected grpcClient to be nil when no address configured")
	}

	state, err := client.FetchDocumentState(context.Background(), 123)
	if err != nil {
		t.Fatalf("FetchDocumentState returned error: %v", err)
	}
	if string(state) != "snapshot-bytes" {
		t.Fatalf("unexpected state: %s", state)
	}

	if err := client.UpdateUserPermission(context.Background(), 123, 55, "editor"); err != nil {
		t.Fatalf("UpdateUserPermission failed: %v", err)
	}

	if err := client.RemoveDocument(context.Background(), 123); err != nil {
		t.Fatalf("RemoveDocument failed: %v", err)
	}
}

// mock GRPC server implementing the syncpb service.
type grpcMock struct {
	syncpb.UnimplementedSyncServerInternalServer
}

func (g *grpcMock) GetState(ctx context.Context, req *syncpb.DocumentIDRequest) (*syncpb.DocumentStateResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md.Get("x-internal-secret")) == 0 || md.Get("x-internal-secret")[0] != config.AppConfig.SyncServerSecret {
		return nil, status.Error(codes.Unauthenticated, "bad secret")
	}
	return &syncpb.DocumentStateResponse{State: []byte("grpc-snapshot")}, nil
}

func (g *grpcMock) DeleteDocument(ctx context.Context, req *syncpb.DocumentIDRequest) (*emptypb.Empty, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if len(md.Get("x-internal-secret")) == 0 {
		return nil, status.Error(codes.Unauthenticated, "")
	}
	return &emptypb.Empty{}, nil
}

func (g *grpcMock) PermissionChanged(ctx context.Context, req *syncpb.PermissionChangedRequest) (*emptypb.Empty, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if len(md.Get("x-internal-secret")) == 0 {
		return nil, status.Error(codes.Unauthenticated, "")
	}
	if req.DocId != 123 || req.UserId != 55 || req.Role != "owner" {
		return nil, status.Error(codes.InvalidArgument, "unexpected payload")
	}
	return &emptypb.Empty{}, nil
}

func startGRPCMockServer(t *testing.T) (addr string, stop func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	server := grpc.NewServer()
	syncpb.RegisterSyncServerInternalServer(server, &grpcMock{})

	go server.Serve(listener)
	return listener.Addr().String(), func() {
		server.Stop()
	}
}

func TestSyncClient_GRPC(t *testing.T) {
	prepareConfig()
	addr, stop := startGRPCMockServer(t)
	defer stop()

	config.AppConfig.SyncServerGRPCAddress = addr
	config.AppConfig.SyncServerSecret = "test-secret"

	client := NewSyncClient()
	defer client.Close()

	if client.grpcClient == nil {
		t.Fatal("expected grpcClient to be non-nil when address set")
	}

	state, err := client.FetchDocumentState(context.Background(), 123)
	if err != nil {
		t.Fatalf("FetchDocumentState failed: %v", err)
	}
	if string(state) != "grpc-snapshot" {
		t.Fatalf("unexpected grpc state: %s", state)
	}

	if err := client.UpdateUserPermission(context.Background(), 123, 55, "owner"); err != nil {
		t.Fatalf("UpdateUserPermission grpc failed: %v", err)
	}

	if err := client.RemoveDocument(context.Background(), 123); err != nil {
		t.Fatalf("RemoveDocument grpc failed: %v", err)
	}
}
