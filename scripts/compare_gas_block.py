#!/usr/bin/env python3
"""
Compare Erigon gas usage with public RPC for block 23800003.
Focus on finding where cumulative gas diverges.
"""

import json
import urllib.request
import time
import sys
import ssl

ssl._create_default_https_context = ssl._create_unverified_context

RPC_URL = "https://eth.llamarpc.com"
BLOCK_NUM = 23800003  # 0x16b28c3
EXPECTED_TOTAL_GAS = 26758527  # From header
ERIGON_TOTAL_GAS = 26760656  # Erigon computed
GAS_DIFF = ERIGON_TOTAL_GAS - EXPECTED_TOTAL_GAS  # 2129

def get_block_tx_hashes(block_num):
    """Get all transaction hashes in a block"""
    payload = {
        "jsonrpc": "2.0",
        "method": "eth_getBlockByNumber",
        "params": [hex(block_num), False],
        "id": 1
    }
    
    req = urllib.request.Request(
        RPC_URL,
        data=json.dumps(payload).encode('utf-8'),
        headers={'Content-Type': 'application/json'}
    )
    
    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            data = json.loads(response.read().decode())
            if data.get("result"):
                return data["result"]["transactions"]
    except Exception as e:
        print(f"Error getting block: {e}")
    return []

def get_receipt_from_rpc(tx_hash):
    """Get transaction receipt from public RPC"""
    payload = {
        "jsonrpc": "2.0",
        "method": "eth_getTransactionReceipt",
        "params": [tx_hash],
        "id": 1
    }
    
    req = urllib.request.Request(
        RPC_URL,
        data=json.dumps(payload).encode('utf-8'),
        headers={'Content-Type': 'application/json'}
    )
    
    try:
        with urllib.request.urlopen(req, timeout=10) as response:
            data = json.loads(response.read().decode())
            if data.get("result"):
                result = data["result"]
                return {
                    "gasUsed": int(result["gasUsed"], 16),
                    "cumulativeGasUsed": int(result["cumulativeGasUsed"], 16),
                    "type": int(result.get("type", "0x0"), 16),
                }
    except Exception as e:
        pass
    return None

def main():
    print(f"Block {BLOCK_NUM} Gas Analysis")
    print(f"Expected total gas (header): {EXPECTED_TOTAL_GAS}")
    print(f"Erigon computed total:       {ERIGON_TOTAL_GAS}")
    print(f"Difference:                  {GAS_DIFF}")
    print("-" * 100)
    
    # Get all transactions in the block
    print("Fetching block transactions...")
    tx_hashes = get_block_tx_hashes(BLOCK_NUM)
    print(f"Total transactions: {len(tx_hashes)}")
    print("-" * 100)
    
    if not tx_hashes:
        print("Failed to get transactions")
        return
    
    # Check every Nth transaction to find where cumulative gas diverges
    # First check first and last few
    check_indices = [0, 1, 2, 50, 100, 150, 200, 210, 215, 218, 219, 220]
    check_indices = [i for i in check_indices if i < len(tx_hashes)]
    
    print("Checking selected transactions for cumulative gas (RPC value)...")
    results = []
    
    for idx in check_indices:
        tx_hash = tx_hashes[idx]
        receipt = get_receipt_from_rpc(tx_hash)
        if receipt:
            results.append({
                "index": idx,
                "gasUsed": receipt["gasUsed"],
                "cumulativeGas": receipt["cumulativeGasUsed"],
                "type": receipt["type"]
            })
            print(f"TX {idx:3d}: gasUsed={receipt['gasUsed']:8d}, cumulative={receipt['cumulativeGasUsed']:10d}, type={receipt['type']}")
        time.sleep(0.3)
    
    print("-" * 100)
    
    # Get last transaction's cumulative gas
    print("Getting last transaction (TX 220) cumulative gas from RPC...")
    last_receipt = get_receipt_from_rpc(tx_hashes[220])
    if last_receipt:
        print(f"TX 220: cumulative={last_receipt['cumulativeGasUsed']} (should equal header gas {EXPECTED_TOTAL_GAS})")
        print(f"Match: {last_receipt['cumulativeGasUsed'] == EXPECTED_TOTAL_GAS}")

if __name__ == "__main__":
    main()
