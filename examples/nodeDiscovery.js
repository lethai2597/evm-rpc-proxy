import axios from 'axios';
import WebSocket from 'ws';

/**
 * Filter and normalize the list of peers by extracting only the valid nodes (with port 8545 or 80).
 * If the port is not 8545 or 80, it will default to port 8545.
 * 
 * @param {Array} peers - List of peers (nodes).
 * @returns {Array} - List of normalized nodes with valid ports.
 */
function filterNodes(peers) {
    console.log("Filtering nodes...");

    const filteredNodes = peers
        .map(peer => {
            if (peer.network && peer.network.remoteAddress) {
                return peer.network.remoteAddress;
            } else if (peer.address) {
                return peer.address;
            }
            console.warn("Peer does not have a valid network or remoteAddress:", peer);
            return null;
        })
        .filter(address => address !== null)
        .map(address => {
            const [ip, port] = address.split(":");
            
            if (port === "8545" || port === "80") {
                return `${ip}:${port}`;
            } else {
                return `${ip}:8545`;
            }
        })
        .filter((value, index, self) => self.indexOf(value) === index);

    console.log(`Filtered nodes: ${filteredNodes.length} nodes found.`);
    return filteredNodes;
}

/**
 * Remove duplicate nodes based on IP address (ignoring ports)
 * If a node is accessible on both ports, prefer port 8545
 *
 * @param {Array} nodes - Array of nodes in format "ip:port"
 * @returns {Array} - Array of unique nodes, preferring port 8545 when duplicate IPs exist
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
 * Check if a node is reachable by checking its blocknumber and chainId.
 * It also includes a timeout to avoid waiting indefinitely.
 * 
 * @param {String} node - The node IP to check.
 * @param {Number} chainId - The expected chain ID.
 * @param {Number} timeout - Timeout duration in milliseconds.
 * @returns {Promise<Boolean>} - True if node is reachable, returns valid blocknumber and chainId.
 */
async function checkNodeStatus(node, chainId, timeout = 1000) {
    const url = `http://${node}/`;
    const batchRequestPayload = [
        {
            jsonrpc: "2.0",
            method: "eth_blockNumber",
            params: [],
            id: 1,
        },
        {
            jsonrpc: "2.0",
            method: "eth_chainId",
            params: [],
            id: 2,
        }
    ];

    try {
        const response = await axios.post(url, batchRequestPayload, { timeout });

        const blockNumberResponse = response.data.find(item => item.id === 1);
        const chainIdResponse = response.data.find(item => item.id === 2);

        if (blockNumberResponse && chainIdResponse) {
            const blockNumber = parseInt(blockNumberResponse.result, 16);
            const nodeChainId = parseInt(chainIdResponse.result, 16);

            if (blockNumber === 0) {
                console.log(`Node ${node} has block number 0, skipping it.`);
                return false;
            }

            if (nodeChainId !== chainId) {
                console.log(`Node ${node} is on chain ${nodeChainId} but expected chain ${chainId}, skipping it.`);
                return false;
            }

            console.log(`Node ${node} is reachable with block number ${blockNumber} and chain ID ${nodeChainId}.`);
            return true;
        }

        return false;

    } catch (error) {
        return false;
    }
}

/**
 * Chunk an array into smaller arrays of a specific size.
 * 
 * @param {Array} array - The array to chunk.
 * @param {Number} size - The size of each chunk.
 * @returns {Array} - A new array containing the chunked arrays.
 */
function chunkArray(array, size) {
    const chunks = [];
    for (let i = 0; i < array.length; i += size) {
        chunks.push(array.slice(i, i + size));
    }
    return chunks;
}

/**
 * Discover nodes by sending RPC requests to a list of admin nodes and filtering out valid peers.
 * The peers are filtered based on valid IP addresses with port 8545 or 80.
 * Then, it checks if those nodes are reachable by making HTTP requests to ports 8545 and 80.
 * 
 * @param {Array} adminNodeUrls - List of admin node URLs (e.g., ["http://localhost:8545"]).
 * @param {Number} chainId - The chain ID to validate nodes against.
 * @returns {Promise<Array>} - List of reachable nodes.
 */
export async function nodeDiscovery(adminNodeUrls, chainId) {
    console.log("Starting node discovery...");

    // Collect peers from all admin nodes using Promise.allSettled
    const peerPromises = adminNodeUrls.map(nodeUrl => 
        axios.post(nodeUrl, {
            jsonrpc: "2.0",
            method: "admin_peers",
            params: [],
            id: 1,
        }, {
            timeout: 1000 // timeout after 1 second
        }).then(response => response.data.result)
          .catch(error => {
              console.error(`Error fetching peers from ${nodeUrl}`);
              return [];
          })
    );

    const allPeersResults = await Promise.allSettled(peerPromises);
    const allPeers = allPeersResults
        .filter(result => result.status === 'fulfilled')
        .flatMap(result => result.value);

    // Filter and remove duplicates
    const validPeers = filterNodes(allPeers);
    const uniqueNodes = removeDuplicateNodes(validPeers);

    // Check node status
    const chunkedNodes = chunkArray(uniqueNodes, 500);
    const reachableNodes = [];

    for (const chunk of chunkedNodes) {
        const results = await Promise.all(chunk.map(node => checkNodeStatus(node, chainId)));
        chunk.forEach((node, index) => {
            if (results[index]) {
                reachableNodes.push(node);
            }
        });
    }

    console.log(`Found ${reachableNodes.length} reachable nodes.`);
    return reachableNodes;
}

/**
 * Filter and normalize the list of peers for WebSocket nodes (port 8546).
 * 
 * @param {Array} peers - List of peers (nodes).
 * @returns {Array} - List of normalized nodes with valid ports.
 */
function filterSocketNodes(peers) {
    console.log("Filtering socket nodes...");

    const filteredNodes = peers
        .map(peer => {
            if (peer.network && peer.network.remoteAddress) {
                return peer.network.remoteAddress;
            } else if (peer.address) {
                return peer.address;
            }
            console.warn("Peer does not have a valid network or remoteAddress:", peer);
            return null;
        })
        .filter(address => address !== null)
        .map(address => {
            const [ip] = address.split(":");
            return `${ip}:8546`;
        })
        .filter((value, index, self) => self.indexOf(value) === index);

    console.log(`Filtered socket nodes: ${filteredNodes.length} nodes found.`);
    return filteredNodes;
}

/**
 * Check if a WebSocket node is reachable.
 * 
 * @param {String} node - The node IP to check.
 * @param {Number} chainId - The expected chain ID.
 * @param {Number} timeout - Timeout duration in milliseconds.
 * @returns {Promise<Boolean>} - True if node is reachable.
 */
export async function checkSocketNodeStatus(node, chainId, timeout = 1000) {
    return new Promise((resolve) => {
        try {
            const ws = new WebSocket(`ws://${node}`);
            const timeoutId = setTimeout(() => {
                ws.terminate();
                console.log(`Socket node ${node} connection timeout`);
                resolve(false);
            }, timeout);

            ws.on('open', () => {
                clearTimeout(timeoutId);
                console.log(`Socket node ${node} is reachable`);
                ws.close();
                resolve(true);
            });

            ws.on('error', (error) => {
                clearTimeout(timeoutId);
                if (error.code === 'ECONNREFUSED') {
                    console.log(`Socket node ${node} refused connection`);
                } else {
                    console.error(`Error checking socket node ${node}:`.message);
                }
                resolve(false);
            });
        } catch (error) {
            console.error(`Error creating WebSocket for ${node}:`.message);
            resolve(false);
        }
    });
}

/**
 * Discover WebSocket nodes by sending RPC requests to a list of admin nodes
 * and filtering out valid peers. The peers are filtered based on valid IP addresses
 * with port 8546.
 * 
 * @param {Array} adminNodeUrls - List of admin node URLs.
 * @param {Number} chainId - The chain ID to validate nodes against.
 * @returns {Promise<Array>} - List of reachable nodes.
 */
export async function socketNodeDiscovery(adminNodeUrls, chainId) {
    console.log("Starting socket node discovery...");

    const peerPromises = adminNodeUrls.map(nodeUrl => 
        axios.post(nodeUrl, {
            jsonrpc: "2.0",
            method: "admin_peers",
            params: [],
            id: 1,
        }, {
            timeout: 1000
        }).then(response => response.data.result)
          .catch(error => {
              console.error(`Error fetching socket peers from ${nodeUrl}`);
              return [];
          })
    );

    const allPeersResults = await Promise.allSettled(peerPromises);
    const allPeers = allPeersResults
        .filter(result => result.status === 'fulfilled')
        .flatMap(result => result.value);

    // Filter and remove duplicates
    const validPeers = filterSocketNodes(allPeers);
    const uniqueNodes = removeDuplicateNodes(validPeers);

    // Check node status
    const chunkedNodes = chunkArray(uniqueNodes, 500);
    const reachableNodes = [];

    for (const chunk of chunkedNodes) {
        const results = await Promise.all(chunk.map(node => checkSocketNodeStatus(node, chainId)));
        chunk.forEach((node, index) => {
            if (results[index]) {
                reachableNodes.push(node);
            }
        });
    }

    console.log(`Found ${reachableNodes.length} reachable socket nodes.`);
    return reachableNodes;
}
