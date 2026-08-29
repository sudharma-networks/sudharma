#pragma once

#include <array>
#include <cstddef>
#include <cstdint>

#include "gpupow_v1_program_loop.h"

#ifdef __CUDACC__
#define SUDHARMA_FINAL_HD __host__ __device__
#else
#define SUDHARMA_FINAL_HD
#endif

namespace sudharma::gpupowv1 {

constexpr std::uint32_t kFinalFNVOffsetBasis = 0x811c9dc5u;
constexpr std::uint32_t kFinalFNVPrime = 0x01000193u;
constexpr std::uint32_t kFinalNumLanes = 16u;
constexpr std::uint32_t kFinalMixWords = 8u;

SUDHARMA_FINAL_HD inline std::uint32_t final_fnv1a(std::uint32_t a, std::uint32_t b) {
    return (a ^ b) * kFinalFNVPrime;
}

SUDHARMA_FINAL_HD inline std::uint32_t reduce_lane(
    const std::array<std::uint32_t, kProgramNumRegs>& lane) {
    std::uint32_t digest = kFinalFNVOffsetBasis;
    for (std::size_t i = 0; i < lane.size(); ++i) {
        digest = final_fnv1a(digest, lane[i]);
    }
    return digest;
}

template <std::size_t CacheNodes>
SUDHARMA_FINAL_HD inline std::array<std::uint32_t, kFinalMixWords> group_digest(
    std::uint64_t work_seed,
    const std::array<std::uint8_t, 32>& program_seed,
    const std::array<std::array<std::uint8_t, 64>, CacheNodes>& cache) {
    static_assert(CacheNodes > 0u, "final digest cache must not be empty");

    std::array<std::uint32_t, kFinalMixWords> digest{};
    for (std::size_t i = 0; i < digest.size(); ++i) {
        digest[i] = kFinalFNVOffsetBasis;
    }

    for (std::uint32_t lane = 0; lane < kFinalNumLanes; ++lane) {
        const auto lane_mix = programmatic_lane_mix(work_seed, lane, program_seed, cache);
        const std::uint32_t word = lane % kFinalMixWords;
        digest[word] = final_fnv1a(digest[word], reduce_lane(lane_mix));
    }
    return digest;
}

SUDHARMA_FINAL_HD inline std::uint32_t final_rotr32(std::uint32_t value, std::uint32_t shift) {
    return (value >> shift) | (value << (32u - shift));
}

SUDHARMA_FINAL_HD inline void final_sha256_compress(std::uint32_t state[8],
                                                     const std::uint8_t block[64]) {
    const std::uint32_t k[64] = {
        0x428a2f98u, 0x71374491u, 0xb5c0fbcfu, 0xe9b5dba5u,
        0x3956c25bu, 0x59f111f1u, 0x923f82a4u, 0xab1c5ed5u,
        0xd807aa98u, 0x12835b01u, 0x243185beu, 0x550c7dc3u,
        0x72be5d74u, 0x80deb1feu, 0x9bdc06a7u, 0xc19bf174u,
        0xe49b69c1u, 0xefbe4786u, 0x0fc19dc6u, 0x240ca1ccu,
        0x2de92c6fu, 0x4a7484aau, 0x5cb0a9dcu, 0x76f988dau,
        0x983e5152u, 0xa831c66du, 0xb00327c8u, 0xbf597fc7u,
        0xc6e00bf3u, 0xd5a79147u, 0x06ca6351u, 0x14292967u,
        0x27b70a85u, 0x2e1b2138u, 0x4d2c6dfcu, 0x53380d13u,
        0x650a7354u, 0x766a0abbu, 0x81c2c92eu, 0x92722c85u,
        0xa2bfe8a1u, 0xa81a664bu, 0xc24b8b70u, 0xc76c51a3u,
        0xd192e819u, 0xd6990624u, 0xf40e3585u, 0x106aa070u,
        0x19a4c116u, 0x1e376c08u, 0x2748774cu, 0x34b0bcb5u,
        0x391c0cb3u, 0x4ed8aa4au, 0x5b9cca4fu, 0x682e6ff3u,
        0x748f82eeu, 0x78a5636fu, 0x84c87814u, 0x8cc70208u,
        0x90befffau, 0xa4506cebu, 0xbef9a3f7u, 0xc67178f2u,
    };

    std::uint32_t w[64] = {};
    for (std::size_t i = 0; i < 16u; ++i) {
        const std::size_t j = i * 4u;
        w[i] = (static_cast<std::uint32_t>(block[j]) << 24u) |
               (static_cast<std::uint32_t>(block[j + 1u]) << 16u) |
               (static_cast<std::uint32_t>(block[j + 2u]) << 8u) |
               static_cast<std::uint32_t>(block[j + 3u]);
    }
    for (std::size_t i = 16u; i < 64u; ++i) {
        const std::uint32_t s0 = final_rotr32(w[i - 15u], 7u) ^
                                 final_rotr32(w[i - 15u], 18u) ^
                                 (w[i - 15u] >> 3u);
        const std::uint32_t s1 = final_rotr32(w[i - 2u], 17u) ^
                                 final_rotr32(w[i - 2u], 19u) ^
                                 (w[i - 2u] >> 10u);
        w[i] = w[i - 16u] + s0 + w[i - 7u] + s1;
    }

    std::uint32_t a = state[0];
    std::uint32_t b = state[1];
    std::uint32_t c = state[2];
    std::uint32_t d = state[3];
    std::uint32_t e = state[4];
    std::uint32_t f = state[5];
    std::uint32_t g = state[6];
    std::uint32_t h = state[7];

    for (std::size_t i = 0; i < 64u; ++i) {
        const std::uint32_t s1 = final_rotr32(e, 6u) ^ final_rotr32(e, 11u) ^ final_rotr32(e, 25u);
        const std::uint32_t ch = (e & f) ^ ((~e) & g);
        const std::uint32_t temp1 = h + s1 + ch + k[i] + w[i];
        const std::uint32_t s0 = final_rotr32(a, 2u) ^ final_rotr32(a, 13u) ^ final_rotr32(a, 22u);
        const std::uint32_t maj = (a & b) ^ (a & c) ^ (b & c);
        const std::uint32_t temp2 = s0 + maj;

        h = g;
        g = f;
        f = e;
        e = d + temp1;
        d = c;
        c = b;
        b = a;
        a = temp1 + temp2;
    }

    state[0] += a;
    state[1] += b;
    state[2] += c;
    state[3] += d;
    state[4] += e;
    state[5] += f;
    state[6] += g;
    state[7] += h;
}

SUDHARMA_FINAL_HD inline std::array<std::uint8_t, 32> final_sha256_90(
    const std::array<std::uint8_t, 90>& input) {
    std::uint32_t state[8] = {
        0x6a09e667u, 0xbb67ae85u, 0x3c6ef372u, 0xa54ff53au,
        0x510e527fu, 0x9b05688cu, 0x1f83d9abu, 0x5be0cd19u,
    };

    std::uint8_t block0[64] = {};
    for (std::size_t i = 0; i < 64u; ++i) block0[i] = input[i];
    final_sha256_compress(state, block0);

    std::uint8_t block1[64] = {};
    for (std::size_t i = 64u; i < input.size(); ++i) block1[i - 64u] = input[i];
    block1[input.size() - 64u] = 0x80u;
    const std::uint64_t bit_length = static_cast<std::uint64_t>(input.size()) * 8u;
    for (std::size_t i = 0; i < 8u; ++i) {
        block1[63u - i] = static_cast<std::uint8_t>(bit_length >> (i * 8u));
    }
    final_sha256_compress(state, block1);

    std::array<std::uint8_t, 32> digest{};
    for (std::size_t i = 0; i < 8u; ++i) {
        digest[i * 4u] = static_cast<std::uint8_t>(state[i] >> 24u);
        digest[i * 4u + 1u] = static_cast<std::uint8_t>(state[i] >> 16u);
        digest[i * 4u + 2u] = static_cast<std::uint8_t>(state[i] >> 8u);
        digest[i * 4u + 3u] = static_cast<std::uint8_t>(state[i]);
    }
    return digest;
}

SUDHARMA_FINAL_HD inline std::uint64_t header_work_seed(
    const std::array<std::uint8_t, 32>& header_digest) {
    std::uint64_t seed = 0u;
    for (std::size_t i = 0; i < 8u; ++i) {
        seed |= static_cast<std::uint64_t>(header_digest[i]) << (i * 8u);
    }
    return seed;
}

template <std::size_t CacheNodes>
SUDHARMA_FINAL_HD inline std::array<std::uint8_t, 32> final_digest_from_header(
    const std::array<std::uint8_t, 32>& header_digest,
    const std::array<std::uint8_t, 32>& program_seed,
    const std::array<std::array<std::uint8_t, 64>, CacheNodes>& cache) {
    const auto mix = group_digest(header_work_seed(header_digest), program_seed, cache);

    std::array<std::uint8_t, 90> input{};
    const std::uint8_t domain[26] = {
        'S', 'U', 'D', 'H', 'A', 'R', 'M', 'A', '-', 'G', 'P', 'U', '-',
        'P', 'O', 'W', '-', 'V', '1', '-', 'F', 'I', 'N', 'A', 'L', 0u,
    };
    for (std::size_t i = 0; i < 26u; ++i) input[i] = domain[i];
    for (std::size_t i = 0; i < header_digest.size(); ++i) input[26u + i] = header_digest[i];
    for (std::size_t word = 0; word < mix.size(); ++word) {
        const std::uint32_t value = mix[word];
        const std::size_t offset = 58u + word * 4u;
        input[offset] = static_cast<std::uint8_t>(value);
        input[offset + 1u] = static_cast<std::uint8_t>(value >> 8u);
        input[offset + 2u] = static_cast<std::uint8_t>(value >> 16u);
        input[offset + 3u] = static_cast<std::uint8_t>(value >> 24u);
    }
    return final_sha256_90(input);
}

}  // namespace sudharma::gpupowv1

#undef SUDHARMA_FINAL_HD
