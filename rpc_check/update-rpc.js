import * as fs from "fs";
import * as path from "path";
import WebSocket from "ws";

// Cấu hình cho nhiều chuỗi
const CHAIN_CONFIGS = [
  {
    chainId: 1,
    name: "Ethereum",
    configFile: "../goevm/main/conf.json",
    outputFile: "admin-nodes-eth.json",
    ports: {
      http: [8545, 80],
      ws: [8546]
    }
  },
  // Có thể thêm cấu hình cho các chuỗi khác ở đây
  // {
  //   chainId: 56,
  //   name: "BSC",
  //   configFile: "../goevm/bsc/conf.json",
  //   outputFile: "admin-nodes-bsc.json",
  //   ports: {
  //     http: [8545, 80],
  //     ws: [8546]
  //   }
  // },
];

// Các cài đặt chung
const DEFAULT_CHAIN_ID = 1; // Ethereum mainnet làm mặc định
const DEFAULT_CONFIG_FILE = "../goevm/main/conf.json";
const TIMEOUT_WHEN_CHECK_RPC = 3000;
const BATCH_SIZE = 200; // Kích thước batch khi kiểm tra nhiều node
const PERFORMANCE_CHECK_RETRIES = 3; // Số lần thử để đánh giá hiệu suất

// Khởi tạo proxyUrl dựa trên tham số dòng lệnh hoặc giá trị mặc định
const proxyUrl = process.env.PROXY_URL || "http://127.0.0.1:8545";
const chainId = process.env.CHAIN_ID ? parseInt(process.env.CHAIN_ID) : DEFAULT_CHAIN_ID;

// Tìm cấu hình dựa trên chainId
const currentChainConfig = CHAIN_CONFIGS.find(config => config.chainId === chainId) || CHAIN_CONFIGS[0];
const configFile = currentChainConfig.configFile;

console.log(`Đang chạy cho chuỗi: ${currentChainConfig.name} (chainId: ${chainId})`);
console.log(`Sử dụng tệp cấu hình: ${configFile}`);
console.log(`Proxy URL: ${proxyUrl}`);

(async () => {
  try {
    // Đọc file cấu hình
    let conf;
    try {
      conf = JSON.parse(fs.readFileSync(configFile));
    } catch (error) {
      console.error(`Lỗi khi đọc file cấu hình ${configFile}: ${error.message}`);
      conf = { EVM_NODES: [] };
    }

    if (!conf || !conf.EVM_NODES || conf.EVM_NODES.length === 0) {
      console.log("Không tìm thấy cấu hình hoặc danh sách node trống");
    } else {
      console.log(`Đã tìm thấy ${conf.EVM_NODES.length} node trong cấu hình`);
    }

    // Chạy chu trình kiểm tra và cập nhật
    await checkRunningRpcAndUpdateProxy();
    console.log("====HOÀN THÀNH====");
  } catch (e) {
    console.error("Lỗi trong quá trình thực thi:", e);
  }
})();

async function checkRunningRpcAndUpdateProxy() {
  console.log(`Bắt đầu chu trình kiểm tra cho chainId ${chainId}`);
  
  // Thực hiện lấy danh sách node HTTP
  console.log("Tìm kiếm các node HTTP đang hoạt động...");
  const runningHttpNodes = await checkRunningRpc();
  
  // Thực hiện lấy danh sách node WebSocket
  console.log("Tìm kiếm các node WebSocket đang hoạt động...");
  const runningWsNodes = await checkRunningWebSocketRpc();
  
  // Chuẩn hóa URL cho các node HTTP
  const formattedHttpNodes = runningHttpNodes.map((item) =>
    item.startsWith("http") ? item : "http://" + item
  );
  console.log(`Tìm thấy ${formattedHttpNodes.length} node HTTP đang hoạt động`);
  console.log(`Tìm thấy ${runningWsNodes.length} node WebSocket đang hoạt động`);

  // Lưu kết quả vào file để tái sử dụng
  saveNodesToFile(formattedHttpNodes, runningWsNodes);

  // Lấy danh sách node hiện tại từ proxy
  const currentNodes = await getNodesFromProxy();
  console.log(`Đang có ${currentNodes.length} node trên proxy`);
  
  // Tìm các node không có trong proxy để thêm vào
  const rpcsNotInProxy = formattedHttpNodes.filter((rpcIp) => {
    const included = currentNodes.filter((node) => {
      return node.Endpoint.trim() === rpcIp.trim();
    });
    return included.length === 0;
  });
  console.log(`Có ${rpcsNotInProxy.length} node HTTP cần thêm vào proxy`);

  // Tìm các node không hoạt động trên proxy để loại bỏ
  const disabledNodes = currentNodes.filter((node) => {
    return node.Is_disabled;
  });

  // Đọc cấu hình
  let conf;
  try {
    conf = JSON.parse(fs.readFileSync(configFile));
  } catch (error) {
    console.error(`Lỗi khi đọc file cấu hình ${configFile}: ${error.message}`);
    conf = { EVM_NODES: [] };
  }

  // Tìm các node có thể loại bỏ, không nằm trong whitelist
  const removableNodes = disabledNodes.filter((nodeDisabled) => {
    if (!conf || !conf.EVM_NODES) return true;
    
    const included = conf.EVM_NODES.filter((nodeWhitelist) => {
      return nodeWhitelist.url.trim() === nodeDisabled.Endpoint.trim();
    });
    return included.length === 0;
  });
  console.log(`Có ${removableNodes.length} node có thể loại bỏ khỏi proxy`);

  // Thêm các node mới vào proxy
  await addRpcsToProxy(rpcsNotInProxy);
  
  // Loại bỏ các node không hoạt động khỏi proxy
  await removeNodesFromProxy(removableNodes);

  // In thống kê cuối cùng
  console.log("=== BÁO CÁO THỐNG KÊ ===");
  console.log(`Tổng số node HTTP đang hoạt động: ${formattedHttpNodes.length}`);
  console.log(`Tổng số node WebSocket đang hoạt động: ${runningWsNodes.length}`);
  console.log(`Số node đã thêm vào proxy: ${rpcsNotInProxy.length}`);
  console.log(`Số node đã loại bỏ khỏi proxy: ${removableNodes.length}`);
  console.log(`Tổng số node hiện tại trên proxy: ${currentNodes.length - removableNodes.length + rpcsNotInProxy.length}`);
}

// Lưu danh sách node vào file để tái sử dụng
function saveNodesToFile(httpNodes, wsNodes) {
  const outputFileName = currentChainConfig.outputFile || `admin-nodes-${chainId}.json`;
  
  try {
    const data = {
      lastUpdated: new Date().toISOString(),
      chainId: chainId,
      nodes: httpNodes.map((node, index) => ({
        http: node,
        ws: index < wsNodes.length ? wsNodes[index] : null,
        lastChecked: new Date().toISOString(),
        performanceScore: 0 // Điểm khởi tạo
      }))
    };
    
    fs.writeFileSync(outputFileName, JSON.stringify(data, null, 2));
    console.log(`Đã lưu ${httpNodes.length} node vào file ${outputFileName}`);
  } catch (error) {
    console.error(`Lỗi khi lưu danh sách node: ${error.message}`);
  }
}

// Lấy danh sách node từ proxy
async function getNodesFromProxy() {
  try {
    const response = await fetch(proxyUrl + "?action=evm_admin");
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    const responseJson = await response.json();
    return Object.values(responseJson);
  } catch (error) {
    console.error(`Lỗi khi lấy danh sách node từ proxy: ${error.message}`);
    return [];
  }
}

// Thêm các node mới vào proxy
async function addRpcsToProxy(nodes) {
  if (!nodes || nodes.length === 0) {
    console.log("Không có node mới để thêm vào proxy");
    return;
  }

  console.log(`Bắt đầu thêm ${nodes.length} node vào proxy...`);
  
  // Chia thành các batch nhỏ để tránh quá tải
  const nodeBatches = chunk(nodes, BATCH_SIZE);
  
  let addedCount = 0;
  for (const [batchIndex, batch] of nodeBatches.entries()) {
    console.log(`Xử lý batch ${batchIndex + 1}/${nodeBatches.length} (${batch.length} node)`);
    
    const result = await Promise.allSettled(
      batch.map((node) => {
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
    
    // Đếm số node thêm thành công
    const successCount = result.filter(r => r.status === 'fulfilled').length;
    addedCount += successCount;
    
    console.log(`Đã thêm ${successCount}/${batch.length} node trong batch này`);
  }
  
  console.log(`Tổng số node đã thêm thành công: ${addedCount}/${nodes.length}`);
}

// Loại bỏ các node không hoạt động
async function removeNodesFromProxy(nodes) {
  if (!nodes || nodes.length === 0) {
    console.log("Không có node cần loại bỏ khỏi proxy");
    return;
  }

  console.log(`Bắt đầu loại bỏ ${nodes.length} node khỏi proxy...`);
  
  // Chia thành các batch nhỏ để tránh quá tải
  const nodeBatches = chunk(nodes, BATCH_SIZE);
  
  let removedCount = 0;
  for (const [batchIndex, batch] of nodeBatches.entries()) {
    console.log(`Xử lý batch ${batchIndex + 1}/${nodeBatches.length} (${batch.length} node)`);
    
    const result = await Promise.allSettled(
      batch.map((node) =>
        callApi(`${proxyUrl}?action=evm_admin_remove&id=${node["ID"]}`)
      )
    );
    
    // Đếm số node loại bỏ thành công
    const successCount = result.filter(r => r.status === 'fulfilled').length;
    removedCount += successCount;
    
    console.log(`Đã loại bỏ ${successCount}/${batch.length} node trong batch này`);
  }
  
  console.log(`Tổng số node đã loại bỏ thành công: ${removedCount}/${nodes.length}`);
}

async function checkRunningRpc() {
  console.log("Bắt đầu kiểm tra các node RPC...");
  
  const runningRpcs = await checkRunningRpcFromPublicNodes();
  console.log(`Tìm thấy ${runningRpcs.length} node từ public nodes`);

  const runningRpcsExt = await checkRunningRpcFromExtNodes();
  console.log(`Tìm thấy ${runningRpcsExt.length} node từ ext nodes`);

  // Loại bỏ node trùng lặp
  const runningRpcsMap = runningRpcs.reduce((rpc, cur) => {
    rpc[cur] = 1;
    return rpc;
  }, {});
  
  // Thêm các node từ ext không trùng lặp
  runningRpcs.push(...runningRpcsExt.filter((rpc) => !runningRpcsMap[rpc]));

  // Kiểm tra hiệu suất của node
  console.log(`Bắt đầu đánh giá hiệu suất cho ${runningRpcs.length} node...`);
  const nodeWithPerformance = await evaluateNodePerformance(runningRpcs);
  
  // Sắp xếp node theo hiệu suất
  const sortedNodes = nodeWithPerformance
    .sort((a, b) => b.performanceScore - a.performanceScore)
    .map(node => node.url);
    
  return sortedNodes;
}

// Hàm mới: Kiểm tra WebSocket node
async function checkRunningWebSocketRpc() {
  console.log("Bắt đầu kiểm tra các WebSocket node...");
  
  try {
    // Đọc các node HTTP đang hoạt động từ file (nếu có)
    let httpNodes = [];
    try {
      const outputFileName = currentChainConfig.outputFile || `admin-nodes-${chainId}.json`;
      if (fs.existsSync(outputFileName)) {
        const data = JSON.parse(fs.readFileSync(outputFileName, 'utf8'));
        httpNodes = data.nodes.map(node => node.http);
      }
    } catch (error) {
      console.error(`Lỗi khi đọc file node: ${error.message}`);
    }
    
    // Nếu không có node HTTP, sử dụng các node từ cấu hình
    if (httpNodes.length === 0) {
      let conf;
      try {
        conf = JSON.parse(fs.readFileSync(configFile));
        if (conf && conf.EVM_NODES) {
          httpNodes = conf.EVM_NODES.map(node => node.url);
        }
      } catch (error) {
        console.error(`Lỗi khi đọc file cấu hình: ${error.message}`);
      }
    }
    
    // Chuyển đổi từ HTTP URL sang WebSocket URL
    const wsNodes = httpNodes.map(httpUrl => {
      // Thay http:// thành ws:// và http port thành ws port
      return httpUrl
        .replace('http://', 'ws://')
        .replace(':8545', ':8546')
        .replace(':80', ':8546');
    });
    
    // Kiểm tra kết nối WebSocket
    console.log(`Kiểm tra ${wsNodes.length} node WebSocket...`);
    const wsNodeBatches = chunk(wsNodes, BATCH_SIZE);
    
    const runningWsNodes = [];
    for (const [batchIndex, batch] of wsNodeBatches.entries()) {
      console.log(`Kiểm tra batch WebSocket ${batchIndex + 1}/${wsNodeBatches.length} (${batch.length} node)`);
      
      const results = await Promise.allSettled(
        batch.map(wsUrl => checkWebSocketConnection(wsUrl))
      );
      
      // Thu thập các node chạy thành công
      const successNodes = results
        .filter((result, index) => result.status === 'fulfilled' && result.value)
        .map((_, index) => batch[index]);
        
      runningWsNodes.push(...successNodes);
      console.log(`Tìm thấy ${successNodes.length}/${batch.length} node WebSocket trong batch này`);
    }
    
    return runningWsNodes;
  } catch (error) {
    console.error(`Lỗi khi kiểm tra WebSocket node: ${error.message}`);
    return [];
  }
}

// Kiểm tra kết nối WebSocket
async function checkWebSocketConnection(wsUrl) {
  return new Promise((resolve) => {
    try {
      const ws = new WebSocket(wsUrl);
      const timeoutId = setTimeout(() => {
        try {
          ws.terminate();
        } catch (error) {}
        resolve(false);
      }, TIMEOUT_WHEN_CHECK_RPC);

      ws.on('open', () => {
        clearTimeout(timeoutId);
        
        // Gửi request để kiểm tra chainId
        const subscribeRequest = {
          jsonrpc: '2.0',
          id: 1,
          method: 'eth_chainId',
          params: []
        };
        
        try {
          ws.send(JSON.stringify(subscribeRequest));
        } catch (error) {
          ws.close();
          resolve(false);
        }
        
        // Thiết lập timeout cho response
        const responseTimeout = setTimeout(() => {
          try {
            ws.close();
          } catch (error) {}
          resolve(false);
        }, 2000);
        
        ws.on('message', (data) => {
          clearTimeout(responseTimeout);
          try {
            const response = JSON.parse(data.toString());
            ws.close();
            
            // Kiểm tra chainId
            if (response.result) {
              const nodeChainId = parseInt(response.result, 16);
              if (nodeChainId === chainId) {
                resolve(true);
              } else {
                console.log(`Node WebSocket ${wsUrl} có chainId ${nodeChainId}, mong đợi chainId ${chainId}`);
                resolve(false);
              }
            } else {
              resolve(false);
            }
          } catch (error) {
            ws.close();
            resolve(false);
          }
        });
      });

      ws.on('error', () => {
        clearTimeout(timeoutId);
        resolve(false);
      });
    } catch (error) {
      resolve(false);
    }
  });
}

async function checkRunningRpcFromPublicNodes() {
  console.log("Kiểm tra node từ public nodes...");
  
  try {
    // Fetch từ Ethereum Public Node
    const response = await callRpc(
      "https://ethereum-rpc.publicnode.com",
      "admin_peers",
      []
    );

    if (!response || !response.result) {
      console.log("Không thể lấy thông tin từ public node, đọc từ file cấu hình...");
      
      try {
        const adminNodesData = JSON.parse(fs.readFileSync(`admin-nodes-${chainId}.json`, "utf8"));
        if (adminNodesData && adminNodesData.nodes) {
          // Đọc node từ file
          const nodes = adminNodesData.nodes.map(node => ({ rpc: node.http }));
          const rpcs = filterRpcExistIp(nodes);
          const runningRpcs = await checkRpcsRunning(rpcs, chainId);
          const privateRpcs = await checkPrivateRpc(nodes, chainId);
          return [...runningRpcs, ...privateRpcs];
        }
      } catch (err) {
        console.error(`Lỗi khi đọc file admin-nodes-${chainId}.json: ${err.message}`);
      }
      
      return [];
    }

    const rpcs = filterRpcExistIp(response.result);
    const runningRpcs = await checkRpcsRunning(rpcs, chainId);
    const privateRpcs = await checkPrivateRpc(response.result, chainId);

    return [...runningRpcs, ...privateRpcs];
  } catch (error) {
    console.error("Lỗi khi kiểm tra các node từ public nodes:", error.message);
    return [];
  }
}

async function checkRunningRpcFromExtNodes() {
  console.log("Kiểm tra node từ extrnode.com...");
  
  try {
    let response = await callApi(
      "https://api.extrnode.com/endpoints?is_rpc=true"
    );

    if (!response) {
      console.log("Không thể kết nối tới API extrnode.com");
      return [];
    }

    const fieldFilter = "endpoint";
    const rpcs = filterRpcExistIp(response, fieldFilter);

    const ipField = "endpoint";
    const runningRpcs = await checkRpcsRunning(rpcs, chainId, ipField);

    return runningRpcs;
  } catch (error) {
    console.error("Lỗi khi kiểm tra node từ extrnode.com:", error.message);
    return [];
  }
}

// Đánh giá hiệu suất node dựa trên thời gian phản hồi và độ tin cậy
async function evaluateNodePerformance(nodes) {
  console.log(`Đánh giá hiệu suất ${nodes.length} node...`);
  
  const resultArray = [];
  const nodeBatches = chunk(nodes, BATCH_SIZE);
  
  for (const [batchIndex, batch] of nodeBatches.entries()) {
    console.log(`Đánh giá batch ${batchIndex + 1}/${nodeBatches.length} (${batch.length} node)`);
    
    const batchResults = await Promise.allSettled(
      batch.map(async (nodeUrl) => {
        try {
          let totalResponseTime = 0;
          let successCount = 0;
          
          // Thực hiện nhiều lần để đo hiệu suất
          for (let i = 0; i < PERFORMANCE_CHECK_RETRIES; i++) {
            const startTime = Date.now();
            try {
              // Kiểm tra độ trễ khi lấy block hiện tại
              const response = await callRpc(
                nodeUrl,
                "eth_blockNumber",
                [],
                nodeUrl,
                3000 // Timeout ngắn hơn để đo hiệu suất
              );
              
              if (response && !response.error && response.result) {
                successCount++;
                totalResponseTime += (Date.now() - startTime);
              }
            } catch (error) {
              // Bỏ qua lỗi khi kiểm tra hiệu suất
            }
            
            // Tạm dừng giữa các lần thử
            await delay(100);
          }
          
          // Tính điểm hiệu suất
          // Công thức: (tỉ lệ thành công * 0.7) + (1 - (thời gian phản hồi trung bình / 1000) * 0.3)
          const reliabilityScore = successCount / PERFORMANCE_CHECK_RETRIES;
          const avgResponseTime = successCount > 0 ? totalResponseTime / successCount : 3000;
          const responseTimeScore = Math.max(0, 1 - (avgResponseTime / 3000));
          
          const performanceScore = (reliabilityScore * 0.7) + (responseTimeScore * 0.3);
          
          return {
            url: nodeUrl,
            performanceScore: performanceScore.toFixed(2) * 1,
            reliability: reliabilityScore.toFixed(2) * 1,
            avgResponseTime: avgResponseTime.toFixed(0) * 1
          };
        } catch (error) {
          return {
            url: nodeUrl,
            performanceScore: 0,
            reliability: 0,
            avgResponseTime: 3000
          };
        }
      })
    );
    
    // Thu thập kết quả
    batchResults.forEach((result, index) => {
      if (result.status === 'fulfilled') {
        resultArray.push(result.value);
      } else {
        resultArray.push({
          url: batch[index],
          performanceScore: 0,
          reliability: 0,
          avgResponseTime: 3000
        });
      }
    });
  }
  
  console.log("Đánh giá hiệu suất hoàn tất");
  return resultArray;
}

// Kiểm tra các RPC có hoạt động không và có đúng chainId không
async function checkRpcsRunning(rpcs, targetChainId, ipField = "rpc") {
  console.log(`Kiểm tra ${rpcs.length} node...`);
  
  // Chia thành các batch nhỏ
  const rpcBatches = chunk(rpcs, BATCH_SIZE);
  const runningRpcs = [];
  
  for (const [batchIndex, batch] of rpcBatches.entries()) {
    console.log(`Kiểm tra batch ${batchIndex + 1}/${rpcBatches.length} (${batch.length} node)`);
    
    const rpcPromises = batch.map((rpc) => {
      let rpcUrl = rpc[ipField];
      rpcUrl = rpcUrl.startsWith("http") ? rpcUrl : "http://" + rpcUrl;

      // Kiểm tra cả blockNumber và chainId
      return fetchWithTimeout(
        rpcUrl,
        "eth_chainId", // Kiểm tra chainId
        [],
        rpcUrl
      );
    });

    const rpcResponses = await Promise.allSettled(rpcPromises);
    
    // Lọc các node thành công
    for (let i = 0; i < rpcResponses.length; i++) {
      const rpcResponse = rpcResponses[i];
      if (rpcResponse.status === 'fulfilled' && !rpcResponse.value.error) {
        // Kiểm tra chainId
        try {
          const nodeChainId = parseInt(rpcResponse.value.result, 16);
          if (nodeChainId === targetChainId) {
            runningRpcs.push(rpcResponse.value.id);
          } else {
            console.log(`Node ${batch[i][ipField]} có chainId ${nodeChainId}, mong đợi chainId ${targetChainId}`);
          }
        } catch (error) {
          // Bỏ qua node lỗi
        }
      }
    }
    
    console.log(`Tìm thấy ${runningRpcs.length} node đang chạy trong tất cả các batch đã kiểm tra`);
  }

  return runningRpcs;
}

function filterRpcExistIp(rpcList, ipField = "rpc") {
  const validRpcList = rpcList.filter((rpc) => {
    // Kiểm tra rpc có hợp lệ không
    return !!rpc[ipField];
  });
  return validRpcList;
}

// Fetch rpc function với timeout
const fetchWithTimeout = (url, method, params, id = 1, timeout = TIMEOUT_WHEN_CHECK_RPC) => {
  return Promise.race([
    callRpc(url, method, params, id),
    new Promise((_, reject) =>
      setTimeout(() => reject(new Error("timeout")), timeout)
    ),
  ]);
};

async function callRpc(url, method, params, id = 1) {
  try {
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
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    
    const responseJson = await response.json();
    return responseJson;
  } catch (error) {
    return { error: error.message, id: id };
  }
}

async function callApi(url, options) {
  try {
    const response = await fetch(url, options);
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    
    const responseJson = await response.json();
    return responseJson;
  } catch (error) {
    console.error(`Lỗi khi gọi API ${url}: ${error.message}`);
    return null;
  }
}

// Hàm delay
function delay(time) {
  return new Promise((resolve) => {
    setTimeout(resolve, time);
  });
}

/* -------------------------------------------------------------------------- */
/*                             Check private node                             */
/* -------------------------------------------------------------------------- */
/**
 * Chia một mảng thành các phần nhỏ (chunks) với kích thước xác định.
 * @param {Array} array - Mảng cần chia.
 * @param {number} size - Kích thước của mỗi phần.
 * @returns {Array} - Mảng chứa các phần nhỏ.
 */
function chunk(array, size) {
  if (!Array.isArray(array)) {
    throw new TypeError("Tham số đầu tiên phải là mảng.");
  }
  if (typeof size !== "number" || size <= 0) {
    throw new RangeError("Kích thước chunk phải là số dương.");
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

async function checkPrivateRpc(rpcs, targetChainId) {
  console.log("Kiểm tra các private node...");
  
  try {
    const result = {
      success: [],
      failed: [],
      ips: [],
    };

    // Thay đổi logic tìm kiếm private node từ Solana sang EVM
    const validRpcList = rpcs
      .filter((rpc) => {
        // Kiểm tra các trường có thể chứa thông tin node
        return rpc["address"] || rpc["remoteAddress"] || rpc["enode"];
      })
      .map((rpc) => {
        let ip;
        if (rpc["address"]) {
          ip = rpc["address"].split(":")[0];
        } else if (rpc["remoteAddress"]) {
          ip = rpc["remoteAddress"].split(":")[0];
        } else if (rpc["enode"]) {
          // Trích xuất IP từ enode URL nếu có
          const match = rpc["enode"].match(/@([^:]+):/);
          ip = match ? match[1] : null;
        }

        if (ip) {
          // Các port phổ biến của EVM RPC
          const httpPorts = currentChainConfig.ports.http || [8545, 30303, 80];
          return httpPorts.map(port => `http://${ip}:${port}`);
        }
        return [];
      });

    const handle = async (rpcUrl) => {
      try {
        // Kiểm tra node bằng eth_chainId
        const response = await callRpc(
          rpcUrl,
          "eth_chainId",
          []
        );

        // Kiểm tra node trả về chainId phù hợp
        const responseHex = response?.result;
        
        if (responseHex && responseHex.startsWith('0x')) {
          const nodeChainId = parseInt(responseHex, 16);
          
          // Kiểm tra chainId
          if (nodeChainId !== targetChainId) {
            console.log(`Node ${rpcUrl} có chainId ${nodeChainId}, mong đợi ${targetChainId}`);
            result.failed.push(rpcUrl);
            return false;
          }
          
          const ip = getRpcIp(rpcUrl);
          if (result.ips.includes(ip)) return false;
          result.ips.push(ip);
          result.success.push(rpcUrl);
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
    
    console.log(`Tổng số node cần kiểm tra: ${validRpcList.length}`);
    const chunked = chunk(validRpcList, BATCH_SIZE);

    console.log(`Tổng số batch: ${chunked.length}`);
    for (const [i, rpcList] of chunked.entries()) {
      console.log(`Kiểm tra batch private node ${i + 1}/${chunked.length}`);
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

    console.log(`Tìm thấy ${result.success.length} private node`);
    return result.success;
  } catch (error) {
    console.error("Lỗi khi kiểm tra private node:", error.message);
    return [];
  }
}
