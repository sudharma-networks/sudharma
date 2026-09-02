#include <cuda_runtime.h>

#include <array>
#include <chrono>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <limits>

#include "../gpu/gpu_tuning_profile.h"
#include "gpupow_v1_chunks.cuh"
#include "gpupow_v1_search.cuh"

namespace {

namespace tuning = sudharma::gpupowv1::tuning;
using sudharma::gpupowv1::SearchCache;
using sudharma::gpupowv1::SearchJob;
using sudharma::gpupowv1::allocate_dataset_chunks;
using sudharma::gpupowv1::kMinimumDedicatedVRAMBytes;
using sudharma::gpupowv1::kProductionCacheBytes;
using sudharma::gpupowv1::kProductionChunkCount;
using sudharma::gpupowv1::kProductionDatasetBytes;
using sudharma::gpupowv1::kProductionRequiredVRAMBytes;
using sudharma::gpupowv1::kProductionRuntimeReserveBytes;
using sudharma::gpupowv1::kSearchNoNonce;
using sudharma::gpupowv1::release_dataset_chunks;

constexpr char kProgramZeroSeedHex[] =
    "613684e3f3b42773073fb9c99e71f2933eed301d450866fe9a5a5c0530a769bd";
constexpr char kGenesisExpectedDigestHex[] =
    "2a7c15fc6c84a67d43ff7074ac5835aa433145f89d10d1d9e36a99fe22da4b2b";
constexpr const char* kGenesisCacheHex[8] = {
    "68fae850a5cc8cddba29c7a56913c7340e69ba0d92830144aab66584e01a20e86b919e515046196e7ef9c006150fff8affc13fc252dea4490ef1bb4527adcb6b",
    "25c2c0f117e806e1a832bc4bbcc444043633f5100f9ddf3714988c1b2de377b6e9d6979803b8deca82d5267d53eccc89fa92e21984e535f4e8193881ab309741",
    "a8620b2ebeca41fbc773bb837b5e724d6eb2de570d99858df0d7d97067fb8103b21757873b735097b35d3bea8fd1c359a9e8a63c1540c76c9784cf8d975e995c",
    "605f70fb5d9b8f1553027dbac7648e70e314ca31e643521da51b78b8b25d2ab35a5842931cda4e39a20efac8290b6d16a890c5a3f867b19260f85ba788bdeebc",
    "44216b2915d6beffc4ca2c8169950f36294462ac53503471c204035586544ae1a1b7dae7aad22d7cc5501578b0b85378b118bea7a0f545d8985c9f426b05cb00",
    "874ec883bf09e6f15d3a54576fe070194925367ff5302438550ac881314e84cd7b1ea8d8cc2f45d54ddaee552996ff856ad27c240ef6b3a769c253c6d0e8cf9b",
    "d4301c92c2addfc7c7983981263e64dafb2d186e147f508ad346b0b002ae4933efebdced686a61f15282bdfc226bb932ced7d5346f296cf8d2c89f38c2adbc59",
    "14d42ce1d735d05d233dccb89532ee7fdbb10acb45d97f2010c04122677b21375a9ddd9dff63010306414d2ecf8c3fb007df86898b2bb55b61c64f19ebffe140",
};

int selected_device = 0;

int cuda_error(const char* operation, cudaError_t err) {
    if (err == cudaSuccess) return 0;
    std::fprintf(stderr, "%s: %s\n", operation, cudaGetErrorString(err));
    return 1;
}

int hex_nibble(char c) {
    if (c >= '0' && c <= '9') return c - '0';
    if (c >= 'a' && c <= 'f') return c - 'a' + 10;
    if (c >= 'A' && c <= 'F') return c - 'A' + 10;
    return -1;
}

template <std::size_t N>
bool decode_fixed_hex(const char* text, std::array<std::uint8_t, N>* output) {
    if (std::strlen(text) != N * 2u) return false;
    for (std::size_t i = 0; i < N; ++i) {
        const int hi = hex_nibble(text[i * 2u]);
        const int lo = hex_nibble(text[i * 2u + 1u]);
        if (hi < 0 || lo < 0) return false;
        (*output)[i] = static_cast<std::uint8_t>((hi << 4) | lo);
    }
    return true;
}

bool decode_header_hex(const char* text, SearchJob* job) {
    const std::size_t length = std::strlen(text);
    if (length == 0u || (length & 1u) != 0u || length / 2u > job->header_prefix.size()) return false;
    job->header_length = static_cast<std::uint32_t>(length / 2u);
    for (std::size_t i = 0; i < job->header_length; ++i) {
        const int hi = hex_nibble(text[i * 2u]);
        const int lo = hex_nibble(text[i * 2u + 1u]);
        if (hi < 0 || lo < 0) return false;
        job->header_prefix[i] = static_cast<std::uint8_t>((hi << 4) | lo);
    }
    return true;
}

SearchCache genesis_vector_cache() {
    SearchCache cache{};
    for (std::size_t node = 0; node < cache.size(); ++node) {
        decode_fixed_hex(kGenesisCacheHex[node], &cache[node]);
    }
    return cache;
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
    constexpr char header[] = "khushi-algorithm-generic-gpu-benchmark";
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

bool checked_add(std::size_t a, std::size_t b, std::size_t* out) {
    if (b > std::numeric_limits<std::size_t>::max() - a) return false;
    *out = a + b;
    return true;
}

std::size_t required_vram_bytes(std::size_t epoch_bytes, std::size_t runtime_bytes) {
    constexpr std::size_t runtime_reserve = 64u * 1024u * 1024u;
    std::size_t required = 0u;
    if (!checked_add(epoch_bytes, runtime_bytes, &required)) return std::numeric_limits<std::size_t>::max();
    if (!checked_add(required, runtime_reserve, &required)) return std::numeric_limits<std::size_t>::max();
    return required;
}

int select_device(int device) {
    int count = 0;
    if (cuda_error("cudaGetDeviceCount", cudaGetDeviceCount(&count)) != 0) return 1;
    if (count == 0) {
        std::fputs("Khushi Algorithm: no CUDA device found\n", stderr);
        return 2;
    }
    if (device < 0 || device >= count) {
        std::fprintf(stderr, "invalid CUDA device index %d (devices=%d)\n", device, count);
        return 2;
    }
    selected_device = device;
    return cuda_error("cudaSetDevice", cudaSetDevice(selected_device));
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
        const auto profile = tuning::cuda_profile(prop.major, prop.minor);
        std::printf("device=%d name=%s capability=%d.%d family=%s vram_bytes=%llu multiprocessors=%d max_threads_per_block=%d\n",
                    device,
                    prop.name,
                    prop.major,
                    prop.minor,
                    tuning::family_name(profile.family),
                    static_cast<unsigned long long>(prop.totalGlobalMem),
                    prop.multiProcessorCount,
                    prop.maxThreadsPerBlock);
    }
    return 0;
}

int memory_preflight(std::size_t epoch_bytes, std::size_t runtime_bytes) {
    std::size_t available_vram_bytes = 0u;
    std::size_t total_vram_bytes = 0u;
    if (cuda_error("cudaMemGetInfo", cudaMemGetInfo(&available_vram_bytes, &total_vram_bytes)) != 0) return 1;
    const std::size_t needed = required_vram_bytes(epoch_bytes, runtime_bytes);
    std::printf("selected_device=%d required_vram_bytes=%llu available_vram_bytes=%llu total_vram_bytes=%llu\n",
                selected_device,
                static_cast<unsigned long long>(needed),
                static_cast<unsigned long long>(available_vram_bytes),
                static_cast<unsigned long long>(total_vram_bytes));
    if (needed == std::numeric_limits<std::size_t>::max() || available_vram_bytes < needed) {
        std::fprintf(stderr,
                     "insufficient GPU memory: required_vram_bytes=%llu available_vram_bytes=%llu\n",
                     static_cast<unsigned long long>(needed),
                     static_cast<unsigned long long>(available_vram_bytes));
        return 2;
    }
    return 0;
}

int production_memory_preflight() {
    std::size_t available_vram_bytes = 0u;
    std::size_t total_vram_bytes = 0u;
    if (cuda_error("cudaMemGetInfo(production)", cudaMemGetInfo(&available_vram_bytes, &total_vram_bytes)) != 0) return 1;

    std::printf(
        "production-memory-policy dataset_bytes=%llu cache_bytes=%llu runtime_reserve_bytes=%llu required_vram_bytes=%llu minimum_dedicated_vram_bytes=%llu available_vram_bytes=%llu total_vram_bytes=%llu\n",
        static_cast<unsigned long long>(kProductionDatasetBytes),
        static_cast<unsigned long long>(kProductionCacheBytes),
        static_cast<unsigned long long>(kProductionRuntimeReserveBytes),
        static_cast<unsigned long long>(kProductionRequiredVRAMBytes),
        static_cast<unsigned long long>(kMinimumDedicatedVRAMBytes),
        static_cast<unsigned long long>(available_vram_bytes),
        static_cast<unsigned long long>(total_vram_bytes));

    if (total_vram_bytes < kMinimumDedicatedVRAMBytes ||
        available_vram_bytes < kProductionRequiredVRAMBytes) {
        std::fprintf(
            stderr,
            "insufficient production GPU memory: minimum_dedicated_vram_bytes=%llu required_vram_bytes=%llu available_vram_bytes=%llu total_vram_bytes=%llu\n",
            static_cast<unsigned long long>(kMinimumDedicatedVRAMBytes),
            static_cast<unsigned long long>(kProductionRequiredVRAMBytes),
            static_cast<unsigned long long>(available_vram_bytes),
            static_cast<unsigned long long>(total_vram_bytes));
        return 2;
    }
    return 0;
}

int run_production_memory_self_test() {
    if (select_device(selected_device) != 0) return 2;
    if (production_memory_preflight() != 0) return 2;

    void* device_cache = nullptr;
    if (cuda_error("cudaMalloc(production cache)", cudaMalloc(&device_cache, static_cast<std::size_t>(kProductionCacheBytes))) != 0) return 1;

    std::array<void*, kProductionChunkCount> dataset_chunks{};
    auto release = [](void* value) { if (value != nullptr) cudaFree(value); };
    auto allocate = [](void** output, std::size_t bytes) {
        return cuda_error("cudaMalloc(production dataset chunk)", cudaMalloc(output, bytes)) == 0;
    };

    if (!allocate_dataset_chunks(&dataset_chunks, allocate, release)) {
        cudaFree(device_cache);
        std::fputs("production-memory-self-test=failed dataset allocation\n", stderr);
        return 3;
    }

    std::size_t remaining_vram_bytes = 0u;
    std::size_t total_vram_bytes = 0u;
    if (cuda_error("cudaMemGetInfo(production allocated)", cudaMemGetInfo(&remaining_vram_bytes, &total_vram_bytes)) != 0) {
        release_dataset_chunks(&dataset_chunks, release);
        cudaFree(device_cache);
        return 1;
    }
    if (remaining_vram_bytes < kProductionRuntimeReserveBytes) {
        std::fprintf(stderr,
            "production-memory-self-test=failed runtime reserve remaining_vram_bytes=%llu required_reserve_bytes=%llu\n",
            static_cast<unsigned long long>(remaining_vram_bytes),
            static_cast<unsigned long long>(kProductionRuntimeReserveBytes));
        release_dataset_chunks(&dataset_chunks, release);
        cudaFree(device_cache);
        return 3;
    }

    release_dataset_chunks(&dataset_chunks, release);
    cudaFree(device_cache);
    std::puts("production-memory-self-test=ok");
    return 0;
}

int run_telemetry() {
    char command[512] = {};
    std::snprintf(command, sizeof(command),
                  "nvidia-smi -i %d --query-gpu=name,driver_version,temperature.gpu,power.draw,utilization.gpu,memory.used --format=csv,noheader,nounits",
                  selected_device);
    std::puts("telemetry-columns=name,driver_version,temperature.gpu,power.draw,utilization.gpu,memory.used");
    const int result = std::system(command);
    if (result != 0) {
        std::fputs("telemetry=unavailable (nvidia-smi failed)\n", stderr);
        return 5;
    }
    return 0;
}

int run_vector_self_test() {
    if (select_device(selected_device) != 0) return 2;
    constexpr std::size_t runtime_bytes = 32u;
    if (memory_preflight(sizeof(SearchCache), runtime_bytes) != 0) return 2;

    SearchCache host_cache = genesis_vector_cache();
    SearchJob job{};
    if (!decode_fixed_hex(kProgramZeroSeedHex, &job.program_seed)) {
        std::fputs("vector-self-test=failed invalid program seed fixture\n", stderr);
        return 1;
    }
    std::array<std::uint8_t, 32> expected{};
    if (!decode_fixed_hex(kGenesisExpectedDigestHex, &expected)) {
        std::fputs("vector-self-test=failed invalid digest fixture\n", stderr);
        return 1;
    }

    SearchCache* device_cache = nullptr;
    std::uint8_t* device_digest = nullptr;
    if (cuda_error("cudaMalloc(vector cache)", cudaMalloc(reinterpret_cast<void**>(&device_cache), sizeof(SearchCache))) != 0) return 1;
    if (cuda_error("cudaMalloc(vector digest)", cudaMalloc(reinterpret_cast<void**>(&device_digest), expected.size())) != 0) return 1;
    if (cuda_error("cudaMemcpy(vector cache)", cudaMemcpy(device_cache, &host_cache, sizeof(host_cache), cudaMemcpyHostToDevice)) != 0) return 1;

    sudharma::gpupowv1::khushi_vector_kernel<<<1u, 1u>>>(job, device_cache, 0u, device_digest);
    if (cuda_error("khushi_vector_kernel launch", cudaGetLastError()) != 0) return 1;
    if (cuda_error("khushi_vector_kernel sync", cudaDeviceSynchronize()) != 0) return 1;

    std::array<std::uint8_t, 32> got{};
    if (cuda_error("cudaMemcpy(vector digest host)", cudaMemcpy(got.data(), device_digest, got.size(), cudaMemcpyDeviceToHost)) != 0) return 1;
    cudaFree(device_digest);
    cudaFree(device_cache);

    std::fputs("vector-digest=", stdout);
    for (std::uint8_t value : got) std::printf("%02x", value);
    std::fputc('\n', stdout);
    if (got != expected) {
        std::fprintf(stderr, "vector-self-test=failed expected=%s\n", kGenesisExpectedDigestHex);
        return 4;
    }
    std::puts("vector-self-test=ok");
    return 0;
}

int run_benchmark(unsigned seconds) {
    if (seconds == 0u) seconds = 10u;
    if (select_device(selected_device) != 0) return 2;
    constexpr std::size_t runtime_bytes = sizeof(std::uint32_t) + 2u * sizeof(unsigned long long);
    if (memory_preflight(sizeof(SearchCache), runtime_bytes) != 0) return 2;

    cudaDeviceProp prop{};
    if (cuda_error("cudaGetDeviceProperties(benchmark)", cudaGetDeviceProperties(&prop, selected_device)) != 0) return 1;
    cudaFuncAttributes kernel_attributes{};
    if (cuda_error(
            "cudaFuncGetAttributes(khushi_search_kernel)",
            cudaFuncGetAttributes(&kernel_attributes, sudharma::gpupowv1::khushi_search_kernel)) != 0) {
        return 1;
    }
    const unsigned device_max_threads =
        prop.maxThreadsPerBlock > 0 ? static_cast<unsigned>(prop.maxThreadsPerBlock) : 1u;
    const unsigned kernel_max_threads =
        kernel_attributes.maxThreadsPerBlock > 0 ? static_cast<unsigned>(kernel_attributes.maxThreadsPerBlock) : 1u;
    const unsigned launch_max_threads =
        device_max_threads < kernel_max_threads ? device_max_threads : kernel_max_threads;
    const tuning::Profile profile = tuning::cuda_profile(prop.major, prop.minor);
    const auto launch_candidates = tuning::candidates(profile, launch_max_threads);

    std::printf(
        "autotune-limits backend=cuda device=%d family=%s device_max_threads=%u kernel_max_threads=%u launch_max_threads=%u\n",
        selected_device,
        tuning::family_name(profile.family),
        device_max_threads,
        kernel_max_threads,
        launch_max_threads);

    SearchCache host_cache = benchmark_cache();
    SearchCache* device_cache = nullptr;
    std::uint32_t* stale_generation = nullptr;
    unsigned long long* found_nonce = nullptr;
    unsigned long long* hashes_done = nullptr;

    if (cuda_error("cudaMalloc(cache)", cudaMalloc(reinterpret_cast<void**>(&device_cache), sizeof(SearchCache))) != 0) return 1;
    if (cuda_error("cudaMalloc(stale_generation)", cudaMalloc(reinterpret_cast<void**>(&stale_generation), sizeof(std::uint32_t))) != 0) return 1;
    if (cuda_error("cudaMalloc(found_nonce)", cudaMalloc(reinterpret_cast<void**>(&found_nonce), sizeof(unsigned long long))) != 0) return 1;
    if (cuda_error("cudaMalloc(hashes_done)", cudaMalloc(reinterpret_cast<void**>(&hashes_done), sizeof(unsigned long long))) != 0) return 1;

    const std::uint32_t generation = 1u;
    const unsigned long long no_nonce = kSearchNoNonce;
    if (cuda_error("cudaMemcpy(cache)", cudaMemcpy(device_cache, &host_cache, sizeof(host_cache), cudaMemcpyHostToDevice)) != 0) return 1;
    if (cuda_error("cudaMemcpy(generation)", cudaMemcpy(stale_generation, &generation, sizeof(generation), cudaMemcpyHostToDevice)) != 0) return 1;

    const unsigned long long total_milliseconds = static_cast<unsigned long long>(seconds) * 1000ull;
    const unsigned long long candidate_milliseconds =
        launch_candidates.empty() ? total_milliseconds :
        (total_milliseconds / static_cast<unsigned long long>(launch_candidates.size()) < 250ull
             ? 250ull
             : total_milliseconds / static_cast<unsigned long long>(launch_candidates.size()));

    double best_rate = -1.0;
    unsigned best_local_size = 0u;
    std::size_t best_work_items = 0u;
    unsigned long long best_hashes = 0ull;
    double best_elapsed = 0.0;

    for (const tuning::Candidate candidate : launch_candidates) {
        unsigned long long zero = 0ull;
        if (cuda_error("cudaMemcpy(benchmark nonce reset)", cudaMemcpy(found_nonce, &no_nonce, sizeof(no_nonce), cudaMemcpyHostToDevice)) != 0) return 1;
        if (cuda_error("cudaMemcpy(benchmark hashes reset)", cudaMemcpy(hashes_done, &zero, sizeof(zero), cudaMemcpyHostToDevice)) != 0) return 1;

        const unsigned threads = candidate.local_size;
        const std::size_t work_items = tuning::work_items(
            candidate,
            prop.multiProcessorCount > 0 ? static_cast<unsigned>(prop.multiProcessorCount) : 1u);
        const unsigned blocks = static_cast<unsigned>((work_items + threads - 1u) / threads);
        const std::uint64_t nonces_per_launch = static_cast<std::uint64_t>(work_items);
        std::uint64_t nonce_start = 0u;
        const auto started = std::chrono::steady_clock::now();
        const auto deadline = started + std::chrono::milliseconds(candidate_milliseconds);

        do {
            SearchJob job = benchmark_job(nonce_start, nonces_per_launch);
            sudharma::gpupowv1::khushi_search_kernel<<<blocks, threads>>>(
                job, device_cache, stale_generation, generation, found_nonce, hashes_done);
            if (cuda_error("khushi_search_kernel autotune launch", cudaGetLastError()) != 0) return 1;
            if (cuda_error("khushi_search_kernel autotune sync", cudaDeviceSynchronize()) != 0) return 1;
            nonce_start += nonces_per_launch;
        } while (std::chrono::steady_clock::now() < deadline);

        unsigned long long hashes = 0ull;
        if (cuda_error("cudaMemcpy(benchmark hashes host)", cudaMemcpy(&hashes, hashes_done, sizeof(hashes), cudaMemcpyDeviceToHost)) != 0) return 1;
        const auto ended = std::chrono::steady_clock::now();
        const double elapsed = std::chrono::duration<double>(ended - started).count();
        const double rate = elapsed > 0.0 ? static_cast<double>(hashes) / elapsed : 0.0;
        std::printf(
            "autotune-candidate backend=cuda device=%d family=%s local_size=%u blocks=%u work_items=%llu seconds=%.3f hashes=%llu hashrate_hps=%.6f\n",
            selected_device,
            tuning::family_name(profile.family),
            threads,
            blocks,
            static_cast<unsigned long long>(work_items),
            elapsed,
            hashes,
            rate);

        if (rate > best_rate) {
            best_rate = rate;
            best_local_size = threads;
            best_work_items = work_items;
            best_hashes = hashes;
            best_elapsed = elapsed;
        }
    }

    if (best_local_size == 0u) {
        std::fputs("Khushi Algorithm CUDA autotune produced no safe launch candidate\n", stderr);
        return 6;
    }
    std::printf(
        "autotune-selected backend=cuda device=%d family=%s local_size=%u work_items=%llu seconds=%.3f hashes=%llu hashrate_hps=%.6f\n",
        selected_device,
        tuning::family_name(profile.family),
        best_local_size,
        static_cast<unsigned long long>(best_work_items),
        best_elapsed,
        best_hashes,
        best_rate);
    std::printf("Khushi Algorithm benchmark backend=cuda device=%d seconds=%.3f hashes=%llu hashrate_hps=%.6f\n",
                selected_device, best_elapsed, best_hashes, best_rate);

    cudaFree(hashes_done);
    cudaFree(found_nonce);
    cudaFree(stale_generation);
    cudaFree(device_cache);
    return 0;
}

int run_staging_search(const char* header_hex, const char* target_hex, std::uint64_t height, std::uint32_t cache_nodes) {
    if (height != 0u || cache_nodes != 8u) {
        std::fputs("staging search only supports height=0 cache_nodes=8\n", stderr);
        return 64;
    }
    if (select_device(selected_device) != 0) return 2;
    constexpr std::size_t runtime_bytes = sizeof(std::uint32_t) + 2u * sizeof(unsigned long long);
    if (memory_preflight(sizeof(SearchCache), runtime_bytes) != 0) return 2;

    SearchJob job{};
    if (!decode_header_hex(header_hex, &job)) {
        std::fputs("invalid --header-prefix-hex value\n", stderr);
        return 64;
    }
    if (!decode_fixed_hex(target_hex, &job.target)) {
        std::fputs("invalid --target-hex value\n", stderr);
        return 64;
    }
    if (!decode_fixed_hex(kProgramZeroSeedHex, &job.program_seed)) {
        std::fputs("invalid staging program seed fixture\n", stderr);
        return 1;
    }

    SearchCache host_cache = genesis_vector_cache();
    SearchCache* device_cache = nullptr;
    std::uint32_t* stale_generation = nullptr;
    unsigned long long* found_nonce = nullptr;
    unsigned long long* hashes_done = nullptr;

    if (cuda_error("cudaMalloc(staging cache)", cudaMalloc(reinterpret_cast<void**>(&device_cache), sizeof(SearchCache))) != 0) return 1;
    if (cuda_error("cudaMalloc(staging generation)", cudaMalloc(reinterpret_cast<void**>(&stale_generation), sizeof(std::uint32_t))) != 0) return 1;
    if (cuda_error("cudaMalloc(staging nonce)", cudaMalloc(reinterpret_cast<void**>(&found_nonce), sizeof(unsigned long long))) != 0) return 1;
    if (cuda_error("cudaMalloc(staging hashes)", cudaMalloc(reinterpret_cast<void**>(&hashes_done), sizeof(unsigned long long))) != 0) return 1;

    const std::uint32_t generation = 1u;
    const unsigned long long no_nonce = kSearchNoNonce;
    unsigned long long zero = 0ull;
    if (cuda_error("cudaMemcpy(staging cache)", cudaMemcpy(device_cache, &host_cache, sizeof(host_cache), cudaMemcpyHostToDevice)) != 0) return 1;
    if (cuda_error("cudaMemcpy(staging generation)", cudaMemcpy(stale_generation, &generation, sizeof(generation), cudaMemcpyHostToDevice)) != 0) return 1;
    if (cuda_error("cudaMemcpy(staging nonce)", cudaMemcpy(found_nonce, &no_nonce, sizeof(no_nonce), cudaMemcpyHostToDevice)) != 0) return 1;
    if (cuda_error("cudaMemcpy(staging hashes)", cudaMemcpy(hashes_done, &zero, sizeof(zero), cudaMemcpyHostToDevice)) != 0) return 1;

    constexpr unsigned threads = 32u;
    constexpr std::uint64_t nonces_per_launch = 32u;
    constexpr std::uint64_t max_nonces = 65536u;
    unsigned long long host_nonce = no_nonce;
    for (std::uint64_t nonce_start = 0u; nonce_start < max_nonces && host_nonce == no_nonce; nonce_start += nonces_per_launch) {
        job.nonce_start = nonce_start;
        job.nonce_count = nonces_per_launch;
        sudharma::gpupowv1::khushi_search_kernel<<<1u, threads>>>(
            job, device_cache, stale_generation, generation, found_nonce, hashes_done);
        if (cuda_error("khushi staging search launch", cudaGetLastError()) != 0) return 1;
        if (cuda_error("khushi staging search sync", cudaDeviceSynchronize()) != 0) return 1;
        if (cuda_error("cudaMemcpy(staging nonce host)", cudaMemcpy(&host_nonce, found_nonce, sizeof(host_nonce), cudaMemcpyDeviceToHost)) != 0) return 1;
    }

    unsigned long long hashes = 0ull;
    if (cuda_error("cudaMemcpy(staging hashes host)", cudaMemcpy(&hashes, hashes_done, sizeof(hashes), cudaMemcpyDeviceToHost)) != 0) return 1;
    cudaFree(hashes_done);
    cudaFree(found_nonce);
    cudaFree(stale_generation);
    cudaFree(device_cache);

    if (host_nonce == no_nonce) {
        std::printf("staging-search=not-found hashes=%llu\n", hashes);
        return 4;
    }
    std::printf("staging-solution-nonce=%llu hashes=%llu\n", host_nonce, hashes);
    return 0;
}

bool parse_device(const char* text, int* device) {
    char* end = nullptr;
    const long value = std::strtol(text, &end, 10);
    if (text == end || *end != '\0' || value < 0 || value > std::numeric_limits<int>::max()) return false;
    *device = static_cast<int>(value);
    return true;
}

bool parse_u64(const char* text, std::uint64_t* value) {
    char* end = nullptr;
    const unsigned long long parsed = std::strtoull(text, &end, 10);
    if (text == end || *end != '\0') return false;
    *value = static_cast<std::uint64_t>(parsed);
    return true;
}

bool parse_u32(const char* text, std::uint32_t* value) {
    std::uint64_t parsed = 0u;
    if (!parse_u64(text, &parsed) || parsed > std::numeric_limits<std::uint32_t>::max()) return false;
    *value = static_cast<std::uint32_t>(parsed);
    return true;
}

void usage() {
    std::fputs(
        "usage: khushi-miner [--device N] --list-devices | --device-info | --telemetry | --vector-self-test | --production-memory-self-test | --benchmark [seconds] | --staging-search --header-prefix-hex HEX --target-hex HEX --height N --cache-nodes N | --mine\n",
        stderr);
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
    if (arg >= argc) {
        usage();
        return 64;
    }

    if ((std::strcmp(argv[arg], "--list-devices") == 0 || std::strcmp(argv[arg], "--device-info") == 0) && arg + 1 == argc) {
        return print_device_info();
    }
    if (std::strcmp(argv[arg], "--telemetry") == 0 && arg + 1 == argc) {
        if (select_device(selected_device) != 0) return 2;
        return run_telemetry();
    }
    if (std::strcmp(argv[arg], "--vector-self-test") == 0 && arg + 1 == argc) {
        return run_vector_self_test();
    }
    if (std::strcmp(argv[arg], "--production-memory-self-test") == 0 && arg + 1 == argc) {
        return run_production_memory_self_test();
    }
    if (std::strcmp(argv[arg], "--benchmark") == 0 && (arg + 1 == argc || arg + 2 == argc)) {
        std::uint32_t seconds = 10u;
        if (arg + 2 == argc && !parse_u32(argv[arg + 1], &seconds)) {
            std::fputs("invalid --benchmark seconds\n", stderr);
            return 64;
        }
        return run_benchmark(seconds);
    }
    if (std::strcmp(argv[arg], "--staging-search") == 0 && arg + 9 == argc &&
        std::strcmp(argv[arg + 1], "--header-prefix-hex") == 0 &&
        std::strcmp(argv[arg + 3], "--target-hex") == 0 &&
        std::strcmp(argv[arg + 5], "--height") == 0 &&
        std::strcmp(argv[arg + 7], "--cache-nodes") == 0) {
        std::uint64_t height = 0u;
        std::uint32_t cache_nodes = 0u;
        if (!parse_u64(argv[arg + 6], &height) || !parse_u32(argv[arg + 8], &cache_nodes)) {
            std::fputs("invalid staging height or cache node count\n", stderr);
            return 64;
        }
        return run_staging_search(argv[arg + 2], argv[arg + 4], height, cache_nodes);
    }
    if (std::strcmp(argv[arg], "--mine") == 0) {
        std::fputs("Khushi Algorithm network mining is gated until hardware interoperability passes; CPU fallback prohibited\n", stderr);
        return 3;
    }

    usage();
    return 64;
}
