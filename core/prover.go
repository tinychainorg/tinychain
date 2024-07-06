package core

import (
	_ "github.com/vocdoni/go-snark/parsers"
	_ "github.com/vocdoni/go-snark/prover"
	_ "github.com/vocdoni/go-snark/verifier"
)

// Recursion
// - https://hackmd.io/@soowon/gkr?utm_source=preview-mode&utm_medium=rec
// - https://github.com/Consensys/gnark/blob/v0.9.1/std/recursion/groth16/verifier.go
// Aggregation:
// - SnarkPack https://eprint.iacr.org/2021/529.pdf
// Poseidon
// - https://eprint.iacr.org/2019/458.pdf
