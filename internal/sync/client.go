package sync

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type SyncClient struct {
	baseURL    string
	httpClient *http.Client
}

type Client interface {
	FetchDocumentState(ctx context.Context, docID uint64) ([]byte, error) 
	UpdateUserPermission(ctx context.Context, docID uint64, userID uint64, role string) error
	RemoveDocument(ctx context.Context, docID uint64) error 
}

func NewSyncClient(baseURL string) *SyncClient {
	return &SyncClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type StateResponse struct {
	Binary string `json:"binary"`
}

// call sync server to get current doc state
func (s *SyncClient) FetchDocumentState(ctx context.Context, docID uint64) ([]byte, error) {
	url := fmt.Sprintf(
		"%s/internal/documents/%d/state",
		s.baseURL,
		docID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"sync server fetch state error: status=%d body=%s",
			resp.StatusCode,
			string(b),
		)
	}

	var payload StateResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(payload.Binary)
}

type SyncPermissionRequest struct {
	UserID uint64 `json:"user_id"`
	Role   string `json:"role"`
}

func (s *SyncClient) UpdateUserPermission(
	ctx context.Context,
	docID uint64,
	userID uint64,
	role string,
) error {

	url := fmt.Sprintf(
		"%s/internal/documents/%d/permission",
		s.baseURL,
		docID,
	)

	payload := SyncPermissionRequest{
		UserID: userID,
		Role:   role,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("X-Internal-Secret", os.Getenv("SYNC_INTERNAL_SECRET"))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"sync server error: status=%d body=%s",
			resp.StatusCode,
			string(b),
		)
	}

	return nil
}

func (s *SyncClient) RemoveDocument(ctx context.Context, docID uint64) error {
	url := fmt.Sprintf(
		"%s/internal/documents/%d",
		s.baseURL,
		docID,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		url,
		nil,
	)
	if err != nil {
		return err
	}

	// req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("X-Internal-Secret", os.Getenv("SYNC_INTERNAL_SECRET"))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"sync server delete error: status=%d body=%s",
			resp.StatusCode,
			string(b),
		)
	}

	return nil
}