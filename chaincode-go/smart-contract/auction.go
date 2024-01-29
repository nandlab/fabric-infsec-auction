/*
SPDX-License-Identifier: Apache-2.0
*/

package auction

import (
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
)

// Auction data
type Auction struct {
	Status AuctionStatus `json:"status"`
	// TODO: Other parameters
	// ...
}

// *************************************************************
// TODO: Any struct that you need goes here

//**************************************************************

// create on auction
func (s *SmartContract) CreateAuction(ctx contractapi.TransactionContextInterface /* TODO: Your paameters go here */) error {

	// get ID of submitting client
	clientID, err := s.GetSubmittingClientIdentity(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	/* 	TODO: Your code goes here */

	return nil
}

// update auction status
func (s *SmartContract) UpdateAuctionStatus(ctx contractapi.TransactionContextInterface /* TODO: Insert here your parameters to update the auction state */) error {

	/* 	TODO: Your code goes here */

}

// make bid
func (s *SmartContract) Bid(ctx contractapi.TransactionContextInterface /* TODO: Insert your parameter to make a (private) bid here */) (string, error) {

	/* 	TODO: Your code goes here */

}

// reveal bid
func (s *SmartContract) OpenBid(ctx contractapi.TransactionContextInterface /* TODO: Insert your parameter to reveal a (private) bid here */) error {

	/* 	TODO: Your code goes here */

}

// close auction
func (s *SmartContract) EndAuction(ctx contractapi.TransactionContextInterface /* TODO: Insert your parameter to end an auction here */) error {

	/* 	TODO: Your code goes here */

}

// directly buy without waiting for the auction to end first
func (s *SmartContract) DirectBuy(ctx contractapi.TransactionContextInterface /* TODO: Insert your parameter to directly buy without waiting for the auction to close/end */) error {

	/* 	TODO: Your code goes here */

}

// feel free to define other functions / modify the above ones
// ...
