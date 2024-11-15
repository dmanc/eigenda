//go:build icicle

package icicle

import (
	"log"
	"time"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/ingonyama-zk/icicle/v3/wrappers/golang/core"
	icicle_bn254 "github.com/ingonyama-zk/icicle/v3/wrappers/golang/curves/bn254"
	"github.com/ingonyama-zk/icicle/v3/wrappers/golang/curves/bn254/msm"
	"github.com/ingonyama-zk/icicle/v3/wrappers/golang/runtime"
)

// SetupMsmG1 initializes the MSM configuration for G1 points.
func SetupMsmG1(rowsG1 [][]bn254.G1Affine) (core.HostOrDeviceSlice, core.MSMConfig, runtime.EIcicleError) {
	rowsG1Icicle := make([]icicle_bn254.Affine, 0)

	for _, row := range rowsG1 {
		rowsG1Icicle = append(rowsG1Icicle, BatchConvertGnarkAffineToIcicleAffine(row)...)
	}

	// Setup MSM configuration
	cfgBn254 := core.GetDefaultMSMConfig()
	cfgBn254.IsAsync = true
	cfgBn254.PrecomputeFactor = 8

	streamBn254, err := runtime.CreateStream()
	if err != runtime.Success {
		return nil, cfgBn254, err
	}
	cfgBn254.StreamHandle = streamBn254

	// Precompute points
	start := time.Now()
	log.Println("precompute start")
	rowsG1IcicleCopy := core.HostSliceFromElements[icicle_bn254.Affine](rowsG1Icicle)
	var precomputeOut core.DeviceSlice
	precomputeSize := len(rowsG1Icicle) * int(cfgBn254.PrecomputeFactor)

	_, err = precomputeOut.MallocAsync(rowsG1IcicleCopy[0].Size(), precomputeSize, streamBn254)
	if err != runtime.Success {
		return nil, cfgBn254, err
	}

	err = msm.PrecomputeBases(rowsG1IcicleCopy, &cfgBn254, precomputeOut)
	if err != runtime.Success {
		precomputeOut.FreeAsync(streamBn254)
		return nil, cfgBn254, err
	}
	log.Println("precompute done", time.Since(start))

	return precomputeOut, cfgBn254, runtime.Success
}
