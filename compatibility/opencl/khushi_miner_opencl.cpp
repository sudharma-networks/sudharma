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

namespace {

constexpr const char* kExpectedDigest = "2a7c15fc6c84a67d43ff7074ac5835aa433145f89d10d1d9e36a99fe22da4b2b";
constexpr const char* kProgramSeed = "613684e3f3b42773073fb9c99e71f2933eed301d450866fe9a5a5c0530a769bd";
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
    cl_ulong global_memory{};
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
            cl_ulong memory = 0;
            check(clGetDeviceInfo(device, CL_DEVICE_NAME, sizeof(name), name, nullptr), "CL_DEVICE_NAME");
            check(clGetDeviceInfo(device, CL_DEVICE_GLOBAL_MEM_SIZE, sizeof(memory), &memory, nullptr), "CL_DEVICE_GLOBAL_MEM_SIZE");
            out.push_back(DeviceRef{platform, device, name, memory});
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
        std::printf("device=%zu name=%s vram_bytes=%llu backend=opencl\n",
                    i, devices[i].name.c_str(), static_cast<unsigned long long>(devices[i].global_memory));
    }
    return devices.empty() ? 2 : 0;
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
    out->clear(); out->reserve(n / 2u);
    for (std::size_t i = 0; i < n; i += 2u) {
        int a = nibble(text[i]), b = nibble(text[i + 1u]);
        if (a < 0 || b < 0) return false;
        out->push_back(static_cast<unsigned char>((a << 4) | b));
    }
    return true;
}

std::string kernel_source() {
    const char* paths[] = {"compatibility/opencl/khushi_pow.cl", "khushi_pow.cl"};
    for (const char* path : paths) {
        std::ifstream in(path, std::ios::binary);
        if (!in) continue;
        std::ostringstream ss; ss << in.rdbuf(); return ss.str();
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
    rt.context = clCreateContext(nullptr, 1, &rt.device, nullptr, nullptr, &rc); check(rc, "clCreateContext");
    rt.queue = clCreateCommandQueue(rt.context, rt.device, 0, &rc); check(rc, "clCreateCommandQueue");
    std::string source = kernel_source(); const char* src = source.c_str(); std::size_t size = source.size();
    rt.program = clCreateProgramWithSource(rt.context, 1, &src, &size, &rc); check(rc, "clCreateProgramWithSource");
    rc = clBuildProgram(rt.program, 1, &rt.device, "-cl-std=CL1.2", nullptr, nullptr);
    if (rc != CL_SUCCESS) {
        std::size_t log_size = 0; clGetProgramBuildInfo(rt.program, rt.device, CL_PROGRAM_BUILD_LOG, 0, nullptr, &log_size);
        std::vector<char> log(log_size + 1u); clGetProgramBuildInfo(rt.program, rt.device, CL_PROGRAM_BUILD_LOG, log_size, log.data(), nullptr);
        std::fprintf(stderr, "OpenCL build failed:\n%s\n", log.data()); std::exit(1);
    }
    return rt;
}

std::vector<unsigned char> vector_cache() {
    std::vector<unsigned char> cache; cache.reserve(512u);
    for (const char* node : kCacheHex) { std::vector<unsigned char> bytes; hex_bytes(node, &bytes); cache.insert(cache.end(), bytes.begin(), bytes.end()); }
    return cache;
}

int vector_self_test() {
    Runtime rt = make_runtime();
    cl_int rc = 0;
    cl_kernel kernel = clCreateKernel(rt.program, "khushi_vector", &rc); check(rc, "clCreateKernel(khushi_vector)");
    std::vector<unsigned char> seed, expected; hex_bytes(kProgramSeed, &seed); hex_bytes(kExpectedDigest, &expected);
    auto cache = vector_cache(); unsigned char dummy_header = 0; cl_uint header_len = 0, cache_nodes = 8; cl_ulong nonce = 0;
    cl_mem h = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, 1, &dummy_header, &rc); check(rc,"header buffer");
    cl_mem s = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, seed.size(), seed.data(), &rc); check(rc,"seed buffer");
    cl_mem c = clCreateBuffer(rt.context, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, cache.size(), cache.data(), &rc); check(rc,"cache buffer");
    cl_mem o = clCreateBuffer(rt.context, CL_MEM_WRITE_ONLY, 32, nullptr, &rc); check(rc,"output buffer");
    check(clSetKernelArg(kernel,0,sizeof(h),&h),"arg0"); check(clSetKernelArg(kernel,1,sizeof(header_len),&header_len),"arg1");
    check(clSetKernelArg(kernel,2,sizeof(nonce),&nonce),"arg2"); check(clSetKernelArg(kernel,3,sizeof(s),&s),"arg3");
    check(clSetKernelArg(kernel,4,sizeof(c),&c),"arg4"); check(clSetKernelArg(kernel,5,sizeof(cache_nodes),&cache_nodes),"arg5"); check(clSetKernelArg(kernel,6,sizeof(o),&o),"arg6");
    std::size_t global = 1; check(clEnqueueNDRangeKernel(rt.queue,kernel,1,nullptr,&global,nullptr,0,nullptr,nullptr),"vector enqueue"); check(clFinish(rt.queue),"vector finish");
    std::vector<unsigned char> got(32); check(clEnqueueReadBuffer(rt.queue,o,CL_TRUE,0,got.size(),got.data(),0,nullptr,nullptr),"vector read");
    std::fputs("vector-digest=",stdout); for (auto b : got) std::printf("%02x",b); std::fputc('\n',stdout);
    clReleaseMemObject(o);clReleaseMemObject(c);clReleaseMemObject(s);clReleaseMemObject(h);clReleaseKernel(kernel);
    if (got != expected) { std::fprintf(stderr,"vector-self-test=failed expected=%s\n",kExpectedDigest); return 4; }
    std::puts("vector-self-test=ok"); return 0;
}

int benchmark(unsigned seconds) {
    if (seconds == 0) seconds = 10;
    Runtime rt = make_runtime(); cl_int rc=0; cl_kernel kernel=clCreateKernel(rt.program,"khushi_search",&rc); check(rc,"clCreateKernel(khushi_search)");
    const char header_text[]="khushi-algorithm-generic-opencl-benchmark";
    cl_uint header_len=sizeof(header_text)-1u,cache_nodes=8,generation=1,found_flag=0,hashes=0;
    cl_ulong nonce_start=0,nonce_count=1,found=~(cl_ulong)0;
    std::vector<unsigned char> seed(32),cache(512),target(32,0);
    for(std::size_t i=0;i<seed.size();++i)seed[i]=(unsigned char)((i*17u+3u)&0xffu);
    for(std::size_t i=0;i<cache.size();++i)cache[i]=(unsigned char)((i*29u+11u)&0xffu);
    cl_mem h=clCreateBuffer(rt.context,CL_MEM_READ_ONLY|CL_MEM_COPY_HOST_PTR,header_len,(void*)header_text,&rc);check(rc,"bench header");
    cl_mem s=clCreateBuffer(rt.context,CL_MEM_READ_ONLY|CL_MEM_COPY_HOST_PTR,32,seed.data(),&rc);check(rc,"bench seed");
    cl_mem c=clCreateBuffer(rt.context,CL_MEM_READ_ONLY|CL_MEM_COPY_HOST_PTR,cache.size(),cache.data(),&rc);check(rc,"bench cache");
    cl_mem t=clCreateBuffer(rt.context,CL_MEM_READ_ONLY|CL_MEM_COPY_HOST_PTR,32,target.data(),&rc);check(rc,"bench target");
    cl_mem g=clCreateBuffer(rt.context,CL_MEM_READ_WRITE|CL_MEM_COPY_HOST_PTR,sizeof(generation),&generation,&rc);check(rc,"bench generation");
    cl_mem ff=clCreateBuffer(rt.context,CL_MEM_READ_WRITE|CL_MEM_COPY_HOST_PTR,sizeof(found_flag),&found_flag,&rc);check(rc,"bench found flag");
    cl_mem f=clCreateBuffer(rt.context,CL_MEM_READ_WRITE|CL_MEM_COPY_HOST_PTR,sizeof(found),&found,&rc);check(rc,"bench found");
    cl_mem hd=clCreateBuffer(rt.context,CL_MEM_READ_WRITE|CL_MEM_COPY_HOST_PTR,sizeof(hashes),&hashes,&rc);check(rc,"bench hashes");
    auto start=std::chrono::steady_clock::now(),deadline=start+std::chrono::seconds(seconds);
    do{
        hashes=0;found_flag=0;found=~(cl_ulong)0;
        check(clEnqueueWriteBuffer(rt.queue,hd,CL_TRUE,0,sizeof(hashes),&hashes,0,nullptr,nullptr),"hash reset");
        check(clEnqueueWriteBuffer(rt.queue,ff,CL_TRUE,0,sizeof(found_flag),&found_flag,0,nullptr,nullptr),"found flag reset");
        check(clEnqueueWriteBuffer(rt.queue,f,CL_TRUE,0,sizeof(found),&found,0,nullptr,nullptr),"found nonce reset");
        int a=0;
        check(clSetKernelArg(kernel,a++,sizeof(h),&h),"a0");check(clSetKernelArg(kernel,a++,sizeof(header_len),&header_len),"a1");
        check(clSetKernelArg(kernel,a++,sizeof(s),&s),"a2");check(clSetKernelArg(kernel,a++,sizeof(c),&c),"a3");
        check(clSetKernelArg(kernel,a++,sizeof(cache_nodes),&cache_nodes),"a4");check(clSetKernelArg(kernel,a++,sizeof(t),&t),"a5");
        check(clSetKernelArg(kernel,a++,sizeof(nonce_start),&nonce_start),"a6");check(clSetKernelArg(kernel,a++,sizeof(nonce_count),&nonce_count),"a7");
        check(clSetKernelArg(kernel,a++,sizeof(g),&g),"a8");check(clSetKernelArg(kernel,a++,sizeof(generation),&generation),"a9");
        check(clSetKernelArg(kernel,a++,sizeof(ff),&ff),"a10");check(clSetKernelArg(kernel,a++,sizeof(f),&f),"a11");
        check(clSetKernelArg(kernel,a++,sizeof(hd),&hd),"a12");
        std::size_t one=1;check(clEnqueueNDRangeKernel(rt.queue,kernel,1,nullptr,&one,nullptr,0,nullptr,nullptr),"bench enqueue");check(clFinish(rt.queue),"bench finish");++nonce_start;
    }while(std::chrono::steady_clock::now()<deadline);
    auto end=std::chrono::steady_clock::now();double elapsed=std::chrono::duration<double>(end-start).count();double rate=elapsed>0?static_cast<double>(nonce_start)/elapsed:0;
    std::printf("Khushi Algorithm benchmark backend=opencl device=%d seconds=%.3f hashes=%llu hashrate_hps=%.6f\n",selected_device,elapsed,(unsigned long long)nonce_start,rate);
    clReleaseMemObject(hd);clReleaseMemObject(f);clReleaseMemObject(ff);clReleaseMemObject(g);clReleaseMemObject(t);clReleaseMemObject(c);clReleaseMemObject(s);clReleaseMemObject(h);clReleaseKernel(kernel);return 0;
}

void usage(){std::fputs("usage: khushi-miner-opencl [--device N] --list-devices | --vector-self-test | --benchmark [seconds] | --mine\n",stderr);}

} // namespace

int main(int argc,char** argv){int arg=1;if(argc>=3&&std::strcmp(argv[1],"--device")==0){selected_device=std::atoi(argv[2]);arg=3;}if(arg>=argc){usage();return 64;}if(std::strcmp(argv[arg],"--list-devices")==0)return list_devices();if(std::strcmp(argv[arg],"--vector-self-test")==0)return vector_self_test();if(std::strcmp(argv[arg],"--benchmark")==0){unsigned seconds=(arg+1<argc)?(unsigned)std::strtoul(argv[arg+1],nullptr,10):10u;return benchmark(seconds);}if(std::strcmp(argv[arg],"--mine")==0){std::fputs("Khushi Algorithm OpenCL network mining remains interoperability-gated; CPU fallback prohibited\n",stderr);return 3;}usage();return 64;}
