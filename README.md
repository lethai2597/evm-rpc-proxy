
<p align="center">
  <img src="doc/logo_light.svg" width="350">
</p>

# HowRare Solana RPC proxy
HowRare Solana RPC proxy is there to allow project creators to freely route Solana RPC calls to different Solana nodes utilizing prioritization and capping.

It allows to:
- Route requests between fast local node(s) with partial chain data and remote node(s) with full chain history
- Keep requests below allocated limits with per-function, per-request and per-transfer capping
- Spread the load across many nodes/providers
- Automatically detect and skip failed/overloaded/timeouting nodes
- Automatically re-do failed requests on different node if they timed-out or returned an error

What's new in Solproxy 0.3?
 - Removed genesysgo support 
 - Authorization. You can add authorization or any other header using "header" parameter. Separate individual headers using newline.
```json
"SOL_NODES":[{"header":"Authorization:Bearer b6082xxxxx\nCache-Control:no-cache", "url":"https://rpc.hellomoon.io/", "public":false, "score_modifier":-90000}],
```

What's new in Solproxy 0.2?
- GenesysGo support, see [adding GenesysGo node](doc/GENESYS.md)
- Health check for pausing lagging nodes, see [setup instructions](doc/HEALTH_CHECK.md)
- Updated dashboard UI
- Allow to throttle using any time interval (eg. cap requests daily or per second)

<p align="center">
  <img src="doc/server-status-solproxy.png" width="550">
</p>

## Building the software
Run following commands to build for windows / linux. Golang 1.18 required. 
<pre>cd solproxy/goevm/main
go build main.go
</pre>

Now you can run main or see [Installation Instructions](doc/INSTALL.md)

## Node types
There are 2 node types defined
- Public - this node stores full archival chain data
- Private - fast local node (usually with partial chain data)
If you don't need to distinct and you want to use the proxy just to route your requests to different providers for loadbalancing / failover - you should setup all nodes as a private type.

This should be default, simplest mode of operation. You'll setup all your nodes as private nodes, and then you can connect to Solana proxy via any api just like you'd do to a "normal" Solana node, using port 8545.

## Configuration
```json
{
"BIND_TO": "h127.0.0.1:8545,h8.8.8.8:8545,",

"FORCE_START":true,
"DEBUG":false,
"VERBOSE":false,
"RUN_SERVICES":"*",

"SOL_NODES":[{"url":"http://127.0.0.1:8899", "public":false, "score_modifier":-90000},
		{"url":"https://api.mainnet-beta.solana.com", "public":false, "throttle":"r,80,10;f,30,10;d,80000000,30", "probe_time":20},
		{"url":"https://solana-api.projectserum.com", "public":false, "throttle":"r,80,10;f,30,10;d,80000000,30", "probe_time":20}],
}
```
Configuration should be self-explanatory. You need to add h prefix before each IP the proxy will bind to. It'll listen for new connection on this IP/Port. There's a possibility to communicate with proxy using pure TCP by skipping the prefix.

Throttle can be configured in following way:
- r[equests],time_in_seconds,limit
- f[unction call],time_in_seconds,limit
- d[ata received],time_in_seconds,limit in bytes

## Accessing proxy information
http://127.0.0.1:8545/?action=server-status
You can access server-status page by using server-status action. There's also PHP script available to password-protect the status page so it can be accessible from outside.

http://127.0.0.1:8545/?action=getSolanaInfo
This url will return throttling status for public and private nodes.

http://127.0.0.1:8545/?action=getFirstAvailableBlock
Gets first available block for public and private nodes.

## Throttling
There is automatic throttling/routing implemented. If node is throttled the request will be routed to different node. If all available nodes are throttled so there's no node to pick to run the request - you will get response with error attribute and issue description.
```json
{"error":"Throttled public node, please wait","throttle_info":{"requests":{"description":"requests made","max":99,"value":3},"requests_fn":{"description":"requests made calling single function","max":39,"value":3},"received":{"description":"bytes received","max":1000000,"value":4735645}},"throttle_timespan_seconds":12,"throttled":true,"throttled_comment":"Too much data received 4735645/1000000"}
```


Please see [Advanced usage](doc/ADVANCED.md) for information about more complex usage scenarios.

## Sử dụng Ethereum RPC Proxy

Server hiện hỗ trợ gọi Ethereum RPC thông qua endpoint `/action/ethereumRaw`. 

### Các phương thức Ethereum phổ biến

Proxy hỗ trợ các phương thức Ethereum JSON-RPC chuẩn, bao gồm:

1. `eth_blockNumber` - Lấy số block mới nhất
2. `eth_getBalance` - Lấy số dư ETH của một địa chỉ
3. `eth_call` - Thực hiện lệnh gọi đến smart contract (read-only)
4. `eth_getTransactionByHash` - Lấy thông tin giao dịch từ hash
5. `eth_getLogs` - Lấy logs từ smart contract

### Cách gọi RPC

Proxy hỗ trợ hai cách gọi RPC:

#### 1. Cách gọi chuẩn JSON-RPC 2.0 (Khuyến nghị)

Sử dụng phương thức POST với body JSON theo định dạng JSON-RPC 2.0:

```bash
curl --location 'http://127.0.0.1:8545/action/ethereumRaw' \
--header 'Content-Type: application/json' \
--data '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "eth_blockNumber",
  "params": []
}'
```

#### 2. Cách gọi qua URL (Hỗ trợ cho tương thích ngược)

```
/action/ethereumRaw?method=eth_blockNumber
```

### Ví dụ gọi RPC chuẩn

#### Lấy số block mới nhất
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "eth_blockNumber",
  "params": []
}
```

#### Lấy số dư ETH
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "eth_getBalance",
  "params": ["0xYourEthereumAddress", "latest"]
}
```

#### Gọi đến một hàm trên smart contract
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "eth_call",
  "params": [
    {
      "to": "0xContractAddress",
      "data": "0xfunctionSelector"
    },
    "latest"
  ]
}
```

### Cấu hình

Có thể thêm nhiều node RPC bằng cách chỉnh sửa hàm `_read_node_config()` trong `goevm/main/main.go`.
