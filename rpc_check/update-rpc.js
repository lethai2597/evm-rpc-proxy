import * as fs from "fs";

const conf = JSON.parse(fs.readFileSync("../gosol/main/conf.json"));

const timeoutWhenCheckRpc = 3000;

(async () => {
  // while (true) {
  try {
    if (!conf || Object.values(conf).length === 0) {
      console.log("No config found");
      return;
    }

    await checkRunningRpcAndUpdateProxy();
    console.log("====DONE====");
  } catch (e) {
    console.log(e);
  }
  // await delay(3000);
  // }
})();

async function checkRunningRpcAndUpdateProxy() {
  let runningRpcIps = await checkRunningRpc();
  runningRpcIps = runningRpcIps.map((item) =>
    item.startsWith("http") ? item : "http://" + item
  );
  console.log("runningRpcIps: ", runningRpcIps.length);

  // get current list node running from proxy
  const currentNodes = await getNodesFromProxy();
  console.log("currentNodes", currentNodes.length);
  // get list node not exist in proxy
  const rpcsNotInProxy = runningRpcIps.filter((rpcIp) => {
    const included = currentNodes.filter((node) => {
      return node.Endpoint.trim() === rpcIp.trim();
    });
    return included.length === 0;
  });
  console.log("rpcsNotInProxy: ", rpcsNotInProxy.length);

  // get list node exist on proxy but not running, exclude whitelist
  const disabledNodes = currentNodes.filter((node) => {
    return node.Is_disabled;
  });

  const remoableNodes = disabledNodes.filter((nodeDisabled) => {
    const included = conf.EVM_NODES.filter((nodeWhitelis) => {
      return nodeWhitelis.url.trim() === nodeDisabled.Endpoint.trim();
    });
    return included.length === 0;
  });
  console.log("remoableNodes: ", remoableNodes.length);

  // call api add rpcsNotInProxy into proxy
  await addRpcsToProxy(rpcsNotInProxy);
  // call api remove nodesNotRunning
  await removeNodesFromProxy(remoableNodes);
}

const proxyUrl = "http://127.0.0.1:8545";

// return list nodes: {ID: 1, Endpoint: ""}
async function getNodesFromProxy() {
  const response = await fetch(proxyUrl + "?action=evm_admin");
  const responseJson = await response.json();
  return Object.values(responseJson);
}

async function addRpcsToProxy(nodes) {
  if (!nodes || nodes.length === 0) {
    return;
  }

  const result = await Promise.allSettled(
    nodes.map((node) => {
      let nodeInfo = JSON.stringify({
        url: node,
        public: false,
        throttle: "r,15000,10",
        score_modifier: 1,
        probe_time: 10,
      });
      nodeInfo = nodeInfo.replace(/;/g, encodeURIComponent(";"));

      return callApi(`${proxyUrl}?action=evm_admin_add&node=${nodeInfo}`);
    })
  );
  console.log("addNodesToProxy: ", result.length);
}

async function removeNodesFromProxy(nodes) {
  if (!nodes || nodes.length === 0) {
    return;
  }

  const result = await Promise.allSettled(
    nodes.map((node) =>
      callApi(`${proxyUrl}?action=evm_admin_remove&id=${node["ID"]}`)
    )
  );
  console.log("removeNodesFromProxy: ", result.length);
}

async function checkRunningRpc() {
  const runningRpcs = await checkRunningRpcFromPublicNodes();
  console.log("runningRpcs: ", runningRpcs.length);

  const runningRpcsExt = await checkRunningRpcFromExtNodes();
  console.log("runningRpcsExt: ", runningRpcsExt.length);

  // reduce rpcs to object use reduce function in js
  const runningRpcsMap = runningRpcs.reduce((rpc, cur) => {
    rpc[cur] = 1;
    return rpc;
  }, {});
  // avoid duplicate rpcs
  runningRpcs.push(...runningRpcsExt.filter((rpc) => !runningRpcsMap[rpc]));

  return runningRpcs;
}

async function checkRunningRpcFromPublicNodes() {
  // fetch rpc with fetch
  const response = await callRpc(
    "https://ethereum-rpc.publicnode.com",
    "admin_peers",
    []
  );
  //   console.log("response: ", response);

  if (!response.result) {
    // Nếu không thể lấy thông tin từ public node, đọc từ file cấu hình
    try {
      const adminNodesData = JSON.parse(fs.readFileSync("admin-nodes-eth.json", "utf8"));
      if (adminNodesData && adminNodesData.nodes) {
        // Đọc các node từ admin-nodes-eth.json
        const nodes = adminNodesData.nodes.map(node => ({ rpc: node }));
        const rpcs = filterRpcExistIp(nodes);
        const runningRpcs = await checkRpcsRunning(rpcs);
        const privateRpcs = await checkPrivateRpc(nodes);
        return [...runningRpcs, ...privateRpcs];
      }
    } catch (err) {
      console.error("Error reading admin-nodes-eth.json:", err);
    }
    throw new Error("No result found");
  }

  const rpcs = filterRpcExistIp(response.result);

  const runningRpcs = await checkRpcsRunning(rpcs);
  const privateRpcs = await checkPrivateRpc(response.result);

  return [...runningRpcs, ...privateRpcs];
}

async function checkRunningRpcFromExtNodes() {
  // fetch rpc with fetch
  let response = await callApi(
    "https://api.extrnode.com/endpoints?is_rpc=true"
  );
  // console.log("response: ", response);

  if (!response) {
    // Nếu không lấy được từ API, trả về mảng rỗng thay vì sử dụng cache
    console.log("[checkRunningRpcFromExtNodes] Không thể kết nối đến API extrnode.com");
    return [];
  }

  const fieldFilter = "endpoint";
  const rpcs = filterRpcExistIp(response, fieldFilter);

  const ipField = "endpoint";
  const runningRpcs = await checkRpcsRunning(rpcs, ipField);

  return runningRpcs;
}

// promiss all fetch rpcs to check call success
async function checkRpcsRunning(rpcs, ipField = "rpc") {
  const rpcPromises = rpcs.map((rpc) => {
    let rpcUrl = rpc[ipField];
    rpcUrl = rpcUrl.startsWith("http") ? rpcUrl : "http://" + rpcUrl;

    // Thay thế phương thức Solana 'getSignaturesForAddress' bằng phương thức EVM 'eth_blockNumber'
    return fetchWithTimeout(
      rpcUrl,
      "eth_blockNumber",
      [],
      rpcUrl
    );
  });

  const rpcResponses = await Promise.allSettled(rpcPromises);
  //   console.log("rpcResponses: ", rpcResponses);

  const rpcRunning = rpcResponses
    .filter(
      (rpcResponse) =>
        rpcResponse.status === "fulfilled" && !rpcResponse.value.error
    )
    .map((rpcResponse) => rpcResponse.value.id);

  return rpcRunning;
}

function filterRpcExistIp(rpcList, ipField = "rpc") {
  const validRpcList = rpcList.filter((rpc) => {
    // check if rpc is valid
    return !!rpc[ipField];
  });
  return validRpcList;
}

// fetch rpc function with timeout 5s
const fetchWithTimeout = (url, method, params, id = 1) => {
  return Promise.race([
    callRpc(url, method, params, id),
    new Promise((_, reject) =>
      setTimeout(() => reject(new Error("timeout")), timeoutWhenCheckRpc)
    ),
  ]);
};

async function callRpc(url, method, params, id = 1) {
  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      jsonrpc: "2.0",
      id: id,
      method: method,
      params: params,
    }),
  });
  const responseJson = await response.json();
  return responseJson;
}

async function callApi(url, options) {
  try {
    const response = await fetch(url, options);
    const responseJson = await response.json();
    return responseJson;
  } catch (e) {
    console.log(e);
  }
}

// delay function
function delay(time) {
  return new Promise((resolve) => {
    setTimeout(resolve, time);
  });
}

/* -------------------------------------------------------------------------- */
/*                             Check private node                             */
/* -------------------------------------------------------------------------- */
/**
 * Splits an array into chunks of a specified size.
 * @param {Array} array - The array to split into chunks.
 * @param {number} size - The size of each chunk.
 * @returns {Array} - An array containing the chunks.
 */
function chunk(array, size) {
  if (!Array.isArray(array)) {
    throw new TypeError("First argument must be an array.");
  }
  if (typeof size !== "number" || size <= 0) {
    throw new RangeError("Chunk size must be a positive number.");
  }
  const result = [];
  for (let i = 0; i < array.length; i += size) {
    result.push(array.slice(i, i + size));
  }
  return result;
}

function getRpcIp(rpc) {
  try {
    const items = rpc.replace("http://", "").split(":");
    if (items.length) return items[0];
    return rpc;
  } catch (error) {
    console.log(error);
    return rpc;
  }
}

async function checkPrivateRpc(rpcs) {
  console.log("Checking private nodes...");
  try {
    const result = {
      success: [],
      failed: [],
      ips: [],
    };

    // Thay đổi logic tìm kiếm private nodes từ Solana sang EVM
    const validRpcList = rpcs
      .filter((rpc) => {
        // check if rpc is valid for EVM nodes
        return rpc["address"] || rpc["remoteAddress"] || rpc["enode"];
      })
      .map((rpc) => {
        let ip;
        if (rpc["address"]) {
          ip = rpc["address"].split(":")[0];
        } else if (rpc["remoteAddress"]) {
          ip = rpc["remoteAddress"].split(":")[0];
        } else if (rpc["enode"]) {
          // Extract IP from enode URL if available
          const match = rpc["enode"].match(/@([^:]+):/);
          ip = match ? match[1] : null;
        }

        if (ip) {
          // Common EVM RPC ports
          return [`http://${ip}:8545`, `http://${ip}:30303`];
        }
        return [];
      });

    const handle = async (rpcUrl) => {
      try {
        // Thay thế Solana method bằng EVM method
        const response = await fetchWithTimeout(
          rpcUrl,
          "eth_blockNumber",
          [],
          rpcUrl
        );
        
        // Kiểm tra xem node có trả về block number hợp lệ hay không
        const hasValidBlockNumber = response.result && 
                                  typeof response.result === 'string' && 
                                  response.result.startsWith('0x');
                                  
        if (hasValidBlockNumber) {
          const ip = getRpcIp(rpcUrl);
          if (result.ips.includes(ip)) return false;
          result.ips.push(ip);
          result.success.push(rpcUrl);
          console.log("Success:", rpcUrl);
          return true;
        } else {
          result.failed.push(rpcUrl);
          return false;
        }
      } catch (error) {
        result.failed.push(rpcUrl);
        return false;
      }
    };
    console.log(`Total rpcs to check ${validRpcList.length}`);
    const chunked = chunk(validRpcList, 250);

    console.log(`Total chunked ${chunked.length}`);
    for (const [i, rpcList] of chunked.entries()) {
      await Promise.allSettled(
        rpcList.map(async (rpcs) => {
          await Promise.all(
            rpcs.map(async (rpcUrl) => {
              return await handle(rpcUrl);
            })
          );
        })
      );
    }

    console.log(`Found ${result.success.length} private nodes`);
    return result.success;
  } catch (error) {
    console.error(error);
    return [];
  }
}
