#define CL_TARGET_OPENCL_VERSION 120
#include <CL/cl.h>

#include <array>
#include <cstddef>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <fstream>
#include <sstream>
#include <string>
#include <vector>

#include "../cuda/gpupow_v1_dataset.h"

namespace {

using CacheNode = std::array<std::uint8_t, 64>;
using sudharma::gpupowv1::keccak512_64;
using sudharma::gpupowv1::keccak_f1600;
using sudharma::gpupowv1::word64;

constexpr std::size_t kProductionCacheBytes = 16u * 1024u * 1024u;
constexpr std::size_t kProductionItemBytes = 64u;
constexpr std::uint32_t kProductionCacheNodes =
    static_cast<std::uint32_t>(kProductionCacheBytes / kProductionItemBytes);
constexpr cl_ulong kMinimumDedicatedVRAMBytes = 4ull << 30u;
static_assert(kProductionCacheNodes == 262144u, "production cache node count changed");
static_assert(sizeof(CacheNode) == 64u, "cache node layout changed");

constexpr std::array<std::uint8_t, 32> kEpoch0Seed = {
    0xa2, 0x0a, 0xc9, 0x1b, 0x09, 0x2f, 0xf0, 0xf1,
    0xac, 0x89, 0xfe, 0xdd, 0xcf, 0x16, 0xca, 0x2f,
    0x47, 0xb0, 0x30, 0xeb, 0x84, 0x1d, 0x75, 0xbc,
    0xc0, 0x26, 0xcd, 0xe7, 0x51, 0xf9, 0xed, 0x7a,
};

constexpr std::array<cl_uint, 4> kBoundaryIndices = {
    0u, 4194303u, 4194304u, 33554431u,
};

constexpr const char* kExpectedDigests[4] = {
    "eb2d4ea4f291aa352d7e6788aa56703a3b1fe7d47564dc80d109d9f8de75e4de5b875e27d6aae37523753a88f00e337f15983e38819c630e2128d04f0548ad8b",
    "9b906e7a49f6b4de879f935d7fb68815481f9091540f083d81f54dcce44543bf95ceb282b3877aaaeb530e756752e79c7295282b008c54e8c05735daef437500",
    "5ff3da521e9a3ad3b2722b8dbc4a7d0d95aab31abba2bfee881ac1552912f754b0819dbd047b9b6b1b05b897968aed49f7002daaaccbd877e83c9812951d5a3e",
    "baade0e152b5a0741b4c865e5d547a4a10887abf3edbc9a93dc80361d205368aa3f7914076210afd2bd233966f32e264efb498719c5afd947c4a5226d7fd83a9",
};

struct DeviceRef {
    cl_platform_id platform{};
    cl_device_id device{};
    std::string name;
    cl_ulong global_memory{};
};

void report_cl_error(const char* operation, cl_int code) {
    std::fprintf(stderr, "%s failed: OpenCL error %d\n", operation, code);
}

std::vector<DeviceRef> gpu_devices() {
    cl_uint platform_count = 0;
    cl_int rc = clGetPlatformIDs(0, nullptr, &platform_count);
    if (rc != CL_SUCCESS || platform_count == 0u) return {};

    std::vector<cl_platform_id> platforms(platform_count);
    if (clGetPlatformIDs(platform_count, platforms.data(), nullptr) != CL_SUCCESS) return {};

    std::vector<DeviceRef> output;
    for (cl_platform_id platform : platforms) {
        cl_uint device_count = 0;
        rc = clGetDeviceIDs(platform, CL_DEVICE_TYPE_GPU, 0, nullptr, &device_count);
        if (rc == CL_DEVICE_NOT_FOUND) continue;
        if (rc != CL_SUCCESS || device_count == 0u) continue;

        std::vector<cl_device_id> devices(device_count);
        if (clGetDeviceIDs(platform, CL_DEVICE_TYPE_GPU, device_count, devices.data(), nullptr) != CL_SUCCESS) {
            continue;
        }
        for (cl_device_id device : devices) {
            char name[256] = {};
            cl_ulong memory = 0;
            if (clGetDeviceInfo(device, CL_DEVICE_NAME, sizeof(name), name, nullptr) != CL_SUCCESS) continue;
            if (clGetDeviceInfo(device, CL_DEVICE_GLOBAL_MEM_SIZE, sizeof(memory), &memory, nullptr) != CL_SUCCESS) continue;
            output.push_back(DeviceRef{platform, device, name, memory});
        }
    }
    return output;
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

std::string read_text_file(const char* packaged, const char* repository) {
    for (const char* path : {packaged, repository}) {
        std::ifstream input(path, std::ios::binary);
        if (!input) continue;
        std::ostringstream text;
        text << input.rdbuf();
        return text.str();
    }
    return {};
}

bool parse_device(const char* text, int* output) {
    char* end = nullptr;
    const long value = std::strtol(text, &end, 10);
    if (text == end || *end != '\0' || value < 0 || value > 0x7fffffffL) return false;
    *output = static_cast<int>(value);
    return true;
}

int run(int selected_device) {
    const auto devices = gpu_devices();
    if (devices.empty()) {
        std::fputs("Khushi Algorithm requires an OpenCL GPU; CPU fallback prohibited\n", stderr);
        return 2;
    }
    if (selected_device < 0 || static_cast<std::size_t>(selected_device) >= devices.size()) {
        std::fprintf(stderr, "invalid OpenCL device=%d devices=%zu\n", selected_device, devices.size());
        return 2;
    }

    const DeviceRef& chosen = devices[static_cast<std::size_t>(selected_device)];
    std::printf(
        "production-vector-device=%d name=%s backend=opencl total_vram_bytes=%llu cache_bytes=%llu cache_nodes=%u\n",
        selected_device,
        chosen.name.c_str(),
        static_cast<unsigned long long>(chosen.global_memory),
        static_cast<unsigned long long>(kProductionCacheBytes),
        kProductionCacheNodes);
    if (chosen.global_memory < kMinimumDedicatedVRAMBytes) {
        std::fputs("production-vector-self-test=failed dedicated VRAM below 4 GiB target\n", stderr);
        return 2;
    }

    std::puts("production-vector-cache-build=started epoch=0 nodes=262144");
    const auto host_cache = build_epoch0_production_cache();
    std::puts("production-vector-cache-build=ok");

    const std::string base_kernel = read_text_file(
        "khushi_pow.cl", "compatibility/opencl/khushi_pow.cl");
    const std::string vector_kernel = read_text_file(
        "gpupow_v1_production_vectors.cl",
        "compatibility/opencl/gpupow_v1_production_vectors.cl");
    if (base_kernel.empty() || vector_kernel.empty()) {
        std::fputs("cannot locate OpenCL production vector kernels\n", stderr);
        return 1;
    }
    const std::string source = base_kernel + "\n" + vector_kernel;

    cl_int rc = CL_SUCCESS;
    cl_context context = clCreateContext(nullptr, 1, &chosen.device, nullptr, nullptr, &rc);
    if (rc != CL_SUCCESS || context == nullptr) {
        report_cl_error("clCreateContext", rc);
        return 1;
    }
    cl_command_queue queue = clCreateCommandQueue(context, chosen.device, 0, &rc);
    if (rc != CL_SUCCESS || queue == nullptr) {
        report_cl_error("clCreateCommandQueue", rc);
        clReleaseContext(context);
        return 1;
    }

    const char* source_ptr = source.c_str();
    const std::size_t source_size = source.size();
    cl_program program = clCreateProgramWithSource(context, 1, &source_ptr, &source_size, &rc);
    if (rc != CL_SUCCESS || program == nullptr) {
        report_cl_error("clCreateProgramWithSource", rc);
        clReleaseCommandQueue(queue);
        clReleaseContext(context);
        return 1;
    }
    rc = clBuildProgram(program, 1, &chosen.device, "-cl-std=CL1.2", nullptr, nullptr);
    if (rc != CL_SUCCESS) {
        std::size_t log_size = 0;
        clGetProgramBuildInfo(program, chosen.device, CL_PROGRAM_BUILD_LOG, 0, nullptr, &log_size);
        std::vector<char> log(log_size + 1u, '\0');
        if (log_size != 0u) {
            clGetProgramBuildInfo(program, chosen.device, CL_PROGRAM_BUILD_LOG,
                                  log_size, log.data(), nullptr);
        }
        std::fprintf(stderr, "OpenCL build failed:\n%s\n", log.data());
        clReleaseProgram(program);
        clReleaseCommandQueue(queue);
        clReleaseContext(context);
        return 1;
    }

    cl_kernel kernel = clCreateKernel(program, "khushi_production_vectors", &rc);
    if (rc != CL_SUCCESS || kernel == nullptr) {
        report_cl_error("clCreateKernel(khushi_production_vectors)", rc);
        clReleaseProgram(program);
        clReleaseCommandQueue(queue);
        clReleaseContext(context);
        return 1;
    }

    cl_mem cache = nullptr;
    cl_mem indices = nullptr;
    cl_mem output = nullptr;
    auto cleanup = [&]() {
        if (output) clReleaseMemObject(output);
        if (indices) clReleaseMemObject(indices);
        if (cache) clReleaseMemObject(cache);
        clReleaseKernel(kernel);
        clReleaseProgram(program);
        clReleaseCommandQueue(queue);
        clReleaseContext(context);
    };

    cache = clCreateBuffer(context, CL_MEM_READ_ONLY,
                           kProductionCacheBytes, nullptr, &rc);
    if (rc != CL_SUCCESS || cache == nullptr) {
        report_cl_error("clCreateBuffer(cache)", rc);
        cleanup();
        return 1;
    }
    indices = clCreateBuffer(context, CL_MEM_READ_ONLY,
                             sizeof(kBoundaryIndices), nullptr, &rc);
    if (rc != CL_SUCCESS || indices == nullptr) {
        report_cl_error("clCreateBuffer(indices)", rc);
        cleanup();
        return 1;
    }
    constexpr std::size_t kOutputBytes = kBoundaryIndices.size() * sizeof(CacheNode);
    output = clCreateBuffer(context, CL_MEM_WRITE_ONLY, kOutputBytes, nullptr, &rc);
    if (rc != CL_SUCCESS || output == nullptr) {
        report_cl_error("clCreateBuffer(output)", rc);
        cleanup();
        return 1;
    }

    rc = clEnqueueWriteBuffer(queue, cache, CL_TRUE, 0, kProductionCacheBytes,
                              host_cache.data(), 0, nullptr, nullptr);
    if (rc != CL_SUCCESS) {
        report_cl_error("clEnqueueWriteBuffer(cache)", rc);
        cleanup();
        return 1;
    }
    rc = clEnqueueWriteBuffer(queue, indices, CL_TRUE, 0, sizeof(kBoundaryIndices),
                              kBoundaryIndices.data(), 0, nullptr, nullptr);
    if (rc != CL_SUCCESS) {
        report_cl_error("clEnqueueWriteBuffer(indices)", rc);
        cleanup();
        return 1;
    }

    const cl_uint cache_nodes = kProductionCacheNodes;
    const cl_uint vector_count = static_cast<cl_uint>(kBoundaryIndices.size());
    if ((rc = clSetKernelArg(kernel, 0, sizeof(cache), &cache)) != CL_SUCCESS ||
        (rc = clSetKernelArg(kernel, 1, sizeof(cache_nodes), &cache_nodes)) != CL_SUCCESS ||
        (rc = clSetKernelArg(kernel, 2, sizeof(indices), &indices)) != CL_SUCCESS ||
        (rc = clSetKernelArg(kernel, 3, sizeof(vector_count), &vector_count)) != CL_SUCCESS ||
        (rc = clSetKernelArg(kernel, 4, sizeof(output), &output)) != CL_SUCCESS) {
        report_cl_error("clSetKernelArg", rc);
        cleanup();
        return 1;
    }

    const std::size_t global_work_size = kBoundaryIndices.size();
    rc = clEnqueueNDRangeKernel(queue, kernel, 1, nullptr, &global_work_size,
                                nullptr, 0, nullptr, nullptr);
    if (rc != CL_SUCCESS) {
        report_cl_error("clEnqueueNDRangeKernel", rc);
        cleanup();
        return 1;
    }
    rc = clFinish(queue);
    if (rc != CL_SUCCESS) {
        report_cl_error("clFinish", rc);
        cleanup();
        return 1;
    }

    std::array<CacheNode, kBoundaryIndices.size()> host_output{};
    rc = clEnqueueReadBuffer(queue, output, CL_TRUE, 0, kOutputBytes,
                             host_output.data(), 0, nullptr, nullptr);
    if (rc != CL_SUCCESS) {
        report_cl_error("clEnqueueReadBuffer", rc);
        cleanup();
        return 1;
    }
    cleanup();

    for (std::size_t i = 0; i < host_output.size(); ++i) {
        const std::string digest = encode_hex(host_output[i].data(), host_output[i].size());
        const bool matches = digest == kExpectedDigests[i];
        std::printf("production-vector index=%u digest=%s status=%s backend=opencl\n",
                    static_cast<unsigned>(kBoundaryIndices[i]),
                    digest.c_str(), matches ? "ok" : "mismatch");
        if (!matches) {
            std::fprintf(stderr, "expected=%s\n", kExpectedDigests[i]);
            return 4;
        }
    }

    std::puts("production-vector-self-test=ok");
    return 0;
}

void usage() {
    std::fputs("usage: khushi-production-vectors-opencl.exe [--device N]\n", stderr);
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
