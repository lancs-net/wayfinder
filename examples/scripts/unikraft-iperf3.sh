#!/usr/bin/env bash

set -xe

QEMU_GUEST=${QEMU_GUEST:-$(which qemu-guest)}
BRIDGE=ukbench$UKBENCH_CORE_ID0 # create a unique bridge
BRIDGE_IP="172.130.0.1"
UNIKERNEL_IMAGE=${UNIKERNEL_IMAGE:-"/usr/src/unikraft/apps/iperf3/build/iperf3_kvm-x86_64"}
UNIKERNEL_IP="172.130.0.${UKBENCH_CORE_ID0}"

if [[ ! -f $UNIKERNEL_IMAGE ]]; then
  echo "Missing unikernel image!"
  exit 1
fi

function cleanup {
  ifconfig $BRIDGE down || true
  brctl delbr $BRIDGE || true
}

trap "cleanup" EXIT

# Install testing tools
apt-get update
apt-get install -y \
  jq \
  iperf3 \
  procps \
  bridge-utils \
  net-tools

echo "Creating bridge..."
brctl addbr $BRIDGE || true
ifconfig $BRIDGE $BRIDGE_IP

echo "Starting unikernel..."
taskset -c $UKBENCH_CORE_ID0 \
  $QEMU_GUEST \
    -x \
    -k $UNIKERNEL_IMAGE \
    -m 1024 \
    -b $BRIDGE \
    -p $UKBENCH_CORE_ID1 \
    -a "netdev.ipv4_addr=${UNIKERNEL_IP} netdev.ipv4_gw_addr=172.130.0.254 netdev.ipv4_subnet_mask=255.255.255.0 -- -s"

# make sure that the server has properly started
sleep 5

echo "Starting experiment..."
taskset -c $UKBENCH_CORE_ID2 \
  iperf3 -4 -c $UNIKERNEL_IP -J &> /results.json

echo "Done!"
