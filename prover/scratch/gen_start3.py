import os

with open('cmd/prover/start.go', 'r') as f:
    text = f.read()

# Fix the broken string literals caused by python multiline string interpolation
text = text.replace('\n", ', '\\n", ')
text = text.replace('\n")', '\\n")')
text = text.replace('\n",', '\\n",')

with open('cmd/prover/start.go', 'w') as f:
    f.write(text)
