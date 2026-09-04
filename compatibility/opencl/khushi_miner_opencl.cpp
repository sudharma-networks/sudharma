#define CL_TARGET_OPENCL_VERSION 120
#include <CL/cl.h>

#include <chrono>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <fstream>
#include <limits>
#include <sstream>
#include <string>
#include <vector>

#include "../gpu/gpu_tuning_profile.h"

namespace {

namespace tuning = sudharma::gpupowv1::tuning;

constexpr const char* kExpectedDigest = "2a7c15fc6c84a67d43ff7074ac5835aa433145f89d10d1d9e36a99fe22da4b2b";
constexpr const char* kProgramSeed = "613684e3f3b42773073fb9c99e71f2933eed301d450866fe9a5a5c0530a769bd";
constexpr cl_ulong kProductionDatasetBytes = 2ull << 30u;
constexpr cl_ulong kProductionCacheBytes = 16ull << 20u;
constexpr cl_ulong kProductionRuntimeReserveBytes = 256ull << 20u;
constexpr cl_ulong kProductionChunkBytes = 256ull << 20u;
constexpr cl_uint kProductionChunkCount = 8u;
constexpr cl_ulong kMinimumDedicatedVRAMBytes = 4ull << 30u;
constexpr cl_ulong kProductionRequiredVRAMBytes =
    kProductionDatasetBytes + kProductionCacheBytes + kProductionRuntimeReserveBytes;
constexpr unsigned long long kAutotuneCandidateMilliseconds = 1000ull;
constexpr const char* kCacheHex[8] = {
    "68fae850a5cc8cddba29c7a56913c7340e69ba0d92830144aab66584e01a20e86b919e515046196e7ef9c006150fff8affc13fc252dea4490ef1bb4527adcb6b",
    "25c2c0f117e806e1a832bc4bbcc444043633f5100f9ddf3714988c1b2de377b6e9d6979803b8deca82d5267d53eccc89fa92e21984e535f4e8193881ab309741",
    "a8620b2ebeca41fbc773bb837b5e724d6eb2de570d99858df0d7d97067fb8103b21757873b735097b35d3bea8fd1c359a9e8a63c1540c76c9784cf8d975e995c",
    "605f70fb5d9b8f1553027dbac7648e70e314ca31e643521da51b78b8b25d2ab35a5842931cda4e39a20efac8290b6d16a890c5a3f867b19260f85ba788bdeebc",
    "44216b2915d6beffc4ca2c8169950f36294462ac53503471c204035586544ae1a1b7dae7aad22d7cc5501578b0b85378b118bea7a0f545d8985c9f426b05cb00",
    "874ec883bf09e6f15d3a54576fe070194925367ff5302438550ac881314e84cd7b1ea8d8cc2f45d54ddaee552996ff856ad27c240ef6b3a769c253c6d0e8cf9b",
    "d4301c92c2addfc7c7983981263e64dafb2d186e147f508ad346b0b002ae4933efebdced686a61f15282bdfc226bb932ced7d5346f296cf8d2c89f38c2adbc59",
    "14d42ce1d735d05d233dccb89532ee7fdbb10acb45d97f2010c04122677b21375a9ddd9dff63010306414d2ecf8c3fb007df86898b2bb55b61c64f19ebffe140",
};

struct DeviceRef {
    cl_platform_id platform{};
    cl_device_id device{};
    std::string name;
    std::string vendor;
    cl_ulong global_memory{};
    cl_ulong max_allocation{};
    cl_uint compute_units{};
    std::size_t max_work_group{};
};

int selected_device = 0;

void check(cl_int code, const char* what) {
    if (code == CL_SUCCESS) return;
    std::fprintf(stderr, "%s failed: OpenCL error %d\n", what, code);
    std::exit(1);
}

std::vector<DeviceRef> gpu_devices() {
    cl_uint platform_count = 0;
    check(clGetPlatformIDs(0, nullptr, &platform_count), "clGetPlatformIDs(count)");
    std::vector<cl_platform_id> platforms(platform_count);
    if (platform_count != 0) check(clGetPlatformIDs(platform_count, platforms.data(), nullptr), "clGetPlatformIDs");
    std::vector<DeviceRef> out;
    for (auto platform : platforms) {
        cl_uint count = 0;
        cl_int rc = clGetDeviceIDs(platform, CL_DEVICE_TYPE_GPU, 0, nullptr, &count);
        if (rc == CL_DEVICE_NOT_FOUND) continue;
        check(rc, "clGetDeviceIDs(count)");
        std::vector<cl_device_id> devices(count);
        check(clGetDeviceIDs(platform, CL_DEVICE_TYPE_GPU, count, devices.data(), nullptr), "clGetDeviceIDs");
        for (auto device : devices) {
            char name[256] = {};
            char vendor[256] = {};
            cl_ulong memory = 0;
            cl_ulong max_allocation = 0;
            cl_uint compute_units = 0;
            std::size_t max_work_group = 0;
            check(clGetDeviceInfo(device, CL_DEVICE_NAME, sizeof(name), name, nullptr), "CL_DEVICE_NAME");
            check(clGetDeviceInfo(device, CL_DEVICE_VENDOR, sizeof(vendor), vendor, nullptr), "CL_DEVICE_VENDOR");
            check(clGetDeviceInfo(device, CL_DEVICE_GLOBAL_MEM_SIZE, sizeof(memory), &memory, nullptr), "CL_DEVICE_GLOBAL_MEM_SIZE");
            check(clGetDeviceInfo(device, CL_DEVICE_MAX_MEM_ALLOC_SIZE, sizeof(max_allocation), &max_allocation, nullptr), "CL_DEVICE_MAX_MEM_ALLOC_SIZE");
            check(clGetDeviceInfo(device, CL_DEVICE_MAX_COMPUTE_UNITS, sizeof(compute_units), &compute_units, nullptr), "CL_DEVICE_MAX_COMPUTE_UNITS");
            check(clGetDeviceInfo(device, CL_DEVICE_MAX_WORK_GROUP_SIZE, sizeof(max_work_group), &max_work_group, nullptr), "CL_DEVICE_MAX_WORK_GROUP_SIZE");
            out.push_back(DeviceRef{platform, device, name, vendor, memory, max_allocation, compute_units, max_work_group});
        }
    }
    return out;
}

std::size_t required_vram_bytes(std::size_t epoch_bytes, std::size_t runtime_bytes) {
    constexpr std::size_t reserve = 64u * 1024u * 1024u;
    if (epoch_bytes > std::numeric_limits<std::size_t>::max() - runtime_bytes) return std::numeric_limits<std::size_t>::max();
    std::size_t value = epoch_bytes + runtime_bytes;
    if (value > std::numeric_limits<std::size_t>::max() - reserve) return std::numeric_limits<std::size_t>::max();
    return value + reserve;
}

int list_devices() {
    auto devices = gpu_devices();
    std::printf("Khushi Algorithm OpenCL GPU devices=%zu\n", devices.size());
    for (std::size_t i = 0; i < devices.size(); ++i) {
        const auto profile = tuning::opencl_profile(devices[i].vendor);
        std::printf("device=%zu name=%s vendor=%s family=%s vram_bytes=%llu max_allocation_bytes=%llu compute_units=%u max_work_group=%llu backend=opencl\n",
                    i, devices[i].name.c_str(), devices[i].vendor.c_str(), tuning::family_name(profile.family),
                    static_cast<unsigned long long>(devices[i].global_memory),
                    static_cast<unsigned long long>(devices[i].max_allocation),
                    static_cast<unsigned>(devices[i].compute_units),
                    static_cast<unsigned long long>(devices[i].max_work_group));
    }
    return devices.empty() ? 2 : 0;
}

int production_memory_self_test() {
    auto devices = gpu_devices();
    if (devices.empty()) {
        std::fputs("Khushi Algorithm requires an OpenCL GPU; CPU fallback prohibited\n", stderr);
        return 2;
    }
    if (selected_device < 0 || static_cast<std::size_t>(selected_device) >= devices.size()) {
        std::fprintf(stderr, "invalid --device %d\n", selected_device);
        return 2;
    }

    const auto& chosen = devices[static_cast<std::size_t>(selected_device)];
    std::printf(
        "production-memory-policy backend=opencl device=%d name=%s dataset_bytes=%llu cache_bytes=%llu runtime_reserve_bytes=%llu chunk_bytes=%llu chunks=%u required_vram_bytes=%llu minimum_dedicated_vram_bytes=%llu total_vram_bytes=%llu max_allocation_bytes=%llu\n",
        selected_device,
        chosen.name.c_str(),
        static_cast<unsigned long long>(kProductionDatasetBytes),
        static_cast<unsigned long long>(kProductionCacheBytes),
        static_cast<unsigned long long>(kProductionRuntimeReserveBytes),
        static_cast<unsigned long long>(kProductionChunkBytes),
        static_cast<unsigned>(kProductionChunkCount),
        static_cast<unsigned long long>(kProductionRequiredVRAMBytes),
        static_cast<unsigned long long>(kMinimumDedicatedVRAMBytes),
        static_cast<unsigned long long>(chosen.global_memory),
        static_cast<unsigned long long>(chosen.max_allocation));

    if (chosen.global_memory < kMinimumDedicatedVRAMBytes ||
        chosen.global_memory < kProductionRequiredVRAMBytes) {
        std::fputs("production-memory-self-test=failed insufficient dedicated VRAM\n", stderr);
        return 2;
    }
    if (chosen.max_allocation < kProductionChunkBytes ||
        chosen.max_allocation < kProductionRuntimeReserveBytes) {
        std::fputs("production-memory-self-test=failed OpenCL max allocation is below 256 MiB\n", stderr);
        return 2;
    }

    cl_int rc = CL_SUCCESS;
    cl_context context = clCreateContext(nullptr, 1, &chosen.device, nullptr, nullptr, &rc);
    if (rc != CL_SUCCESS || context == nullptr) {
        std::fprintf(stderr, "clCreateContext(production) failed: OpenCL error %d\n", rc);
        return 1;
    }

    cl_mem cache = nullptr;
    cl_mem reserve = nullptr;
    std::vector<cl_mem> chunks;
    chunks.reserve(kProductionChunkCount);
    auto cleanup = [&]() {
        if (reserve) clReleaseMemObject(reserve);
        for (cl_mem chunk : chunks) {
            if (chunk) clReleaseMemObject(chunk);
        }
        if (cache) clReleaseMemObject(cache);
        clReleaseContext(context);
    };

    cache = clCreateBuffer(context, CL_MEM_READ_WRITE,
                           static_cast<std::size_t>(kProductionCacheBytes), nullptr, &rc);
    if (rc != CL_SUCCESS || cache == nullptr) {
        std::fprintf(stderr, "production cache allocation failed: OpenCL error %d\n", rc);
        cleanup();
        return 3;
    }

    for (cl_uint i = 0; i < kProductionChunkCount; ++i) {
        cl_mem chunk = clCreateBuffer(context, CL_MEM_READ_WRITE,
                                      static_cast<std::size_t>(kProductionChunkBytes), nullptr, &rc);
        if (rc != CL_SUCCESS || chunk == nullptr) {
            std::fprintf(stderr, "production dataset chunk %u allocation failed: OpenCL error %d\n",
                         static_cast<unsigned>(i), rc);
            cleanup();
            return 3;
        }
        chunks.push_back(chunk);
    }

    reserve = clCreateBuffer(context, CL_MEM_READ_WRITE,
                             static_cast<std::size_t>(kProductionRuntimeReserveBytes), nullptr, &rc);
    if (rc != CL_SUCCESS || reserve == nullptr) {
        std::fprintf(stderr, "production runtime reserve allocation failed: OpenCL error %d\n", rc);
        cleanup();
        return 3;
    }

    cleanup();
    std::puts("production-memory-self-test=ok");
    return 0;
}

bool hex_bytes(const char* text, std::vector<unsigned char>* out) {
    const std::size_t n = std::strlen(text);
    if (n % 2u != 0u) return false;
    auto nibble = [](char c) -> int {
        if (c >= '0' && c <= '9') return c - '0';
        if (c >= 'a' && c <= 'f') return c - 'a' + 10;
        if (c >= 'A' && c <= 'F') return c - 'A' + 10;
        return -1;
    };
    out->clear();
    out->reserve(n / 2u);
    for (std::size_t i = 0; i < n; i += 2u) {
        int a = nibble(text[i]), b = nibble(text[i + 1u]);
        if (a < 0 || b < 0) return false;
        out->push_back(static_cast<unsigned char>((a << 4) | b));
    }
    return true;
}

bool parse_u64(const char* text, cl_ulong* out) {
    if (text == nullptr || *text == '\0') return false;
    char* end = nullptr;
    const unsigned long long value = std::strtoull(text, &end, 10);
    if (end == text || *end != '\0') return false;
    *out = static_cast<cl_ulong>(value);
    return true;
}

bool parse_u32(const char* text, cl_uint* out) {
    cl_ulong value = 0;
    if (!parse_u64(text, &value) || value > std::numeric_limits<cl_uint>::max()) return false;
    *out = static_cast<cl_uint>(value);
    return true;
}

std::string kernel_source() {
    const char* paths[] = {"compatibility/opencl/khushi_pow.cl", "khushi_pow.cl"};
    for (const char* path : paths) {
        std::ifstream in(path, std::ios::binary);
        if (!in) continue;
        std::ostringstream ss;
        ss << in.rdbuf();
        return ss.str();
    }
    std::fputs("cannot locate khushi_pow.cl\n", stderr);
    std::exit(1);
}

struct Runtime {
    cl_context context{};
    cl_command_queue queue{};
    cl_program program{};
    cl_device_id device{};
    ~Runtime() {
        if (program) clReleaseProgram(program);
        if (queue) clReleaseCommandQueue(queue);
        if (context) clReleaseContext(context);
    }
};

Runtime make_runtime() {
    auto devices = gpu_devices();
    if (devices.empty()) {
        std::fputs("Khushi Algorithm requires an OpenCL GPU; CPU fallback prohibited\n", stderr);
        std::exit(2);
    }
    if (selected_device < 0 || static_cast<std::size_t>(selected_device) >= devices.size()) {
        std::fprintf(stderr, "invalid --device %d\n", selected_device);
        std::exit(2);
    }
    const auto& chosen = devices[static_cast<std::size_t>(selected_device)];
    const std::size_t needed = required_vram_bytes(8u * 64u, 4096u);
    std::printf("selected_device=%d name=%s required_vram_bytes=%llu available_vram_bytes=%llu backend=opencl\n",
                selected_device, chosen.name.c_str(), static_cast<unsigned long long>(needed),
                static_cast<unsigned long long>(chosen.global_memory));
    if (needed == std::numeric_limits<std::size_t>::max() || chosen.global_memory < needed) {
        std::fputs("insufficient GPU memory\n", stderr);
        std::exit(2);
    }

    Runtime rt;
    rt.device = chosen.device;
    cl_int rc = 0;
    rt.context = clCreateContext(nullptr, 1, &rt.device, nullptr, nullptr, &rc);
    check(rc, "clCreateContext");
    rt.queue = clCreateCommandQueue(rt.context, rt.device, 0, &rc);
    check(rc, "clCreateCommandQueue");
    std::string source = kernel_source();
    const char* src = source.c_str();
    std::size_t size = source.size();
    rt.program = clCreateProgramWithSource(rt.context, 1, &src, &size, &rc);
    check(rc, "clCreateProgramWithSource");
    rc = clBuildProgram(rt.program, 1, &rt.device, "-cl-std=CL1.2", nullptr, nullptr);
    if (rc != CL_SUCCESS) {
        std::size_t log_size = 0;
        clGetProgramBuildInfo(rt.program, rt.device, CL_PROGRAM_BUILD_LOG, 0, nullptr, &log_size);
        std::vector<char> log(log_size + 1u);
        clGetProgramBuildInfo(rt.program, rt.device, CL_PROGRAM_BUILD_LOG, log_size, log.data(), nullptr);
        std::fprintf(stderr, "OpenCL build failed:\n%s\n", log.data());
        std::exit(1);
    }
    return rt;
}

std::vector<unsigned char> vector_cache() {
    std::vector<unsigned char> cache;
    cache.reserve(512u);
    for (const char* node : kCacheHex) {
        std::vector<unsigned char> bytes;
        if (!hex_bytes(node, &bytes)) {
            std::fputs("invalid embedded cache fixture\n", stderr);
            std::exit(1);
        }
        cache.insert(cache.end(), bytes.begin(), bytes.end());
    }
    return cache;
}

int vector_self_test() {
    Runtime rt = make_runtime();
    cl_int rc = 0;
    cl_kernel kernel = clCreateKernel(rt.program, "khushi_vector", &rc);
    check(rc, "clCreateKernel(khushi_vector)");
    std::vector<unsigned char> seed, expected;
    if (!hex_bytes(kProgramSeed, &seed) || !hex_bytes(kExpectedDigest, &expected)) return 1;
    auto cache = vector_cache();
    unsigned char dummy_header = 0;
    cl_uint header_len = 0, cache_nodes = 8;
    cl_ulong nonce = 0;
    cl_mem h = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, 1, &dummy_header, &rc);
    check(rc, "header buffer");
    cl_mem s = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, seed.size(), seed.data(), &rc);
    check(rc, "seed buffer");
    cl_mem c = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, cache.size(), cache.data(), &rc);
    check(rc, "cache buffer");
    cl_mem o = clCreateBuffer(rt.context, CL_MEM_WRITE_ONLY, 32, nullptr, &rc);
    check(rc, "output buffer");
    check(clSetKernelArg(kernel, 0, sizeof(h), &h), "arg0");
    check(clSetKernelArg(kernel, 1, sizeof(header_len), &header_len), "arg1");
    check(clSetKernelArg(kernel, 2, sizeof(nonce), &nonce), "arg2");
    check(clSetKernelArg(kernel, 3, sizeof(s), &s), "arg3");
    check(clSetKernelArg(kernel, 4, sizeof(c), &c), "arg4");
    check(clSetKernelArg(kernel, 5, sizeof(cache_nodes), &cache_nodes), "arg5");
    check(clSetKernelArg(kernel, 6, sizeof(o), &o), "arg6");
    std::size_t global = 1;
    check(clEnqueueNDRangeKernel(rt.queue, kernel, 1, nullptr, &global, nullptr, 0, nullptr, nullptr), "vector enqueue");
    check(clFinish(rt.queue), "vector finish");
    std::vector<unsigned char> got(32);
    check(clEnqueueReadBuffer(rt.queue, o, CL_TRUE, 0, got.size(), got.data(), 0, nullptr, nullptr), "vector read");
    std::fputs("vector-digest=", stdout);
    for (auto b : got) std::printf("%02x", b);
    std::fputc('\n', stdout);
    clReleaseMemObject(o);
    clReleaseMemObject(c);
    clReleaseMemObject(s);
    clReleaseMemObject(h);
    clReleaseKernel(kernel);
    if (got != expected) {
        std::fprintf(stderr, "vector-self-test=failed expected=%s\n", kExpectedDigest);
        return 4;
    }
    std::puts("vector-self-test=ok");
    return 0;
}

int benchmark(unsigned seconds) {
    if (seconds == 0) seconds = 10;
    auto devices = gpu_devices();
    if (devices.empty()) {
        std::fputs("Khushi Algorithm requires an OpenCL GPU; CPU fallback prohibited\n", stderr);
        return 2;
    }
    if (selected_device < 0 || static_cast<std::size_t>(selected_device) >= devices.size()) {
        std::fprintf(stderr, "invalid --device %d\n", selected_device);
        return 2;
    }
    const auto& chosen = devices[static_cast<std::size_t>(selected_device)];
    const tuning::Profile profile = tuning::opencl_profile(chosen.vendor);

    Runtime rt = make_runtime();
    cl_int rc = 0;
    cl_kernel kernel = clCreateKernel(rt.program, "khushi_search", &rc);
    check(rc, "clCreateKernel(khushi_search)");

    std::size_t kernel_max_work_group = 0u;
    check(clGetKernelWorkGroupInfo(
              kernel,
              chosen.device,
              CL_KERNEL_WORK_GROUP_SIZE,
              sizeof(kernel_max_work_group),
              &kernel_max_work_group,
              nullptr),
          "CL_KERNEL_WORK_GROUP_SIZE");
    if (chosen.max_work_group == 0u || kernel_max_work_group == 0u) {
        std::fputs("Khushi Algorithm OpenCL kernel reported no safe work-group size\n", stderr);
        clReleaseKernel(kernel);
        return 6;
    }
    const std::size_t safe_max_local_size =
        chosen.max_work_group < kernel_max_work_group ? chosen.max_work_group : kernel_max_work_group;
    const unsigned safe_max_local = safe_max_local_size > static_cast<std::size_t>(std::numeric_limits<unsigned>::max())
        ? std::numeric_limits<unsigned>::max()
        : static_cast<unsigned>(safe_max_local_size);
    if (safe_max_local == 0u) {
        std::fputs("Khushi Algorithm OpenCL kernel work-group bound resolved to zero\n", stderr);
        clReleaseKernel(kernel);
        return 6;
    }
    const auto launch_candidates = tuning::candidates(profile, safe_max_local);
    if (launch_candidates.empty()) {
        std::fputs("Khushi Algorithm OpenCL autotune produced no kernel-safe launch candidate\n", stderr);
        clReleaseKernel(kernel);
        return 6;
    }

    const char header_text[] = "khushi-algorithm-generic-opencl-benchmark";
    cl_uint header_len = sizeof(header_text) - 1u, cache_nodes = 8, generation = 1, found_flag = 0, hashes = 0;
    cl_ulong nonce_start = 0, found = ~(cl_ulong)0;
    std::vector<unsigned char> seed(32), cache(512), target(32, 0);
    for (std::size_t i = 0; i < seed.size(); ++i) seed[i] = (unsigned char)((i * 17u + 3u) & 0xffu);
    for (std::size_t i = 0; i < cache.size(); ++i) cache[i] = (unsigned char)((i * 29u + 11u) & 0xffu);
    cl_mem h = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, header_len, (void*)header_text, &rc);
    check(rc, "bench header");
    cl_mem s = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, 32, seed.data(), &rc);
    check(rc, "bench seed");
    cl_mem c = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, cache.size(), cache.data(), &rc);
    check(rc, "bench cache");
    cl_mem t = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, 32, target.data(), &rc);
    check(rc, "bench target");
    cl_mem g = clCreateBuffer(rt.context, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR, sizeof(generation), &generation, &rc);
    check(rc, "bench generation");
    cl_mem ff = clCreateBuffer(rt.context, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR, sizeof(found_flag), &found_flag, &rc);
    check(rc, "bench found flag");
    cl_mem f = clCreateBuffer(rt.context, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR, sizeof(found), &found, &rc);
    check(rc, "bench found");
    cl_mem hd = clCreateBuffer(rt.context, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR, sizeof(hashes), &hashes, &rc);
    check(rc, "bench hashes");

    double best_rate = -1.0;
    std::size_t best_local = 0u;
    std::size_t best_global = 0u;
    cl_uint best_hashes = 0u;
    double best_elapsed = 0.0;

    for (const tuning::Candidate candidate : launch_candidates) {
        hashes = 0u;
        found_flag = 0u;
        found = ~(cl_ulong)0;
        nonce_start = 0u;
        check(clEnqueueWriteBuffer(rt.queue, hd, CL_TRUE, 0, sizeof(hashes), &hashes, 0, nullptr, nullptr), "hash reset");
        check(clEnqueueWriteBuffer(rt.queue, ff, CL_TRUE, 0, sizeof(found_flag), &found_flag, 0, nullptr, nullptr), "found flag reset");
        check(clEnqueueWriteBuffer(rt.queue, f, CL_TRUE, 0, sizeof(found), &found, 0, nullptr, nullptr), "found nonce reset");

        const std::size_t local = candidate.local_size;
        const std::size_t global = tuning::work_items(candidate, chosen.compute_units);
        const cl_ulong nonce_count = static_cast<cl_ulong>(global);
        const auto start = std::chrono::steady_clock::now();
        const auto deadline = start + std::chrono::milliseconds(kAutotuneCandidateMilliseconds);
        do {
            int a = 0;
            check(clSetKernelArg(kernel, a++, sizeof(h), &h), "a0");
            check(clSetKernelArg(kernel, a++, sizeof(header_len), &header_len), "a1");
            check(clSetKernelArg(kernel, a++, sizeof(s), &s), "a2");
            check(clSetKernelArg(kernel, a++, sizeof(c), &c), "a3");
            check(clSetKernelArg(kernel, a++, sizeof(cache_nodes), &cache_nodes), "a4");
            check(clSetKernelArg(kernel, a++, sizeof(t), &t), "a5");
            check(clSetKernelArg(kernel, a++, sizeof(nonce_start), &nonce_start), "a6");
            check(clSetKernelArg(kernel, a++, sizeof(nonce_count), &nonce_count), "a7");
            check(clSetKernelArg(kernel, a++, sizeof(g), &g), "a8");
            check(clSetKernelArg(kernel, a++, sizeof(generation), &generation), "a9");
            check(clSetKernelArg(kernel, a++, sizeof(ff), &ff), "a10");
            check(clSetKernelArg(kernel, a++, sizeof(f), &f), "a11");
            check(clSetKernelArg(kernel, a++, sizeof(hd), &hd), "a12");
            check(clEnqueueNDRangeKernel(rt.queue, kernel, 1, nullptr, &global, &local, 0, nullptr, nullptr), "bench autotune enqueue");
            check(clFinish(rt.queue), "bench autotune finish");
            nonce_start += nonce_count;
        } while (std::chrono::steady_clock::now() < deadline);

        check(clEnqueueReadBuffer(rt.queue, hd, CL_TRUE, 0, sizeof(hashes), &hashes, 0, nullptr, nullptr), "bench hashes read");
        const auto end = std::chrono::steady_clock::now();
        const double elapsed = std::chrono::duration<double>(end - start).count();
        const double rate = elapsed > 0.0 ? static_cast<double>(hashes) / elapsed : 0.0;
        std::printf(
            "autotune-candidate backend=opencl device=%d family=%s local_size=%llu work_items=%llu seconds=%.3f hashes=%u hashrate_hps=%.6f\n",
            selected_device,
            tuning::family_name(profile.family),
            static_cast<unsigned long long>(local),
            static_cast<unsigned long long>(global),
            elapsed,
            static_cast<unsigned>(hashes),
            rate);

        if (rate > best_rate) {
            best_rate = rate;
            best_local = local;
            best_global = global;
            best_hashes = hashes;
            best_elapsed = elapsed;
        }
    }

    if (best_local == 0u || best_global == 0u) {
        std::fputs("Khushi Algorithm OpenCL autotune produced no safe launch candidate\n", stderr);
        clReleaseMemObject(hd);
        clReleaseMemObject(f);
        clReleaseMemObject(ff);
        clReleaseMemObject(g);
        clReleaseMemObject(t);
        clReleaseMemObject(c);
        clReleaseMemObject(s);
        clReleaseMemObject(h);
        clReleaseKernel(kernel);
        return 6;
    }
    std::printf(
        "autotune-selected backend=opencl device=%d family=%s local_size=%llu work_items=%llu seconds=%.3f hashes=%u hashrate_hps=%.6f\n",
        selected_device,
        tuning::family_name(profile.family),
        static_cast<unsigned long long>(best_local),
        static_cast<unsigned long long>(best_global),
        best_elapsed,
        static_cast<unsigned>(best_hashes),
        best_rate);

    hashes = 0u;
    found_flag = 0u;
    found = ~(cl_ulong)0;
    nonce_start = 0u;
    check(clEnqueueWriteBuffer(rt.queue, hd, CL_TRUE, 0, sizeof(hashes), &hashes, 0, nullptr, nullptr), "final hash reset");
    check(clEnqueueWriteBuffer(rt.queue, ff, CL_TRUE, 0, sizeof(found_flag), &found_flag, 0, nullptr, nullptr), "final found flag reset");
    check(clEnqueueWriteBuffer(rt.queue, f, CL_TRUE, 0, sizeof(found), &found, 0, nullptr, nullptr), "final found nonce reset");

    const std::size_t final_local = best_local;
    const std::size_t final_global = best_global;
    const cl_ulong final_nonce_count = static_cast<cl_ulong>(final_global);
    const auto final_started = std::chrono::steady_clock::now();
    const auto final_deadline = final_started + std::chrono::seconds(seconds);
    do {
        int a = 0;
        check(clSetKernelArg(kernel, a++, sizeof(h), &h), "final a0");
        check(clSetKernelArg(kernel, a++, sizeof(header_len), &header_len), "final a1");
        check(clSetKernelArg(kernel, a++, sizeof(s), &s), "final a2");
        check(clSetKernelArg(kernel, a++, sizeof(c), &c), "final a3");
        check(clSetKernelArg(kernel, a++, sizeof(cache_nodes), &cache_nodes), "final a4");
        check(clSetKernelArg(kernel, a++, sizeof(t), &t), "final a5");
        check(clSetKernelArg(kernel, a++, sizeof(nonce_start), &nonce_start), "final a6");
        check(clSetKernelArg(kernel, a++, sizeof(final_nonce_count), &final_nonce_count), "final a7");
        check(clSetKernelArg(kernel, a++, sizeof(g), &g), "final a8");
        check(clSetKernelArg(kernel, a++, sizeof(generation), &generation), "final a9");
        check(clSetKernelArg(kernel, a++, sizeof(ff), &ff), "final a10");
        check(clSetKernelArg(kernel, a++, sizeof(f), &f), "final a11");
        check(clSetKernelArg(kernel, a++, sizeof(hd), &hd), "final a12");
        check(clEnqueueNDRangeKernel(rt.queue, kernel, 1, nullptr, &final_global, &final_local, 0, nullptr, nullptr), "bench selected-profile enqueue");
        check(clFinish(rt.queue), "bench selected-profile finish");
        nonce_start += final_nonce_count;
    } while (std::chrono::steady_clock::now() < final_deadline);

    check(clEnqueueReadBuffer(rt.queue, hd, CL_TRUE, 0, sizeof(hashes), &hashes, 0, nullptr, nullptr), "final hashes read");
    const auto final_ended = std::chrono::steady_clock::now();
    const double final_elapsed = std::chrono::duration<double>(final_ended - final_started).count();
    const double final_rate = final_elapsed > 0.0 ? static_cast<double>(hashes) / final_elapsed : 0.0;
    std::printf(
        "selected-profile-benchmark backend=opencl device=%d family=%s local_size=%llu work_items=%llu requested_seconds=%u seconds=%.3f hashes=%u hashrate_hps=%.6f\n",
        selected_device,
        tuning::family_name(profile.family),
        static_cast<unsigned long long>(final_local),
        static_cast<unsigned long long>(final_global),
        seconds,
        final_elapsed,
        static_cast<unsigned>(hashes),
        final_rate);
    std::printf("Khushi Algorithm benchmark backend=opencl device=%d seconds=%.3f hashes=%u hashrate_hps=%.6f\n",
                selected_device, final_elapsed, static_cast<unsigned>(hashes), final_rate);

    clReleaseMemObject(hd);
    clReleaseMemObject(f);
    clReleaseMemObject(ff);
    clReleaseMemObject(g);
    clReleaseMemObject(t);
    clReleaseMemObject(c);
    clReleaseMemObject(s);
    clReleaseMemObject(h);
    clReleaseKernel(kernel);
    return 0;
}

int staging_search(const char* header_hex, const char* target_hex, cl_ulong height, cl_uint cache_nodes) {
    if (height != 0u || cache_nodes != 8u) {
        std::fputs("staging search only supports height=0 cache_nodes=8\n", stderr);
        return 64;
    }

    std::vector<unsigned char> header, target, seed;
    if (!hex_bytes(header_hex, &header) || header.empty() || header.size() > 256u) {
        std::fputs("invalid --header-prefix-hex value\n", stderr);
        return 64;
    }
    if (!hex_bytes(target_hex, &target) || target.size() != 32u) {
        std::fputs("invalid --target-hex value\n", stderr);
        return 64;
    }
    if (!hex_bytes(kProgramSeed, &seed) || seed.size() != 32u) {
        std::fputs("invalid staging program seed fixture\n", stderr);
        return 1;
    }

    Runtime rt = make_runtime();
    auto cache = vector_cache();
    cl_int rc = 0;
    cl_kernel kernel = clCreateKernel(rt.program, "khushi_search", &rc);
    check(rc, "clCreateKernel(khushi_search staging)");

    const cl_uint header_len = static_cast<cl_uint>(header.size());
    const cl_uint generation = 1u;
    cl_uint found_flag = 0u;
    cl_uint hashes = 0u;
    const cl_ulong no_nonce = ~(cl_ulong)0;
    cl_ulong found = no_nonce;

    cl_mem h = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, header.size(), header.data(), &rc);
    check(rc, "staging header");
    cl_mem s = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, seed.size(), seed.data(), &rc);
    check(rc, "staging seed");
    cl_mem c = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, cache.size(), cache.data(), &rc);
    check(rc, "staging cache");
    cl_mem t = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, target.size(), target.data(), &rc);
    check(rc, "staging target");
    cl_mem g = clCreateBuffer(rt.context, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR, sizeof(generation), (void*)&generation, &rc);
    check(rc, "staging generation");
    cl_mem ff = clCreateBuffer(rt.context, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR, sizeof(found_flag), &found_flag, &rc);
    check(rc, "staging found flag");
    cl_mem f = clCreateBuffer(rt.context, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR, sizeof(found), &found, &rc);
    check(rc, "staging found nonce");
    cl_mem hd = clCreateBuffer(rt.context, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR, sizeof(hashes), &hashes, &rc);
    check(rc, "staging hashes");

    const cl_ulong max_nonces = 65536;
    const cl_ulong nonces_per_launch = 64u;
    for (cl_ulong nonce_start = 0u; nonce_start < max_nonces && found == no_nonce; nonce_start += nonces_per_launch) {
        cl_ulong nonce_count = nonces_per_launch;
        if (nonce_start + nonce_count > max_nonces) nonce_count = max_nonces - nonce_start;

        int a = 0;
        check(clSetKernelArg(kernel, a++, sizeof(h), &h), "staging a0");
        check(clSetKernelArg(kernel, a++, sizeof(header_len), &header_len), "staging a1");
        check(clSetKernelArg(kernel, a++, sizeof(s), &s), "staging a2");
        check(clSetKernelArg(kernel, a++, sizeof(c), &c), "staging a3");
        check(clSetKernelArg(kernel, a++, sizeof(cache_nodes), &cache_nodes), "staging a4");
        check(clSetKernelArg(kernel, a++, sizeof(t), &t), "staging a5");
        check(clSetKernelArg(kernel, a++, sizeof(nonce_start), &nonce_start), "staging a6");
        check(clSetKernelArg(kernel, a++, sizeof(nonce_count), &nonce_count), "staging a7");
        check(clSetKernelArg(kernel, a++, sizeof(g), &g), "staging a8");
        check(clSetKernelArg(kernel, a++, sizeof(generation), &generation), "staging a9");
        check(clSetKernelArg(kernel, a++, sizeof(ff), &ff), "staging a10");
        check(clSetKernelArg(kernel, a++, sizeof(f), &f), "staging a11");
        check(clSetKernelArg(kernel, a++, sizeof(hd), &hd), "staging a12");

        const std::size_t global = static_cast<std::size_t>(nonce_count);
        check(clEnqueueNDRangeKernel(rt.queue, kernel, 1, nullptr, &global, nullptr, 0, nullptr, nullptr), "staging enqueue");
        check(clFinish(rt.queue), "staging finish");
        check(clEnqueueReadBuffer(rt.queue, f, CL_TRUE, 0, sizeof(found), &found, 0, nullptr, nullptr), "staging nonce read");
    }

    check(clEnqueueReadBuffer(rt.queue, hd, CL_TRUE, 0, sizeof(hashes), &hashes, 0, nullptr, nullptr), "staging hashes read");

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
        std::printf("staging-search=not-found hashes=%u\n", static_cast<unsigned>(hashes));
        return 4;
    }
    std::printf("staging-solution-nonce=%llu hashes=%u\n",
                static_cast<unsigned long long>(found), static_cast<unsigned>(hashes));
    return 0;
}

void usage() {
    std::fputs(
        "usage: khushi-miner-opencl [--device N] --list-devices | --vector-self-test | --production-memory-self-test | --benchmark [seconds] | --staging-search --header-prefix-hex HEX --target-hex HEX --height N --cache-nodes N | --mine\n",
        stderr);
}

}  // namespace

int main(int argc, char** argv) {
    int arg = 1;
    if (argc >= 3 && std::strcmp(argv[1], "--device") == 0) {
        selected_device = std::atoi(argv[2]);
        arg = 3;
    }
    if (arg >= argc) {
        usage();
        return 64;
    }
    if (std::strcmp(argv[arg], "--list-devices") == 0 && arg + 1 == argc) return list_devices();
    if (std::strcmp(argv[arg], "--vector-self-test") == 0 && arg + 1 == argc) return vector_self_test();
    if (std::strcmp(argv[arg], "--production-memory-self-test") == 0 && arg + 1 == argc) return production_memory_self_test();
    if (std::strcmp(argv[arg], "--benchmark") == 0 && (arg + 1 == argc || arg + 2 == argc)) {
        unsigned seconds = (arg + 2 == argc) ? (unsigned)std::strtoul(argv[arg + 1], nullptr, 10) : 10u;
        return benchmark(seconds);
    }
    if (std::strcmp(argv[arg], "--staging-search") == 0 && arg + 9 == argc &&
        std::strcmp(argv[arg + 1], "--header-prefix-hex") == 0 &&
        std::strcmp(argv[arg + 3], "--target-hex") == 0 &&
        std::strcmp(argv[arg + 5], "--height") == 0 &&
        std::strcmp(argv[arg + 7], "--cache-nodes") == 0) {
        cl_ulong height = 0;
        cl_uint cache_nodes = 0;
        if (!parse_u64(argv[arg + 6], &height) || !parse_u32(argv[arg + 8], &cache_nodes)) {
            std::fputs("invalid staging height or cache node count\n", stderr);
            return 64;
        }
        return staging_search(argv[arg + 2], argv[arg + 4], height, cache_nodes);
    }
    if (std::strcmp(argv[arg], "--mine") == 0) {
        std::fputs("Khushi Algorithm OpenCL network mining remains interoperability-gated; CPU fallback prohibited\n", stderr);
        return 3;
    }
    usage();
    return 64;
}