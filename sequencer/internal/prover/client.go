package prover

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client communicates with the Prover API
type Client struct {
	apiURL     string
	httpClient *http.Client
}

// BatchResponse represents the API response from prover (matches prover's BatchResponse)
type BatchResponse struct {
	BatchNumber  uint64   `json:"batchNumber"`
	BatchHash    string   `json:"batchHash"`
	OldStateRoot string   `json:"oldStateRoot"`
	NewStateRoot string   `json:"newStateRoot"`
	Submitter    string   `json:"submitter"`
	Timestamp    uint64   `json:"timestamp"`
	VerifiedAt   uint64   `json:"verifiedAt"`
	Status       uint8    `json:"status"`
	TxHash       string   `json:"txHash,omitempty"`
	TxHashes     []string `json:"txHashes,omitempty"`
}

// NewClient creates a new prover API client
func NewClient(apiURL string) *Client {
	return &Client{
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetLatestProvenBatch queries the prover for the latest proven batch number
func (c *Client) GetLatestProvenBatch() (uint64, error) {
	url := fmt.Sprintf("%s/batches/latest", c.apiURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to query prover API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No proven batches yet
		return 0, nil
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("prover API returned status %d", resp.StatusCode)
	}

	var result BatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.BatchNumber, nil
}
