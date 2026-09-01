package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// blobscanHTTPClient bounds both requests fetchBlob makes (Blobscan metadata + the
// IPFS/storage content fetch). Plain http.Get has no timeout at all — a stalled
// Blobscan or IPFS gateway would hang this tool indefinitely. 30s is generous for a
// slow gateway while still guaranteeing the tool eventually fails instead of hanging.
var blobscanHTTPClient = &http.Client{Timeout: 30 * time.Second}

// blobscanBlobResponse mirrors the subset of Blobscan's GET /blobs/{versionedHash}
// response this tool needs. Verified live against api.sepolia.blobscan.com this
// session — the IPFS-referenced content is exactly the raw 131072-byte
// kzg4844.Blob (4096 limbs * 32 bytes), the same format prover/internal/da/blob.go
// already packs/unpacks. No new decoding format needed.
type blobscanBlobResponse struct {
	VersionedHash          string `json:"versionedHash"`
	DataStorageReferences []struct {
		Storage string `json:"storage"`
		URL     string `json:"url"`
	} `json:"dataStorageReferences"`
}

// fetchBlob retrieves the raw 131072-byte blob for a versioned hash from Blobscan.
// blobscanAPI is e.g. "https://api.sepolia.blobscan.com".
func fetchBlob(blobscanAPI, versionedHash string) ([blobLimbs * 32]byte, error) {
	var blob [blobLimbs * 32]byte

	metaURL := fmt.Sprintf("%s/blobs/%s", blobscanAPI, versionedHash)
	resp, err := blobscanHTTPClient.Get(metaURL)
	if err != nil {
		return blob, fmt.Errorf("query Blobscan for %s: %w", versionedHash, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return blob, fmt.Errorf("Blobscan returned %s for %s: %s", resp.Status, versionedHash, string(body))
	}

	var meta blobscanBlobResponse
	if err := json.Unmarshal(body, &meta); err != nil {
		return blob, fmt.Errorf("decode Blobscan response: %w", err)
	}
	if len(meta.DataStorageReferences) == 0 {
		return blob, fmt.Errorf("Blobscan has no storage reference for %s — blob may not be indexed yet", versionedHash)
	}

	// Prefer an ipfs reference if multiple are present; otherwise take the first.
	dataURL := meta.DataStorageReferences[0].URL
	for _, ref := range meta.DataStorageReferences {
		if ref.Storage == "ipfs" {
			dataURL = ref.URL
			break
		}
	}

	dataResp, err := blobscanHTTPClient.Get(dataURL)
	if err != nil {
		return blob, fmt.Errorf("fetch blob content from %s: %w", dataURL, err)
	}
	defer dataResp.Body.Close()
	raw, err := io.ReadAll(dataResp.Body)
	if err != nil {
		return blob, fmt.Errorf("read blob content: %w", err)
	}
	if len(raw) != len(blob) {
		return blob, fmt.Errorf("unexpected blob size from %s: got %d bytes, want %d", dataURL, len(raw), len(blob))
	}
	copy(blob[:], raw)
	return blob, nil
}
