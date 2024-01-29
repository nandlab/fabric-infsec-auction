## InfSec auction

## Deploy the chaincode

Change into the test network directory.
```
cd fabric-samples/test-network
```

If the test network is already running, run the following command to bring the network down and start from a clean initial state.
```
./network.sh down
```

You can then run the following command to deploy a new network.
```
./network.sh up createChannel -ca
```

Run the following command to deploy the auction smart contract.
```
./network.sh deployCC -ccn auction -ccp ../infsec_auction/chaincode-go/ -ccl go
```


## Install the application dependencies

We will run an auction using a series of Node.js applications. Change into the `application-javascript` directory:
```
cd infsec_auction/application-javascript
```

From this directory, run the following command to download the application dependencies if you have not done so already:
```
npm install
```

## Register and enroll the application identities

To interact with the network, you will need to enroll at least one Certificate Authority administrator. You can use the `enrollAdmin.js` program for this task. Run the following command to enroll the e.g. Org1 admin:
```
node enrollAdmin.js org1
```

We can use CA admins to register and enroll the identities of the seller that will create the auction and the bidders. Run the following command to register and enroll the seller identity that will create the auction. Here, the seller will belong to Org1.
```
node registerEnrollUser.js org1 seller
```
You should see the logs of the seller wallet being created as well. 

Furthermore, run the following commands to e.g. register and enroll two bidders from Org1:
```
node registerEnrollUser.js org1 bidder1
node registerEnrollUser.js org1 bidder2
```

## Clean up

When your are done using the auction smart contract, you can bring down the network and clean up the environment. In the `infsec_auction/application-javascript` directory, run the following command to remove the wallets used to run the applications:
```
rm -rf wallet
```

You can then navigate to the test network directory and bring down the network:
````
cd ../../test-network/
./network.sh down
````
