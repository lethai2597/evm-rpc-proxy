import { ensureAdminNodesFile, processChain } from './utils.js';
import fs from 'fs';
import path from 'path';

const chainConfig = {
  chainId: 56,
  configFile: "../goevm/main/bsc.json",
  validNodesFile: "data/valid-nodes-bsc.json",
  adminNodesFile: "data/admin-nodes-bsc.json",
};

// Ensure data directory exists
const dataDir = path.join(process.cwd(), 'data');
if (!fs.existsSync(dataDir)) {
  fs.mkdirSync(dataDir, { recursive: true });
}

const INTERVAL = 10 * 60 * 1000;

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function main() {
  while (true) {
    try {
      console.log(`\n[${new Date().toISOString()}] Starting BSC update...`);
      
      // Initialize admin nodes file
      ensureAdminNodesFile(chainConfig.adminNodesFile);

      if (!fs.existsSync(chainConfig.configFile)) {
        console.error(`Config file not found: ${chainConfig.configFile}`);
        await sleep(INTERVAL);
        continue;
      }

      await processChain(chainConfig);
      
      console.log(`Sleeping for 10 minutes...`);
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