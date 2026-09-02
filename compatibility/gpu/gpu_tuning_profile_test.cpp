#include "gpu_tuning_profile.h"

#include <cassert>

using namespace sudharma::gpupowv1::tuning;

int main() {
    assert(cuda_profile(7, 5).family == Family::NvidiaTuring);
    assert(cuda_profile(8, 6).family == Family::NvidiaAmpere);
    assert(cuda_profile(8, 9).family == Family::NvidiaAda);
    assert(cuda_profile(12, 0).family == Family::NvidiaBlackwell);
    assert(cuda_profile(6, 1).family == Family::NvidiaGeneric);

    assert(opencl_profile("Advanced Micro Devices, Inc.").family == Family::AmdGeneric);
    assert(opencl_profile("AMD").family == Family::AmdGeneric);
    assert(opencl_profile("NVIDIA Corporation").family == Family::NvidiaGeneric);
    assert(opencl_profile("Unknown GPU Vendor").family == Family::Generic);

    const auto safe = candidates(Profile{Family::Generic, 32u}, 64u);
    assert(!safe.empty());
    for (const auto candidate : safe) {
        assert(candidate.local_size > 0u);
        assert(candidate.local_size <= 64u);
        assert(candidate.groups_per_unit > 0u);
    }

    const auto amd = candidates(Profile{Family::AmdGeneric, 64u}, 256u);
    assert(!amd.empty());
    assert(amd.front().local_size == 64u);

    const auto tiny = candidates(Profile{Family::NvidiaBlackwell, 32u}, 16u);
    assert(!tiny.empty());
    for (const auto candidate : tiny) assert(candidate.local_size <= 16u);

    assert(work_items(Candidate{64u, 4u}, 10u) == 2560u);
    assert(work_items(Candidate{32u, 2u}, 0u) == 64u);
}
