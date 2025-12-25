#!/usr/bin/env python3
"""
Compare Erigon gas with RPC gas.
Usage: 
1. Run Erigon with debug logging
2. Copy the TX receipt lines from Erigon log to a file (erigon_gas.txt)
3. Run: python3 compare_erigon_log.py erigon_gas.txt
"""

import json
import re
import sys
import os

def parse_erigon_log(filename):
    """Parse Erigon debug log to extract gasUsed per transaction"""
    erigon_gas = {}
    pattern = r'txIndex=(\d+).*?gasUsed=(\d+)'
    
    with open(filename, 'r') as f:
        for line in f:
            match = re.search(pattern, line)
            if match:
                idx = int(match.group(1))
                gas = int(match.group(2))
                erigon_gas[idx] = gas
    return erigon_gas

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 compare_erigon_log.py <erigon_log_file>")
        print("\nThe log file should contain lines like:")
        print("[DEBUG] TX receipt txIndex=0 ... gasUsed=117778 ...")
        sys.exit(1)
    
    erigon_file = sys.argv[1]
    rpc_file = "rpc_gas_23800003.json"
    
    # Parse Erigon log
    erigon_gas = parse_erigon_log(erigon_file)
    print(f"Parsed {len(erigon_gas)} transactions from Erigon log")
    
    # Load RPC data if available
    if os.path.exists(rpc_file):
        with open(rpc_file, 'r') as f:
            rpc_gas = {int(k): v for k, v in json.load(f).items()}
        print(f"Loaded {len(rpc_gas)} transactions from RPC cache")
    else:
        print(f"RPC cache file not found: {rpc_file}")
        print("Run scripts/find_diff_tx.py first to fetch RPC data")
        rpc_gas = {}
    
    if not erigon_gas:
        print("No Erigon gas data found!")
        return
    
    # Calculate totals
    erigon_total = sum(erigon_gas.values())
    print(f"\nErigon total gas: {erigon_total}")
    
    # Compare
    if rpc_gas:
        print("\n" + "=" * 70)
        print("Comparing Erigon vs RPC gas...")
        print("-" * 70)
        
        differences = []
        cumulative_diff = 0
        
        for i in sorted(erigon_gas.keys()):
            e_gas = erigon_gas.get(i)
            r_gas = rpc_gas.get(i)
            
            if e_gas is not None and r_gas is not None:
                diff = e_gas - r_gas
                if diff != 0:
                    cumulative_diff += diff
                    differences.append((i, e_gas, r_gas, diff, cumulative_diff))
                    print(f"*** TX {i:3d}: Erigon={e_gas:8d}, RPC={r_gas:8d}, Diff={diff:+6d}, CumulDiff={cumulative_diff:+6d}")
        
        print("-" * 70)
        print(f"Transactions with gas differences: {len(differences)}")
        print(f"Total gas difference: {cumulative_diff}")
        
        # Expected vs Actual
        print("\n" + "=" * 70)
        print("Summary:")
        print(f"  Erigon total gas:   {erigon_total}")
        print(f"  Expected (header):  26758527")
        print(f"  Difference:         {erigon_total - 26758527}")
        
        if differences:
            print("\nFirst transaction with difference:")
            idx, e, r, d, cd = differences[0]
            print(f"  TX {idx}: Erigon={e}, RPC={r}, Diff={d}")
    else:
        print("\nNo RPC data to compare. Showing Erigon gas only:")
        for i in sorted(erigon_gas.keys())[:20]:
            print(f"  TX {i}: {erigon_gas[i]}")

if __name__ == "__main__":
    main()

