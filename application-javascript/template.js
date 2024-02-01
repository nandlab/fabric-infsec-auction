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

async function /* TODO: Your function name to interact goes here */ (ccp, wallet, user, /* TODO: Insert here parameters to call the CreateAuction function */) {
	try {
		const gateway = new Gateway();
		// connect using Discovery enabled

		await gateway.connect(ccp,
			{ wallet: wallet, identity: user, discovery: { enabled: true, asLocalhost: true } });

		const network = await gateway.getNetwork(myChannel);
		const contract = network.getContract(myChaincodeName);

		const statefulTxn = contract.createTransaction(/* TODO: Insert the name of the function zou want to interact with */);

		console.log('\n--> Submit Transaction: Propose a new auction');
		await statefulTxn.submit( /* TODO: Insert here parameters to call the respetive function function */);
		console.log('*** Result: committed');

		gateway.disconnect();
	} catch (error) {
		console.error(`******** FAILED to do something: ${error}`);
	}
}

async function main () {
	try {
		if (process.argv[2] === undefined || process.argv[3] === undefined /* TODO: other function arguments */) {
			process.exit(1);
		}

		const org = process.argv[2].toLowerCase();
		const user = process.argv[3];
		// TODO: other function arguments
		// ...

		if (org === 'Org1' || org === 'org1') {
			const ccp = buildCCPOrg1();
			const walletPath = path.join(__dirname, 'wallet/org1');
			const wallet = await buildWallet(Wallets, walletPath);
			await /* TODO: Insert the name of the function to interact with here */(ccp, wallet, user, /* TODO: Insert your parameters here */);
		/* Optional TODO: You might want to use more than one orgnization
		} else if (org === 'Org2' || org === 'org2') {
			const ccp = buildCCPOrg2();
			const walletPath = path.join(__dirname, 'wallet/org2');
			const wallet = await buildWallet(Wallets, walletPath);
			await TODO: Insert the name of the function to interact with here (ccp, wallet, user, TODO: Insert your parameters here );
		*/
		} else {
			console.log('Org must be Org1 ...');
		}
	} catch (error) {
		console.error(`******** FAILED to run the application: ${error}`);
	}
}

main();
