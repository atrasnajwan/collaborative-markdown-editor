package sync

import (
	"bytes"
	"collaborative-markdown-editor/internal/config"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	log "github.com/rs/zerolog/log"
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
