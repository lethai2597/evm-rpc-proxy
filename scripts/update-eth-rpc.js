import { ensureAdminNodesFile, processChain } from './utils.js';
import fs from 'fs';

const chainConfig = {
  chainId: 1,
  configFile: "../goevm/main/eth.json",
  validNodesFile: "data/valid-nodes-eth.json",
  adminNodesFile: "data/admin-nodes-eth.json",
};

const INTERVAL = 2 * 60 * 1000; // 2 phút

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function main() {
  while (true) {
    try {
      console.log(`\n[${new Date().toISOString()}] Starting ETH update...`);
      
      // Initialize admin nodes file
      ensureAdminNodesFile(chainConfig.adminNodesFile);

      if (!fs.existsSync(chainConfig.configFile)) {
        console.error(`Config file not found: ${chainConfig.configFile}`);
        await sleep(INTERVAL);
        continue;
      }

      await processChain(chainConfig);
      
      console.log(`Sleeping for 2 minutes...`);
      await sleep(INTERVAL);
    } catch (error) {
      console.error("Error in main process:", error);
      await sleep(INTERVAL);
    }
  }
}

// Start the process
main().catch(error => {
  console.error("Fatal error:", error);
  process.exit(1);
}); 