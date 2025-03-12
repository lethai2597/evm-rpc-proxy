import fs from 'fs';
import path from 'path';
import axios from 'axios';

function readNodesFromFile(filePath) {
    try {
        const data = fs.readFileSync(filePath, 'utf8');
        const jsonData = JSON.parse(data);
        // Read nodes from valid-nodes file which contains both HTTP and WebSocket endpoints
        return jsonData
            .filter(node => node.http || node.ws)  // Filter nodes that have at least one endpoint
            .map(node => ({
                http: node.http,
                ws: node.ws,
                performanceScore: node.performanceScore
            }))
            .sort((a, b) => (b.performanceScore || 0) - (a.performanceScore || 0))  // Sort by performance score
            .map(node => node.http)  // Get only HTTP endpoints for admin node discovery
            .filter(Boolean);  // Remove null/undefined values
    } catch (error) {
        console.error(`Error reading file from disk: ${error}`);
        return [];
    }
}

/**
 * Split array into smaller chunks for batch processing
 * @param {Array} array - Array to be chunked
 * @param {Number} chunkSize - Size of each chunk
 * @returns {Array} Array of chunks
 */
function chunkArray(array, chunkSize) {
    const chunks = [];
    for (let i = 0; i < array.length; i += chunkSize) {
        chunks.push(array.slice(i, i + chunkSize));
    }
    return chunks;
}

/**
 * Remove duplicate nodes based on IP address, preferring port 8545
 * @param {Array} nodes - Array of node URLs
 * @returns {Array} Array of unique nodes
 */
function removeDuplicateNodes(nodes) {
    const ipMap = new Map();

    nodes.forEach((node) => {
        const cleanNode = node.replace('http://', '');
        const [ip, port] = cleanNode.split(":");

        if (!ipMap.has(ip) || port === "8545") {
            ipMap.set(ip, node);
        }
    });

    return Array.from(ipMap.values());
}

/**
 * Check if a node has admin_peers RPC method available
 * @param {String} url - Node URL to check
 * @returns {Promise<String|null>} URL if available, null if not
 */
async function checkRpcAvailability(url) {
    const data = {
        jsonrpc: "2.0",
        method: "admin_peers",
        params: [],
        id: 1
    };

    try {
        const response = await axios.post(url, data, {
            headers: {
                "Content-Type": "application/json"
            },
            timeout: 5000
        });

        if (response.data.result) {
            console.log(`Node ${url} is available. Number of peers: ${response.data.result.length}`);
            return url;
        } else {
            console.log(`Node ${url} is not available. Error: ${response.data.error || 'Unknown error'}`);
        }
    } catch (error) {
        console.log(`Node ${url} is not available. Exception: ${error.message}`);
    }
    return null;
}

/**
 * Process a chain configuration to find admin nodes
 * @param {Object} chainConfig - Chain configuration object
 * @param {Number} chainConfig.chainId - Chain ID
 * @param {String} chainConfig.inputFile - Input file path
 * @param {String} chainConfig.outputFile - Output file path
 */
async function processChain(chainConfig) {
    const { chainId, inputFile, outputFile } = chainConfig;
    console.log(`Processing chain ID ${chainId}...`);

    const filePath = path.resolve(inputFile);
    const nodeUrls = readNodesFromFile(filePath);
    const chunkSize = 100;
    const nodeChunks = chunkArray(nodeUrls, chunkSize);
    const availableNodes = [];

    for (const chunk of nodeChunks) {
        const results = await Promise.all(chunk.map(url => checkRpcAvailability(`http://${url}`)));
        availableNodes.push(...results.filter(url => url !== null));
    }

    const uniqueNodes = removeDuplicateNodes(availableNodes);

    const outputFilePath = path.resolve(outputFile);
    fs.writeFileSync(outputFilePath, JSON.stringify({ nodes: uniqueNodes }, null, 2));
    console.log(`Saved ${uniqueNodes.length} nodes to ${outputFilePath}`);
}

async function main() {
    const chainConfigs = [
        { chainId: 1, inputFile: 'valid-nodes-eth.json', outputFile: 'admin-nodes-eth.json' },
        { chainId: 56, inputFile: 'valid-nodes-bsc.json', outputFile: 'admin-nodes-bsc.json' },
        { chainId: 8453, inputFile: 'valid-nodes-base.json', outputFile: 'admin-nodes-base.json' }
    ];

    for (const chainConfig of chainConfigs) {
        await processChain(chainConfig);
    }
}

main();
