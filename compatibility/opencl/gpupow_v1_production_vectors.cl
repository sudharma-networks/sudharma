__kernel void khushi_production_vectors(
    __global const uchar* cache,
    uint cache_nodes,
    __global const uint* indices,
    uint count,
    __global uchar* output) {
    const uint vector = (uint)get_global_id(0);
    if (vector >= count) return;

    uchar item[64];
    khushi_dataset_item(cache, cache_nodes, indices[vector], item);
    for (uint byte = 0u; byte < 64u; ++byte) {
        output[(size_t)vector * 64u + byte] = item[byte];
    }
}
