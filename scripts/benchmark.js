// Import required modules using ES modules
import axios from 'axios';

// Configuration settings
const CONFIG = {
    url: 'http://127.0.0.1:8545/?action=ethereumRaw', // HAProxy endpoint URL
    concurrency: 100,           // Number of concurrent requests
    totalRequests: 1000,        // Total number of requests to perform
    timeoutMs: 10000,          // Timeout for each request in milliseconds
    fromBlock: 'latest',          // Start block for getLogs query
    toBlock: 'latest'          // End block for getLogs query
};

// Create axios instance with timeout
const client = axios.create({
    timeout: CONFIG.timeoutMs,
    headers: {
        'Content-Type': 'application/json'
    }
});

// Tracking variables
let completed = 0;
let successful = 0;
let failed = 0;
const startTime = Date.now();

// Function to make a single request
async function makeRequest() {
    try {
        // JSON-RPC request for eth_getLogs with a large block range
        const responseLogs = await client.post(CONFIG.url, {
            jsonrpc: '2.0',
            method: 'eth_getLogs',
            params: [{
                fromBlock: CONFIG.fromBlock,
                toBlock: CONFIG.toBlock,
                address: null,    // Optional: specify contract address
                topics: []        // Optional: specify event topics
            }],
            id: 1
        });
        if (responseLogs.data.result !== undefined) {
            successful++;
        } else {
            failed++;
            console.error('Invalid response for eth_getLogs:', responseLogs.data);
        }

        // JSON-RPC request for eth_chainId
        const responseChainId = await client.post(CONFIG.url, {
            jsonrpc: '2.0',
            method: 'eth_chainId',
            params: [],
            id: 2
        });
        if (responseChainId.data.result !== undefined) {
            successful++;
        } else {
            failed++;
            console.error('Invalid response for eth_chainId:', responseChainId.data);
        }

        // JSON-RPC request for eth_getBalance
        const responseBalance = await client.post(CONFIG.url, {
            jsonrpc: '2.0',
            method: 'eth_getBalance',
            params: ['0x0000000000000000000000000000000000000000', 'latest'],
            id: 3
        });
        if (responseBalance.data.result !== undefined) {
            successful++;
        } else {
            failed++;
            console.error('Invalid response for eth_getBalance:', responseBalance.data);
        }

    } catch (error) {
        failed++;
        console.error(`Error: ${error.message}`);
    }
    completed += 3;
    
    // Print progress every 100 requests
    if (completed % (CONFIG.concurrency * 3) === 0) {
        console.log(`Progress: ${completed / 3}/${CONFIG.totalRequests}`);
    }
}

// Function to run concurrent requests
async function runBenchmark() {
    try {
        // Create batches of concurrent requests
        for (let i = 0; i < CONFIG.totalRequests; i += CONFIG.concurrency) {
            const batchSize = Math.min(CONFIG.concurrency, CONFIG.totalRequests - i);
            const promises = Array(batchSize).fill().map(() => makeRequest());
            await Promise.all(promises);
        }

        // Print final results
        const endTime = Date.now();
        const duration = (endTime - startTime) / 1000;
        const rps = CONFIG.totalRequests / duration;
        
        console.log('\n=== Benchmark Results ===');
        console.log(`Total duration: ${duration.toFixed(2)} seconds`);
        console.log(`Requests/second: ${rps.toFixed(2)}`);
        console.log(`Successful requests: ${successful}`);
        console.log(`Failed requests: ${failed}`);
        console.log(`Success rate: ${((successful/(successful + failed))*100).toFixed(2)}%`);
    } catch (error) {
        console.error('Benchmark failed:', error);
    }
}

// Run the benchmark
runBenchmark();
