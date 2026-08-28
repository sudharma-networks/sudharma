#pragma once

#include <cstdint>
#include <limits>

#ifdef __CUDACC__
#define SUDHARMA_CHUNK_HD __host__ __device__
#else
#define SUDHARMA_CHUNK_HD
#endif

namespace sudharma::gpupowv1 {

constexpr std::uint64_t kProductionDatasetBytes = 2ull << 30u;
constexpr std::uint64_t kProductionCacheBytes = 16ull << 20u;
constexpr std::uint64_t kProductionChunkBytes = 256ull << 20u;
constexpr std::uint64_t kProductionItemBytes = 64ull;
constexpr std::uint32_t kProductionChunkCount = 8u;
constexpr std::uint64_t kProductionItemCount =
    kProductionDatasetBytes / kProductionItemBytes;

struct DatasetLocation {
    std::uint32_t chunk = 0u;
    std::uint64_t offset = 0u;
};

SUDHARMA_CHUNK_HD inline bool dataset_item_location(
    std::uint64_t index, DatasetLocation* output) {
    if (output == nullptr || index >= kProductionItemCount) return false;
    if (index > std::numeric_limits<std::uint64_t>::max() / kProductionItemBytes) {
        return false;
    }
    const std::uint64_t byte_offset = index * kProductionItemBytes;
    const std::uint64_t chunk = byte_offset / kProductionChunkBytes;
    if (chunk >= kProductionChunkCount) return false;
    output->chunk = static_cast<std::uint32_t>(chunk);
    output->offset = byte_offset % kProductionChunkBytes;
    return true;
}

}  // namespace sudharma::gpupowv1

#undef SUDHARMA_CHUNK_HD
