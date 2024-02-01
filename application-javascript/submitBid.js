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

const jsSHA = require("jssha");

const { getRandomValues } = require('Crypto');

function uint64EncodeBidEndian(n) {
	let buffer = new ArrayBuffer(8);
	let view = new DataView(buffer);
	view.setBigUint64(0, n);
	return buffer;
}

function arrayToHexString(byteArray) {
    return Array.from(byteArray, function(byte) {
        return ('0' + (byte & 0xFF).toString(16)).slice(-2);
    }).join('');
}

function hashBid(clientID, bidPrice, salt) {
	const shake = new jsSHA("SHAKE256");
	shake.update(clientID);
	shake.update(uint64EncodeBidEndian(bidPrice));
	shake.update(salt);
	return shaObj.getHash("BUFFER", {outputLen: 512});
}

function generateSalt() {
	let salt = Uint8Array(64);
	getRandomValues(salt);
	return salt;
}

async function submitBid (ccp, wallet, user, auctionName, bidPrice) {
	try {
		const gateway = new Gateway();
		// connect using Discovery enabled

		await gateway.connect(ccp,
			{ wallet: wallet, identity: user, discovery: { enabled: true, asLocalhost: true } });

		const network = await gateway.getNetwork(myChannel);
		const contract = network.getContract(myChaincodeName);
		const clientID = gateway.getIdentity();

		console.log(`Client ID is: ${clientID}`);

		const statefulTxn = contract.createTransaction('Bid');

		let salt = generateSalt();
		let bidHash = hashBid(clientID, bidPrice, salt);
		
		console.log('\n--> Submit Transaction: Bid');
		await statefulTxn.submit(auctionName, bidHash);
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
			console.error(`Usage: ${process.argv[0]} ${process.argv[1]} org user auctionName bidPrice`);
			process.exit(1);
		}

		const org = process.argv[2];
		const user = process.argv[3];
		const auctionName = process.argv[4];
		const bidPrice = BigInt(process.argv[5]);
		
		org = org.toLowerCase();
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
		const salt = await submitBid(ccp, wallet, user, auctionName, bidPrice);
		console.log(`Please save the salt:\n${arrayToHexString(salt)}`);
	}
	catch (error) {
		console.error(`******** FAILED to run the application: ${error}`);
	}
}

main();
