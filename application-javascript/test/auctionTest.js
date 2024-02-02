const assert = require('assert');

/* Auction Seller */
const { createAuction } = require("../createAuction.js");
const { closeAuction } = require("../closeAuction.js");
const { endAuction } = require("../endAuction.js");

/* Auction Bidders */
const { submitBid } = require("../submitBid.js");
const { openBid } = require("../openBid.js");

const { Wallets, Gateway } = require('fabric-network');
const path = require('node:path');
const process = require('node:process');
const { buildCCPOrg1, buildCCPOrg2, buildWallet, prettyJSONString } = require('/home/fabric-user/fabric-samples/test-application/javascript/AppUtil.js');
const { randomUUID } = require('node:crypto');
const { uint8ArrayToHex } = require('../encode-utils.js');

function zip(...arr) {
	return Array(Math.max(...arr.map(a => a.length))).fill().map((_,i) => arr.map(a => a[i]));
}

const myChannel = 'mychannel';
const myChaincodeName = 'auction';

describe('Auction', function () {
  	it('simulate auction', async function () {
		this.timeout(120000);
		let contract = null;
		let gateway = null;
		let contractListener = null;

		try {
			const org = "org1";
			const seller = "seller";
			const bidders = ["bidder1", "bidder2", "bidder3"];
			const bids = [10n, 40n, 20n];
			const expectedWinner = 1;
			const expectedHammerPrice = 20n;
			const auctionName = "testAuction_" + randomUUID();
			const directBuyPrice = 1000;
		
			console.log(`Auction name: ${auctionName}`);

			let ccp = null;
			let walletPath = null;
			if (org === 'org1') {
				ccp = buildCCPOrg1();
				walletPath = path.join(process.cwd(), 'wallet/org1');
			}
			else if (org === 'org2') {
				ccp = buildCCPOrg2();
				walletPath = path.join(process.cwd(), 'wallet/org2');
			}
			else {
				console.error('Org must be org1 or org2 ...');
				process.exit(1);
			}
			const wallet = await buildWallet(Wallets, walletPath);

			// Establish one persistent blockchain connection to listen for auction summary events
			gateway = new Gateway();
			await gateway.connect(ccp,
				{ wallet: wallet, identity: "admin", discovery: { enabled: true, asLocalhost: true } });
		
			const network = await gateway.getNetwork(myChannel);
			contract = network.getContract(myChaincodeName);
			const auctionKey = `auction ${auctionName}`;

			let resultPromiseResolver = null;
			let auctionResult = null

			contractListener = (event) => {
				if (event.eventName == auctionKey) {
					const auctionSummary = JSON.parse(event.payload.toString("utf8"));
					console.log(`Auction status: ${auctionSummary.status}`);
					auctionResult = auctionSummary.result;
					if (resultPromiseResolver !== null) {
						resultPromiseResolver(auctionResult);
					}
				}
			};

			contract.addContractListener(contractListener);

			// Create auction
			console.log("Creating auction...");
			await createAuction(ccp, wallet, seller, auctionName, directBuyPrice);
			console.log("Done.");

			let salts = [];

			// Commit bid phase
			console.log("Submitting bids...");
			for (const [buyer, bid] of zip(bidders, bids)) {
				let salt = await submitBid(ccp, wallet, buyer, auctionName, bid);
				console.log(`Salt: ${uint8ArrayToHex(salt)}\n`);
				salts.push(salt);
			}
			console.log("Done.");

			// Close auction
			console.log("Closing auction...");
			await closeAuction(ccp, wallet, seller, auctionName);
			console.log("Done.");

			// Reveal bid phase
			console.log("Revealing bids...");
			for (const [buyer, bid, salt] of zip(bidders, bids, salts)) {
				await openBid(ccp, wallet, buyer, auctionName, bid, salt);
			}
			console.log("Done.");

			// End auction and determine winner
			console.log("Closing auction...");
			await endAuction(ccp, wallet, seller, auctionName);
			console.log("Done.");

			// Wait for the auction to end
			await new Promise((resolve, reject) => {
				if (auctionResult !== null) {
					resolve(auctionResult);
				}
				else {
					resultPromiseResolver = resolve;
				}
			});

			// auctionResult now holds the auction result
			console.log("Auction finished:");
			console.log(auctionResult);

			const winner = auctionResult.winner;
			const hammerPrice = BigInt(auctionResult.hammerPrice);
			
			assert(winner.indexOf(bidders[1]) != 0, "The second bidder should win");
			assert.equal(hammerPrice, 20n, "The hammer price should be 20");
		}
		finally {
			if (contract !== null && contractListener !== null) {
				contract.removeContractListener(contractListener);
			}
			if (gateway !== null) {
				gateway.disconnect();
			}
		}
	});
});