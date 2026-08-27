#include <cstdint>
#include <cstdio>
#include <cstring>

#ifdef __CUDACC__
#define SUDHARMA_HD __host__ __device__
#else
#define SUDHARMA_HD
#endif

namespace sudharma::gpupowv1 {

constexpr std::uint32_t kFNVOffsetBasis = 0x811c9dc5u;
constexpr std::uint32_t kFNVPrime = 0x01000193u;

SUDHARMA_HD inline std::uint32_t fnv1a(std::uint32_t a, std::uint32_t b) {
    return (a ^ b) * kFNVPrime;
}

struct KISS99 {
    std::uint32_t z;
    std::uint32_t w;
    std::uint32_t jsr;
    std::uint32_t jcong;

    SUDHARMA_HD std::uint32_t next() {
        z = 36969u * (z & 0xffffu) + (z >> 16u);
        w = 18000u * (w & 0xffffu) + (w >> 16u);
        jcong = 69069u * jcong + 1234567u;
        jsr ^= jsr << 17u;
        jsr ^= jsr >> 13u;
        jsr ^= jsr << 5u;
        return (((z << 16u) + w) ^ jcong) + jsr;
    }
};

SUDHARMA_HD inline KISS99 new_kiss99(std::uint32_t seed_lo, std::uint32_t seed_hi) {
    const std::uint32_t z = fnv1a(kFNVOffsetBasis, seed_lo);
    const std::uint32_t w = fnv1a(z, seed_hi);
    const std::uint32_t jsr = fnv1a(w, seed_lo);
    const std::uint32_t jcong = fnv1a(jsr, seed_hi);
    return KISS99{z, w, jsr, jcong};
}

}  // namespace sudharma::gpupowv1

static int run_self_test() {
    using sudharma::gpupowv1::new_kiss99;

    auto rng = new_kiss99(0x01234567u, 0x89abcdefu);
    std::uint32_t got[4] = {rng.next(), rng.next(), rng.next(), rng.next()};
    constexpr std::uint32_t expected[4] = {
        0x5f502f5eu,
        0x5065034fu,
        0x0b7649f5u,
        0x6759296du,
    };

    std::printf("kiss99=%08x,%08x,%08x,%08x\n", got[0], got[1], got[2], got[3]);
    for (int i = 0; i < 4; ++i) {
        if (got[i] != expected[i]) {
            std::fputs("self-test=failed\n", stderr);
            return 1;
        }
    }
    std::puts("self-test=ok");
    return 0;
}

int main(int argc, char** argv) {
    if (argc == 2 && std::strcmp(argv[1], "--self-test") == 0) {
        return run_self_test();
    }

    if (argc == 2 && std::strcmp(argv[1], "--mine") == 0) {
#ifndef __CUDACC__
        std::fputs("CUDA backend required; CPU fallback prohibited\n", stderr);
        return 2;
#else
        std::fputs("CUDA mining kernel not implemented in this checkpoint\n", stderr);
        return 3;
#endif
    }

    std::fputs("usage: sudharma-gpupow-cuda --self-test | --mine\n", stderr);
    return 64;
}
