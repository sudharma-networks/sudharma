#pragma once

#include <array>
#include <cstddef>
#include <cstdint>

#include "gpupow_v1_dataset.h"
#include "gpupow_v1_permutation.h"

#ifdef __CUDACC__
#define SUDHARMA_PROGRAM_HD __host__ __device__
#else
#define SUDHARMA_PROGRAM_HD
#endif

namespace sudharma::gpupowv1 {

constexpr std::uint32_t kProgramNumRegs = 32u;
constexpr std::uint32_t kProgramDAGRounds = 64u;
constexpr std::uint32_t kProgramDAGLoads = 4u;
constexpr std::uint32_t kProgramCacheAccesses = 11u;
constexpr std::uint32_t kProgramMathOperations = 18u;
constexpr std::uint32_t kProgramFNVOffsetBasis = 0x811c9dc5u;
constexpr std::uint32_t kProgramFNVPrime = 0x01000193u;

SUDHARMA_PROGRAM_HD inline std::uint32_t program_fnv1a(std::uint32_t a, std::uint32_t b) {
    return (a ^ b) * kProgramFNVPrime;
}

SUDHARMA_PROGRAM_HD inline std::uint32_t program_fnv1(std::uint32_t a, std::uint32_t b) {
    return (a * kProgramFNVPrime) ^ b;
}

struct ProgramKISS99 {
    std::uint32_t z;
    std::uint32_t w;
    std::uint32_t jsr;
    std::uint32_t jcong;

    SUDHARMA_PROGRAM_HD std::uint32_t next() {
        z = 36969u * (z & 0xffffu) + (z >> 16u);
        w = 18000u * (w & 0xffffu) + (w >> 16u);
        jcong = 69069u * jcong + 1234567u;
        jsr ^= jsr << 17u;
        jsr ^= jsr >> 13u;
        jsr ^= jsr << 5u;
        return (((z << 16u) + w) ^ jcong) + jsr;
    }
};

SUDHARMA_PROGRAM_HD inline ProgramKISS99 program_kiss99(std::uint32_t seed_lo,
                                                         std::uint32_t seed_hi) {
    const std::uint32_t z = program_fnv1a(kProgramFNVOffsetBasis, seed_lo);
    const std::uint32_t w = program_fnv1a(z, seed_hi);
    const std::uint32_t jsr = program_fnv1a(w, seed_lo);
    const std::uint32_t jcong = program_fnv1a(jsr, seed_hi);
    return ProgramKISS99{z, w, jsr, jcong};
}

SUDHARMA_PROGRAM_HD inline std::array<std::uint32_t, kProgramNumRegs> program_init_lane(
    std::uint64_t seed, std::uint32_t lane) {
    const std::uint32_t seed_lo = static_cast<std::uint32_t>(seed);
    const std::uint32_t seed_hi = static_cast<std::uint32_t>(seed >> 32u);
    ProgramKISS99 rng{
        program_fnv1a(kProgramFNVOffsetBasis, seed_lo),
        program_fnv1a(kProgramFNVOffsetBasis, seed_hi),
        program_fnv1a(kProgramFNVOffsetBasis, lane),
        program_fnv1a(kProgramFNVOffsetBasis, lane),
    };

    std::array<std::uint32_t, kProgramNumRegs> mix{};
    for (std::size_t i = 0; i < mix.size(); ++i) mix[i] = rng.next();
    return mix;
}

SUDHARMA_PROGRAM_HD inline std::uint32_t program_rotl32(std::uint32_t value,
                                                        std::uint32_t amount) {
    const std::uint32_t shift = amount & 31u;
    return shift == 0u ? value : ((value << shift) | (value >> (32u - shift)));
}

SUDHARMA_PROGRAM_HD inline std::uint32_t program_rotr32(std::uint32_t value,
                                                        std::uint32_t amount) {
    const std::uint32_t shift = amount & 31u;
    return shift == 0u ? value : ((value >> shift) | (value << (32u - shift)));
}

SUDHARMA_PROGRAM_HD inline std::uint32_t program_mul_hi32(std::uint32_t a, std::uint32_t b) {
    return static_cast<std::uint32_t>((static_cast<std::uint64_t>(a) * b) >> 32u);
}

SUDHARMA_PROGRAM_HD inline std::uint32_t program_clz32(std::uint32_t value) {
    if (value == 0u) return 32u;
    std::uint32_t count = 0u;
    for (std::uint32_t bit = 0x80000000u; (value & bit) == 0u; bit >>= 1u) ++count;
    return count;
}

SUDHARMA_PROGRAM_HD inline std::uint32_t program_popcount32(std::uint32_t value) {
    std::uint32_t count = 0u;
    while (value != 0u) {
        value &= value - 1u;
        ++count;
    }
    return count;
}

SUDHARMA_PROGRAM_HD inline std::uint32_t program_random_math(std::uint32_t a,
                                                             std::uint32_t b,
                                                             std::uint32_t selector) {
    switch (selector % 11u) {
        case 0u: return a + b;
        case 1u: return a * b;
        case 2u: return program_mul_hi32(a, b);
        case 3u: return a < b ? a : b;
        case 4u: return program_rotl32(a, b);
        case 5u: return program_rotr32(a, b);
        case 6u: return a & b;
        case 7u: return a | b;
        case 8u: return a ^ b;
        case 9u: return program_clz32(a) + program_clz32(b);
        default: return program_popcount32(a) + program_popcount32(b);
    }
}

SUDHARMA_PROGRAM_HD inline std::uint32_t program_random_merge(std::uint32_t a,
                                                              std::uint32_t b,
                                                              std::uint32_t selector) {
    const std::uint32_t x = ((selector >> 16u) % 31u) + 1u;
    switch (selector % 4u) {
        case 0u: return (a * 33u) + b;
        case 1u: return (a ^ b) * 33u;
        case 2u: return program_rotl32(a, x) ^ b;
        default: return program_rotr32(a, x) ^ b;
    }
}

SUDHARMA_PROGRAM_HD inline std::uint32_t program_seed_word(
    const std::array<std::uint8_t, 32>& seed, std::size_t offset) {
    return static_cast<std::uint32_t>(seed[offset]) |
           (static_cast<std::uint32_t>(seed[offset + 1u]) << 8u) |
           (static_cast<std::uint32_t>(seed[offset + 2u]) << 16u) |
           (static_cast<std::uint32_t>(seed[offset + 3u]) << 24u);
}

template <std::size_t CacheNodes>
SUDHARMA_PROGRAM_HD inline std::array<std::uint32_t, kProgramNumRegs> programmatic_lane_mix(
    std::uint64_t work_seed,
    std::uint32_t lane,
    const std::array<std::uint8_t, 32>& program_seed,
    const std::array<std::array<std::uint8_t, 64>, CacheNodes>& cache) {
    static_assert(CacheNodes > 0u, "programmatic lane cache must not be empty");

    auto mix = program_init_lane(work_seed, lane);
    const std::uint32_t seed_lo = program_seed_word(program_seed, 0u);
    const std::uint32_t seed_hi = program_seed_word(program_seed, 4u);
    const auto schedules = register_permutations(seed_lo, seed_hi);
    auto rng = program_kiss99(seed_lo, seed_hi);

    for (std::uint32_t round = 0; round < kProgramDAGRounds; ++round) {
        const std::uint32_t dag_index = program_fnv1(mix[0] ^ round, lane ^ rng.next());
        const auto dag_item = dataset_item(cache, dag_index);

        for (std::uint32_t load = 0; load < kProgramDAGLoads; ++load) {
            const std::uint32_t dst = schedules.first[(round * kProgramDAGLoads + load) % kProgramNumRegs];
            mix[dst] = program_random_merge(mix[dst], word64(dag_item, load), rng.next());
        }

        for (std::uint32_t access = 0; access < kProgramCacheAccesses; ++access) {
            const std::uint32_t src = schedules.second[(round + access) % kProgramNumRegs];
            const std::uint32_t selector = mix[src] ^ rng.next();
            const auto& cache_node = cache[selector % static_cast<std::uint32_t>(CacheNodes)];
            const std::uint32_t cache_word = word64(cache_node, (selector >> 16u) % 16u);
            const std::uint32_t dst = schedules.first[
                (round + kProgramDAGLoads + access) % kProgramNumRegs];
            mix[dst] = program_random_merge(mix[dst], cache_word, rng.next());
        }

        for (std::uint32_t op = 0; op < kProgramMathOperations; ++op) {
            const std::uint32_t dst = schedules.first[
                (round + kProgramDAGLoads + kProgramCacheAccesses + op) % kProgramNumRegs];
            const std::uint32_t src_a = schedules.second[(round + op) % kProgramNumRegs];
            const std::uint32_t src_b = schedules.second[(round + op + 1u) % kProgramNumRegs];
            const std::uint32_t value = program_random_math(mix[src_a], mix[src_b], rng.next());
            mix[dst] = program_random_merge(mix[dst], value, rng.next());
        }
    }

    return mix;
}

}  // namespace sudharma::gpupowv1

#undef SUDHARMA_PROGRAM_HD
