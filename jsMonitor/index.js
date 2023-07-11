const express = require('express');
const { SecretNetworkClient }= require( "secretjs");
const fs = require('fs');

// Load config
const configFilePath = process.argv[2];
const config = JSON.parse(fs.readFileSync(configFilePath, 'utf8'));
const { url, chain_id,code_hash } = config;

const app = express();
const secretjs = new SecretNetworkClient({
    url,
    chainId: chain_id,
  });
  

app.get('/cosmos/bank/v1beta1/balances/:userAddress', async (req, res) => {
  try {
    const userAddress = req.params.userAddress;
    const account = await secretjs.query.bank.balance(
        {
            address:userAddress,
            denom:"uscrt",
        }
            );
   
    const response = {
      balances: [
        {
            amount:account.balance.amount,
            denom:account.balance.denom
        }
      ],
      pagination: {
        next_key: null, 
        total: "1",
      },
    };

    res.json(response);
  } catch (err) {
    console.error(err);
    res.status(500).send(err.toString());
  }
});

app.get('/cosmwasm/wasm/v1/contract/:contractAddress/smart/:request', async (req, res) => {
  try {
    const contractAddress = req.params.contractAddress;
    const request= req.params.request;
    const decodedString = Buffer.from(request, 'base64').toString('utf8');
    const queryMsg = JSON.parse(decodedString);

    const contractQueryResponse = await secretjs.query.compute.queryContract(
        {
            contract_address: contractAddress,
            code_hash: code_hash,
            query: queryMsg,
          }
    );

    const response = {
      data: {
        request_id: contractQueryResponse.request_id,
      },
    };

    res.json(response);
  } catch (err) {
    console.error(err);
    res.status(500).send(err.toString());
  }
});

app.listen(3000, () => console.log('Server running on http://localhost:3000'));
