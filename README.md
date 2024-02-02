## Fabric Vickrey Auction

This is a tutorial on how to deploy and interact with the Vickrey auction contract on the [Hyperledger Fabric](https://www.hyperledger.org/projects/fabric) blockchain. This tutorial is adjusted from the InfSec homework template.

## Deploy the chaincode

We set the `TESTNETDIR` variable to the directory of the test-network.
```
TESTNETDIR=~/fabric-samples/test-network
```

If the test network is already running, run the following command to bring the network down and start from a clean initial state.
```
"${TESTNETDIR}/network.sh" down
```

You can then run the following command to deploy a new network.
```
"${TESTNETDIR}/network.sh" up createChannel -ca
```

Run the following command to deploy the auction smart contract.
```
"${TESTNETDIR}/network.sh" deployCC -ccn auction -ccp chaincode-go/ -ccl go
```

## Install the application dependencies

We will run an auction using a series of Node.js applications. Go to `application-javascript` in the project directory.
```
cd application-javascript
```

From this directory, run the following command to download the application dependencies if you have not done so already:
```
npm install
```

## Register and enroll the application identities

We use the following script to register an admin, an auction seller and three bidders in the org1 organization:
```
node ./enrollAdmin.js org1
for u in seller bidder1 bidder2 bidder3 ; do
node ./registerEnrollUser.js org1 "$u"
done
```

## Run unit tests
We can run the unit test with `npm run test`. It will simulate an auction and check if the winner and the hammer price at the end are correct. The unit test code can be found in `test/auctionTest.js`, it can be used as an example of how a user can interact with the auction from JavaScript.

## Command line interface
The JavaScript files can also be called from the console to interact with the contract.
The auction seller can execute the following files:
```
# Create an auction
# Optionally, a directBuyPrice can be given.
node ./createAuction.js org user auctionName [directBuyPrice]

# Close the auction, so that no further bids can be submitted
node ./closeAuction.js org user auctionName

# End the auction and determine the winner
node ./endAuction.js org user auctionName
```
The bidders can do the following:
```
# Submit a bid secretly
# It prints a secret salt which should be saved for later
node ./submitBid.js org user auctionName bidPrice

# Reveal the bid using the salt generated before
node ./openBid.js org user auctionName bidPrice salt

# Directly buy the item for the specified price (>= directBuyPrice)
node ./directBuy.js org user auctionName price
```

## Clean up

When your are done using the auction smart contract, you can bring down the network and clean up the environment. In the `application-javascript` directory, run the following command to remove the wallets used to run the applications:
```
rm -rf wallet
```

You can then bring down the test network:
````
"${TESTNETDIR}/network.sh" down
````
