/*
SPDX-License-Identifier: Apache-2.0
*/

package auction

import (
	"crypto/x509"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func (s *VickreyAuctionContract) GetSubmittingClientIdentity(ctx contractapi.TransactionContextInterface) (*x509.Certificate, error) {

	cert, err := ctx.GetClientIdentity().GetX509Certificate()
	if err != nil {
		return nil, fmt.Errorf("failed to read clientID: %v", err)
	}
	return cert, nil
}

func intToByteArray(val int) []byte {
	arr := make([]byte, 4)
	arr[0] = byte(val)
	arr[1] = byte(val >> 8)
	arr[2] = byte(val >> 16)
	arr[3] = byte(val >> 24)
	return arr
}
