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

async function createAuction (ccp, wallet, user, auctionName, directBuyPrice) {
	try {
		const gateway = new Gateway();
		// connect using Discovery enabled

		await gateway.connect(ccp,
			{ wallet: wallet, identity: user, discovery: { enabled: true, asLocalhost: true } });

		const network = await gateway.getNetwork(myChannel);
		const contract = network.getContract(myChaincodeName);

		const statefulTxn = contract.createTransaction('CreateAuction');

		console.log('\n--> Submit Transaction: Propose a new auction');
		await statefulTxn.submit(auctionName, directBuyPrice);
		console.log('*** Result: committed');

		gateway.disconnect();
	} catch (error) {
		console.error(`******** FAILED to submit auction: ${error}`);
	}
}

async function main () {
	try {
		if (process.argv.length < 5) {
			process.exit(1);
		}

		const org = process.argv[2];
		const user = process.argv[3];
		const auctionName = process.argv[4]
		const directBuyPrice = process.argv[5] ?? 0;
		
		if (org === 'Org1' || org === 'org1') {
			const ccp = buildCCPOrg1();
			const walletPath = path.join(__dirname, 'wallet/org1');
			const wallet = await buildWallet(Wallets, walletPath);
			await createAuction(ccp, wallet, user, auctionName, directBuyPrice);
		/* Optional TODO: You might want to use more than one orgnization
		} else if (org === 'Org2' || org === 'org2') {
			const ccp = buildCCPOrg2();
			const walletPath = path.join(__dirname, 'wallet/org2');
			const wallet = await buildWallet(Wallets, walletPath);
			await createAuction(ccp, wallet, user, TODO: Insert your parameters here );
		*/
		} else {
			console.log('Org must be Org1 ...');
		}
	} catch (error) {
		console.error(`******** FAILED to run the application: ${error}`);
	}
}

main();
