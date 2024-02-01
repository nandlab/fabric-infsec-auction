var assert = require('assert');

const { createAuction } = require("../createAuction.js");
const { Gateway, Wallets } = require('fabric-network');
const path = require('path');
const { buildCCPOrg1, buildCCPOrg2, buildWallet, prettyJSONString } = require('/home/fabric-user/fabric-samples/test-application/javascript/AppUtil.js');
const { randomUUID } = require('crypto');

describe('Auction', function () {
  it('simulate auction', async function () {
    this.timeout(5000);
    
    const org = "org1";
		const seller = "seller";
		const bidders = ["bidder1", "bidder2", "bidder3"];
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

    await createAuction(ccp, wallet, seller, auctionName, directBuyPrice);
  });
});
