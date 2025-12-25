#!/usr/bin/env python3
"""
Find which transaction has gas difference between Erigon and RPC.
Paste Erigon debug log output (TX receipts) when prompted.
"""

import json
import urllib.request
import re
import ssl
import time

ssl._create_default_https_context = ssl._create_unverified_context

RPC_URL = "https://eth.llamarpc.com"

def get_block_receipts(block_num):
    """Get all transaction hashes and fetch receipts"""
    # First get block
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
    
    with urllib.request.urlopen(req, timeout=30) as response:
        data = json.loads(response.read().decode())
        tx_hashes = data["result"]["transactions"]
    
    print(f"Fetching {len(tx_hashes)} receipts from RPC...")
    
    rpc_gas = {}
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
                    rpc_gas[i] = int(result["gasUsed"], 16)
        except:
            pass
        
        if i % 50 == 0:
            print(f"  {i}/{len(tx_hashes)}...")
        time.sleep(0.12)
    
    return rpc_gas

def parse_erigon_log(log_text):
    """Parse Erigon debug log to extract gasUsed per transaction"""
    erigon_gas = {}
    # Pattern: txIndex=N ... gasUsed=M
    pattern = r'txIndex=(\d+).*?gasUsed=(\d+)'
    for match in re.finditer(pattern, log_text):
        idx = int(match.group(1))
        gas = int(match.group(2))
        erigon_gas[idx] = gas
    return erigon_gas

def main():
    print("Block 23800003 Gas Comparison Tool")
    print("=" * 60)
    
    block_num = 23800003
    
    # Get RPC data
    print(f"\nStep 1: Fetching RPC receipts for block {block_num}...")
    rpc_gas = get_block_receipts(block_num)
    print(f"Fetched {len(rpc_gas)} receipts from RPC")
    
    # Save RPC data
    with open("rpc_gas_23800003.json", "w") as f:
        json.dump(rpc_gas, f)
    print("Saved RPC data to rpc_gas_23800003.json")
    
    print("\n" + "=" * 60)
    print("Step 2: Paste Erigon debug log output below")
    print("(Lines containing 'txIndex=... gasUsed=...')")
    print("End input with empty line or Ctrl+D")
    print("-" * 60)
    
    lines = []
    try:
        while True:
            line = input()
            if not line.strip():
                break
            lines.append(line)
    except EOFError:
        pass
    
    log_text = "\n".join(lines)
    erigon_gas = parse_erigon_log(log_text)
    
    if not erigon_gas:
        print("\nNo Erigon gas data parsed. Using saved RPC data only.")
        print("\nRPC gas values (first 20):")
        for i in range(min(20, len(rpc_gas))):
            if i in rpc_gas:
                print(f"  TX {i}: {rpc_gas[i]}")
        return
    
    print(f"\nParsed {len(erigon_gas)} transactions from Erigon log")
    
    # Compare
    print("\n" + "=" * 60)
    print("Step 3: Comparing gas values...")
    print("-" * 60)
    
    differences = []
    cumulative_diff = 0
    
    for i in sorted(set(erigon_gas.keys()) | set(rpc_gas.keys())):
        e_gas = erigon_gas.get(i)
        r_gas = rpc_gas.get(i)
        
        if e_gas is not None and r_gas is not None:
            diff = e_gas - r_gas
            if diff != 0:
                cumulative_diff += diff
                differences.append((i, e_gas, r_gas, diff, cumulative_diff))
                print(f"*** TX {i}: Erigon={e_gas}, RPC={r_gas}, Diff={diff:+d}, CumulDiff={cumulative_diff:+d}")
    
    print("\n" + "=" * 60)
    print(f"Total differences found: {len(differences)}")
    print(f"Total gas difference: {cumulative_diff}")
    
    if differences:
        print("\nSummary of differing transactions:")
        for idx, e, r, d, cd in differences:
            print(f"  TX {idx}: diff={d:+d} (Erigon={e}, RPC={r})")

if __name__ == "__main__":
    main()

