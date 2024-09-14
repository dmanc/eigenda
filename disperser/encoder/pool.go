package encoder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/Layr-Labs/eigenda/core"
	"github.com/Layr-Labs/eigenda/disperser"
	"github.com/Layr-Labs/eigenda/encoding"
)

type Route struct {
	MinSize       uint64 `json:"minSize"`
	TargetService string `json:"targetService"`
	Timeout       string `json:"timeout"`
}

type EncoderPoolConfig struct {
	Routes          []Route `json:"routes"`
	RefreshInterval string  `json:"refreshInterval"`
}

type EncoderPool struct {
	MinSize       uint64
	TargetService string
	Client        disperser.EncoderClient
	Timeout       time.Duration
}

type PoolManager struct {
	Pools           []EncoderPool
	refreshInterval time.Duration
}

func NewPoolManager(pools []EncoderPool, refreshInterval time.Duration) (*PoolManager, error) {
	for i, pool := range pools {
		client, err := NewEncoderClient(pool.TargetService, pool.Timeout)
		if err != nil {
			return nil, fmt.Errorf("error creating client for pool %d: %v", i, err)
		}
		pools[i].Client = client
	}

	// Sort pools by MinSize in descending order
	sort.Slice(pools, func(i, j int) bool {
		return pools[i].MinSize >= pools[j].MinSize
	})

	return &PoolManager{
		Pools:           pools,
		refreshInterval: refreshInterval,
	}, nil
}

func (pm *PoolManager) GetEncoderForSize(size uint64) (disperser.EncoderClient, time.Duration, error) {
	for _, pool := range pm.Pools {
		if size >= pool.MinSize {
			return pool.Client, pool.Timeout, nil
		}
	}
	return nil, time.Duration(0), fmt.Errorf("no suitable encoder found for size: %d", size)
}

func (pm *PoolManager) EncodeBlob(ctx context.Context, data []byte, encodingParams encoding.EncodingParams) (*encoding.BlobCommitments, *core.ChunksData, error) {
	size := uint64(len(data))
	encoder, _, err := pm.GetEncoderForSize(size)
	if err != nil {
		return nil, nil, err
	}
	return encoder.EncodeBlob(ctx, data, encodingParams)
}

func CreatePoolManagerFromConfig(config EncoderPoolConfig) (*PoolManager, error) {
	encoderPools := make([]EncoderPool, len(config.Routes))
	for i, route := range config.Routes {
		encoderPools[i] = EncoderPool{
			MinSize:       route.MinSize,
			TargetService: route.TargetService,
		}

		if route.Timeout != "" {
			timeout, err := time.ParseDuration(route.Timeout)
			if err != nil {
				return nil, fmt.Errorf("error parsing timeout: %v", err)
			}
			encoderPools[i].Timeout = timeout
		}
	}

	// Parse the refresh interval
	refreshInterval, err := time.ParseDuration(config.RefreshInterval)
	if err != nil {
		return nil, fmt.Errorf("error parsing refresh interval: %v", err)
	}

	return NewPoolManager(encoderPools, refreshInterval)
}

func ReadConfig(filename string) (EncoderPoolConfig, error) {
	// Read the JSON file
	data, err := os.ReadFile(filename)
	if err != nil {
		return EncoderPoolConfig{}, fmt.Errorf("error reading file: %v", err)
	}

	// Parse the JSON data
	var config EncoderPoolConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return EncoderPoolConfig{}, fmt.Errorf("error parsing JSON: %v", err)
	}

	return config, nil
}
