#!/usr/bin/env python3
"""
Binary search to find which transaction causes cumulative gas to diverge.
"""

import json
import urllib.request
import time
import sys
import ssl

ssl._create_default_https_context = ssl._create_unverified_context

RPC_URL = "https://eth.llamarpc.com"
BLOCK_NUM = 23800003

# Erigon debug log cumulative gas (parsed from log, first 50 transactions for testing)
# Format: txIndex -> cumulativeGas
ERIGON_CUMULATIVE = {
    0: 117778,
    1: 239182,
    2: 260182,
    # Add more from Erigon log output...
}

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

def get_receipt(tx_hash):
    """Get transaction receipt"""
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
                    "from": result.get("from", ""),
                    "to": result.get("to", ""),
                    "type": int(result.get("type", "0x0"), 16),
                }
    except Exception as e:
        pass
    return None

def get_tx_details(tx_hash):
    """Get transaction details"""
    payload = {
        "jsonrpc": "2.0",
        "method": "eth_getTransactionByHash",
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
                    "type": int(result.get("type", "0x0"), 16),
                    "from": result.get("from", ""),
                    "to": result.get("to", ""),
                    "authorizationList": result.get("authorizationList", []),
                }
    except Exception as e:
        pass
    return None

def main():
    print("Finding gas difference location in block 23800003")
    print("=" * 80)
    
    # Get all transactions
    print("Fetching transactions...")
    tx_hashes = get_block_tx_hashes(BLOCK_NUM)
    print(f"Total transactions: {len(tx_hashes)}")
    
    # Binary search to find first difference
    # Check at indices: 0, 50, 100, 150, 200, 220
    print("\nPhase 1: Checking key positions...")
    key_indices = [0, 10, 20, 30, 40, 50, 75, 100, 125, 150, 175, 200, 210, 215, 220]
    
    prev_cumulative = 0
    erigon_prev_cumulative = 0
    
    for idx in key_indices:
        if idx >= len(tx_hashes):
            continue
        receipt = get_receipt(tx_hashes[idx])
        if receipt:
            # We need Erigon cumulative for this to work...
            # For now, just print RPC values
            print(f"TX {idx:3d}: RPC_cumulative={receipt['cumulativeGasUsed']:10d}, gasUsed={receipt['gasUsed']:8d}, type={receipt['type']}")
        time.sleep(0.25)
    
    print("\n" + "=" * 80)
    print("Next steps:")
    print("1. Add the Erigon cumulative gas values to ERIGON_CUMULATIVE dict")
    print("2. Or compare Erigon debug log output directly")
    print("\nLooking for type=4 (EIP-7702) transactions...")
    
    # Find type=4 transactions
    type4_txs = []
    for i, tx_hash in enumerate(tx_hashes):
        tx = get_tx_details(tx_hash)
        if tx and tx["type"] == 4:
            type4_txs.append((i, tx_hash, tx))
            print(f"Found type=4 TX at index {i}: {tx_hash[:20]}...")
            receipt = get_receipt(tx_hash)
            if receipt:
                print(f"  gasUsed={receipt['gasUsed']}, cumulative={receipt['cumulativeGasUsed']}")
                if tx.get("authorizationList"):
                    print(f"  authorizationList has {len(tx['authorizationList'])} entries")
        time.sleep(0.15)
        # Check all
        if i % 20 == 0:
            print(f"Checked {i} transactions...")
    
    if type4_txs:
        print(f"\nFound {len(type4_txs)} type=4 (EIP-7702) transactions in first 50")
    else:
        print("\nNo type=4 transactions found in first 50")

if __name__ == "__main__":
    main()

