/*
 * Copyright IBM Corp. All Rights Reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

'use strict';

const { Gateway, Wallets } = require('fabric-network');
const path = require('path');
const { buildCCPOrg1, buildCCPOrg2, buildWallet, prettyJSONString } = require('/home/fabric-user/fabric-samples/test-application/javascript/AppUtil.js');

const myChannel = 'mychannel';
const myChaincodeName = 'auction';

async function closeAuction (ccp, wallet, user, auctionName) {
	try {
		const gateway = new Gateway();
		// connect using Discovery enabled

		await gateway.connect(ccp,
			{ wallet: wallet, identity: user, discovery: { enabled: true, asLocalhost: true } });

		const network = await gateway.getNetwork(myChannel);
		const contract = network.getContract(myChaincodeName);

		const statefulTxn = contract.createTransaction('CloseAuction');

		console.log('\n--> Submit Transaction: Close the auction');
		await statefulTxn.submit(auctionName);
		console.log('*** Result: committed');

		gateway.disconnect();
	} catch (error) {
		console.error(`******** FAILED to submit auction: ${error}`);
	}
}

async function main () {
	try {
		if (process.argv.length < 5) {
			console.error(`Usage: ${process.argv[0]} org user auctionName`);
			process.exit(1);
		}

		const org = process.argv[2].toLowerCase();
		const user = process.argv[3];
		const auctionName = process.argv[4];
		
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
		await closeAuction(ccp, wallet, user, auctionName);
	}
	catch (error) {
		console.error(`******** FAILED to run the application: ${error}`);
	}
}

main();
