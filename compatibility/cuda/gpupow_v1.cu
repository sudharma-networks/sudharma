#include <array>
#include <cstdint>
#include <cstdio>
#include <cstring>
#include <string>
#include <vector>

#ifdef __CUDACC__
#define SUDHARMA_HD __host__ __device__
#else
#define SUDHARMA_HD
#endif

namespace sudharma::gpupowv1 {

constexpr std::uint32_t kFNVOffsetBasis = 0x811c9dc5u;
constexpr std::uint32_t kFNVPrime = 0x01000193u;
constexpr char kHeaderDomain[] = "SUDHARMA-GPU-POW-V1-REFERENCE-HEADER";

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

inline std::uint32_t rotr(std::uint32_t x, std::uint32_t n) {
    return (x >> n) | (x << (32u - n));
}

std::array<std::uint8_t, 32> sha256(const std::vector<std::uint8_t>& input) {
    static constexpr std::uint32_t k[64] = {
        0x428a2f98u, 0x71374491u, 0xb5c0fbcfu, 0xe9b5dba5u,
        0x3956c25bu, 0x59f111f1u, 0x923f82a4u, 0xab1c5ed5u,
        0xd807aa98u, 0x12835b01u, 0x243185beu, 0x550c7dc3u,
        0x72be5d74u, 0x80deb1feu, 0x9bdc06a7u, 0xc19bf174u,
        0xe49b69c1u, 0xefbe4786u, 0x0fc19dc6u, 0x240ca1ccu,
        0x2de92c6fu, 0x4a7484aau, 0x5cb0a9dcu, 0x76f988dau,
        0x983e5152u, 0xa831c66du, 0xb00327c8u, 0xbf597fc7u,
        0xc6e00bf3u, 0xd5a79147u, 0x06ca6351u, 0x14292967u,
        0x27b70a85u, 0x2e1b2138u, 0x4d2c6dfcu, 0x53380d13u,
        0x650a7354u, 0x766a0abbu, 0x81c2c92eu, 0x92722c85u,
        0xa2bfe8a1u, 0xa81a664bu, 0xc24b8b70u, 0xc76c51a3u,
        0xd192e819u, 0xd6990624u, 0xf40e3585u, 0x106aa070u,
        0x19a4c116u, 0x1e376c08u, 0x2748774cu, 0x34b0bcb5u,
        0x391c0cb3u, 0x4ed8aa4au, 0x5b9cca4fu, 0x682e6ff3u,
        0x748f82eeu, 0x78a5636fu, 0x84c87814u, 0x8cc70208u,
        0x90befffau, 0xa4506cebu, 0xbef9a3f7u, 0xc67178f2u,
    };

    std::vector<std::uint8_t> msg = input;
    const std::uint64_t bit_len = static_cast<std::uint64_t>(msg.size()) * 8u;
    msg.push_back(0x80u);
    while ((msg.size() % 64u) != 56u) {
        msg.push_back(0u);
    }
    for (int shift = 56; shift >= 0; shift -= 8) {
        msg.push_back(static_cast<std::uint8_t>((bit_len >> shift) & 0xffu));
    }

    std::uint32_t h[8] = {
        0x6a09e667u, 0xbb67ae85u, 0x3c6ef372u, 0xa54ff53au,
        0x510e527fu, 0x9b05688cu, 0x1f83d9abu, 0x5be0cd19u,
    };

    for (std::size_t offset = 0; offset < msg.size(); offset += 64u) {
        std::uint32_t w[64] = {};
        for (std::size_t i = 0; i < 16; ++i) {
            const std::size_t j = offset + i * 4u;
            w[i] = (static_cast<std::uint32_t>(msg[j]) << 24u) |
                   (static_cast<std::uint32_t>(msg[j + 1]) << 16u) |
                   (static_cast<std::uint32_t>(msg[j + 2]) << 8u) |
                   static_cast<std::uint32_t>(msg[j + 3]);
        }
        for (std::size_t i = 16; i < 64; ++i) {
            const std::uint32_t s0 = rotr(w[i - 15], 7u) ^ rotr(w[i - 15], 18u) ^ (w[i - 15] >> 3u);
            const std::uint32_t s1 = rotr(w[i - 2], 17u) ^ rotr(w[i - 2], 19u) ^ (w[i - 2] >> 10u);
            w[i] = w[i - 16] + s0 + w[i - 7] + s1;
        }

        std::uint32_t a = h[0];
        std::uint32_t b = h[1];
        std::uint32_t c = h[2];
        std::uint32_t d = h[3];
        std::uint32_t e = h[4];
        std::uint32_t f = h[5];
        std::uint32_t g = h[6];
        std::uint32_t hh = h[7];

        for (std::size_t i = 0; i < 64; ++i) {
            const std::uint32_t s1 = rotr(e, 6u) ^ rotr(e, 11u) ^ rotr(e, 25u);
            const std::uint32_t ch = (e & f) ^ ((~e) & g);
            const std::uint32_t temp1 = hh + s1 + ch + k[i] + w[i];
            const std::uint32_t s0 = rotr(a, 2u) ^ rotr(a, 13u) ^ rotr(a, 22u);
            const std::uint32_t maj = (a & b) ^ (a & c) ^ (b & c);
            const std::uint32_t temp2 = s0 + maj;

            hh = g;
            g = f;
            f = e;
            e = d + temp1;
            d = c;
            c = b;
            b = a;
            a = temp1 + temp2;
        }

        h[0] += a;
        h[1] += b;
        h[2] += c;
        h[3] += d;
        h[4] += e;
        h[5] += f;
        h[6] += g;
        h[7] += hh;
    }

    std::array<std::uint8_t, 32> digest{};
    for (std::size_t i = 0; i < 8; ++i) {
        digest[i * 4] = static_cast<std::uint8_t>(h[i] >> 24u);
        digest[i * 4 + 1] = static_cast<std::uint8_t>(h[i] >> 16u);
        digest[i * 4 + 2] = static_cast<std::uint8_t>(h[i] >> 8u);
        digest[i * 4 + 3] = static_cast<std::uint8_t>(h[i]);
    }
    return digest;
}

bool decode_hex(const char* text, std::vector<std::uint8_t>* out) {
    const std::size_t n = std::strlen(text);
    if ((n % 2u) != 0u) {
        return false;
    }
    out->clear();
    out->reserve(n / 2u);
    for (std::size_t i = 0; i < n; i += 2u) {
        auto nibble = [](char c) -> int {
            if (c >= '0' && c <= '9') return c - '0';
            if (c >= 'a' && c <= 'f') return c - 'a' + 10;
            if (c >= 'A' && c <= 'F') return c - 'A' + 10;
            return -1;
        };
        const int hi = nibble(text[i]);
        const int lo = nibble(text[i + 1]);
        if (hi < 0 || lo < 0) {
            return false;
        }
        out->push_back(static_cast<std::uint8_t>((hi << 4) | lo));
    }
    return true;
}

bool parse_u64_hex(const char* text, std::uint64_t* out) {
    if (std::strlen(text) != 16u) {
        return false;
    }
    std::vector<std::uint8_t> bytes;
    if (!decode_hex(text, &bytes) || bytes.size() != 8u) {
        return false;
    }
    std::uint64_t value = 0;
    for (std::uint8_t b : bytes) {
        value = (value << 8u) | b;
    }
    *out = value;
    return true;
}

std::array<std::uint8_t, 32> header_digest(const std::vector<std::uint8_t>& header, std::uint64_t nonce) {
    std::vector<std::uint8_t> input;
    input.reserve(sizeof(kHeaderDomain) + header.size() + 8u);
    input.insert(input.end(), kHeaderDomain, kHeaderDomain + sizeof(kHeaderDomain));
    input.insert(input.end(), header.begin(), header.end());
    for (unsigned shift = 0; shift < 64u; shift += 8u) {
        input.push_back(static_cast<std::uint8_t>((nonce >> shift) & 0xffu));
    }
    return sha256(input);
}

std::uint64_t work_seed(const std::array<std::uint8_t, 32>& digest) {
    std::uint64_t seed = 0;
    for (unsigned i = 0; i < 8u; ++i) {
        seed |= static_cast<std::uint64_t>(digest[i]) << (8u * i);
    }
    return seed;
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

static int run_header_seed(const char* header_hex, const char* nonce_hex) {
    std::vector<std::uint8_t> header;
    std::uint64_t nonce = 0;
    if (!sudharma::gpupowv1::decode_hex(header_hex, &header) ||
        !sudharma::gpupowv1::parse_u64_hex(nonce_hex, &nonce)) {
        std::fputs("invalid header or nonce hex\n", stderr);
        return 65;
    }

    const auto digest = sudharma::gpupowv1::header_digest(header, nonce);
    std::fputs("header-digest=", stdout);
    for (std::uint8_t b : digest) {
        std::printf("%02x", b);
    }
    std::printf(" work-seed=%016llx\n",
                static_cast<unsigned long long>(sudharma::gpupowv1::work_seed(digest)));
    return 0;
}

int main(int argc, char** argv) {
    if (argc == 2 && std::strcmp(argv[1], "--self-test") == 0) {
        return run_self_test();
    }

    if (argc == 4 && std::strcmp(argv[1], "--header-seed") == 0) {
        return run_header_seed(argv[2], argv[3]);
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

    std::fputs("usage: sudharma-gpupow-cuda --self-test | --header-seed HEADER_HEX NONCE_HEX | --mine\n", stderr);
    return 64;
}
