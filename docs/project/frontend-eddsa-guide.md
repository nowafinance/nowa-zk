# 🖥️ Frontend Guide: EdDSA (BabyJubJub) Hidden Key Flow

Because Nowa-ZK uses a highly optimized ZK-SNARK Prover, we require all trades to be signed using **EdDSA (BabyJubJub)** on the BN254 curve. 

However, standard Ethereum wallets (like MetaMask) **do not** natively support BabyJubJub. They only support ECDSA. To solve this without ruining the user experience, we use the "Hidden Key" flow popularized by dYdX.

## The "Hidden Key" Flow

When a user connects their wallet to the Nowa-ZK frontend for the first time, you must generate a temporary BabyJubJub keypair for them.

### Step 1: Request a standard MetaMask Signature
Ask the user to sign a standard EIP-712 or personal message. This signature costs no gas.
**Message:** `"Sign this message to generate your Nowa-ZK Trading Key. This key allows you to trade instantly with zero gas fees."`

### Step 2: Generate the BabyJubJub Seed
Take the resulting ECDSA signature string from MetaMask. Pass this string through a deterministic hash function (like `Poseidon` or `SHA-256`) to generate a 32-byte seed.

### Step 3: Derive the BabyJubJub Keypair
Use the 32-byte seed to generate a valid BabyJubJub Private/Public keypair. (You can use libraries like `circomlibjs` or `ffjavascript` for this math in the browser).

### Step 4: Store the Key
Save this generated BabyJubJub Private Key securely in the browser's `LocalStorage` or `SessionStorage`.

### Step 5: Sign Trades Instantly
When the user clicks "Buy" or "Sell", **do not** prompt MetaMask. 
Instead, fetch the hidden BabyJubJub private key from LocalStorage, use it to sign the `Order` payload in the background, and instantly send the JSON payload to the Go Sequencer's REST API.

---

## Security Considerations
*   **Key Expiration:** If the user clears their cache or uses a new device, they simply re-sign the MetaMask message in Step 1. Because the hash is deterministic, it will regenerate the exact same BabyJubJub keypair!
*   **Revocation:** If a user fears their LocalStorage was compromised, they can submit a transaction on Ethereum L1 to increment their global nonce, invalidating the old BabyJubJub key.
