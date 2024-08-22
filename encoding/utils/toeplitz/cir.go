package toeplitz

import (
	"errors"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	gnark_fft "github.com/consensys/gnark-crypto/ecc/bn254/fr/fft"
)

type Circular struct {
	V       []fr.Element
	GnarkFs *gnark_fft.Domain
}

func NewCircular(v []fr.Element, gnarkFs *gnark_fft.Domain) *Circular {
	return &Circular{
		V:       v,
		GnarkFs: gnarkFs,
	}
}

// Matrix multiplication between a circular matrix and a vector using FFT
func (c *Circular) Multiply(x []fr.Element) ([]fr.Element, error) {
	if len(x) != len(c.V) {
		return nil, errors.New("dimension inconsistent")
	}
	n := len(x)

	colV := make([]fr.Element, n)
	for i := 0; i < n; i++ {
		colV[i] = c.V[(n-i)%n]
	}

	c.GnarkFs.FFT(x, gnark_fft.DIT)
	gnark_fft.BitReverse(x)

	c.GnarkFs.FFT(colV, gnark_fft.DIT)
	gnark_fft.BitReverse(colV)

	gnark_fft.BitReverse(x)
	gnark_fft.BitReverse(colV)

	u := make([]fr.Element, n)
	err := Hadamard(x, colV, u)
	if err != nil {
		return nil, err
	}
	// gnark_fft.BitReverse(u)

	c.GnarkFs.FFTInverse(u, gnark_fft.DIT)

	// Apply bit-reversal permutation
	gnark_fft.BitReverse(u)

	return u, nil
}

// Taking FFT on the circular matrix vector
func (c *Circular) GetFFTCoeff() ([]fr.Element, error) {
	n := len(c.V)

	colV := make([]fr.Element, n)
	for i := 0; i < n; i++ {
		colV[i] = c.V[(n-i)%n]
	}

	c.GnarkFs.FFT(colV, gnark_fft.DIT)
	gnark_fft.BitReverse(colV)
	return colV, nil
}

// Taking FFT on the circular matrix vector
func (c *Circular) GetCoeff() ([]fr.Element, error) {
	n := len(c.V)

	colV := make([]fr.Element, n)
	for i := 0; i < n; i++ {
		colV[i] = c.V[(n-i)%n]
	}
	return colV, nil
}

// Hadamard product between 2 vectors containing Fr elements
func Hadamard(a, b, u []fr.Element) error {
	if len(a) != len(b) {
		return errors.New("dimension inconsistent. Cannot do Hadamard Product on Fr")
	}

	for i := 0; i < len(a); i++ {
		u[i].Mul(&a[i], &b[i])
	}
	return nil
}
