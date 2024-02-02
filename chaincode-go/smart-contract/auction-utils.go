package auction

import (
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"golang.org/x/crypto/sha3"
)

// auctionKey gets a world state key from the auction name
func auctionKey(auctionName string) string {
	return fmt.Sprintf("auction %s", auctionName)
}

// doesAuctionExist checks if an auction with the given name exists in the world state
func doesAuctionExist(ctx contractapi.TransactionContextInterface, auctionName string) (bool, error) {
	auctionBin, err := ctx.GetStub().GetState(auctionKey(auctionName))
	if err != nil {
		return false, err
	}
	exists := auctionBin != nil
	return exists, nil
}

// getAuction retrieves the auction with the given name from the world state
func getAuction(ctx contractapi.TransactionContextInterface, auctionName string) (*Auction, error) {
	auctionBin, errGetState := ctx.GetStub().GetState(auctionKey(auctionName))
	if errGetState != nil {
		return nil, errGetState
	}
	var auction Auction
	err := json.Unmarshal(auctionBin, &auction)
	if err != nil {
		return nil, err
	}
	return &auction, nil
}

// putAuction saves the given auction in the contract world state
func putAuction(ctx contractapi.TransactionContextInterface, auction *Auction) error {
	auctionBin, err := json.Marshal(auction)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(auctionKey(auction.Name), auctionBin)
}

// setAuctionSummaryEvent sets an event about the current auction status which can be received by contract users
func setAuctionSummaryEvent(ctx contractapi.TransactionContextInterface, auctionSummary *AuctionSummary) error {
	if auctionSummary == nil {
		return fmt.Errorf("auctionSummary cannot be nil")
	}
	auctionSummaryBin, err := json.Marshal(auctionSummary)
	if err != nil {
		return err
	}
	return ctx.GetStub().SetEvent(auctionKey(auctionSummary.Name), auctionSummaryBin)
}

// hashBid hashes a bid
// It takes a random salt and the client's ID (X.509 certificate) into account
func hashBid(clientCert *x509.Certificate, bidPrice uint64, salt []byte) ([]byte, error) {
	shake := sha3.NewShake256()
	bidPriceBytes := [8]byte{}
	binary.BigEndian.PutUint64(bidPriceBytes[:], bidPrice)
	for _, data := range [][]byte{clientCert.Raw, bidPriceBytes[:], salt} {
		_, errShakeWrite := shake.Write(data)
		if errShakeWrite != nil {
			return nil, fmt.Errorf("failed to write data to SHAKE: %v", errShakeWrite)
		}
	}
	hash := make([]byte, 64)
	_, errShakeRead := shake.Read(hash)
	if errShakeRead != nil {
		return nil, fmt.Errorf("failed to read data from SHAKE: %v", errShakeRead)
	}
	return hash, nil
}
