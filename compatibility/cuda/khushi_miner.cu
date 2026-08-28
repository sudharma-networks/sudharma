#include <cuda_runtime.h>

#include <array>
#include <chrono>
#include <cstdint>
#include <cstdio>
#include <cstring>

#include "gpupow_v1_search.cuh"

namespace {

using sudharma::gpupowv1::SearchCache;
using sudharma::gpupowv1::SearchJob;
using sudharma::gpupowv1::kSearchNoNonce;

int cuda_error(const char* operation, cudaError_t err) {
    if (err == cudaSuccess) return 0;
    std::fprintf(stderr, "%s: %s\n", operation, cudaGetErrorString(err));
    return 1;
}

SearchCache benchmark_cache() {
    SearchCache cache{};
    for (std::size_t node = 0; node < cache.size(); ++node) {
        for (std::size_t i = 0; i < cache[node].size(); ++i) {
            cache[node][i] = static_cast<std::uint8_t>((node * 67u + i * 29u + 11u) & 0xffu);
        }
    }
    return cache;
}

SearchJob benchmark_job(std::uint64_t nonce_start, std::uint64_t nonce_count) {
    SearchJob job{};
    constexpr char header[] = "khushi-algorithm-rtx2060-benchmark";
    job.header_length = static_cast<std::uint32_t>(sizeof(header) - 1u);
    for (std::size_t i = 0; i < sizeof(header) - 1u; ++i) {
        job.header_prefix[i] = static_cast<std::uint8_t>(header[i]);
    }
    for (std::size_t i = 0; i < job.program_seed.size(); ++i) {
        job.program_seed[i] = static_cast<std::uint8_t>((i * 17u + 3u) & 0xffu);
    }
    job.target.fill(0u);
    job.nonce_start = nonce_start;
    job.nonce_count = nonce_count;
    return job;
}

int print_device_info() {
    int count = 0;
    if (cuda_error("cudaGetDeviceCount", cudaGetDeviceCount(&count)) != 0) return 1;
    if (count == 0) {
        std::fputs("Khushi Algorithm: no CUDA device found\n", stderr);
        return 2;
    }

    std::printf("Khushi Algorithm CUDA devices=%d\n", count);
    for (int device = 0; device < count; ++device) {
        cudaDeviceProp prop{};
        if (cuda_error("cudaGetDeviceProperties", cudaGetDeviceProperties(&prop, device)) != 0) return 1;
        std::printf("device=%d name=%s capability=%d.%d vram_bytes=%llu multiprocessors=%d\n",
                    device,
                    prop.name,
                    prop.major,
                    prop.minor,
                    static_cast<unsigned long long>(prop.totalGlobalMem),
                    prop.multiProcessorCount);
    }
    return 0;
}

int run_benchmark(unsigned seconds) {
    if (seconds == 0u) seconds = 10u;
    if (print_device_info() != 0) return 2;

    SearchCache host_cache = benchmark_cache();
    SearchCache* device_cache = nullptr;
    std::uint32_t* stale_generation = nullptr;
    unsigned long long* found_nonce = nullptr;
    unsigned long long* hashes_done = nullptr;

    if (cuda_error("cudaMalloc(cache)", cudaMalloc(&device_cache, sizeof(SearchCache))) != 0) return 1;
    if (cuda_error("cudaMalloc(stale_generation)", cudaMalloc(&stale_generation, sizeof(std::uint32_t))) != 0) return 1;
    if (cuda_error("cudaMalloc(found_nonce)", cudaMalloc(&found_nonce, sizeof(unsigned long long))) != 0) return 1;
    if (cuda_error("cudaMalloc(hashes_done)", cudaMalloc(&hashes_done, sizeof(unsigned long long))) != 0) return 1;

    const std::uint32_t generation = 1u;
    const unsigned long long no_nonce = kSearchNoNonce;
    unsigned long long zero = 0ull;
    if (cuda_error("cudaMemcpy(cache)", cudaMemcpy(device_cache, &host_cache, sizeof(host_cache), cudaMemcpyHostToDevice)) != 0) return 1;
    if (cuda_error("cudaMemcpy(generation)", cudaMemcpy(stale_generation, &generation, sizeof(generation), cudaMemcpyHostToDevice)) != 0) return 1;
    if (cuda_error("cudaMemcpy(found_nonce)", cudaMemcpy(found_nonce, &no_nonce, sizeof(no_nonce), cudaMemcpyHostToDevice)) != 0) return 1;
    if (cuda_error("cudaMemcpy(hashes_done)", cudaMemcpy(hashes_done, &zero, sizeof(zero), cudaMemcpyHostToDevice)) != 0) return 1;

    constexpr unsigned threads = 32u;
    constexpr std::uint64_t nonces_per_launch = 32u;
    std::uint64_t nonce_start = 0u;
    const auto started = std::chrono::steady_clock::now();
    const auto deadline = started + std::chrono::seconds(seconds);

    do {
        SearchJob job = benchmark_job(nonce_start, nonces_per_launch);
        sudharma::gpupowv1::khushi_search_kernel<<<1u, threads>>>(
            job, device_cache, stale_generation, generation, found_nonce, hashes_done);
        if (cuda_error("khushi_search_kernel launch", cudaGetLastError()) != 0) return 1;
        if (cuda_error("khushi_search_kernel sync", cudaDeviceSynchronize()) != 0) return 1;
        nonce_start += nonces_per_launch;
    } while (std::chrono::steady_clock::now() < deadline);

    unsigned long long hashes = 0ull;
    if (cuda_error("cudaMemcpy(hashes_done host)", cudaMemcpy(&hashes, hashes_done, sizeof(hashes), cudaMemcpyDeviceToHost)) != 0) return 1;

    const auto ended = std::chrono::steady_clock::now();
    const double elapsed = std::chrono::duration<double>(ended - started).count();
    const double rate = elapsed > 0.0 ? static_cast<double>(hashes) / elapsed : 0.0;
    std::printf("Khushi Algorithm benchmark seconds=%.3f hashes=%llu hashrate_hps=%.6f\n",
                elapsed, hashes, rate);

    cudaFree(hashes_done);
    cudaFree(found_nonce);
    cudaFree(stale_generation);
    cudaFree(device_cache);
    return 0;
}

}  // namespace

int main(int argc, char** argv) {
    if (argc == 2 && std::strcmp(argv[1], "--device-info") == 0) {
        return print_device_info();
    }
    if ((argc == 2 || argc == 3) && std::strcmp(argv[1], "--benchmark") == 0) {
        unsigned seconds = 10u;
        if (argc == 3) {
            seconds = static_cast<unsigned>(std::strtoul(argv[2], nullptr, 10));
        }
        return run_benchmark(seconds);
    }
    if (argc >= 2 && std::strcmp(argv[1], "--mine") == 0) {
        std::fputs("Khushi Algorithm network mining is gated until RTX 2060 interoperability passes; CPU fallback prohibited\n", stderr);
        return 3;
    }

    std::fputs("usage: khushi-miner --device-info | --benchmark [seconds] | --mine\n", stderr);
    return 64;
}
