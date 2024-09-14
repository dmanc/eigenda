package encoder_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Layr-Labs/eigenda/core"
	"github.com/Layr-Labs/eigenda/disperser"
	"github.com/Layr-Labs/eigenda/disperser/encoder"
	"github.com/Layr-Labs/eigenda/encoding"
)

// Mock EncoderClient for testing
type mockEncoderClient struct {
	targetService string
}

func (m mockEncoderClient) EncodeBlob(ctx context.Context, data []byte, encodingParams encoding.EncodingParams) (*encoding.BlobCommitments, *core.ChunksData, error) {
	return &encoding.BlobCommitments{}, &core.ChunksData{}, nil
}

// Mock NewEncoderClient function for testing
func mockNewEncoderClient(addr string, timeout time.Duration) (disperser.EncoderClient, error) {
	return mockEncoderClient{targetService: addr}, nil
}

func TestReadConfigAndCreateEncoderPools(t *testing.T) {
	// Create a temporary JSON file with the provided configuration
	tmpfile, err := os.CreateTemp("", "config*.json")
	if err != nil {
		t.Fatalf("Error creating temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	configJSON := `{
		"routes": [
			{
				"minSize": 8388608,
				"targetService": "http://encoder-medium-svc:8080"
			},
			{
				"minSize": 16777216,
				"targetService": "http://encoder-large-svc:8080"
			},
			{
				"minSize": 0,
				"targetService": "http://encoder-small-svc:8080"
			}
		],
		"refreshInterval": "60s"
	}`

	if _, err := tmpfile.Write([]byte(configJSON)); err != nil {
		t.Fatalf("Error writing to temporary file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Error closing temporary file: %v", err)
	}

	// Call the function with the temporary file
	encoderPoolConfig, err := encoder.ReadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Error calling ReadConfig: %v", err)
	}

	// Create the expected EncoderPoolConfig
	expectedEncoderPoolConfig := encoder.EncoderPoolConfig{
		Routes: []encoder.Route{
			{
				MinSize:       8388608,
				TargetService: "http://encoder-medium-svc:8080",
				Timeout:       "",
			},
			{
				MinSize:       16777216,
				TargetService: "http://encoder-large-svc:8080",
				Timeout:       "",
			},
			{
				MinSize:       0,
				TargetService: "http://encoder-small-svc:8080",
				Timeout:       "",
			},
		},
		RefreshInterval: "60s",
	}

	// Compare the actual and expected EncoderPoolConfig
	if encoderPoolConfig.RefreshInterval != expectedEncoderPoolConfig.RefreshInterval {
		t.Fatalf("RefreshInterval mismatch: got %v, want %v", encoderPoolConfig.RefreshInterval, expectedEncoderPoolConfig.RefreshInterval)
	}

	for i, route := range encoderPoolConfig.Routes {
		if route.MinSize != expectedEncoderPoolConfig.Routes[i].MinSize {
			t.Fatalf("MinSize mismatch: got %v, want %v", route.MinSize, expectedEncoderPoolConfig.Routes[i].MinSize)
		}
		if route.TargetService != expectedEncoderPoolConfig.Routes[i].TargetService {
			t.Fatalf("TargetService mismatch: got %v, want %v", route.TargetService, expectedEncoderPoolConfig.Routes[i].TargetService)
		}
		if route.Timeout != expectedEncoderPoolConfig.Routes[i].Timeout {
			t.Fatalf("Timeout mismatch: got %v, want %v", route.Timeout, expectedEncoderPoolConfig.Routes[i].Timeout)
		}
	}
}
