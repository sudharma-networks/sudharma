#include <cuda_runtime.h>

#include <array>
#include <chrono>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <cstring>

#include "gpupow_v1_search.cuh"

namespace {

using sudharma::gpupowv1::SearchCache;
using sudharma::gpupowv1::SearchJob;
using sudharma::gpupowv1::kSearchNoNonce;

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

int run_telemetry() {
    std::puts("telemetry-columns=name,driver_version,temperature.gpu,power.draw,utilization.gpu,memory.used");
    const int result = std::system(
        "nvidia-smi --query-gpu=name,driver_version,temperature.gpu,power.draw,utilization.gpu,memory.used --format=csv,noheader,nounits");
    if (result != 0) {
        std::fputs("telemetry=unavailable (nvidia-smi failed)\n", stderr);
        return 5;
    }
    return 0;
}

int run_vector_self_test() {
    if (print_device_info() != 0) return 2;

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
    if (print_device_info() != 0) return 2;

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
    if (argc == 2 && std::strcmp(argv[1], "--telemetry") == 0) {
        return run_telemetry();
    }
    if (argc == 2 && std::strcmp(argv[1], "--vector-self-test") == 0) {
        return run_vector_self_test();
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

    std::fputs("usage: khushi-miner --device-info | --telemetry | --vector-self-test | --benchmark [seconds] | --mine\n", stderr);
    return 64;
}
