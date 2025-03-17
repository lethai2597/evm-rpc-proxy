
# HowRare Evm RPC proxy
HowRare Evm RPC proxy is there to allow project creators to freely route Evm RPC calls to different Evm nodes utilizing prioritization and capping.

It allows to:
- Route requests between fast local node(s) with partial chain data and remote node(s) with full chain history
- Keep requests below allocated limits with per-function, per-request and per-transfer capping
- Spread the load across many nodes/providers
- Automatically detect and skip failed/overloaded/timeouting nodes
- Automatically re-do failed requests on different node if they timed-out or returned an error


## Building the software
Run following commands to build for windows / linux. Golang 1.18 required. 
<pre>cd scripts
./run-eth-proxy.sh
./run-update-eth-rpc.sh
</pre>

Now you can run main or see [Installation Instructions](doc/INSTALL.md)

## Node types
There are 2 node types defined
- Public - this node stores full archival chain data
- Private - fast local node (usually with partial chain data)
If you don't need to distinct and you want to use the proxy just to route your requests to different providers for loadbalancing / failover - you should setup all nodes as a private type.

This should be default, simplest mode of operation. You'll setup all your nodes as private nodes, and then you can connect to Evm proxy via any api just like you'd do to a "normal" Evm node, using port 8545.

## Configuration
```json
{
"BIND_TO": "h127.0.0.1:8545,h8.8.8.8:8545,",

"FORCE_START":true,
"DEBUG":false,
"VERBOSE":false,
"RUN_SERVICES":"*",

"EVM_NODES":[{"url":"http://127.0.0.1:8545", "public":false, "score_modifier":-90000}],
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

## Throttling
There is automatic throttling/routing implemented. If node is throttled the request will be routed to different node. If all available nodes are throttled so there's no node to pick to run the request - you will get response with error attribute and issue description.
```json
{"error":"Throttled public node, please wait","throttle_info":{"requests":{"description":"requests made","max":99,"value":3},"requests_fn":{"description":"requests made calling single function","max":39,"value":3},"received":{"description":"bytes received","max":1000000,"value":4735645}},"throttle_timespan_seconds":12,"throttled":true,"throttled_comment":"Too much data received 4735645/1000000"}
```