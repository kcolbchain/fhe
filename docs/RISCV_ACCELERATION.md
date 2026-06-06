# RISC-V Hardware Acceleration for FHE

> Roadmap for accelerating Fully Homomorphic Encryption operations on Tenstorrent's RISC-V + Tensix architecture.

## Motivation

FHE's primary computational bottleneck is **bootstrapping**, which reduces to polynomial multiplication, which decomposes into Number Theoretic Transforms (NTTs). These are structured matrix operations — exactly what Tenstorrent's Tensix cores are designed to accelerate.

Current FHE implementations run on CPUs or GPUs. A Tenstorrent backend would provide:
- **664 TFLOPS** matrix compute (Blackhole) for NTT/polynomial operations
- **180 MB SRAM** for keeping ciphertext parameters resident
- **Hardware crypto** (AES, SHA-2/3) via RISC-V Zkn extensions for key management
- **Scale-out** via Galaxy mesh networking for distributed FHE workloads

## Operation Mapping

### Core FHE Operations → Tensix

| FHE Operation | Computational Core | Tensix Mapping | Engine |
|---|---|---|---|
| **Polynomial multiplication** | NTT butterfly + point-wise multiply | Matrix multiplication | Matrix Engine (FPU) |
| **NTT / iNTT** | Butterfly operations across coefficient arrays | Structured matrix ops | Matrix Engine (FPU) |
| **Ciphertext addition** | Element-wise vector add | SIMD vector add | Vector Engine (SFPU) |
| **Key switching** | Large matrix-vector product | Direct matrix multiply | Matrix Engine (FPU) |
| **Bootstrapping** | NTT + modular reduction + key switch | Full pipeline | Both engines |
| **Modular reduction** | Barrett/Montgomery reduction | Custom kernel | Baby RISC-V + SFPU |
| **Random sampling** | Gaussian/uniform noise generation | Hardware entropy (Zkr) + SFPU | Baby RISC-V |

### Crypto Primitives → RISC-V Zkn

| Operation | ISA Extension | Use in FHE |
|---|---|---|
| AES-256 encrypt/decrypt | Zknd, Zkne | Key encapsulation, authenticated encryption of ciphertext |
| SHA-256 | Zknh | Hash-based key derivation, integrity checks |
| Hardware entropy | Zkr | Noise sampling for encryption |
| Bit manipulation | Zbb, Zbs | NTT index computation, modular arithmetic |

## Architecture

```
                    ┌─────────────────────────────────────────┐
                    │            Application Layer             │
                    │    LUX Smart Contract (FHE operations)   │
                    └──────────────────┬──────────────────────┘
                                       │
                    ┌──────────────────▼──────────────────────┐
                    │          Rust FFI Bindings               │
                    │   bindings/rust/ (existing, extend)      │
                    │   Feature flag: --features tenstorrent   │
                    └──────────────────┬──────────────────────┘
                                       │
              ┌────────────────────────┼────────────────────────┐
              │                        │                        │
     ┌────────▼────────┐    ┌─────────▼─────────┐    ┌────────▼────────┐
     │  CPU Backend    │    │  GPU Backend       │    │  TT Backend     │
     │  (existing)     │    │  (existing/CUDA)   │    │  (NEW)          │
     │  Go / C         │    │  CUDA kernels      │    │  TT-Metalium    │
     └─────────────────┘    └───────────────────┘    │  kernels        │
                                                      │  (C++ on Tensix)│
                                                      └────────┬───────┘
                                                               │
                                                      ┌────────▼───────┐
                                                      │  Tenstorrent   │
                                                      │  Hardware      │
                                                      │  Blackhole     │
                                                      │  120 Tensix    │
                                                      │  16 Big RISC-V │
                                                      └────────────────┘
```

## Implementation Phases

### Phase 1: Baseline Benchmarks (no hardware needed)
- Benchmark existing FHE operations (NTT, polynomial multiply, bootstrapping) on RISC-V via Whisper ISS
- Compare: rv64gc baseline vs rv64gc+Zkn vs x86 vs ARM
- Identify which operations dominate wall-clock time
- Tools: [tenstorrent/whisper](https://github.com/tenstorrent/whisper)

### Phase 2: TT-Metalium Kernel Prototypes (needs Wormhole card ~$999)
- Implement NTT butterfly as a Tensix kernel
- Implement polynomial point-wise multiply
- Implement modular reduction (Barrett) on SFPU
- Validate correctness against CPU reference
- Tools: [tenstorrent/tt-metal](https://github.com/tenstorrent/tt-metal)

### Phase 3: Rust Integration
- Extend `bindings/rust/build.rs` to detect TT-Metalium
- Add `tenstorrent` feature flag to Cargo.toml
- Route FHE operations to TT backend when available
- Fallback to CPU when hardware not present

### Phase 4: Benchmarks & Paper
- Full benchmark suite: CPU vs GPU vs Tenstorrent
- Publish results in kcolbchain/research
- Target: demonstrate >10x speedup on bootstrapping vs CPU

## Parameters

Target FHE schemes and parameters:

| Scheme | Ring Dimension (N) | Modulus (q) | Key Size | Notes |
|---|---|---|---|---|
| TFHE | 1024 | 32-bit | ~2 KB | Gate bootstrapping |
| BGV | 4096-32768 | 60-180 bit | ~1-100 MB | Batched arithmetic |
| BFV | 4096-32768 | 60-180 bit | ~1-100 MB | Integer arithmetic |
| CKKS | 4096-65536 | 60-360 bit | ~1-500 MB | Approximate (ML inference) |

Blackhole's 180 MB SRAM can hold keys for TFHE and moderate BGV/BFV parameters without external memory access — a significant advantage over GPU approaches that are memory-bandwidth limited.

## Related Issues

- [kcolbchain/fhe#4](https://github.com/kcolbchain/fhe/issues/4) — RISC-V hardware acceleration tracking issue
- [kcolbchain/proofs#1](https://github.com/kcolbchain/proofs/issues/1) — RISC-V crypto extension verification
- [kcolbchain/papers#1](https://github.com/kcolbchain/papers/issues/1) — Research brief

## References

- [Tenstorrent tt-metal](https://github.com/tenstorrent/tt-metal) — TT-Metalium kernel programming
- [Tenstorrent riscv-ocelot](https://github.com/kcolbchain/riscv-ocelot) — kcolbchain fork with crypto benchmarks
- [RISC-V Scalar Crypto Spec](https://github.com/riscv/riscv-crypto)
- Chillotti et al., "TFHE: Fast Fully Homomorphic Encryption over the Torus" (2020)
- Jung et al., "Over 100x Faster Bootstrapping in FHE through GPU Parallelization" (2021)
