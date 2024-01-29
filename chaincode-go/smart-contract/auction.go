/*
SPDX-License-Identifier: Apache-2.0
*/

package auction

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

// enum possible status: open, closed, ended
type AuctionStatus int

const (
	Open   AuctionStatus = iota // Buyers can send hidden bids or direct buy
	Closed                      // Buyers opens bids
	Ended                       // Auction is closed and winner is set
	NumberOfStatuses
)

// Bid data
type Bid struct {
	Buyer        string `json:"buyer"`        // the potential buyer's address
	BidPrice     uint   `json:"bidPrice"`     // 0 means hidden, later set the actual bid price during reveal
	HiddenCommit []byte `json:"hiddenCommit"` // A hash of the buyer's hidden bid data
}

// Auction data
type Auction struct {
	Name           string        `json:"name"`   // The auction name should be globally unique
	Seller         string        `json:"seller"` // The seller who opened this auction
	Status         AuctionStatus `json:"status"`
	DirectBuyPrice uint          `json:"directBuyPrice"` // A buyer can directly buy the item by paying at least this price
	Bids           []Bid         `json:"bids"`
}

//**************************************************************

// auctionKey gets a world state key from the auction name
func auctionKey(auctionName string) string {
	return fmt.Sprint("auction %s", auctionName)
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

// putAction saves the given auction in the contract world state
func putAuction(ctx contractapi.TransactionContextInterface, auction *Auction) error {
	auctionBin, err := json.Marshal(auction)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(auctionKey(auction.Name), auctionBin)
}

// CreateAuction creates a new auction
func (s *SmartContract) CreateAuction(ctx contractapi.TransactionContextInterface, auctionName string, directBuyPrice uint) error {

	// get ID of submitting client
	clientID, errClientID := s.GetSubmittingClientIdentity(ctx)
	if errClientID != nil {
		return fmt.Errorf("failed to get client identity: %v", errClientID)
	}

	// check if such an auction already exists
	auctionExists, errAuctionExist := doesAuctionExist(ctx, auctionName)
	if errAuctionExist != nil {
		return fmt.Errorf("failed to check if an auction with the same name already exists: %v", errAuctionExist)
	}
	if auctionExists {
		return fmt.Errorf("auction with the same name already exists")
	}

	// create new auction and save it
	auction := Auction{
		Name:           auctionName,
		Seller:         clientID,
		Status:         AuctionStatus(Open),
		DirectBuyPrice: directBuyPrice,
		Bids:           []Bid{},
	}

	errPutAction := putAuction(ctx, &auction)
	if errPutAction != nil {
		return fmt.Errorf("could not save the new auction in the world state: %v", errPutAction)
	}

	return nil
}

func isAuctionStatusValid(status AuctionStatus) bool {
	return uint(status) < uint(AuctionStatus(NumberOfStatuses))
}

// UpdateAuctionStatus updates the auction status (this can only be done by the auction seller)
func (s *SmartContract) UpdateAuctionStatus(ctx contractapi.TransactionContextInterface, auctionName string, newStatus AuctionStatus) error {

	// get ID of submitting client
	clientID, errClientID := s.GetSubmittingClientIdentity(ctx)
	if errClientID != nil {
		return fmt.Errorf("failed to get client identity: %v", errClientID)
	}

	// Get auction from world state
	auction, errGetAuction := getAuction(ctx, auctionName)
	if errGetAuction != nil {
		return fmt.Errorf("could not get the auction: %v", errGetAuction)
	}
	if auction == nil {
		return fmt.Errorf("auction not found")
	}

	// Check if the submitting client is the seller of the auction
	if auction.Seller != clientID {
		return fmt.Errorf("only the auction seller can update the auction status")
	}

	// Check newStatus validity
	if !isAuctionStatusValid(newStatus) {
		return fmt.Errorf("requested new auction status is not valid")
	}
	if uint(newStatus) < uint(auction.Status) {
		return fmt.Errorf("the status cannot be decreased")
	}

	if uint(newStatus) == uint(auction.Status) {
		// Nothing changed
		return nil
	}

	// Finally update the auction status
	auction.Status = newStatus
	errPutAction := putAuction(ctx, auction)
	if errPutAction != nil {
		return fmt.Errorf("failed to save the updated auction")
	}

	return nil
}

// make bid
func (s *SmartContract) Bid(ctx contractapi.TransactionContextInterface, auctionName string, hiddenCommit []byte) error {
	// get ID of submitting client
	clientID, errClientID := s.GetSubmittingClientIdentity(ctx)
	if errClientID != nil {
		return fmt.Errorf("failed to get client identity: %v", errClientID)
	}

	// Get auction from world state
	auction, errGetAuction := getAuction(ctx, auctionName)
	if errGetAuction != nil {
		return fmt.Errorf("could not get the auction: %v", errGetAuction)
	}
	if auction == nil {
		return fmt.Errorf("auction not found")
	}

	// Add bid to auction
	auction.Bids = append(auction.Bids, Bid{
		Buyer:        clientID,
		BidPrice:     0,
		HiddenCommit: hiddenCommit,
	})

	// Save updated auction
	errPutAction := putAuction(ctx, auction)
	if errPutAction != nil {
		return fmt.Errorf("could not save the updated auction: %v", errPutAction)
	}

	return nil
}

// reveal bid
func (s *SmartContract) OpenBid(ctx contractapi.TransactionContextInterface, auctionName string, salt []byte, bidPrice uint) error {
	if bidPrice == 0 {
		return fmt.Errorf("bid price cannot be zero")
	}

	// get ID of submitting client
	clientID, errClientID := s.GetSubmittingClientIdentity(ctx)
	if errClientID != nil {
		return fmt.Errorf("failed to get client identity: %v", errClientID)
	}

	// Get auction from world state
	auction, errGetAuction := getAuction(ctx, auctionName)
	if errGetAuction != nil {
		return fmt.Errorf("could not get the auction: %v", errGetAuction)
	}
	if auction == nil {
		return fmt.Errorf("auction not found")
	}

	for i := range auction.Bids {
		bid := &auction.Bids[i]
		if bid.Buyer == clientID && bid.BidPrice == 0 {
			// TODO
		}
	}

	return nil
}

// close auction
func (s *SmartContract) EndAuction(ctx contractapi.TransactionContextInterface /* TODO: Insert your parameter to end an auction here */) error {

	/* 	TODO: Your code goes here */
	return nil
}

// directly buy without waiting for the auction to end first
func (s *SmartContract) DirectBuy(ctx contractapi.TransactionContextInterface /* TODO: Insert your parameter to directly buy without waiting for the auction to close/end */) error {

	/* 	TODO: Your code goes here */
	return nil
}
