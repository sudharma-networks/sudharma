#include <cuda_runtime.h>

#include <array>
#include <cstddef>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <string>
#include <vector>

#include "gpupow_v1_chunks.cuh"
#include "gpupow_v1_dataset.h"

namespace {

using CacheNode = std::array<std::uint8_t, 64>;
using sudharma::gpupowv1::dataset_item_from_cache;
using sudharma::gpupowv1::keccak512_64;
using sudharma::gpupowv1::keccak_f1600;
using sudharma::gpupowv1::kMinimumDedicatedVRAMBytes;
using sudharma::gpupowv1::kProductionCacheBytes;
using sudharma::gpupowv1::kProductionItemBytes;
using sudharma::gpupowv1::word64;

constexpr std::uint32_t kProductionCacheNodes =
    static_cast<std::uint32_t>(kProductionCacheBytes / kProductionItemBytes);
static_assert(kProductionCacheNodes == 262144u, "production cache node count changed");
static_assert(sizeof(CacheNode) == 64u, "cache node layout changed");

constexpr std::array<std::uint8_t, 32> kEpoch0Seed = {
    0xa2, 0x0a, 0xc9, 0x1b, 0x09, 0x2f, 0xf0, 0xf1,
    0xac, 0x89, 0xfe, 0xdd, 0xcf, 0x16, 0xca, 0x2f,
    0x47, 0xb0, 0x30, 0xeb, 0x84, 0x1d, 0x75, 0xbc,
    0xc0, 0x26, 0xcd, 0xe7, 0x51, 0xf9, 0xed, 0x7a,
};

constexpr std::array<std::uint32_t, 4> kBoundaryIndices = {
    0u, 4194303u, 4194304u, 33554431u,
};

constexpr const char* kExpectedDigests[4] = {
    "eb2d4ea4f291aa352d7e6788aa56703a3b1fe7d47564dc80d109d9f8de75e4de5b875e27d6aae37523753a88f00e337f15983e38819c630e2128d04f0548ad8b",
    "9b906e7a49f6b4de879f935d7fb68815481f9091540f083d81f54dcce44543bf95ceb282b3877aaaeb530e756752e79c7295282b008c54e8c05735daef437500",
    "5ff3da521e9a3ad3b2722b8dbc4a7d0d95aab31abba2bfee881ac1552912f754b0819dbd047b9b6b1b05b897968aed49f7002daaaccbd877e83c9812951d5a3e",
    "baade0e152b5a0741b4c865e5d547a4a10887abf3edbc9a93dc80361d205368aa3f7914076210afd2bd233966f32e264efb498719c5afd947c4a5226d7fd83a9",
};

int cuda_error(const char* operation, cudaError_t error) {
    if (error == cudaSuccess) return 0;
    std::fprintf(stderr, "%s: %s\n", operation, cudaGetErrorString(error));
    return 1;
}

CacheNode keccak512_seed32(const std::array<std::uint8_t, 32>& input) {
    constexpr std::size_t kRateBytes = 72u;
    std::uint64_t state[25] = {};
    for (std::size_t i = 0; i < input.size(); ++i) {
        state[i / 8u] ^= static_cast<std::uint64_t>(input[i]) << ((i % 8u) * 8u);
    }
    state[input.size() / 8u] ^= static_cast<std::uint64_t>(0x01u) <<
        ((input.size() % 8u) * 8u);
    state[(kRateBytes - 1u) / 8u] ^= static_cast<std::uint64_t>(0x80u) <<
        (((kRateBytes - 1u) % 8u) * 8u);
    keccak_f1600(state);

    CacheNode output{};
    for (std::size_t i = 0; i < output.size(); ++i) {
        output[i] = static_cast<std::uint8_t>(state[i / 8u] >> ((i % 8u) * 8u));
    }
    return output;
}

std::vector<CacheNode> build_epoch0_production_cache() {
    std::vector<CacheNode> cache(kProductionCacheNodes);
    cache[0] = keccak512_seed32(kEpoch0Seed);
    for (std::size_t i = 1; i < cache.size(); ++i) {
        cache[i] = keccak512_64(cache[i - 1u]);
    }

    constexpr unsigned kCacheRounds = 3u;
    for (unsigned round = 0; round < kCacheRounds; ++round) {
        for (std::size_t i = 0; i < cache.size(); ++i) {
            const CacheNode previous = cache[(i + cache.size() - 1u) % cache.size()];
            const std::uint32_t selector = word64(cache[i], 0u) % kProductionCacheNodes;
            CacheNode mixed{};
            for (std::size_t byte = 0; byte < mixed.size(); ++byte) {
                mixed[byte] = previous[byte] ^ cache[selector][byte];
            }
            cache[i] = keccak512_64(mixed);
        }
    }
    return cache;
}

std::string encode_hex(const std::uint8_t* bytes, std::size_t size) {
    static constexpr char kHex[] = "0123456789abcdef";
    std::string output(size * 2u, '0');
    for (std::size_t i = 0; i < size; ++i) {
        output[i * 2u] = kHex[bytes[i] >> 4u];
        output[i * 2u + 1u] = kHex[bytes[i] & 0x0fu];
    }
    return output;
}

__global__ void production_dataset_vector_kernel(
    const std::uint8_t* cache,
    std::uint32_t cache_nodes,
    const std::uint32_t* indices,
    std::uint32_t count,
    std::uint8_t* output) {
    const std::uint32_t vector = blockIdx.x * blockDim.x + threadIdx.x;
    if (vector >= count) return;
    const auto item = dataset_item_from_cache(cache, cache_nodes, indices[vector]);
    for (std::size_t byte = 0; byte < item.size(); ++byte) {
        output[static_cast<std::size_t>(vector) * item.size() + byte] = item[byte];
    }
}

bool parse_device(const char* text, int* output) {
    char* end = nullptr;
    const long value = std::strtol(text, &end, 10);
    if (text == end || *end != '\0' || value < 0 || value > 0x7fffffffL) return false;
    *output = static_cast<int>(value);
    return true;
}

int run(int device) {
    int count = 0;
    if (cuda_error("cudaGetDeviceCount", cudaGetDeviceCount(&count)) != 0) return 1;
    if (count == 0 || device < 0 || device >= count) {
        std::fprintf(stderr, "invalid CUDA device=%d devices=%d\n", device, count);
        return 2;
    }
    if (cuda_error("cudaSetDevice", cudaSetDevice(device)) != 0) return 1;

    cudaDeviceProp properties{};
    if (cuda_error("cudaGetDeviceProperties", cudaGetDeviceProperties(&properties, device)) != 0) return 1;
    std::printf(
        "production-vector-device=%d name=%s total_vram_bytes=%llu cache_bytes=%llu cache_nodes=%u\n",
        device,
        properties.name,
        static_cast<unsigned long long>(properties.totalGlobalMem),
        static_cast<unsigned long long>(kProductionCacheBytes),
        kProductionCacheNodes);
    if (properties.totalGlobalMem < kMinimumDedicatedVRAMBytes) {
        std::fprintf(stderr,
                     "production-vector-self-test=failed dedicated VRAM below 4 GiB target\n");
        return 2;
    }

    std::puts("production-vector-cache-build=started epoch=0 nodes=262144");
    const auto host_cache = build_epoch0_production_cache();
    std::puts("production-vector-cache-build=ok");

    std::uint8_t* device_cache = nullptr;
    std::uint32_t* device_indices = nullptr;
    std::uint8_t* device_output = nullptr;
    auto cleanup = [&]() {
        if (device_output != nullptr) cudaFree(device_output);
        if (device_indices != nullptr) cudaFree(device_indices);
        if (device_cache != nullptr) cudaFree(device_cache);
    };

    if (cuda_error(
            "cudaMalloc(production vector cache)",
            cudaMalloc(reinterpret_cast<void**>(&device_cache),
                       static_cast<std::size_t>(kProductionCacheBytes))) != 0) {
        cleanup();
        return 1;
    }
    if (cuda_error(
            "cudaMalloc(production vector indices)",
            cudaMalloc(reinterpret_cast<void**>(&device_indices), sizeof(kBoundaryIndices))) != 0) {
        cleanup();
        return 1;
    }
    constexpr std::size_t kOutputBytes = kBoundaryIndices.size() * sizeof(CacheNode);
    if (cuda_error(
            "cudaMalloc(production vector output)",
            cudaMalloc(reinterpret_cast<void**>(&device_output), kOutputBytes)) != 0) {
        cleanup();
        return 1;
    }

    if (cuda_error(
            "cudaMemcpy(production vector cache)",
            cudaMemcpy(device_cache, host_cache.data(),
                       static_cast<std::size_t>(kProductionCacheBytes),
                       cudaMemcpyHostToDevice)) != 0 ||
        cuda_error(
            "cudaMemcpy(production vector indices)",
            cudaMemcpy(device_indices, kBoundaryIndices.data(), sizeof(kBoundaryIndices),
                       cudaMemcpyHostToDevice)) != 0) {
        cleanup();
        return 1;
    }

    production_dataset_vector_kernel<<<1u, static_cast<unsigned>(kBoundaryIndices.size())>>>(
        device_cache,
        kProductionCacheNodes,
        device_indices,
        static_cast<std::uint32_t>(kBoundaryIndices.size()),
        device_output);
    if (cuda_error("production_dataset_vector_kernel launch", cudaGetLastError()) != 0 ||
        cuda_error("production_dataset_vector_kernel sync", cudaDeviceSynchronize()) != 0) {
        cleanup();
        return 1;
    }

    std::array<CacheNode, kBoundaryIndices.size()> output{};
    if (cuda_error(
            "cudaMemcpy(production vector output host)",
            cudaMemcpy(output.data(), device_output, kOutputBytes,
                       cudaMemcpyDeviceToHost)) != 0) {
        cleanup();
        return 1;
    }
    cleanup();

    for (std::size_t i = 0; i < output.size(); ++i) {
        const std::string digest = encode_hex(output[i].data(), output[i].size());
        const bool matches = digest == kExpectedDigests[i];
        std::printf("production-vector index=%u digest=%s status=%s\n",
                    kBoundaryIndices[i], digest.c_str(), matches ? "ok" : "mismatch");
        if (!matches) {
            std::fprintf(stderr, "expected=%s\n", kExpectedDigests[i]);
            return 4;
        }
    }

    std::puts("production-vector-self-test=ok");
    return 0;
}

void usage() {
    std::fputs("usage: khushi-production-vectors.exe [--device N]\n", stderr);
}

}  // namespace

int main(int argc, char** argv) {
    int device = 0;
    if (argc == 3 && std::string(argv[1]) == "--device") {
        if (!parse_device(argv[2], &device)) {
            usage();
            return 64;
        }
    } else if (argc != 1) {
        usage();
        return 64;
    }
    return run(device);
}
