/*
 * Copyright IBM Corp. All Rights Reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

'use strict';

const { Gateway, Wallets } = require('fabric-network');
const path = require('path');
const { buildCCPOrg1, buildCCPOrg2, buildWallet, prettyJSONString } = require('/home/fabric-user/fabric-samples/test-application/javascript/AppUtil.js');
const { X509Certificate } = require('crypto');

const myChannel = 'mychannel';
const myChaincodeName = 'auction';

async function endAuction (ccp, wallet, user, auctionName) {
	const gateway = new Gateway();
	// connect using Discovery enabled

	await gateway.connect(ccp,
		{ wallet: wallet, identity: user, discovery: { enabled: true, asLocalhost: true } });

	const network = await gateway.getNetwork(myChannel);
	const contract = network.getContract(myChaincodeName);

	const statefulTxn = contract.createTransaction('EndAuction');

	console.log('\n--> Submit Transaction: End the auction');
	await statefulTxn.submit(auctionName);
	console.log('*** Result: committed');

	gateway.disconnect();
}

async function main () {
	try {
		if (process.argv.length < 5) {
			console.error(`Usage: ${process.argv[0]} ${process.argv[1]} org user auctionName`);
			process.exit(1);
		}

		const org = process.argv[2].toLowerCase();
		const user = process.argv[3];
		const auctionName = process.argv[4];

		let contract = null;
		let gateway = null;
		let contractListener = null;

		try {
			let ccp = null;
			let walletPath = null;
			if (org === 'org1') {
				ccp = buildCCPOrg1();
				walletPath = path.join(__dirname, 'wallet/org1');
			}
			else if (org === 'org2') {
				ccp = buildCCPOrg2();
				walletPath = path.join(__dirname, 'wallet/org2');
			}
			else {
				console.error('Org must be org1 or org2 ...');
				process.exit(1);
			}
			const wallet = await buildWallet(Wallets, walletPath);

			// Establish a connection to listen to the auction event
			gateway = new Gateway();
			await gateway.connect(ccp,
				{ wallet: wallet, identity: user, discovery: { enabled: true, asLocalhost: true } });		
			
			// Listen to auction events
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

			// End the auction
			await endAuction(ccp, wallet, user, auctionName);

			// Wait for the auction to end
			await new Promise((resolve, reject) => {
				if (auctionResult !== null) {
					resolve(auctionResult);
				}
				else {
					resultPromiseResolver = resolve;
				}
			});

			const winner = Buffer.from(auctionResult.winner, 'base64');
			const hammerPrice = BigInt(auctionResult.hammerPrice);
			const winnerCert = new X509Certificate(winner);
			const winnerSubject = winnerCert.subject;

			console.log(`The auction winner is: ${winnerSubject}`);
			console.log(`The hammer price is: ${hammerPrice}`);
			console.log("Full X.509 certificate of the winner:");
			console.log(winnerCert);
		}
		finally {
			if (contract !== null && contractListener !== null) {
				contract.removeContractListener(contractListener);
			}
			if (gateway !== null) {
				gateway.disconnect();
			}
		}
	}
	catch (error) {
		console.error(`******** FAILED to run the application: ${error}`);
	}
}

if (require.main === module) {
	main();
}

module.exports = {endAuction};
