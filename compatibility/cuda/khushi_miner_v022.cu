#include <array>
#include <cstddef>
#include <cstdint>
#include <cstdio>
#include <cstring>
#include <vector>

#define main khushi_legacy_main
#include "khushi_miner.cu"
#undef main

#include "gpupow_v1_production_search.cuh"

namespace {

using ProductionCacheNode = std::array<std::uint8_t, 64>;
constexpr std::uint32_t kProductionConsensusCacheNodes = 262144u;
static_assert(
    kProductionConsensusCacheNodes ==
        sudharma::gpupowv1::kProductionCacheBytes / sizeof(ProductionCacheNode),
    "production consensus cache size changed");

ProductionCacheNode production_keccak512_seed32(
    const std::array<std::uint8_t, 32>& input) {
    constexpr std::size_t rate = 72u;
    std::uint64_t state[25] = {};
    for (std::size_t i = 0; i < input.size(); ++i) {
        state[i / 8u] ^= static_cast<std::uint64_t>(input[i]) << ((i % 8u) * 8u);
    }
    state[input.size() / 8u] ^=
        static_cast<std::uint64_t>(0x01u) << ((input.size() % 8u) * 8u);
    state[(rate - 1u) / 8u] ^=
        static_cast<std::uint64_t>(0x80u) << (((rate - 1u) % 8u) * 8u);
    sudharma::gpupowv1::keccak_f1600(state);

    ProductionCacheNode output{};
    for (std::size_t i = 0; i < output.size(); ++i) {
        output[i] = static_cast<std::uint8_t>(state[i / 8u] >> ((i % 8u) * 8u));
    }
    return output;
}

std::vector<ProductionCacheNode> build_production_consensus_cache(
    const std::array<std::uint8_t, 32>& epoch_seed) {
    std::vector<ProductionCacheNode> cache(kProductionConsensusCacheNodes);
    cache[0] = production_keccak512_seed32(epoch_seed);
    for (std::size_t i = 1; i < cache.size(); ++i) {
        cache[i] = sudharma::gpupowv1::keccak512_64(cache[i - 1u]);
    }

    constexpr unsigned cache_rounds = 3u;
    for (unsigned round = 0; round < cache_rounds; ++round) {
        for (std::size_t i = 0; i < cache.size(); ++i) {
            const ProductionCacheNode previous =
                cache[(i + cache.size() - 1u) % cache.size()];
            const std::uint32_t selector =
                sudharma::gpupowv1::word64(cache[i], 0u) % kProductionConsensusCacheNodes;
            ProductionCacheNode mixed{};
            for (std::size_t byte = 0; byte < mixed.size(); ++byte) {
                mixed[byte] = previous[byte] ^ cache[selector][byte];
            }
            cache[i] = sudharma::gpupowv1::keccak512_64(mixed);
        }
    }
    return cache;
}

int run_production_consensus_search(
    const char* header_hex,
    const char* target_hex,
    std::uint64_t height,
    std::uint32_t cache_nodes,
    const char* program_seed_hex,
    const char* epoch_seed_hex) {
    if (cache_nodes != kProductionConsensusCacheNodes) {
        std::fprintf(
            stderr,
            "production consensus search requires cache_nodes=%u, got %u\n",
            kProductionConsensusCacheNodes,
            cache_nodes);
        return 64;
    }
    if (select_device(selected_device) != 0) return 2;
    if (production_memory_preflight() != 0) return 2;

    sudharma::gpupowv1::SearchJob job{};
    if (!decode_header_hex(header_hex, &job)) {
        std::fputs("invalid --header-prefix-hex value\n", stderr);
        return 64;
    }
    if (!decode_fixed_hex(target_hex, &job.target)) {
        std::fputs("invalid --target-hex value\n", stderr);
        return 64;
    }
    if (!decode_fixed_hex(program_seed_hex, &job.program_seed)) {
        std::fputs("invalid --program-seed-hex value\n", stderr);
        return 64;
    }
    std::array<std::uint8_t, 32> epoch_seed{};
    if (!decode_fixed_hex(epoch_seed_hex, &epoch_seed)) {
        std::fputs("invalid --epoch-seed-hex value\n", stderr);
        return 64;
    }

    std::printf(
        "production-consensus-search=enabled backend=cuda height=%llu cache_nodes=%u\n",
        static_cast<unsigned long long>(height),
        cache_nodes);
    std::puts("production-consensus-cache-build=started");
    const auto host_cache = build_production_consensus_cache(epoch_seed);
    std::puts("production-consensus-cache-build=ok");

    std::uint8_t* device_cache = nullptr;
    std::uint32_t* stale_generation = nullptr;
    unsigned long long* found_nonce = nullptr;
    unsigned long long* hashes_done = nullptr;
    auto cleanup = [&]() {
        if (hashes_done != nullptr) cudaFree(hashes_done);
        if (found_nonce != nullptr) cudaFree(found_nonce);
        if (stale_generation != nullptr) cudaFree(stale_generation);
        if (device_cache != nullptr) cudaFree(device_cache);
    };

    if (cuda_error(
            "cudaMalloc(production search cache)",
            cudaMalloc(reinterpret_cast<void**>(&device_cache),
                       static_cast<std::size_t>(sudharma::gpupowv1::kProductionCacheBytes))) != 0 ||
        cuda_error(
            "cudaMalloc(production search generation)",
            cudaMalloc(reinterpret_cast<void**>(&stale_generation), sizeof(std::uint32_t))) != 0 ||
        cuda_error(
            "cudaMalloc(production search nonce)",
            cudaMalloc(reinterpret_cast<void**>(&found_nonce), sizeof(unsigned long long))) != 0 ||
        cuda_error(
            "cudaMalloc(production search hashes)",
            cudaMalloc(reinterpret_cast<void**>(&hashes_done), sizeof(unsigned long long))) != 0) {
        cleanup();
        return 1;
    }

    const std::uint32_t generation = 1u;
    const unsigned long long no_nonce = sudharma::gpupowv1::kSearchNoNonce;
    unsigned long long zero = 0ull;
    if (cuda_error(
            "cudaMemcpy(production search cache)",
            cudaMemcpy(
                device_cache,
                host_cache.data(),
                static_cast<std::size_t>(sudharma::gpupowv1::kProductionCacheBytes),
                cudaMemcpyHostToDevice)) != 0 ||
        cuda_error(
            "cudaMemcpy(production search generation)",
            cudaMemcpy(stale_generation, &generation, sizeof(generation), cudaMemcpyHostToDevice)) != 0 ||
        cuda_error(
            "cudaMemcpy(production search nonce)",
            cudaMemcpy(found_nonce, &no_nonce, sizeof(no_nonce), cudaMemcpyHostToDevice)) != 0 ||
        cuda_error(
            "cudaMemcpy(production search hashes)",
            cudaMemcpy(hashes_done, &zero, sizeof(zero), cudaMemcpyHostToDevice)) != 0) {
        cleanup();
        return 1;
    }

    constexpr unsigned threads = 32u;
    constexpr std::uint64_t nonces_per_launch = 32u;
    constexpr std::uint64_t max_nonces = 65536u;
    unsigned long long host_nonce = no_nonce;
    for (std::uint64_t nonce_start = 0u;
         nonce_start < max_nonces && host_nonce == no_nonce;
         nonce_start += nonces_per_launch) {
        job.nonce_start = nonce_start;
        job.nonce_count = nonces_per_launch;
        sudharma::gpupowv1::khushi_production_search_kernel<<<1u, threads>>>(
            job,
            device_cache,
            cache_nodes,
            stale_generation,
            generation,
            found_nonce,
            hashes_done);
        if (cuda_error("Khushi production search launch", cudaGetLastError()) != 0 ||
            cuda_error("Khushi production search sync", cudaDeviceSynchronize()) != 0 ||
            cuda_error(
                "cudaMemcpy(production nonce host)",
                cudaMemcpy(&host_nonce, found_nonce, sizeof(host_nonce), cudaMemcpyDeviceToHost)) != 0) {
            cleanup();
            return 1;
        }
    }

    unsigned long long hashes = 0ull;
    if (cuda_error(
            "cudaMemcpy(production hashes host)",
            cudaMemcpy(&hashes, hashes_done, sizeof(hashes), cudaMemcpyDeviceToHost)) != 0) {
        cleanup();
        return 1;
    }
    cleanup();

    if (host_nonce == no_nonce) {
        std::printf("production-consensus-search=not-found hashes=%llu\n", hashes);
        return 4;
    }
    std::printf(
        "staging-solution-nonce=%llu hashes=%llu production-consensus-search=solved\n",
        host_nonce,
        hashes);
    return 0;
}

}  // namespace

int main(int argc, char** argv) {
    int arg = 1;
    if (argc >= 3 && std::strcmp(argv[arg], "--device") == 0) {
        if (!parse_device(argv[arg + 1], &selected_device)) {
            std::fputs("invalid --device value\n", stderr);
            return 64;
        }
        arg += 2;
    }

    if (arg < argc && std::strcmp(argv[arg], "--staging-search") == 0 &&
        arg + 13 == argc &&
        std::strcmp(argv[arg + 1], "--header-prefix-hex") == 0 &&
        std::strcmp(argv[arg + 3], "--target-hex") == 0 &&
        std::strcmp(argv[arg + 5], "--height") == 0 &&
        std::strcmp(argv[arg + 7], "--cache-nodes") == 0 &&
        std::strcmp(argv[arg + 9], "--program-seed-hex") == 0 &&
        std::strcmp(argv[arg + 11], "--epoch-seed-hex") == 0) {
        std::uint64_t height = 0u;
        std::uint32_t cache_nodes = 0u;
        if (!parse_u64(argv[arg + 6], &height) || !parse_u32(argv[arg + 8], &cache_nodes)) {
            std::fputs("invalid staging height or cache node count\n", stderr);
            return 64;
        }
        return run_production_consensus_search(
            argv[arg + 2],
            argv[arg + 4],
            height,
            cache_nodes,
            argv[arg + 10],
            argv[arg + 12]);
    }

    return khushi_legacy_main(argc, argv);
}
