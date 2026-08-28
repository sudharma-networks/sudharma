#pragma once

#include <array>
#include <cstddef>
#include <cstdint>

#ifdef __CUDACC__
#define SUDHARMA_GPUPOW_HD __host__ __device__
#else
#define SUDHARMA_GPUPOW_HD
#endif

namespace sudharma::gpupowv1 {

SUDHARMA_GPUPOW_HD inline std::uint32_t cache_index(std::uint32_t selector, std::uint32_t cache_nodes) {
    return cache_nodes == 0u ? 0u : selector % cache_nodes;
}

SUDHARMA_GPUPOW_HD inline std::uint32_t word64(const std::array<std::uint8_t, 64>& node,
                                                std::uint32_t word_index) {
    const std::size_t offset = static_cast<std::size_t>(word_index & 15u) * 4u;
    return static_cast<std::uint32_t>(node[offset]) |
           (static_cast<std::uint32_t>(node[offset + 1u]) << 8u) |
           (static_cast<std::uint32_t>(node[offset + 2u]) << 16u) |
           (static_cast<std::uint32_t>(node[offset + 3u]) << 24u);
}

}  // namespace sudharma::gpupowv1

#undef SUDHARMA_GPUPOW_HD
