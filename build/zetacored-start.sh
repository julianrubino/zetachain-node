#!/bin/bash

NODE_NUMBER=$1
source /etc/environment
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:/root/go/bin

if [ -z ${MYIP} ]; then 
    # If MYIP is not set, use the private IP of the host
    echo "MYIP ENV Variable Not Set -- Setting it automatically using host IP"
    export MYIP=$(hostname -i)
fi

echo "Starting Zetacore Node $NODE_NUMBER"

FILE="/root/.zetacore/config/app.toml"
if  [[ ! -f "$FILE" ]]; then
    echo "Copying Config From /zetashared/node$NODE_NUMBER/"
    cp -rf /zetashared/node$NODE_NUMBER/* /root/.zetacore/
fi


zetacored start \
    --rpc.laddr "tcp://0.0.0.0:26657" \
    --rpc.pprof_laddr "0.0.0.0:6060"  \
    --address "tcp://$MYIP:26658" \
    --moniker "node$NODE_NUMBER" 2>&1 | tee /root/.zetacore/zetacored.log

    # --proxy_app "tcp://0.0.0.0:26658" \


