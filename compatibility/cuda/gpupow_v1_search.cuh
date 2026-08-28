#pragma once

#include <array>
#include <cstddef>
#include <cstdint>

#include "gpupow_v1_final.h"

#ifdef __CUDACC__
#include <cuda_runtime.h>
#endif

namespace sudharma::gpupowv1 {

constexpr std::size_t kSearchCacheNodes = 8u;
constexpr std::size_t kSearchMaxHeaderBytes = 256u;
constexpr unsigned long long kSearchNoNonce = ~0ull;

using SearchCache = std::array<std::array<std::uint8_t, 64>, kSearchCacheNodes>;

struct SearchJob {
    std::array<std::uint8_t, kSearchMaxHeaderBytes> header_prefix{};
    std::uint32_t header_length = 0u;
    std::array<std::uint8_t, 32> program_seed{};
    std::array<std::uint8_t, 32> target{};
    std::uint64_t nonce_start = 0u;
    std::uint64_t nonce_count = 0u;
};

#ifdef __CUDACC__

__device__ inline std::array<std::uint8_t, 32> search_sha256(
    const std::uint8_t* input, std::size_t input_size) {
    std::uint32_t state[8] = {
        0x6a09e667u, 0xbb67ae85u, 0x3c6ef372u, 0xa54ff53au,
        0x510e527fu, 0x9b05688cu, 0x1f83d9abu, 0x5be0cd19u,
    };

    constexpr std::size_t kMaxMessageBytes = 384u;
    std::uint8_t message[kMaxMessageBytes] = {};
    if (input_size > kMaxMessageBytes - 9u) {
        return {};
    }
    for (std::size_t i = 0; i < input_size; ++i) message[i] = input[i];
    message[input_size] = 0x80u;

    std::size_t padded = input_size + 1u;
    while ((padded % 64u) != 56u) ++padded;
    const std::uint64_t bit_length = static_cast<std::uint64_t>(input_size) * 8u;
    for (std::size_t i = 0; i < 8u; ++i) {
        message[padded + 7u - i] = static_cast<std::uint8_t>(bit_length >> (i * 8u));
    }
    padded += 8u;

    for (std::size_t offset = 0; offset < padded; offset += 64u) {
        final_sha256_compress(state, message + offset);
    }

    std::array<std::uint8_t, 32> digest{};
    for (std::size_t i = 0; i < 8u; ++i) {
        digest[i * 4u] = static_cast<std::uint8_t>(state[i] >> 24u);
        digest[i * 4u + 1u] = static_cast<std::uint8_t>(state[i] >> 16u);
        digest[i * 4u + 2u] = static_cast<std::uint8_t>(state[i] >> 8u);
        digest[i * 4u + 3u] = static_cast<std::uint8_t>(state[i]);
    }
    return digest;
}

__device__ inline std::array<std::uint8_t, 32> search_header_digest(
    const SearchJob& job, std::uint64_t nonce) {
    const char domain[] = "SUDHARMA-GPU-POW-V1-REFERENCE-HEADER";
    constexpr std::size_t kMaxInput = sizeof(domain) + kSearchMaxHeaderBytes + 8u;
    std::uint8_t input[kMaxInput] = {};
    std::size_t n = 0u;

    for (std::size_t i = 0; i < sizeof(domain); ++i) {
        input[n++] = static_cast<std::uint8_t>(domain[i]);
    }
    const std::size_t header_length =
        job.header_length <= kSearchMaxHeaderBytes ? job.header_length : kSearchMaxHeaderBytes;
    for (std::size_t i = 0; i < header_length; ++i) input[n++] = job.header_prefix[i];
    for (unsigned shift = 0u; shift < 64u; shift += 8u) {
        input[n++] = static_cast<std::uint8_t>((nonce >> shift) & 0xffu);
    }
    return search_sha256(input, n);
}

__device__ inline bool search_meets_target(
    const std::array<std::uint8_t, 32>& digest,
    const std::array<std::uint8_t, 32>& target) {
    for (std::size_t i = 0; i < digest.size(); ++i) {
        if (digest[i] < target[i]) return true;
        if (digest[i] > target[i]) return false;
    }
    return true;
}

// Each CUDA thread evaluates one complete Khushi Algorithm nonce. Dataset items
// are generated on-device from the copied epoch cache by final_digest_from_header.
// The generation word is checked before and after the expensive digest so the
// host can cancel stale work without accepting a solution from an old template.
__global__ void khushi_search_kernel(
    SearchJob job,
    const SearchCache* cache,
    const std::uint32_t* stale_generation,
    std::uint32_t expected_generation,
    unsigned long long* found_nonce,
    unsigned long long* hashes_done) {
    if (stale_generation != nullptr && *stale_generation != expected_generation) return;

    const std::uint64_t offset =
        static_cast<std::uint64_t>(blockIdx.x) * blockDim.x + threadIdx.x;
    if (offset >= job.nonce_count) return;

    const std::uint64_t nonce = job.nonce_start + offset;
    const auto header = search_header_digest(job, nonce);
    const auto digest = final_digest_from_header(header, job.program_seed, *cache);

    if (stale_generation != nullptr && *stale_generation != expected_generation) return;
    atomicAdd(hashes_done, 1ull);
    if (search_meets_target(digest, job.target)) {
        atomicMin(found_nonce, static_cast<unsigned long long>(nonce));
    }
}

// Hardware interoperability kernel. One thread executes the same complete
// header -> 16-lane program -> reduction -> final digest path used by search,
// then returns the 32-byte digest for comparison with the locked Go vector.
__global__ void khushi_vector_kernel(
    SearchJob job,
    const SearchCache* cache,
    std::uint64_t nonce,
    std::uint8_t* output_digest) {
    if (blockIdx.x != 0u || threadIdx.x != 0u) return;
    const auto header = search_header_digest(job, nonce);
    const auto digest = final_digest_from_header(header, job.program_seed, *cache);
    for (std::size_t i = 0; i < digest.size(); ++i) output_digest[i] = digest[i];
}

#endif  // __CUDACC__

}  // namespace sudharma::gpupowv1
