#pragma once

#include <array>
#include <cstddef>
#include <cstdint>

#include "gpupow_v1_search.cuh"

namespace sudharma::gpupowv1 {

#ifdef __CUDACC__

__device__ inline std::array<std::uint32_t, kProgramNumRegs> production_lane_mix(
    std::uint64_t work_seed,
    std::uint32_t lane,
    const std::array<std::uint8_t, 32>& program_seed,
    const std::uint8_t* cache,
    std::uint32_t cache_nodes) {
    auto mix = program_init_lane(work_seed, lane);
    if (cache == nullptr || cache_nodes == 0u) return mix;

    const std::uint32_t seed_lo = program_seed_word(program_seed, 0u);
    const std::uint32_t seed_hi = program_seed_word(program_seed, 4u);
    const auto schedules = register_permutations(seed_lo, seed_hi);
    auto rng = program_kiss99(seed_lo, seed_hi);

    for (std::uint32_t round = 0; round < kProgramDAGRounds; ++round) {
        const std::uint32_t dag_index = program_fnv1(mix[0] ^ round, lane ^ rng.next());
        const auto dag_item = dataset_item_from_cache(cache, cache_nodes, dag_index);

        for (std::uint32_t load = 0; load < kProgramDAGLoads; ++load) {
            const std::uint32_t dst =
                schedules.first[(round * kProgramDAGLoads + load) % kProgramNumRegs];
            mix[dst] = program_random_merge(mix[dst], word64(dag_item, load), rng.next());
        }

        for (std::uint32_t access = 0; access < kProgramCacheAccesses; ++access) {
            const std::uint32_t src = schedules.second[(round + access) % kProgramNumRegs];
            const std::uint32_t selector = mix[src] ^ rng.next();
            const std::uint32_t cache_index = selector % cache_nodes;
            const std::uint8_t* cache_node =
                cache + static_cast<std::size_t>(cache_index) * 64u;
            const std::uint32_t cache_word =
                word64_raw(cache_node, (selector >> 16u) % 16u);
            const std::uint32_t dst = schedules.first[
                (round + kProgramDAGLoads + access) % kProgramNumRegs];
            mix[dst] = program_random_merge(mix[dst], cache_word, rng.next());
        }

        for (std::uint32_t op = 0; op < kProgramMathOperations; ++op) {
            const std::uint32_t dst = schedules.first[
                (round + kProgramDAGLoads + kProgramCacheAccesses + op) % kProgramNumRegs];
            const std::uint32_t src_a = schedules.second[(round + op) % kProgramNumRegs];
            const std::uint32_t src_b = schedules.second[(round + op + 1u) % kProgramNumRegs];
            const std::uint32_t value =
                program_random_math(mix[src_a], mix[src_b], rng.next());
            mix[dst] = program_random_merge(mix[dst], value, rng.next());
        }
    }

    return mix;
}

__device__ inline std::array<std::uint8_t, 32> production_final_digest(
    const std::array<std::uint8_t, 32>& header_digest,
    const std::array<std::uint8_t, 32>& program_seed,
    const std::uint8_t* cache,
    std::uint32_t cache_nodes) {
    std::array<std::uint32_t, kFinalMixWords> mix{};
    for (std::size_t i = 0; i < mix.size(); ++i) mix[i] = kFinalFNVOffsetBasis;

    const std::uint64_t work_seed = header_work_seed(header_digest);
    for (std::uint32_t lane = 0; lane < kFinalNumLanes; ++lane) {
        const auto lane_mix =
            production_lane_mix(work_seed, lane, program_seed, cache, cache_nodes);
        const std::uint32_t word = lane % kFinalMixWords;
        mix[word] = final_fnv1a(mix[word], reduce_lane(lane_mix));
    }

    std::array<std::uint8_t, 90> input{};
    const std::uint8_t domain[26] = {
        'S', 'U', 'D', 'H', 'A', 'R', 'M', 'A', '-', 'G', 'P', 'U', '-',
        'P', 'O', 'W', '-', 'V', '1', '-', 'F', 'I', 'N', 'A', 'L', 0u,
    };
    for (std::size_t i = 0; i < 26u; ++i) input[i] = domain[i];
    for (std::size_t i = 0; i < header_digest.size(); ++i) input[26u + i] = header_digest[i];
    for (std::size_t word = 0; word < mix.size(); ++word) {
        const std::uint32_t value = mix[word];
        const std::size_t offset = 58u + word * 4u;
        input[offset] = static_cast<std::uint8_t>(value);
        input[offset + 1u] = static_cast<std::uint8_t>(value >> 8u);
        input[offset + 2u] = static_cast<std::uint8_t>(value >> 16u);
        input[offset + 3u] = static_cast<std::uint8_t>(value >> 24u);
    }
    return final_sha256_90(input);
}

// Production-consensus search keeps the legacy 8-node vector kernel untouched.
// It consumes the exact node-issued program seed and 16 MiB epoch cache.
__global__ void khushi_production_search_kernel(
    SearchJob job,
    const std::uint8_t* cache,
    std::uint32_t cache_nodes,
    const std::uint32_t* stale_generation,
    std::uint32_t expected_generation,
    unsigned long long* found_nonce,
    unsigned long long* hashes_done) {
    if (cache == nullptr || cache_nodes == 0u) return;
    if (stale_generation != nullptr && *stale_generation != expected_generation) return;

    const std::uint64_t offset =
        static_cast<std::uint64_t>(blockIdx.x) * blockDim.x + threadIdx.x;
    if (offset >= job.nonce_count) return;

    const std::uint64_t nonce = job.nonce_start + offset;
    const auto header = search_header_digest(job, nonce);
    const auto digest = production_final_digest(header, job.program_seed, cache, cache_nodes);

    if (stale_generation != nullptr && *stale_generation != expected_generation) return;
    atomicAdd(hashes_done, 1ull);
    if (search_meets_target(digest, job.target)) {
        atomicMin(found_nonce, static_cast<unsigned long long>(nonce));
    }
}

#endif  // __CUDACC__

}  // namespace sudharma::gpupowv1
