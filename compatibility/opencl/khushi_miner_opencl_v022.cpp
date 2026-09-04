#include <array>
#include <cstddef>
#include <cstdint>
#include <cstdio>
#include <cstring>
#include <vector>

#define main khushi_opencl_legacy_main
#include "khushi_miner_opencl.cpp"
#undef main

#include "../cuda/gpupow_v1_dataset.h"

namespace {

using ProductionCacheNode = std::array<std::uint8_t, 64>;
constexpr cl_uint kProductionConsensusCacheNodes = 262144u;
static_assert(
    static_cast<cl_ulong>(kProductionConsensusCacheNodes) * sizeof(ProductionCacheNode) ==
        kProductionCacheBytes,
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

int production_consensus_search(
    const char* header_hex,
    const char* target_hex,
    cl_ulong height,
    cl_uint cache_nodes,
    const char* program_seed_hex,
    const char* epoch_seed_hex) {
    if (cache_nodes != kProductionConsensusCacheNodes) {
        std::fprintf(
            stderr,
            "production consensus search requires cache_nodes=%u, got %u\n",
            static_cast<unsigned>(kProductionConsensusCacheNodes),
            static_cast<unsigned>(cache_nodes));
        return 64;
    }

    auto devices = gpu_devices();
    if (devices.empty() || selected_device < 0 ||
        static_cast<std::size_t>(selected_device) >= devices.size()) {
        std::fputs("Khushi Algorithm requires a selected OpenCL GPU\n", stderr);
        return 2;
    }
    if (devices[static_cast<std::size_t>(selected_device)].global_memory <
        kMinimumDedicatedVRAMBytes) {
        std::fputs("production consensus search requires at least 4 GiB dedicated VRAM\n", stderr);
        return 2;
    }

    std::vector<unsigned char> header, target, seed, epoch_seed_bytes;
    if (!hex_bytes(header_hex, &header) || header.empty() || header.size() > 256u) {
        std::fputs("invalid --header-prefix-hex value\n", stderr);
        return 64;
    }
    if (!hex_bytes(target_hex, &target) || target.size() != 32u) {
        std::fputs("invalid --target-hex value\n", stderr);
        return 64;
    }
    if (!hex_bytes(program_seed_hex, &seed) || seed.size() != 32u) {
        std::fputs("invalid --program-seed-hex value\n", stderr);
        return 64;
    }
    if (!hex_bytes(epoch_seed_hex, &epoch_seed_bytes) || epoch_seed_bytes.size() != 32u) {
        std::fputs("invalid --epoch-seed-hex value\n", stderr);
        return 64;
    }

    std::array<std::uint8_t, 32> epoch_seed{};
    for (std::size_t i = 0; i < epoch_seed.size(); ++i) epoch_seed[i] = epoch_seed_bytes[i];

    std::printf(
        "production-consensus-search=enabled backend=opencl height=%llu cache_nodes=%u\n",
        static_cast<unsigned long long>(height),
        static_cast<unsigned>(cache_nodes));
    std::puts("production-consensus-cache-build=started");
    const auto cache_nodes_host = build_production_consensus_cache(epoch_seed);
    const auto* cache_bytes =
        reinterpret_cast<const unsigned char*>(cache_nodes_host.data());
    const std::size_t cache_size =
        cache_nodes_host.size() * sizeof(ProductionCacheNode);
    std::puts("production-consensus-cache-build=ok");

    Runtime rt = make_runtime();
    cl_int rc = 0;
    cl_kernel kernel = clCreateKernel(rt.program, "khushi_search", &rc);
    check(rc, "clCreateKernel(khushi_search production)");

    const cl_uint header_len = static_cast<cl_uint>(header.size());
    const cl_uint generation = 1u;
    cl_uint found_flag = 0u;
    cl_uint hashes = 0u;
    const cl_ulong no_nonce = ~(cl_ulong)0;
    cl_ulong found = no_nonce;

    cl_mem h = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR,
                              header.size(), header.data(), &rc);
    check(rc, "production header");
    cl_mem s = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR,
                              seed.size(), seed.data(), &rc);
    check(rc, "production seed");
    cl_mem c = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR,
                              cache_size, const_cast<unsigned char*>(cache_bytes), &rc);
    check(rc, "production cache");
    cl_mem t = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR,
                              target.size(), target.data(), &rc);
    check(rc, "production target");
    cl_mem g = clCreateBuffer(rt.context, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR,
                              sizeof(generation), (void*)&generation, &rc);
    check(rc, "production generation");
    cl_mem ff = clCreateBuffer(rt.context, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR,
                               sizeof(found_flag), &found_flag, &rc);
    check(rc, "production found flag");
    cl_mem f = clCreateBuffer(rt.context, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR,
                              sizeof(found), &found, &rc);
    check(rc, "production found nonce");
    cl_mem hd = clCreateBuffer(rt.context, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR,
                               sizeof(hashes), &hashes, &rc);
    check(rc, "production hashes");

    const cl_ulong max_nonces = 65536u;
    const cl_ulong nonces_per_launch = 64u;
    for (cl_ulong nonce_start = 0u;
         nonce_start < max_nonces && found == no_nonce;
         nonce_start += nonces_per_launch) {
        cl_ulong nonce_count = nonces_per_launch;
        if (nonce_start + nonce_count > max_nonces) nonce_count = max_nonces - nonce_start;

        int a = 0;
        check(clSetKernelArg(kernel, a++, sizeof(h), &h), "production a0");
        check(clSetKernelArg(kernel, a++, sizeof(header_len), &header_len), "production a1");
        check(clSetKernelArg(kernel, a++, sizeof(s), &s), "production a2");
        check(clSetKernelArg(kernel, a++, sizeof(c), &c), "production a3");
        check(clSetKernelArg(kernel, a++, sizeof(cache_nodes), &cache_nodes), "production a4");
        check(clSetKernelArg(kernel, a++, sizeof(t), &t), "production a5");
        check(clSetKernelArg(kernel, a++, sizeof(nonce_start), &nonce_start), "production a6");
        check(clSetKernelArg(kernel, a++, sizeof(nonce_count), &nonce_count), "production a7");
        check(clSetKernelArg(kernel, a++, sizeof(g), &g), "production a8");
        check(clSetKernelArg(kernel, a++, sizeof(generation), &generation), "production a9");
        check(clSetKernelArg(kernel, a++, sizeof(ff), &ff), "production a10");
        check(clSetKernelArg(kernel, a++, sizeof(f), &f), "production a11");
        check(clSetKernelArg(kernel, a++, sizeof(hd), &hd), "production a12");

        const std::size_t global = static_cast<std::size_t>(nonce_count);
        check(clEnqueueNDRangeKernel(
                  rt.queue, kernel, 1, nullptr, &global, nullptr, 0, nullptr, nullptr),
              "production enqueue");
        check(clFinish(rt.queue), "production finish");
        check(clEnqueueReadBuffer(
                  rt.queue, f, CL_TRUE, 0, sizeof(found), &found, 0, nullptr, nullptr),
              "production nonce read");
    }

    check(clEnqueueReadBuffer(
              rt.queue, hd, CL_TRUE, 0, sizeof(hashes), &hashes, 0, nullptr, nullptr),
          "production hashes read");

    clReleaseMemObject(hd);
    clReleaseMemObject(f);
    clReleaseMemObject(ff);
    clReleaseMemObject(g);
    clReleaseMemObject(t);
    clReleaseMemObject(c);
    clReleaseMemObject(s);
    clReleaseMemObject(h);
    clReleaseKernel(kernel);

    if (found == no_nonce) {
        std::printf("production-consensus-search=not-found hashes=%u\n",
                    static_cast<unsigned>(hashes));
        return 4;
    }
    std::printf(
        "staging-solution-nonce=%llu hashes=%u production-consensus-search=solved\n",
        static_cast<unsigned long long>(found),
        static_cast<unsigned>(hashes));
    return 0;
}

}  // namespace

int main(int argc, char** argv) {
    int arg = 1;
    if (argc >= 3 && std::strcmp(argv[1], "--device") == 0) {
        selected_device = std::atoi(argv[2]);
        arg = 3;
    }

    if (arg < argc && std::strcmp(argv[arg], "--staging-search") == 0 &&
        arg + 13 == argc &&
        std::strcmp(argv[arg + 1], "--header-prefix-hex") == 0 &&
        std::strcmp(argv[arg + 3], "--target-hex") == 0 &&
        std::strcmp(argv[arg + 5], "--height") == 0 &&
        std::strcmp(argv[arg + 7], "--cache-nodes") == 0 &&
        std::strcmp(argv[arg + 9], "--program-seed-hex") == 0 &&
        std::strcmp(argv[arg + 11], "--epoch-seed-hex") == 0) {
        cl_ulong height = 0;
        cl_uint cache_nodes = 0;
        if (!parse_u64(argv[arg + 6], &height) ||
            !parse_u32(argv[arg + 8], &cache_nodes)) {
            std::fputs("invalid staging height or cache node count\n", stderr);
            return 64;
        }
        return production_consensus_search(
            argv[arg + 2],
            argv[arg + 4],
            height,
            cache_nodes,
            argv[arg + 10],
            argv[arg + 12]);
    }

    return khushi_opencl_legacy_main(argc, argv);
}
