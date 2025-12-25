#!/usr/bin/env python3
"""
Compare every transaction's gasUsed between Erigon and RPC for block 23800003.
User needs to provide Erigon output.

Run with: python3 compare_all_gas.py erigon_output.txt
Where erigon_output.txt contains lines like:
[DEBUG] TX receipt txIndex=0 txHash=0x... gasUsed=117778 ...
"""

import json
import urllib.request
import time
import ssl
import sys
import re

ssl._create_default_https_context = ssl._create_unverified_context

RPC_URL = "https://eth.llamarpc.com"
BLOCK_NUM = 23800003

def get_all_receipts():
    """Get all receipts for block 23800003"""
    # Get block transactions first
    payload = {
        "jsonrpc": "2.0",
        "method": "eth_getBlockByNumber",
        "params": [hex(BLOCK_NUM), False],
        "id": 1
    }
    
    req = urllib.request.Request(
        RPC_URL,
        data=json.dumps(payload).encode('utf-8'),
        headers={'Content-Type': 'application/json'}
    )
    
    with urllib.request.urlopen(req, timeout=30) as response:
        data = json.loads(response.read().decode())
        tx_hashes = data["result"]["transactions"]
    
    print(f"Fetching {len(tx_hashes)} receipts from RPC...")
    
    receipts = {}
    for i, tx_hash in enumerate(tx_hashes):
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
                    receipts[i] = {
                        "gasUsed": int(result["gasUsed"], 16),
                        "cumulativeGas": int(result["cumulativeGasUsed"], 16),
                        "hash": tx_hash
                    }
        except Exception as e:
            print(f"Error fetching TX {i}: {e}")
        
        if i % 20 == 0:
            print(f"  Fetched {i}/{len(tx_hashes)}...")
        time.sleep(0.15)
    
    return receipts

def parse_erigon_output(filename):
    """Parse Erigon debug output"""
    erigon_receipts = {}
    pattern = r'txIndex=(\d+).*gasUsed=(\d+)'
    
    with open(filename, 'r') as f:
        for line in f:
            match = re.search(pattern, line)
            if match:
                idx = int(match.group(1))
                gas = int(match.group(2))
                erigon_receipts[idx] = {"gasUsed": gas}
    
    return erigon_receipts

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 compare_all_gas.py <erigon_output.txt>")
        print("Or run without args to just fetch RPC receipts and save to file")
        
        # Just fetch and save RPC data
        receipts = get_all_receipts()
        
        print(f"\nFetched {len(receipts)} receipts")
        print(f"Total gas: {receipts[220]['cumulativeGas']}")
        
        # Save to file
        with open("rpc_receipts.json", "w") as f:
            json.dump(receipts, f, indent=2)
        print("Saved to rpc_receipts.json")
        
        # Print in format user can compare
        print("\n--- RPC Gas Values (format: txIndex=N gasUsed=M) ---")
        for i in range(len(receipts)):
            if i in receipts:
                print(f"txIndex={i} gasUsed={receipts[i]['gasUsed']}")
        return
    
    # Compare mode
    erigon_file = sys.argv[1]
    erigon_receipts = parse_erigon_output(erigon_file)
    print(f"Parsed {len(erigon_receipts)} receipts from Erigon output")
    
    # Fetch RPC receipts
    rpc_receipts = get_all_receipts()
    print(f"Fetched {len(rpc_receipts)} receipts from RPC")
    
    # Compare
    differences = []
    total_diff = 0
    
    for i in sorted(set(erigon_receipts.keys()) | set(rpc_receipts.keys())):
        erigon_gas = erigon_receipts.get(i, {}).get("gasUsed", "N/A")
        rpc_gas = rpc_receipts.get(i, {}).get("gasUsed", "N/A")
        
        if erigon_gas != "N/A" and rpc_gas != "N/A":
            diff = erigon_gas - rpc_gas
            if diff != 0:
                differences.append((i, erigon_gas, rpc_gas, diff))
                total_diff += diff
    
    print(f"\n=== Comparison Results ===")
    print(f"Transactions with differences: {len(differences)}")
    print(f"Total gas difference: {total_diff}")
    
    if differences:
        print("\nDifferences:")
        for idx, erigon, rpc, diff in differences:
            print(f"  TX {idx}: Erigon={erigon}, RPC={rpc}, Diff={diff:+d}")

if __name__ == "__main__":
    main()

