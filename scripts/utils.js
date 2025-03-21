import fs from "fs";
import axios from "axios";

const chainPorts = {
  1: 8545,  // Ethereum
  56: 8547, // BSC
  8453: 8549 // Base
};

export const getProxyUrl = (chainId) => {
  const port = chainPorts[chainId] || 8545; // Default to ETH port if chainId not found
  return `http://127.0.0.1:${port}`;
};

export const proxyUrl = getProxyUrl(1); // Default to Ethereum
export const timeoutWhenCheckRpc = 5000;

// Utility functions
export function chunkArray(array, size) {
  const chunks = [];
  for (let i = 0; i < array.length; i += size) {
    chunks.push(array.slice(i, i + size));
  }
  return chunks;
}

export function removeDuplicateNodes(nodes) {
  const ipMap = new Map();
  nodes.forEach((node) => {
    const cleanNode = node.replace(/^https?:\/\//, "");
    const [ip, port] = cleanNode.split(":");
    if (!ipMap.has(ip) || port === "8545") {
      ipMap.set(ip, node);
    }
  });
  return Array.from(ipMap.values());
}

// File management
export function ensureAdminNodesFile(filePath) {
  try {
    // Ensure directory exists
    const dirPath = filePath.substring(0, filePath.lastIndexOf("/"));
    if (!fs.existsSync(dirPath)) {
      fs.mkdirSync(dirPath, { recursive: true });
      console.log(`Created directory: ${dirPath}`);
    }

    if (!fs.existsSync(filePath)) {
      fs.writeFileSync(filePath, JSON.stringify({ nodes: [] }, null, 2));
      console.log(`Created empty admin nodes file: ${filePath}`);
    }
  } catch (error) {
    console.error(`Error creating admin nodes file ${filePath}:`, error);
  }
}

// Node management
export async function updateAdminNodes(chainConfig, validNodes) {
  try {
    // Check each node for admin_peers availability
    const adminNodePromises = validNodes.map(async (node) => {
      try {
        const response = await axios.post(
          node,
          {
            jsonrpc: "2.0",
            method: "admin_peers",
            params: [],
            id: 1,
          },
          {
            headers: { "Content-Type": "application/json" },
            timeout: 5000,
          }
        );

        if (response.data.result) {
          console.log(
            `Node ${node} is available as admin node. Peers: ${response.data.result.length}`
          );
          return node;
        }
      } catch (error) {
        return null;
      }
      return null;
    });

    const results = await Promise.allSettled(adminNodePromises);
    const adminNodes = results
      .filter((result) => result.status === "fulfilled" && result.value)
      .map((result) => result.value);

    // Update admin nodes file
    fs.writeFileSync(
      chainConfig.adminNodesFile,
      JSON.stringify({ nodes: adminNodes }, null, 2)
    );
    console.log(`Updated admin nodes file with ${adminNodes.length} nodes`);
  } catch (error) {
    console.error("Error updating admin nodes:", error);
  }
}

export function filterNodes(peers) {
  const filteredNodes = peers
    .map((peer) => {
      if (peer.network && peer.network.remoteAddress) {
        return peer.network.remoteAddress;
      } else if (peer.address) {
        return peer.address;
      }
      return null;
    })
    .filter((address) => address !== null)
    .map((address) => {
      const [ip, port] = address.split(":");
      if (port === "8545" || port === "80") {
        return `http://${ip}:${port}`;
      }
      return `http://${ip}:8545`;
    })
    .filter((value, index, self) => self.indexOf(value) === index);

  return filteredNodes;
}

export async function checkNodeStatus(node, chainId) {
  try {
    const response = await axios.post(
      node,
      [
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
        },
      ],
      {
        headers: { "Content-Type": "application/json" },
        timeout: timeoutWhenCheckRpc,
      }
    );

    const blockNumberResponse = response.data.find((item) => item.id === 1);
    const chainIdResponse = response.data.find((item) => item.id === 2);

    if (blockNumberResponse?.result && chainIdResponse?.result) {
      const blockNumber = parseInt(blockNumberResponse.result, 16);
      const nodeChainId = parseInt(chainIdResponse.result, 16);

      if (blockNumber === 0) {
        return false;
      }

      if (nodeChainId !== chainId) {
        return false;
      }

      return true;
    }
    return false;
  } catch (error) {
    return false;
  }
}

export async function nodeDiscovery(adminNodeUrls, chainId) {
  console.log("Starting node discovery...");
  const MAX_NODES = 60;

  // Collect peers from admin nodes
  const peerPromises = adminNodeUrls.map((nodeUrl) =>
    axios
      .post(
        nodeUrl,
        {
          jsonrpc: "2.0",
          method: "admin_peers",
          params: [],
          id: 1,
        },
        {
          headers: { "Content-Type": "application/json" },
          timeout: timeoutWhenCheckRpc,
        }
      )
      .then((response) => response.data.result || [])
      .catch((error) => {
        console.error(`Error fetching peers from ${nodeUrl}:`, error.message);
        return [];
      })
  );

  const allPeersResults = await Promise.allSettled(peerPromises);
  const allPeers = allPeersResults
    .filter((result) => result.status === "fulfilled")
    .flatMap((result) => result.value);

  // Process and filter nodes
  const filteredNodes = filterNodes(allPeers);
  const uniqueNodes = removeDuplicateNodes(filteredNodes);
  const chunkedNodes = chunkArray(uniqueNodes, 50);
  const validNodes = [];

  for (const chunk of chunkedNodes) {
    if (validNodes.length >= MAX_NODES) {
      console.log(`Reached maximum limit of ${MAX_NODES} valid nodes`);
      break;
    }

    const results = await Promise.all(
      chunk.map((node) => checkNodeStatus(node, chainId))
    );
    chunk.forEach((node, index) => {
      if (results[index] && validNodes.length < MAX_NODES) {
        validNodes.push(node);
      }
    });
    // Add small delay between chunks
    await delay(1000);
  }

  console.log(`Found ${validNodes.length} valid nodes (max ${MAX_NODES})`);
  return validNodes;
}

// Proxy management
export async function getNodesFromProxy(chainId) {
  try {
    const response = await axios.get(getProxyUrl(chainId), {
      params: {
        action: "evm_admin",
        chain_id: chainId,
      },
      timeout: 5000,
    });

    if (response.data && typeof response.data === "object") {
      console.log(
        `Successfully got ${
          Object.keys(response.data).length
        } nodes from proxy for chain ${chainId}`
      );
      return Object.values(response.data);
    } else {
      return [];
    }
  } catch (error) {
    return [];
  }
}

export async function updateProxyNodes(
  discoveredNodes,
  currentNodes,
  whitelistNodes,
  chainId
) {
  // Add new nodes
  const nodesToAdd = discoveredNodes.filter(
    (node) => !currentNodes.some((current) => current.Endpoint === node)
  );

  // Remove disabled nodes (except whitelist)
  const disabledNodes = currentNodes.filter((node) => node.Is_disabled);
  const nodesToRemove = disabledNodes.filter(
    (node) =>
      !whitelistNodes.some((whitelist) => whitelist.url === node.Endpoint)
  );
  nodesToRemove.forEach((node) =>
    console.log(`- ${node.Endpoint} (ID: ${node.ID})`)
  );

  await addNodesToProxy(nodesToAdd, chainId);
  await removeNodesFromProxy(nodesToRemove, chainId);

  console.log(`\nSummary for chain ${chainId}:`);
  console.log(`- Added ${nodesToAdd.length} nodes`);
  console.log(`- Removed ${nodesToRemove.length} nodes`);
}

export async function addNodesToProxy(nodes, chainId) {
  if (!nodes || nodes.length === 0) return;

  console.log(
    `\nAttempting to add ${nodes.length} nodes to chain ${chainId}...`
  );

  const results = await Promise.allSettled(
    nodes.map(async (node) => {
      try {
        // Test node trước khi thêm
        console.log(`\nTesting node ${node} before adding...`);
        const testResponse = await axios.post(
          node,
          {
            jsonrpc: "2.0",
            method: "eth_blockNumber",
            params: [],
            id: 1,
          },
          {
            headers: { "Content-Type": "application/json" },
            timeout: 5000,
          }
        );

        if (!testResponse.data?.result) {
          console.error(`Node ${node} is not responding to RPC calls`);
          return false;
        }

        // Get block range of the node
        const [blockNumberHex, firstBlockHex] = await Promise.all([
          axios
            .post(
              node,
              {
                jsonrpc: "2.0",
                method: "eth_blockNumber",
                params: [],
                id: 1,
              },
              {
                headers: { "Content-Type": "application/json" },
                timeout: 5000,
              }
            )
            .then((res) => res.data.result),
          axios
            .post(
              node,
              {
                jsonrpc: "2.0",
                method: "eth_getBlockByNumber",
                params: ["0x1", false],
                id: 1,
              },
              {
                headers: { "Content-Type": "application/json" },
                timeout: 5000,
              }
            )
            .then((res) => (res.data.result ? "0x1" : "0x0")),
        ]);

        const lastBlock = parseInt(blockNumberHex, 16);
        const firstBlock = parseInt(firstBlockHex, 16);
        const now = Date.now();

        const nodeInfo = {
          url: node,
          public: false,
          throttle: "requests;10000;10;0",
          score_modifier: 1,
          probe_time: 10,
          available_block_last: lastBlock,
          available_block_last_ts: now,
          is_disabled: false,
          is_throttled: false,
          is_paused: false,
          attr: 1,
          score: 100,
        };

        const response = await axios.get(getProxyUrl(chainId), {
          params: {
            action: "evm_admin_add",
            chain_id: chainId,
            node: JSON.stringify(nodeInfo),
          },
          timeout: 5000,
        });

        if (response.data && !response.data.error) {
          console.log(`Successfully added node ${node} to chain ${chainId}`);
          return true;
        } else {
          return false;
        }
      } catch (error) {
        return false;
      }
    })
  );

  const successful = results.filter(
    (r) => r.status === "fulfilled" && r.value
  ).length;
  const failed = results.length - successful;

  console.log(`\nAdd nodes results for chain ${chainId}:`);
  console.log(`- Success: ${successful}`);
  console.log(`- Failed: ${failed}`);
}

export async function removeNodesFromProxy(nodes, chainId) {
  if (!nodes || nodes.length === 0) return;

  const results = await Promise.allSettled(
    nodes.map(async (node) => {
      try {
        const response = await axios.get(getProxyUrl(chainId), {
          params: {
            action: "evm_admin_remove",
            chain_id: chainId,
            id: node.ID,
          },
          timeout: 5000,
        });

        if (response.data && !response.data.error) {
          return true;
        } else {
          return false;
        }
      } catch (error) {
        return false;
      }
    })
  );

  const successful = results.filter(
    (r) => r.status === "fulfilled" && r.value
  ).length;
  const failed = results.length - successful;
}

export function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export async function getAdminNodes(chainConfig) {
  try {
    if (!fs.existsSync(chainConfig.adminNodesFile)) {
      ensureAdminNodesFile(chainConfig.adminNodesFile);
    }

    const data = fs.readFileSync(chainConfig.adminNodesFile, "utf8");
    const adminNodes = JSON.parse(data);

    if (adminNodes?.nodes?.length > 0) {
      return adminNodes.nodes;
    }

    // Fallback to config nodes
    const conf = JSON.parse(fs.readFileSync(chainConfig.configFile, "utf8"));
    return conf.EVM_NODES.map((node) => node.url);
  } catch (error) {
    console.error(
      `Error reading admin nodes for chain ${chainConfig.chainId}:`,
      error
    );
    const conf = JSON.parse(fs.readFileSync(chainConfig.configFile, "utf8"));
    return conf.EVM_NODES.map((node) => node.url);
  }
}

export async function processChain(chainConfig) {
  try {
    const conf = JSON.parse(fs.readFileSync(chainConfig.configFile, "utf8"));
    if (!conf || !conf.EVM_NODES || conf.EVM_NODES.length === 0) {
      console.log(`No EVM_NODES config found for chain ${chainConfig.chainId}`);
      return;
    }

    // Get admin nodes
    const adminNodes = await getAdminNodes(chainConfig);
    console.log(`Using ${adminNodes.length} admin nodes for discovery`);

    // Discover nodes
    const discoveredNodes = await nodeDiscovery(
      adminNodes,
      chainConfig.chainId
    );
    console.log(`Found ${discoveredNodes.length} valid nodes`);

    // Update admin nodes list
    await updateAdminNodes(chainConfig, discoveredNodes);

    // Save discovered nodes
    fs.writeFileSync(
      chainConfig.validNodesFile,
      JSON.stringify(discoveredNodes, null, 2)
    );

    // Update proxy
    const currentNodes = await getNodesFromProxy(chainConfig.chainId);
    await updateProxyNodes(
      discoveredNodes,
      currentNodes,
      conf.EVM_NODES,
      chainConfig.chainId
    );
  } catch (error) {
    console.error(`Error processing chain ${chainConfig.chainId}:`, error);
  }
}
