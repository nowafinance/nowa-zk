# Tan-ZK Documentation

Complete documentation for setting up, deploying, and operating the Tan-ZK rollup system.

---

## 📖 Quick Navigation

### 🚀 Getting Started

Start here if you're new to Tan-ZK:

- **[CODEME.md](../CODEME.md)** - Quick start guide with essential commands

### 🌐 Deployment

Choose your deployment method:

- **[Cloud Server Setup](deployment/cloud.md)** - Deploy on a Linux cloud server (VPS, AWS, GCP, etc.)
- **[Docker Setup](deployment/docker.md)** - Deploy using Docker containers

### ⚙️ Operations

Daily operations and maintenance:

- **[Upgrade Guide](operations/upgrade.md)** - Regular code updates (without circuit changes)
- **[Project Update & Reset](operations/project-update.md)** - Key regeneration and contract redeployment when circuit changes
- **[Troubleshooting](operations/troubleshooting.md)** - Common issues and solutions

### 📐 Architecture

System design and technical details:

- **[Data Flow](architecture/data-flow.md)** - How data flows through the system (User → Sequencer → Prover → L1)

### 📊 Project

Development tracking:

- **[Milestones](project/milestones.md)** - Project roadmap and progress tracking

---

## 🎯 Common Tasks

### First Time Setup

1. Read [CODEME.md](../CODEME.md) for overview
2. Follow [Cloud Setup](deployment/cloud.md) or [Docker Setup](deployment/docker.md)
3. Bookmark [Troubleshooting Guide](operations/troubleshooting.md)

### Regular Maintenance

- **Code update only**: Use [Upgrade Guide](operations/upgrade.md)
- **Circuit changed**: Use [Project Update Guide](operations/project-update.md)

### When Things Go Wrong

1. Check [Troubleshooting Guide](operations/troubleshooting.md) first
2. Review service logs: `sudo journalctl -u tan-sequencer -f`
3. Verify environment variables are loaded correctly

---

## 📁 Documentation Structure

```
docs/
├── README.md                        # This file - navigation guide
│
├── deployment/                      # Initial setup guides
│   ├── cloud.md                     # Cloud server deployment
│   └── docker.md                    # Docker deployment
│
├── operations/                      # Operational guides
│   ├── upgrade.md                   # Regular upgrades
│   ├── project-update.md            # Circuit changes & resets
│   └── troubleshooting.md           # Problem solving
│
├── architecture/                    # Technical documentation
│   └── data-flow.md                 # System architecture
│
└── project/                         # Project management
    └── milestones.md                # Development roadmap
```

---

## ⚡ Critical Notes

### Circuit Changes Require Contract Redeployment

> [!CAUTION]
> When the ZK circuit changes:
> - You MUST regenerate prover keys
> - You MUST redeploy all contracts
> - The old contract cannot verify proofs from new keys
> 
> See: [Circuit Update Guide](operations/circuit-update.md)

### Environment Variables

Always load with auto-export:

```bash
set -a
source /etc/tan/.env
set +a
```

### Deployment Commands

Use the correct contract name:

```bash
# ✅ Correct
forge script script/Deploy.s.sol:Deploy --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast

# ❌ Wrong (missing :Deploy)
forge script script/Deploy.s.sol --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast
```

---

## 🆘 Need Help?

1. **Deployment errors**: See [Troubleshooting → Deployment Issues](operations/troubleshooting.md#deployment-issues)
2. **Prover errors**: See [Troubleshooting → ZK Proof Generation](operations/troubleshooting.md#zk-proof-generation-issues)
3. **Service issues**: See [Troubleshooting → Service Not Starting](operations/troubleshooting.md#service-not-starting)

---

## 📝 Contributing to Docs

When updating documentation:
- Keep commands up-to-date with actual code
- Include error messages in troubleshooting
- Test all commands before documenting
- Use alerts ([!CAUTION], [!WARNING], [!NOTE]) for critical information
