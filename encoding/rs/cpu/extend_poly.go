package cpu

import (
	"github.com/Layr-Labs/eigenda/encoding"
	"github.com/Layr-Labs/eigenda/encoding/fft"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	gnark_fft "github.com/consensys/gnark-crypto/ecc/bn254/fr/fft"
)

type RsCpuComputeDevice struct {
	Fs *fft.FFTSettings

	GnarkFs *gnark_fft.Domain

	encoding.EncodingParams
}

// Encoding Reed Solomon using FFT
func (g *RsCpuComputeDevice) ExtendPolyEval(coeffs []fr.Element) ([]fr.Element, error) {
	g.GnarkFs.FFT(coeffs, gnark_fft.DIT)
	gnark_fft.BitReverse(coeffs)
	return coeffs, nil
}
