/*
 * Copyright IBM Corp. All Rights Reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

'use strict';

const { Gateway, Wallets } = require('fabric-network');
const path = require('path');
const { buildCCPOrg1, buildCCPOrg2, buildWallet, prettyJSONString } = require('/home/fabric-user/fabric-samples/test-application/javascript/AppUtil.js');
const { uint8ArrayToHex } = require('./encode-utils.js');

const myChannel = 'mychannel';
const myChaincodeName = 'auction';

async function openBid (ccp, wallet, user, auctionName, bidPrice, salt) {
	const gateway = new Gateway();
	// connect using Discovery enabled

	await gateway.connect(ccp,
		{ wallet: wallet, identity: user, discovery: { enabled: true, asLocalhost: true } });

	const network = await gateway.getNetwork(myChannel);
	const contract = network.getContract(myChaincodeName);

	const statefulTxn = contract.createTransaction('OpenBid');

	console.log('\n--> Submit Transaction: Open Bid');
	await statefulTxn.submit(auctionName, bidPrice, uint8ArrayToHex(salt));
	console.log('*** Result: committed');

	gateway.disconnect();

	return salt;
}

async function main () {
	try {
		if (process.argv.length < 7) {
			console.error(`Usage: ${process.argv[0]} ${process.argv[1]} org user auctionName bidPrice salt`);
			process.exit(1);
		}

		const org = process.argv[2].toLowerCase();
		const user = process.argv[3];
		const auctionName = process.argv[4];
		const bidPrice = BigInt(process.argv[5]);
		const salt = Uint8Array.from(Buffer.from(process.argv[6], 'hex'));

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
		await openBid(ccp, wallet, user, auctionName, bidPrice, salt);
	}
	catch (error) {
		console.error(`******** FAILED to run the application: ${error}`);
	}
}

if (require.main === module) {
	main();
}

module.exports = {openBid};
