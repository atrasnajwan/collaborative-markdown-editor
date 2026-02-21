package sync

import (
	"bytes"
	"collaborative-markdown-editor/internal/config"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type SyncClient struct {
	httpClient *http.Client
}

type Client interface {
	FetchDocumentState(ctx context.Context, docID uint64) ([]byte, error) 
	UpdateUserPermission(ctx context.Context, docID uint64, userID uint64, role string) error
	RemoveDocument(ctx context.Context, docID uint64) error 
}

func NewSyncClient() *SyncClient {
	return &SyncClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (s *SyncClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
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

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Internal-Secret", config.AppConfig.SyncServerSecret)

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

type StateResponse struct {
	Binary string `json:"binary"`
}

// GET /internal/documents/:id/state
func (s *SyncClient) FetchDocumentState(ctx context.Context, docID uint64) ([]byte, error) {
    path := fmt.Sprintf("/internal/documents/%d/state", docID)
    resp, err := s.doRequest(ctx, http.MethodGet, path, nil)
    if err != nil {
        log.Printf("[SYNC SERVER ERROR] Failed to get last state doc %d", docID)
        return nil, err
    }
    defer resp.Body.Close()

    var payload StateResponse
    if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
        return nil, err
    }
    return base64.StdEncoding.DecodeString(payload.Binary)
}

type UpdateRequest struct {
	UserID uint64 `json:"user_id"`
	Role string `json:"role"`
}

// PUT /internal/documents/:id/permission
func (s *SyncClient) UpdateUserPermission(ctx context.Context, docID, userID uint64, role string) error {
    path := fmt.Sprintf("/internal/documents/%d/permission", docID)
    payload := UpdateRequest{userID, role}
    
    resp, err := s.doRequest(ctx, http.MethodPut, path, payload)
    if err != nil {
        log.Printf("[SYNC SERVER ERROR] Failed to notify user %d role on doc %d is changed!", userID, docID)
        return err
    }
    resp.Body.Close()
    return nil
}

// DELETE /internal/documents/:id
func (s *SyncClient) RemoveDocument(ctx context.Context, docID uint64) error {
    path := fmt.Sprintf("/internal/documents/%d", docID)
    resp, err := s.doRequest(ctx, http.MethodDelete, path, nil)
    if err != nil {
        log.Printf("[SYNC SERVER ERROR] Failed to notify user that doc %d is removed!", docID)
        return err
    }
    resp.Body.Close()
    return nil
}