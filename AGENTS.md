<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

Always open `@/openspec/AGENTS.md` when the request:
- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:
- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

<!-- OPENSPEC:END -->

## Multi-Chain Support

This library supports both EVM (Ethereum Virtual Machine) and SVM (Solana Virtual Machine) payment schemes:

- **EVM Support**: Ethereum-based chains using EIP-712 signatures and EIP-3009 transfer with authorization
- **SVM Support**: Solana-based chains using partially-signed SPL token transfer transactions

When implementing new features or making changes:
- Ensure compatibility with both EVM and SVM payment flows
- The `PaymentPayload.Payload` field uses `interface{}` to support both EVM (struct) and SVM (map) payload formats
- Client signers implement the `PaymentSigner` interface for chain-agnostic payment handling
- Server handlers should gracefully handle both payload types during logging and validation