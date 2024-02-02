/*
SPDX-License-Identifier: Apache-2.0
*/

package auction

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"sort"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"golang.org/x/crypto/sha3"
)

// This contract implements a Vickrey auction
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
	Buyer        string `json:"buyer"`    // the potential buyer's address
	BidPrice     uint64 `json:"bidPrice"` // 0 means hidden, later set the actual bid price during reveal
	HiddenCommit []byte `json:"hiddenCommit"`
	/*
		HiddenCommit is the 64 byte SHAKE256 output of (clientCert, bidPrice, salt)
		* clientCert is the X.509 client certificate in DER format
		* the bidPrice is a big endian encoded 64 bit integer
		* salt should be at least 64 bytes long
	*/
}

// Auction data
type Auction struct {
	Name           string        `json:"name"`   // The auction name should be globally unique
	Seller         string        `json:"seller"` // The seller who opened this auction
	Status         AuctionStatus `json:"status"`
	DirectBuyPrice uint64        `json:"directBuyPrice"` // A buyer can directly buy the item by paying at least this price (0 means disabled)
	Bids           []Bid         `json:"bids"`
	Winner         *string       `json:"winner"`
	HammerPrice    uint64        `json:"hammerPrice"`
}

//**************************************************************

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

/**************** AUCTION SELLER METHODS ****************/

// CreateAuction creates a new auction
func (s *SmartContract) CreateAuction(ctx contractapi.TransactionContextInterface, auctionName string, directBuyPrice uint64) error {

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
		Winner:         nil,
		HammerPrice:    0,
	}

	errPutAuction := putAuction(ctx, &auction)
	if errPutAuction != nil {
		return fmt.Errorf("could not save the new auction in the world state: %v", errPutAuction)
	}

	return nil
}

// UpdateAuctionStatus updates the auction status (this can only be done by the auction seller)
func (s *SmartContract) CloseAuction(ctx contractapi.TransactionContextInterface, auctionName string) error {

	// Get ID of submitting client
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

	// If auction is already closed, do nothing
	if auction.Status != AuctionStatus(Open) {
		return nil
	}

	// Change auction status from open to closed
	auction.Status = AuctionStatus(Closed)
	errPutAuction := putAuction(ctx, auction)
	if errPutAuction != nil {
		return fmt.Errorf("failed to save the updated auction")
	}

	return nil
}

// EndAuction determines the highest bidder and the hammer price
func (s *SmartContract) EndAuction(ctx contractapi.TransactionContextInterface, auctionName string) error {
	// Get ID of submitting client
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
		return fmt.Errorf("only the auction seller can end the auction")
	}

	// If the auction has already ended, do nothing
	if auction.Status == AuctionStatus(Ended) {
		return nil
	}

	// Build a mapping from the buyer to their highest bid
	buyerToBid := make(map[string]uint64)
	for i := range auction.Bids {
		bid := &auction.Bids[i]
		if bid.BidPrice == 0 {
			return fmt.Errorf("cannot end auction, because not all bids are revealed yet")
		}
		prevBid, exists := buyerToBid[bid.Buyer]
		if !exists || bid.BidPrice > prevBid {
			buyerToBid[bid.Buyer] = bid.BidPrice
		}
	}

	type BidPriceBuyerPair struct {
		BidPrice uint64
		Buyer    string
	}

	// Convert map to (BidPrice, Buyer) slice
	bidPriceToBuyer := make([]BidPriceBuyerPair, 0, len(buyerToBid))

	for buyer, bidPrice := range buyerToBid {
		bidPriceToBuyer = append(bidPriceToBuyer, BidPriceBuyerPair{
			BidPrice: bidPrice,
			Buyer:    buyer,
		})
	}

	// Sort bidders by descending bid price
	sort.Slice(bidPriceToBuyer, func(i int, j int) bool {
		return bidPriceToBuyer[i].BidPrice > bidPriceToBuyer[j].BidPrice
	})

	if len(bidPriceToBuyer) == 0 {
		// No bids submitted => no winner
		fmt.Printf("No bids were submitted, auction ended without a winner\n")
		return nil
	}

	// Determine hammer price
	highestPrice := bidPriceToBuyer[0].BidPrice
	hammerPrice := highestPrice
	if len(bidPriceToBuyer) > 1 {
		hammerPrice = bidPriceToBuyer[1].BidPrice
	}

	// If there are multiple bidders with the same highest bid, one is chosen at random
	// Potential problem: if there are multiple endorsers, their outcomes might not match
	numberOfCandidates := uint(0)
	for i := range bidPriceToBuyer {
		if bidPriceToBuyer[i].BidPrice < highestPrice {
			break
		}
		numberOfCandidates += 1
	}
	numberOfCandidatesBigInt := new(big.Int).SetUint64(uint64(numberOfCandidates))
	winningCandidate, errRand := rand.Int(rand.Reader, numberOfCandidatesBigInt)
	if errRand != nil {
		return fmt.Errorf("could not get a random number: %v", errRand)
	}

	if !winningCandidate.IsUint64() {
		return fmt.Errorf("winning candidate index cannot be represented as a uint64")
	}
	winner := &bidPriceToBuyer[winningCandidate.Uint64()].Buyer

	// End the auction
	auction.HammerPrice = hammerPrice
	auction.Winner = winner
	auction.Status = AuctionStatus(Ended)
	errPutAuction := putAuction(ctx, auction)
	if errPutAuction != nil {
		return fmt.Errorf("could not save ended auction: %v", errPutAuction)
	}

	// Annouce the auction winner
	fmt.Printf("Auction winner is: %s\n", *winner)
	fmt.Printf("Item sold for: %d\n", hammerPrice)

	return nil
}

/**************** AUCTION BIDDER METHODS ****************/

// Bid is called by a bidder to submit a hidden bid
func (s *SmartContract) Bid(ctx contractapi.TransactionContextInterface, auctionName string, hiddenCommitHex string) error {
	// Decode hidden commit
	hiddenCommit, errDecode := hex.DecodeString(hiddenCommitHex)
	if errDecode != nil {
		return fmt.Errorf("could not decode hidden commit: %v", errDecode)
	}

	// The hiddenCommit should be a 512 bit long hash
	if len(hiddenCommit) != 64 {
		return fmt.Errorf("hiddenCommit is not 512 bit long")
	}

	// Get ID of submitting client
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

	// Can only submit new bid while auction is open
	if auction.Status != AuctionStatus(Open) {
		return fmt.Errorf("auction is closed")
	}

	// Add bid to auction
	auction.Bids = append(auction.Bids, Bid{
		Buyer:        clientID,
		BidPrice:     0,
		HiddenCommit: hiddenCommit,
	})

	// Save updated auction
	errPutAuction := putAuction(ctx, auction)
	if errPutAuction != nil {
		return fmt.Errorf("could not save the updated auction: %v", errPutAuction)
	}

	return nil
}

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

// OpenBid reveals the bid price of a bid
func (s *SmartContract) OpenBid(ctx contractapi.TransactionContextInterface, auctionName string, bidPrice uint64, saltHex string) error {

	// Check if the bidPrice is reasonable
	if bidPrice == 0 {
		return fmt.Errorf("bid price cannot be zero")
	}

	// Decode hidden commit
	salt, errSaltDecode := hex.DecodeString(saltHex)
	if errSaltDecode != nil {
		return fmt.Errorf("could not decode salt: %v", errSaltDecode)
	}

	// Check salt minimum requirements
	if len(salt) < 64 {
		return fmt.Errorf("salt should be at least 64 bytes long")
	}

	// Get ID of submitting client
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

	clientCert, errCert := ctx.GetClientIdentity().GetX509Certificate()
	if errCert != nil {
		return fmt.Errorf("could not get client certificate")
	}

	bidHash, errHashBid := hashBid(clientCert, bidPrice, salt)
	if errHashBid != nil {
		return errHashBid
	}

	// Iterate over the bids and try to reveal any
	for i := range auction.Bids {
		bid := &auction.Bids[i]
		if bid.Buyer == clientID && bid.BidPrice == 0 {
			// Check if hidden commit matches the hash
			if reflect.DeepEqual(bid.HiddenCommit, bidHash) {
				// The bid price is revealed
				bid.BidPrice = bidPrice
			}
		}
	}

	// Save the updated auction
	errPutAuction := putAuction(ctx, auction)
	if errPutAuction != nil {
		return fmt.Errorf("could not save the updated auction: %v", errPutAuction)
	}

	return nil
}

// DirectBuy: The buyer should pay at least auction.DirectBuyPrice to directly purchase the auction item
func (s *SmartContract) DirectBuy(ctx contractapi.TransactionContextInterface, auctionName string, price uint64) error {
	// Get ID of submitting client
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

	// Check auction status
	if auction.Status == AuctionStatus(Ended) {
		return fmt.Errorf("auction has already ended")
	}

	// Check direct buy validity
	if auction.DirectBuyPrice == 0 {
		return fmt.Errorf("direct buy is disabled for this auction")
	}
	if price < auction.DirectBuyPrice {
		return fmt.Errorf("payment amount not sufficient for a direct buy")
	}

	// End the auction
	auction.HammerPrice = price
	auction.Winner = &clientID
	auction.Status = AuctionStatus(Ended)
	errPutAuction := putAuction(ctx, auction)
	if errPutAuction != nil {
		return fmt.Errorf("could not save ended auction: %v", errPutAuction)
	}

	// Announce direct buy winner
	fmt.Printf("Auction item purchased directly by: %s\n", clientID)
	fmt.Printf("Item sold for: %d\n", price)

	return nil
}
