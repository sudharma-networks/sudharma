#pragma once

#include <array>
#include <cstdint>

#ifdef __CUDACC__
#define SUDHARMA_GPUPOW_HD __host__ __device__
#else
#define SUDHARMA_GPUPOW_HD
#endif

namespace sudharma::gpupowv1 {

constexpr std::uint32_t kPermutationFNVOffsetBasis = 0x811c9dc5u;
constexpr std::uint32_t kPermutationFNVPrime = 0x01000193u;
constexpr std::size_t kPermutationNumRegs = 32u;

SUDHARMA_GPUPOW_HD inline std::uint32_t permutation_fnv1a(std::uint32_t a, std::uint32_t b) {
    return (a ^ b) * kPermutationFNVPrime;
}

struct PermutationKISS99 {
    std::uint32_t z;
    std::uint32_t w;
    std::uint32_t jsr;
    std::uint32_t jcong;

    SUDHARMA_GPUPOW_HD std::uint32_t next() {
        z = 36969u * (z & 0xffffu) + (z >> 16u);
        w = 18000u * (w & 0xffffu) + (w >> 16u);
        jcong = 69069u * jcong + 1234567u;
        jsr ^= jsr << 17u;
        jsr ^= jsr >> 13u;
        jsr ^= jsr << 5u;
        return (((z << 16u) + w) ^ jcong) + jsr;
    }
};

SUDHARMA_GPUPOW_HD inline PermutationKISS99 new_permutation_kiss99(
    std::uint32_t seed_lo, std::uint32_t seed_hi) {
    const std::uint32_t z = permutation_fnv1a(kPermutationFNVOffsetBasis, seed_lo);
    const std::uint32_t w = permutation_fnv1a(z, seed_hi);
    const std::uint32_t jsr = permutation_fnv1a(w, seed_lo);
    const std::uint32_t jcong = permutation_fnv1a(jsr, seed_hi);
    return PermutationKISS99{z, w, jsr, jcong};
}

struct RegisterSchedules {
    std::array<std::uint32_t, kPermutationNumRegs> first;
    std::array<std::uint32_t, kPermutationNumRegs> second;
};

SUDHARMA_GPUPOW_HD inline RegisterSchedules register_permutations(
    std::uint32_t seed_lo, std::uint32_t seed_hi) {
    auto rng = new_permutation_kiss99(seed_lo, seed_hi);
    RegisterSchedules schedules{};

    for (std::uint32_t i = 0; i < kPermutationNumRegs; ++i) {
        schedules.first[i] = i;
        schedules.second[i] = i;
    }

    for (std::uint32_t i = kPermutationNumRegs - 1u; i > 0u; --i) {
        std::uint32_t j = rng.next() % (i + 1u);
        const std::uint32_t dst = schedules.first[i];
        schedules.first[i] = schedules.first[j];
        schedules.first[j] = dst;

        j = rng.next() % (i + 1u);
        const std::uint32_t src = schedules.second[i];
        schedules.second[i] = schedules.second[j];
        schedules.second[j] = src;
    }

    return schedules;
}

}  // namespace sudharma::gpupowv1

#undef SUDHARMA_GPUPOW_HD
