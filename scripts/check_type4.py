#!/usr/bin/env python3
"""
Check for type=4 transactions in multiple blocks.
"""

import json
import urllib.request
import time
import ssl

ssl._create_default_https_context = ssl._create_unverified_context

RPC_URL = "https://eth.llamarpc.com"

def get_block(block_num):
    """Get block with full transaction objects"""
    payload = {
        "jsonrpc": "2.0",
        "method": "eth_getBlockByNumber",
        "params": [hex(block_num), True],  # True = include full tx objects
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
                return data["result"]
    except Exception as e:
        print(f"Error getting block: {e}")
    return None

def main():
    blocks_to_check = [23800000, 23800001, 23800002, 23800003]
    
    for block_num in blocks_to_check:
        print(f"\n=== Block {block_num} ===")
        block = get_block(block_num)
        if not block:
            print("  Failed to fetch")
            continue
        
        txs = block.get("transactions", [])
        print(f"  Total transactions: {len(txs)}")
        print(f"  Gas used (header): {int(block.get('gasUsed', '0x0'), 16)}")
        
        type4_count = 0
        for i, tx in enumerate(txs):
            tx_type = int(tx.get("type", "0x0"), 16)
            if tx_type == 4:
                type4_count += 1
                auth_list = tx.get("authorizationList", [])
                print(f"  TX {i} (type=4): hash={tx['hash'][:20]}..., authList={len(auth_list)} entries")
                for j, auth in enumerate(auth_list[:3]):
                    print(f"    auth[{j}]: address={auth.get('address', 'N/A')}")
        
        print(f"  Type=4 transactions: {type4_count}")
        time.sleep(0.5)

if __name__ == "__main__":
    main()

