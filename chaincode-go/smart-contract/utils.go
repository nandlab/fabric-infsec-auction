/*
SPDX-License-Identifier: Apache-2.0
*/

package auction

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func getSubmittingClientIdentity(ctx contractapi.TransactionContextInterface) (*x509.Certificate, error) {
	cert, err := ctx.GetClientIdentity().GetX509Certificate()
	if err != nil {
		return nil, fmt.Errorf("failed to read clientID: %v", err)
	}
	return cert, nil
}

// certDerToPem converts a certificate from binary DER to PEM text format
func certDerToPem(derCert []byte) *string {
	pemCertBytes := pem.EncodeToMemory(&pem.Block{
		Type:    "CERTIFICATE",
		Headers: nil,
		Bytes:   derCert,
	})
	if pemCertBytes == nil {
		return nil
	}
	pemCert := string(pemCertBytes)
	return &pemCert
}

// certPemToDer converts a certificate from PEM text to binary DER format
func certPemToDer(pemCert string) []byte {
	block, _ := pem.Decode([]byte(pemCert))
	if block == nil {
		return nil
	}
	return block.Bytes
}

func intToByteArray(val int) []byte {
	arr := make([]byte, 4)
	arr[0] = byte(val)
	arr[1] = byte(val >> 8)
	arr[2] = byte(val >> 16)
	arr[3] = byte(val >> 24)
	return arr
}
