#pragma once

#include <array>
#include <cstddef>
#include <cstdint>

#include "gpupow_v1_memory.h"

#ifdef __CUDACC__
#define SUDHARMA_GPUPOW_HD __host__ __device__
#else
#define SUDHARMA_GPUPOW_HD
#endif

namespace sudharma::gpupowv1 {

constexpr std::uint32_t kDatasetParents = 512u;
constexpr std::uint32_t kDatasetFNVPrime = 0x01000193u;

SUDHARMA_GPUPOW_HD inline std::uint32_t dataset_fnv1(std::uint32_t a, std::uint32_t b) {
    return (a * kDatasetFNVPrime) ^ b;
}

SUDHARMA_GPUPOW_HD inline std::uint64_t rotl64(std::uint64_t value, unsigned shift) {
    shift &= 63u;
    return shift == 0u ? value : ((value << shift) | (value >> (64u - shift)));
}

SUDHARMA_GPUPOW_HD inline void keccak_f1600(std::uint64_t state[25]) {
    constexpr std::uint64_t round_constants[24] = {
        0x0000000000000001ull, 0x0000000000008082ull,
        0x800000000000808aull, 0x8000000080008000ull,
        0x000000000000808bull, 0x0000000080000001ull,
        0x8000000080008081ull, 0x8000000000008009ull,
        0x000000000000008aull, 0x0000000000000088ull,
        0x0000000080008009ull, 0x000000008000000aull,
        0x000000008000808bull, 0x800000000000008bull,
        0x8000000000008089ull, 0x8000000000008003ull,
        0x8000000000008002ull, 0x8000000000000080ull,
        0x000000000000800aull, 0x800000008000000aull,
        0x8000000080008081ull, 0x8000000000008080ull,
        0x0000000080000001ull, 0x8000000080008008ull,
    };
    constexpr unsigned rotations[25] = {
        0u, 1u, 62u, 28u, 27u,
        36u, 44u, 6u, 55u, 20u,
        3u, 10u, 43u, 25u, 39u,
        41u, 45u, 15u, 21u, 8u,
        18u, 2u, 61u, 56u, 14u,
    };

    for (unsigned round = 0; round < 24u; ++round) {
        std::uint64_t c[5];
        std::uint64_t d[5];
        for (unsigned x = 0; x < 5u; ++x) {
            c[x] = state[x] ^ state[x + 5u] ^ state[x + 10u] ^ state[x + 15u] ^ state[x + 20u];
        }
        for (unsigned x = 0; x < 5u; ++x) {
            d[x] = c[(x + 4u) % 5u] ^ rotl64(c[(x + 1u) % 5u], 1u);
        }
        for (unsigned y = 0; y < 5u; ++y) {
            for (unsigned x = 0; x < 5u; ++x) {
                state[x + 5u * y] ^= d[x];
            }
        }

        std::uint64_t b[25] = {};
        for (unsigned y = 0; y < 5u; ++y) {
            for (unsigned x = 0; x < 5u; ++x) {
                const unsigned source = x + 5u * y;
                const unsigned target_x = y;
                const unsigned target_y = (2u * x + 3u * y) % 5u;
                b[target_x + 5u * target_y] = rotl64(state[source], rotations[source]);
            }
        }

        for (unsigned y = 0; y < 5u; ++y) {
            for (unsigned x = 0; x < 5u; ++x) {
                state[x + 5u * y] = b[x + 5u * y] ^
                    ((~b[(x + 1u) % 5u + 5u * y]) & b[(x + 2u) % 5u + 5u * y]);
            }
        }
        state[0] ^= round_constants[round];
    }
}

SUDHARMA_GPUPOW_HD inline std::array<std::uint8_t, 64> keccak512_64(
    const std::array<std::uint8_t, 64>& input) {
    constexpr std::size_t rate = 72u;
    std::uint64_t state[25] = {};

    for (std::size_t i = 0; i < input.size(); ++i) {
        const std::size_t lane = i / 8u;
        const unsigned shift = static_cast<unsigned>((i % 8u) * 8u);
        state[lane] ^= static_cast<std::uint64_t>(input[i]) << shift;
    }
    state[input.size() / 8u] ^= static_cast<std::uint64_t>(0x01u) <<
        static_cast<unsigned>((input.size() % 8u) * 8u);
    state[(rate - 1u) / 8u] ^= static_cast<std::uint64_t>(0x80u) <<
        static_cast<unsigned>(((rate - 1u) % 8u) * 8u);
    keccak_f1600(state);

    std::array<std::uint8_t, 64> output{};
    for (std::size_t i = 0; i < output.size(); ++i) {
        output[i] = static_cast<std::uint8_t>(state[i / 8u] >> ((i % 8u) * 8u));
    }
    return output;
}

SUDHARMA_GPUPOW_HD inline void put_word64(std::array<std::uint8_t, 64>* node,
                                          std::uint32_t word_index,
                                          std::uint32_t value) {
    const std::size_t offset = static_cast<std::size_t>(word_index & 15u) * 4u;
    (*node)[offset] = static_cast<std::uint8_t>(value);
    (*node)[offset + 1u] = static_cast<std::uint8_t>(value >> 8u);
    (*node)[offset + 2u] = static_cast<std::uint8_t>(value >> 16u);
    (*node)[offset + 3u] = static_cast<std::uint8_t>(value >> 24u);
}

SUDHARMA_GPUPOW_HD inline std::uint32_t word64_raw(const std::uint8_t* node,
                                                   std::uint32_t word_index) {
    const std::size_t offset = static_cast<std::size_t>(word_index & 15u) * 4u;
    return static_cast<std::uint32_t>(node[offset]) |
           (static_cast<std::uint32_t>(node[offset + 1u]) << 8u) |
           (static_cast<std::uint32_t>(node[offset + 2u]) << 16u) |
           (static_cast<std::uint32_t>(node[offset + 3u]) << 24u);
}

SUDHARMA_GPUPOW_HD inline std::array<std::uint8_t, 64> dataset_item_from_cache(
    const std::uint8_t* cache,
    std::uint32_t cache_nodes,
    std::uint32_t index) {
    std::array<std::uint8_t, 64> mix{};
    if (cache == nullptr || cache_nodes == 0u) return mix;

    const std::size_t base =
        static_cast<std::size_t>(index % cache_nodes) * mix.size();
    for (std::size_t i = 0; i < mix.size(); ++i) mix[i] = cache[base + i];
    put_word64(&mix, 0u, word64(mix, 0u) ^ index);
    mix = keccak512_64(mix);

    for (std::uint32_t parent = 0; parent < kDatasetParents; ++parent) {
        const std::uint32_t selector =
            dataset_fnv1(index ^ parent, word64(mix, parent % 16u));
        const std::size_t parent_base =
            static_cast<std::size_t>(selector % cache_nodes) * mix.size();
        for (std::uint32_t word = 0; word < 16u; ++word) {
            put_word64(
                &mix,
                word,
                dataset_fnv1(
                    word64(mix, word),
                    word64_raw(cache + parent_base, word)));
        }
    }

    return keccak512_64(mix);
}

template <std::size_t CacheNodes>
SUDHARMA_GPUPOW_HD inline std::array<std::uint8_t, 64> dataset_item(
    const std::array<std::array<std::uint8_t, 64>, CacheNodes>& cache,
    std::uint32_t index) {
    static_assert(CacheNodes > 0u, "dataset cache must not be empty");

    auto mix = cache[static_cast<std::size_t>(index % static_cast<std::uint32_t>(CacheNodes))];
    put_word64(&mix, 0u, word64(mix, 0u) ^ index);
    mix = keccak512_64(mix);

    for (std::uint32_t parent = 0; parent < kDatasetParents; ++parent) {
        const std::uint32_t selector = dataset_fnv1(index ^ parent, word64(mix, parent % 16u));
        const auto& parent_node = cache[static_cast<std::size_t>(selector % static_cast<std::uint32_t>(CacheNodes))];
        for (std::uint32_t word = 0; word < 16u; ++word) {
            put_word64(&mix, word, dataset_fnv1(word64(mix, word), word64(parent_node, word)));
        }
    }

    return keccak512_64(mix);
}

}  // namespace sudharma::gpupowv1

#undef SUDHARMA_GPUPOW_HD
