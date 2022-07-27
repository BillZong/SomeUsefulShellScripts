#!/usr/bin/env node

const yargs = require('yargs/yargs')
const { hideBin } = require('yargs/helpers')
const keythereum = require("keythereum");

const args = yargs(hideBin(process.argv))
  .option('datadir', {
    alias: 'd',
    type: 'string',
    description: 'data directory, which contains a keystore/ directory',
    default: '.'
  })
  .option('address', {
  	alias: 'a',
  	type: 'string',
  	description: 'ethereum address, hex string'
  })
  .option('password', {
  	alias: 'p',
  	type: 'string',
  	description: 'password'
  })
  .parse()

var keyObject = keythereum.importFromFile(args.address, args.datadir);
var privateKey = keythereum.recover(args.password, keyObject);
console.log(privateKey.toString('hex'));
