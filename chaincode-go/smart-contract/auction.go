/*
SPDX-License-Identifier: Apache-2.0
*/

package auction

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"sort"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Vickrey auction smart contract
type VickreyAuctionContract struct {
	contractapi.Contract
}

/**************** AUCTION SELLER METHODS ****************/

// CreateAuction creates a new auction
func (s *VickreyAuctionContract) CreateAuction(ctx contractapi.TransactionContextInterface, auctionName string, directBuyPrice uint64) error {

	// get ID of submitting client
	clientID, errClientID := getSubmittingClientIdentity(ctx)
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
		Seller:         clientID.Raw,
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

	// Inform the users about the auction creation
	auctionSummaryErr :=
		setAuctionSummaryEvent(ctx, &AuctionSummary{
			Name:           auction.Name,
			Seller:         auction.Seller,
			Status:         auction.Status,
			DirectBuyPrice: auction.DirectBuyPrice,
			Result:         nil,
		})
	if auctionSummaryErr != nil {
		return fmt.Errorf("could not set auction summary event: %v", auctionSummaryErr)
	}

	return nil
}

// UpdateAuctionStatus updates the auction status (this can only be done by the auction seller)
func (s *VickreyAuctionContract) CloseAuction(ctx contractapi.TransactionContextInterface, auctionName string) error {

	// Get ID of submitting client
	clientID, errClientID := getSubmittingClientIdentity(ctx)
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
	if !reflect.DeepEqual(auction.Seller, clientID.Raw) {
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

	// Inform the users about the auction status change
	auctionSummaryErr :=
		setAuctionSummaryEvent(ctx, &AuctionSummary{
			Name:           auction.Name,
			Seller:         auction.Seller,
			Status:         auction.Status,
			DirectBuyPrice: auction.DirectBuyPrice,
			Result:         nil,
		})
	if auctionSummaryErr != nil {
		return fmt.Errorf("could not set auction summary event: %v", auctionSummaryErr)
	}

	return nil
}

// EndAuction determines the highest bidder and the hammer price
func (s *VickreyAuctionContract) EndAuction(ctx contractapi.TransactionContextInterface, auctionName string) error {
	// Get ID of submitting client
	clientID, errClientID := getSubmittingClientIdentity(ctx)
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
	if !reflect.DeepEqual(auction.Seller, clientID.Raw) {
		return fmt.Errorf("only the auction seller can end the auction")
	}

	// If the auction has already ended, do nothing
	if auction.Status == AuctionStatus(Ended) {
		return nil
	}

	// Build a mapping from the buyer (PEM certificate) to their highest bid
	buyerToBid := make(map[string]uint64)
	for i := range auction.Bids {
		bid := &auction.Bids[i]
		if bid.BidPrice == 0 {
			return fmt.Errorf("cannot end auction, because not all bids are revealed yet")
		}
		buyerCertPem := certDerToPem(bid.Buyer)
		if buyerCertPem == nil {
			return fmt.Errorf("could not convert certificate from DER to PEM format")
		}
		prevBid, exists := buyerToBid[*buyerCertPem]
		if !exists || bid.BidPrice > prevBid {
			buyerToBid[*buyerCertPem] = bid.BidPrice
		}
	}

	type BidPriceBuyerPair struct {
		BidPrice uint64
		Buyer    []byte
	}

	// Convert map to (BidPrice, Buyer) slice
	bidPriceToBuyer := make([]BidPriceBuyerPair, 0, len(buyerToBid))

	for buyer, bidPrice := range buyerToBid {
		buyerCertDer := certPemToDer(buyer)
		if buyerCertDer == nil {
			return fmt.Errorf("could not convert certificate from PEM to DER format")
		}
		bidPriceToBuyer = append(bidPriceToBuyer, BidPriceBuyerPair{
			BidPrice: bidPrice,
			Buyer:    buyerCertDer,
		})
	}

	// Sort bidders by descending bid price
	sort.Slice(bidPriceToBuyer, func(i int, j int) bool {
		return bidPriceToBuyer[i].BidPrice > bidPriceToBuyer[j].BidPrice
	})

	var auctionSummary *AuctionSummary = nil
	if len(bidPriceToBuyer) == 0 {
		// No bids submitted => no winner
		// Update auction state
		auction.HammerPrice = 0
		auction.Winner = nil
		auction.Status = AuctionStatus(Ended)

		// Set auction summary
		auctionSummary = &AuctionSummary{
			Name:           auction.Name,
			Seller:         auction.Seller,
			Status:         auction.Status,
			DirectBuyPrice: auction.DirectBuyPrice,
			Result: &AuctionResult{
				Winner:      nil,
				HammerPrice: 0,
				DirectBuy:   false,
			},
		}
	} else {
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
		winner := bidPriceToBuyer[winningCandidate.Uint64()].Buyer

		// Update auction state
		auction.HammerPrice = hammerPrice
		auction.Winner = winner
		auction.Status = AuctionStatus(Ended)

		// Set auction summary
		auctionSummary = &AuctionSummary{
			Name:           auction.Name,
			Seller:         auction.Seller,
			Status:         auction.Status,
			DirectBuyPrice: auction.DirectBuyPrice,
			Result: &AuctionResult{
				Winner:      auction.Winner,
				HammerPrice: auction.HammerPrice,
				DirectBuy:   false,
			},
		}
	}

	// Save new auction state
	errPutAuction := putAuction(ctx, auction)
	if errPutAuction != nil {
		return fmt.Errorf("could not save ended auction: %v", errPutAuction)
	}

	// Set auction summary event
	auctionSummaryErr := setAuctionSummaryEvent(ctx, auctionSummary)
	if auctionSummaryErr != nil {
		return fmt.Errorf("could not set auction summary event: %v", auctionSummaryErr)
	}

	return nil
}

/**************** AUCTION BIDDER METHODS ****************/

// Bid is called by a bidder to submit a hidden bid
// Apparently, it is not possible to pass a byte array to the contract,
// therefore the client has to send the hidden commit hex encoded.
func (s *VickreyAuctionContract) Bid(ctx contractapi.TransactionContextInterface, auctionName string, hiddenCommitHex string) error {
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
	clientID, errClientID := getSubmittingClientIdentity(ctx)
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
		Buyer:        clientID.Raw,
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

// OpenBid reveals the bid price of a bid
func (s *VickreyAuctionContract) OpenBid(ctx contractapi.TransactionContextInterface, auctionName string, bidPrice uint64, saltHex string) error {

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
	clientID, errClientID := getSubmittingClientIdentity(ctx)
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
		if reflect.DeepEqual(bid.Buyer, clientID.Raw) && bid.BidPrice == 0 {
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
func (s *VickreyAuctionContract) DirectBuy(ctx contractapi.TransactionContextInterface, auctionName string, price uint64) error {
	// Get ID of submitting client
	clientID, errClientID := getSubmittingClientIdentity(ctx)
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
	auction.Winner = clientID.Raw
	auction.Status = AuctionStatus(Ended)
	errPutAuction := putAuction(ctx, auction)
	if errPutAuction != nil {
		return fmt.Errorf("could not save ended auction: %v", errPutAuction)
	}

	// Inform the users about the auction result
	auctionSummaryErr :=
		setAuctionSummaryEvent(ctx, &AuctionSummary{
			Name:           auction.Name,
			Seller:         auction.Seller,
			Status:         auction.Status,
			DirectBuyPrice: auction.DirectBuyPrice,
			Result: &AuctionResult{
				Winner:      auction.Winner,
				HammerPrice: auction.HammerPrice,
				DirectBuy:   true,
			},
		})
	if auctionSummaryErr != nil {
		return fmt.Errorf("could not set auction summary event: %v", auctionSummaryErr)
	}

	return nil
}
