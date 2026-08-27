// Sudharma GPU-PoW v1 pre-activation OpenCL compatibility kernel.
//
// This file is intentionally isolated from active consensus. CI executes it
// through a CPU OpenCL runtime first; the same arithmetic is intended to be
// portable to AMD/NVIDIA OpenCL devices after the full vector gate is built.

typedef struct {
    uint z;
    uint w;
    uint jsr;
    uint jcong;
} sudharma_kiss99_state;

static uint sudharma_kiss99_next(__private sudharma_kiss99_state *state) {
    state->z = 36969u * (state->z & 0xffffu) + (state->z >> 16);
    state->w = 18000u * (state->w & 0xffffu) + (state->w >> 16);
    state->jcong = 69069u * state->jcong + 1234567u;

    state->jsr ^= state->jsr << 17;
    state->jsr ^= state->jsr >> 13;
    state->jsr ^= state->jsr << 5;

    return (((state->z << 16) + state->w) ^ state->jcong) + state->jsr;
}

__kernel void gpupow_v1_kiss99_probe(
    const uint z,
    const uint w,
    const uint jsr,
    const uint jcong,
    __global uint *outputs
) {
    if (get_global_id(0) != 0) {
        return;
    }

    sudharma_kiss99_state state;
    state.z = z;
    state.w = w;
    state.jsr = jsr;
    state.jcong = jcong;

    for (uint i = 0; i < 5u; i++) {
        outputs[i] = sudharma_kiss99_next(&state);
    }
}
