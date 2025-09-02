package utils

import (
	"crypto/sha256"
	"encoding/pem"
)

func GetCertFingerprint(pemData []byte) [32]byte {
	if block, _ := pem.Decode(pemData); block != nil {
		pemData = block.Bytes
	}
	return sha256.Sum256(pemData)
}
