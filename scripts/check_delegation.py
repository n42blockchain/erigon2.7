#!/usr/bin/env python3
"""
Check details of the type=4 transaction in block 23800001 
and look for references to delegation address in block 23800003.
"""

import json
import urllib.request
import time
import ssl

ssl._create_default_https_context = ssl._create_unverified_context

RPC_URL = "https://eth.llamarpc.com"

def rpc_call(method, params):
    """Generic RPC call"""
    payload = {
        "jsonrpc": "2.0",
        "method": method,
        "params": params,
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
            return data.get("result")
    except Exception as e:
        print(f"Error: {e}")
    return None

def main():
    # Get the type=4 transaction from block 23800001
    print("=== Block 23800001 Type=4 Transaction (TX 79) ===")
    block = rpc_call("eth_getBlockByNumber", [hex(23800001), True])
    if not block:
        print("Failed to get block")
        return
    
    tx79 = block["transactions"][79]
    tx_hash = tx79["hash"]
    print(f"Hash: {tx_hash}")
    print(f"From: {tx79['from']}")
    print(f"To: {tx79.get('to', 'None')}")
    print(f"Type: {int(tx79['type'], 16)}")
    
    auth_list = tx79.get("authorizationList", [])
    print(f"Authorization List ({len(auth_list)} entries):")
    delegation_addresses = []
    for i, auth in enumerate(auth_list):
        addr = auth.get("address", "N/A")
        delegation_addresses.append(addr)
        print(f"  [{i}] address={addr}")
        print(f"      chainId={auth.get('chainId', 'N/A')}")
        print(f"      nonce={auth.get('nonce', 'N/A')}")
        # The 'authority' is derived from the signature, it's the signer
    
    # Get receipt for gas info
    receipt = rpc_call("eth_getTransactionReceipt", [tx_hash])
    if receipt:
        print(f"\nReceipt:")
        print(f"  Status: {int(receipt['status'], 16)}")
        print(f"  GasUsed: {int(receipt['gasUsed'], 16)}")
        print(f"  CumulativeGas: {int(receipt['cumulativeGasUsed'], 16)}")
    
    # Now check block 23800003 for any references to tx79's from address
    print(f"\n=== Checking Block 23800003 for references to {tx79['from']} ===")
    block3 = rpc_call("eth_getBlockByNumber", [hex(23800003), True])
    if not block3:
        print("Failed to get block")
        return
    
    found_refs = []
    for i, tx in enumerate(block3["transactions"]):
        if tx.get("from", "").lower() == tx79["from"].lower():
            found_refs.append((i, "from", tx["hash"]))
        if tx.get("to", "").lower() == tx79["from"].lower():
            found_refs.append((i, "to", tx["hash"]))
    
    if found_refs:
        print(f"Found {len(found_refs)} references:")
        for idx, field, hash in found_refs:
            print(f"  TX {idx}: {field} = {tx79['from']}, hash={hash[:20]}...")
    else:
        print("No direct references found")
    
    # Also check for the delegation target address
    if delegation_addresses:
        print(f"\n=== Checking Block 23800003 for delegation addresses ===")
        for deleg_addr in delegation_addresses:
            found_deleg = []
            for i, tx in enumerate(block3["transactions"]):
                if tx.get("from", "").lower() == deleg_addr.lower():
                    found_deleg.append((i, "from", tx["hash"]))
                if tx.get("to", "").lower() == deleg_addr.lower():
                    found_deleg.append((i, "to", tx["hash"]))
            
            if found_deleg:
                print(f"Found {len(found_deleg)} references to {deleg_addr}:")
                for idx, field, hash in found_deleg:
                    print(f"  TX {idx}: {field}, hash={hash[:20]}...")
            else:
                print(f"No references to {deleg_addr}")

if __name__ == "__main__":
    main()

