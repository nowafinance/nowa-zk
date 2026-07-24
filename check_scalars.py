import sys

with open('/home/aakash/.nowa-zk/prover/failures/batch_1_witness.bin', 'rb') as f:
    data = f.read()

R = 21888242871839275222246405745257275088548364400416034343698204186575808495617
offset = 12
for i in range(120):
    chunk = data[offset : offset+32]
    val = int.from_bytes(chunk, byteorder='big')
    if val >= R:
        print(f"Index {i} is >= R! val: {val}")
    offset += 32
print("Done checking.")
