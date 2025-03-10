pm2 delete eclipse-update-rpc
pm2 start eclipse-update-rpc.js --name "eclipse-update-rpc" --time --cron "0 */2 * * * *"
