/*
SPDX-License-Identifier: Apache-2.0
*/

package auction

// enum possible status: open, closed, ended
type AuctionStatus int

const (
	Open   AuctionStatus = iota // Buyers can send hidden bids or direct buy
	Closed                      // Buyers opens bids
	Ended                       // Auction is closed and winner is set
)

// Bid data
type Bid struct {
	Buyer        []byte `json:"buyer"`    // the certificate of the potential buyer
	BidPrice     uint64 `json:"bidPrice"` // 0 means hidden, later set the actual bid price during reveal
	HiddenCommit []byte `json:"hiddenCommit"`
	/*
		HiddenCommit is the 64 byte SHAKE256 output of (clientCert, bidPrice, salt)
		* clientCert is the X.509 client certificate in DER format
		* the bidPrice is a big endian encoded 64 bit integer
		* salt should be at least 64 bytes long
		It can be computed using the hashBid function.
	*/
}

type Auction struct {
	Name           string        `json:"name"`   // The auction name should be globally unique
	Seller         []byte        `json:"seller"` // The seller who opened this auction
	Status         AuctionStatus `json:"status"`
	DirectBuyPrice uint64        `json:"directBuyPrice"` // A buyer can directly buy the item by paying at least this price (0 means disabled)
	Bids           []Bid         `json:"bids"`
	Winner         []byte        `json:"winner"`
	HammerPrice    uint64        `json:"hammerPrice"`
}

// Auction status information, which will be presented to the users in an event
type AuctionSummary struct {
	Name           string         `json:"name"`
	Seller         []byte         `json:"seller"`
	Status         AuctionStatus  `json:"status"`
	DirectBuyPrice uint64         `json:"directBuyPrice"`
	Result         *AuctionResult `json:"result"` // It is set when the auction ends
}

type AuctionResult struct {
	Winner      []byte `json:"winner"`
	DirectBuy   bool   `json:"directBuy"` // If true, the winner bought directly, otherwise they were the highest bidder
	HammerPrice uint64 `json:"hammerPrice"`
}
