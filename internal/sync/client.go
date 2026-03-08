package sync

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"collaborative-markdown-editor/internal/config"
	"collaborative-markdown-editor/internal/sync/syncpb"

	log "github.com/rs/zerolog/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"encoding/json"
)

type SyncClient struct {
	httpClient *http.Client

	// If grpcClient is non-nil we will use it instead of the HTTP helpers.
	grpcConn   *grpc.ClientConn
	grpcClient syncpb.SyncServerInternalClient
}

type Client interface {
	FetchDocumentState(ctx context.Context, docID uint64) ([]byte, error)
	UpdateUserPermission(ctx context.Context, docID uint64, userID uint64, role string) error
	RemoveDocument(ctx context.Context, docID uint64) error
}

func NewSyncClient() *SyncClient {
	client := &SyncClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// if a GRPC address is explicitly provided we attempt to dial it and create a gRPC client
	if addr := config.AppConfig.SyncServerGRPCAddress; addr != "" {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		opts := []grpc.DialOption{
			grpc.WithBlock(), // Must be present for initial dial
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		conn, err := grpc.DialContext(timeoutCtx, config.AppConfig.SyncServerGRPCAddress, opts...)

		if err == nil {
			client.grpcConn = conn
			client.grpcClient = syncpb.NewSyncServerInternalClient(conn)
		} else {
			log.Warn().Err(err).Str("address", addr).Msg("failed to dial sync server gRPC, falling back to HTTP")
		}
	}

	return client
}

// Close shuts down any underlying gRPC connection.
func (s *SyncClient) Close() error {
	if s.grpcConn != nil {
		return s.grpcConn.Close()
	}
	return nil
}

func (s *SyncClient) doRequest(ctx context.Context, method, path string, headers map[string]string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	url := fmt.Sprintf("%s%s", config.AppConfig.SyncServerAddress, path)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	// set headers
	req.Header.Set("X-Internal-Secret", config.AppConfig.SyncServerSecret)
	for key, val := range headers {
		req.Header.Set(key, val)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sync server error: status=%d body=%s", resp.StatusCode, string(b))
	}

	return resp, nil
}

// GET /internal/documents/:id/state
func (s *SyncClient) FetchDocumentState(ctx context.Context, docID uint64) ([]byte, error) {
	if s.grpcClient != nil {
		// use gRPC transport
		md := metadata.Pairs("x-internal-secret", config.AppConfig.SyncServerSecret)
		ctx = metadata.NewOutgoingContext(ctx, md)
		resp, err := s.grpcClient.GetState(ctx, &syncpb.DocumentIDRequest{Id: docID})
		if err != nil {
			log.Error().Err(err).Uint64("doc_id", docID).Msg("gRPC call to sync server failed")
			return nil, err
		}
		if len(resp.State) == 0 {
			log.Error().Uint64("doc_id", docID).Msg("gRPC sync server returned empty snapshot")
			return nil, fmt.Errorf("empty state from sync server")
		}
		return resp.State, nil
	}

	path := fmt.Sprintf("/internal/documents/%d/state", docID)
	headers := map[string]string{
		"Content-Type": "application/octet-stream",
	}
	resp, err := s.doRequest(ctx, http.MethodGet, path, headers, nil)
	if err != nil {
		log.Error().Err(err).Uint64("doc_id", docID).Msg("failed to get last state from sync server")
		return nil, err
	}
	defer resp.Body.Close()

	state, err := io.ReadAll(resp.Body)
	if err != nil || len(state) == 0 {
		log.Error().Err(err).Uint64("doc_id", docID).Msg("Can't read doc state or empty state")
		return nil, err
	}

	return state, nil
}

type UpdateRequest struct {
	UserID uint64 `json:"user_id"`
	Role   string `json:"role"`
}

// PUT /internal/documents/:id/permission
func (s *SyncClient) UpdateUserPermission(ctx context.Context, docID, userID uint64, role string) error {
	if s.grpcClient != nil {
		md := metadata.Pairs("x-internal-secret", config.AppConfig.SyncServerSecret)
		ctx = metadata.NewOutgoingContext(ctx, md)
		_, err := s.grpcClient.PermissionChanged(ctx, &syncpb.PermissionChangedRequest{DocId: docID, UserId: userID, Role: role})
		if err != nil {
			log.Error().Err(err).Uint64("user_id", userID).Uint64("doc_id", docID).Msg("gRPC notify sync server of permission change failed")
		}
		return err
	}

	path := fmt.Sprintf("/internal/documents/%d/permission", docID)
	payload := UpdateRequest{userID, role}
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	resp, err := s.doRequest(ctx, http.MethodPut, path, headers, payload)
	if err != nil {
		log.Error().Err(err).Uint64("user_id", userID).Uint64("doc_id", docID).Msg("failed to notify sync server of permission change")
		return err
	}
	resp.Body.Close()
	return nil
}

// DELETE /internal/documents/:id
func (s *SyncClient) RemoveDocument(ctx context.Context, docID uint64) error {
	if s.grpcClient != nil {
		md := metadata.Pairs("x-internal-secret", config.AppConfig.SyncServerSecret)
		ctx = metadata.NewOutgoingContext(ctx, md)
		_, err := s.grpcClient.DeleteDocument(ctx, &syncpb.DocumentIDRequest{Id: docID})
		if err != nil {
			log.Error().Err(err).Uint64("doc_id", docID).Msg("gRPC notify sync server document removed failed")
		}
		return err
	}

	path := fmt.Sprintf("/internal/documents/%d", docID)
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	resp, err := s.doRequest(ctx, http.MethodDelete, path, headers, nil)
	if err != nil {
		log.Error().Err(err).Uint64("doc_id", docID).Msg("failed to notify sync server document removed")
		return err
	}
	resp.Body.Close()
	return nil
}
