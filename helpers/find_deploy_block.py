import urllib.request
import json
import sys

RPC_URL = "https://node1.nowa.finance"
CONTRACT_ADDRESS = "0xD28C36Adcd614d4EC42CA800114eDEd644b28189"

def check_code_at_block(block_number):
    hex_block = hex(block_number)
    payload = {
        "jsonrpc": "2.0",
        "method": "eth_getCode",
        "params": [CONTRACT_ADDRESS, hex_block],
        "id": 1
    }
    req = urllib.request.Request(RPC_URL, data=json.dumps(payload).encode('utf-8'),
                                 headers={'Content-Type': 'application/json'})
    with urllib.request.urlopen(req) as response:
        res = json.loads(response.read().decode())
        code = res.get("result", "0x")
        return len(code) > 2  # Returns True if bytecode exists (more than "0x")

# We know the latest block is around 4228435
latest_block = 4228435

# Binary search
low = 0
high = latest_block
ans = -1

print("Starting binary search...")
while low <= high:
    mid = (low + high) // 2
    try:
        exists = check_code_at_block(mid)
        if exists:
            ans = mid
            high = mid - 1
            print(f"Contract exists at {mid}, searching lower...")
        else:
            low = mid + 1
            print(f"Contract does NOT exist at {mid}, searching higher...")
    except Exception as e:
        print(f"Error at block {mid}: {e}")
        break

if ans != -1:
    print(f"\nSUCCESS! Contract was deployed at block: {ans}")
else:
    print("\nFAILED to find deployment block. Are you sure the contract is deployed on this network?")
