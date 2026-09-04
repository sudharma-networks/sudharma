#pragma once

#include <algorithm>
#include <cctype>
#include <cstddef>
#include <string>
#include <string_view>
#include <vector>

namespace sudharma::gpupowv1::tuning {

enum class Family {
    Generic,
    NvidiaGeneric,
    NvidiaTuring,
    NvidiaAmpere,
    NvidiaAda,
    NvidiaBlackwell,
    AmdGeneric,
};

struct Profile {
    Family family = Family::Generic;
    unsigned native_width = 32u;
};

struct Candidate {
    unsigned local_size = 1u;
    unsigned groups_per_unit = 1u;
};

inline const char* family_name(Family family) {
    switch (family) {
        case Family::NvidiaGeneric: return "nvidia-generic";
        case Family::NvidiaTuring: return "nvidia-turing";
        case Family::NvidiaAmpere: return "nvidia-ampere";
        case Family::NvidiaAda: return "nvidia-ada";
        case Family::NvidiaBlackwell: return "nvidia-blackwell";
        case Family::AmdGeneric: return "amd-generic";
        default: return "generic";
    }
}

inline Profile cuda_profile(int major, int minor) {
    if (major >= 12) return {Family::NvidiaBlackwell, 32u};
    if (major == 8 && minor >= 9) return {Family::NvidiaAda, 32u};
    if (major == 8) return {Family::NvidiaAmpere, 32u};
    if (major == 7 && minor >= 5) return {Family::NvidiaTuring, 32u};
    return {Family::NvidiaGeneric, 32u};
}

inline std::string lower_ascii(std::string_view value) {
    std::string out(value);
    std::transform(out.begin(), out.end(), out.begin(), [](unsigned char c) {
        return static_cast<char>(std::tolower(c));
    });
    return out;
}

inline Profile opencl_profile(std::string_view vendor) {
    const std::string lower = lower_ascii(vendor);
    if (lower.find("advanced micro devices") != std::string::npos ||
        lower.find("amd") != std::string::npos) {
        return {Family::AmdGeneric, 64u};
    }
    if (lower.find("nvidia") != std::string::npos) {
        return {Family::NvidiaGeneric, 32u};
    }
    return {Family::Generic, 32u};
}

inline std::vector<Candidate> candidates(Profile profile, unsigned max_local_size) {
    if (max_local_size == 0u) max_local_size = 1u;

    std::vector<Candidate> preferred;
    if (profile.family == Family::AmdGeneric) {
        preferred = {{64u, 8u}, {128u, 4u}, {256u, 2u}};
    } else if (profile.family == Family::NvidiaGeneric ||
               profile.family == Family::NvidiaTuring ||
               profile.family == Family::NvidiaAmpere ||
               profile.family == Family::NvidiaAda ||
               profile.family == Family::NvidiaBlackwell) {
        preferred = {{32u, 8u}, {64u, 4u}, {128u, 2u}, {256u, 1u}};
    } else {
        preferred = {{32u, 4u}, {64u, 2u}, {128u, 1u}};
    }

    std::vector<Candidate> out;
    for (Candidate candidate : preferred) {
        candidate.local_size = std::min(candidate.local_size, max_local_size);
        if (candidate.local_size == 0u) continue;
        bool duplicate = false;
        for (const Candidate existing : out) {
            if (existing.local_size == candidate.local_size &&
                existing.groups_per_unit == candidate.groups_per_unit) {
                duplicate = true;
                break;
            }
        }
        if (!duplicate) out.push_back(candidate);
    }
    if (out.empty()) out.push_back({max_local_size, 1u});
    return out;
}

inline std::size_t work_items(Candidate candidate, unsigned compute_units) {
    const std::size_t units = compute_units == 0u ? 1u : compute_units;
    return static_cast<std::size_t>(candidate.local_size) *
           units * static_cast<std::size_t>(candidate.groups_per_unit);
}

}  // namespace sudharma::gpupowv1::tuning
