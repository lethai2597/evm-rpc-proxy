pm2 delete update-rpc
pm2 start update-rpc.js --name "update-rpc" --time --cron "0 */2 * * * *"
