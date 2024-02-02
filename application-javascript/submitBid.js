/*
 * Copyright IBM Corp. All Rights Reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

'use strict';

const { Gateway, Wallets } = require('fabric-network');
const path = require('path');
const { buildCCPOrg1, buildCCPOrg2, buildWallet, prettyJSONString } = require('/home/fabric-user/fabric-samples/test-application/javascript/AppUtil.js');
const { X509Certificate } = require('node:crypto');

const myChannel = 'mychannel';
const myChaincodeName = 'auction';

const jsSHA = require("jssha");

const { getRandomValues } = require('node:crypto');
const { uint8ArrayToHex, uint64EncodeBidEndian, arrayToHexString } = require('./encode-utils.js');

function hashBid(clientCert, bidPrice, salt) {
	const shake = new jsSHA("SHAKE256", "UINT8ARRAY");
	for (const data of [clientCert.raw, new Uint8Array(uint64EncodeBidEndian(bidPrice)), salt]) {
		shake.update(data);
	}
	return shake.getHash("UINT8ARRAY", {outputLen: 512});
}

function generateSalt() {
	let salt = new Uint8Array(64);
	getRandomValues(salt);
	return salt;
}

async function submitBid (ccp, wallet, user, auctionName, bidPrice) {
	const gateway = new Gateway();
	// connect using Discovery enabled

	await gateway.connect(ccp,
		{ wallet: wallet, identity: user, discovery: { enabled: true, asLocalhost: true } });

	const network = await gateway.getNetwork(myChannel);
	const contract = network.getContract(myChaincodeName);
	const clientID = gateway.getIdentity();

	if (clientID.type !== "X.509") {
		throw TypeError("Client ID should be a X.509 certificate");
	}

	const clientCert = new X509Certificate(clientID.credentials.certificate);

	let salt = generateSalt();
	let bidHash = hashBid(clientCert, bidPrice, salt);
	let bidHashHex = uint8ArrayToHex(bidHash);

	console.log(`Hidden Bid Hash: ${bidHashHex}`);

	console.log('--> Submit Transaction: Bid');
	await contract.submitTransaction("Bid", auctionName, bidHashHex);
	console.log('*** Result: committed');

	gateway.disconnect();

	return salt;
}

async function main () {
	try {
		if (process.argv.length < 6) {
			console.error(`Usage: ${process.argv[0]} ${process.argv[1]} org user auctionName bidPrice`);
			process.exit(1);
		}

		const org = process.argv[2].toLowerCase();
		const user = process.argv[3];
		const auctionName = process.argv[4];
		const bidPrice = BigInt(process.argv[5]);
		
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

if (require.main === module) {
	main();
}

module.exports = {submitBid};
