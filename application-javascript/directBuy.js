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


async function directBuy (ccp, wallet, user, auctionName, price) {
	try {
		const gateway = new Gateway();
		// connect using Discovery enabled

		await gateway.connect(ccp,
			{ wallet: wallet, identity: user, discovery: { enabled: true, asLocalhost: true } });

		const network = await gateway.getNetwork(myChannel);
		const contract = network.getContract(myChaincodeName);
		const clientID = gateway.getIdentity();

		console.log(`Client ID is: ${clientID}`);

		const statefulTxn = contract.createTransaction('DirectBuy');

		console.log('\n--> Submit Transaction: Direct Buy');
		await statefulTxn.submit(auctionName, price);
		console.log('*** Result: committed');

		gateway.disconnect();

		return salt;
	} catch (error) {
		console.error(`******** FAILED to submit auction: ${error}`);
	}
}

async function main () {
	try {
		if (process.argv.length < 6) {
			console.error(`Usage: ${process.argv[0]} ${process.argv[1]} org user auctionName price`);
			process.exit(1);
		}

		const org = process.argv[2].toLowerCase();
		const user = process.argv[3];
		const auctionName = process.argv[4];
		const price = BigInt(process.argv[5]);
		
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
		await directBuy(ccp, wallet, user, auctionName, price);
	}
	catch (error) {
		console.error(`******** FAILED to run the application: ${error}`);
	}
}

main();
