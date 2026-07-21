import os
import re

def replace_in_file(file_path):
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
    except Exception as e:
        return False
        
    new_content = content
    # Case sensitive replacements
    new_content = re.sub(r'\bRPC_INDEXER\b', 'L2_RPC_URL', new_content)
    new_content = re.sub(r'\bRPC_PROVER\b', 'L1_RPC_URL', new_content)
    
    if new_content != content:
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(new_content)
        return True
    return False

exclude_dirs = {'.git', 'build', 'node_modules', 'out', 'cache', '.nowa-zk', 'dir_data'}

modified_files = []
for root, dirs, files in os.walk('.'):
    dirs[:] = [d for d in dirs if d not in exclude_dirs]
    for file in files:
        if file.endswith(('.png', '.jpg', '.jpeg', '.gif', '.ico', '.pdf', '.zip', '.tar', '.gz')):
            continue
        file_path = os.path.join(root, file)
        if file_path.endswith('rename_rpc.py'):
            continue
        if replace_in_file(file_path):
            modified_files.append(file_path)

for m in modified_files:
    print(f"Modified: {m}")
