## Installation
It's not required to install, you can run directly from console and the proxy will work right after compiling, using standard Evm public nodes.

## Installing as a service
- Create a new user, it can be called evmproxy
- Copy the content of goevm/main into /home/evmproxy (it needs to contain compiled sources) or just download a package from this website and put everything under /home/evmproxy
- Create a directory /home/evmproxy/log owned by evmproxy user
- Run sudo vi /etc/systemd/system/evmproxy.service and place the following content into evmproxy.service file

<pre>[Unit]
After=network-online.target
Wants=network-online.target
Description=Evm Proxy Service

[Service]
User=evmproxy
LimitNOFILE=524288
LimitMEMLOCK=1073741824
LimitNICE=-10
Nice=-10
ExecStart=/bin/sh -c 'cd /home/evmproxy; export GODEBUG=gctrace=1; started=`date --rfc-3339=seconds`; echo Starting Evm Proxy $started; ./main 1>"log/log-$started.txt" 2>"log/error-$started.log.txt";'
Type=simple
PrivateNetwork=false
PrivateTmp=false
ProtectSystem=false
ProtectHome=false
KillMode=control-group
Restart=always
DefaultTasksMax=65536
TasksMax=65536
RestartSec=30
StartLimitIntervalSec=200
StartLimitBurst=10

[Install]
WantedBy=multi-user.target</pre>
- Run systemctl daemon-reload
- Run systemctl enable worker-node

The proxy should be now running
