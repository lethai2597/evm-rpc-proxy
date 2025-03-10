#!/bin/bash

LIST_RPC=$(curl https://api.mainnet-beta.solana.com -X POST -H "Content-Type: application/json" -d '
{
"jsonrpc": "2.0", "id": 1,
"method": "getClusterNodes"
}
' | jq .result)

# echo $LIST_RPC

# count total valid call rpc
CALL_SUCCESS=0

for row in $(echo "${LIST_RPC}" | jq -r ".[] | @base64"); do
    _jq() {
        echo ${row} | base64 --decode | jq -r ${1}
    }

    RPC_URL=$(_jq '.rpc')

    curl -s --max-time 5 --connect-timeout 5 ${RPC_URL} -X POST -H "Content-Type: application/json" -d '
    {
    "jsonrpc": "2.0", "id": 1,
    "method": "getAccountInfo",
    "params": ["So11111111111111111111111111111111111111112"]
    }
    '
    # curl success then increase CALL_SUCCESS 1
    if [ $? -eq 0 ]; then
        CALL_SUCCESS=$((CALL_SUCCESS + 1))
        echo "RPC_URL: ${RPC_URL}"
        echo "CALL_SUCCESS: ${CALL_SUCCESS}"
    fi
done

echo "CALL_SUCCESS: ${CALL_SUCCESS}"
