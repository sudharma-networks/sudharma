#define SUDHARMA_GPU_POW_V1_REFERENCE_HEADER 1
#define SUDHARMA_GPU_POW_V1_FINAL 1
#define KHUSHI_NUM_REGS 32
#define KHUSHI_NUM_LANES 16
#define KHUSHI_DAG_ROUNDS 64
#define KHUSHI_DAG_LOADS 4
#define KHUSHI_CACHE_ACCESSES 11
#define KHUSHI_MATH_OPERATIONS 18
#define KHUSHI_DATASET_PARENTS 512
#define KHUSHI_FNV_OFFSET 0x811c9dc5u
#define KHUSHI_FNV_PRIME 0x01000193u
#define KHUSHI_MAX_HEADER 256u

typedef struct {
    uint z;
    uint w;
    uint jsr;
    uint jcong;
} khushi_kiss99;

inline uint khushi_fnv1a(uint a, uint b) { return (a ^ b) * KHUSHI_FNV_PRIME; }
inline uint khushi_fnv1(uint a, uint b) { return (a * KHUSHI_FNV_PRIME) ^ b; }

inline uint khushi_kiss99_next(__private khushi_kiss99* s) {
    s->z = 36969u * (s->z & 0xffffu) + (s->z >> 16u);
    s->w = 18000u * (s->w & 0xffffu) + (s->w >> 16u);
    s->jcong = 69069u * s->jcong + 1234567u;
    s->jsr ^= s->jsr << 17u;
    s->jsr ^= s->jsr >> 13u;
    s->jsr ^= s->jsr << 5u;
    return (((s->z << 16u) + s->w) ^ s->jcong) + s->jsr;
}

inline khushi_kiss99 khushi_program_rng(uint seed_lo, uint seed_hi) {
    uint z = khushi_fnv1a(KHUSHI_FNV_OFFSET, seed_lo);
    uint w = khushi_fnv1a(z, seed_hi);
    uint jsr = khushi_fnv1a(w, seed_lo);
    uint jcong = khushi_fnv1a(jsr, seed_hi);
    khushi_kiss99 out = {z, w, jsr, jcong};
    return out;
}

inline uint khushi_le32_private(__private const uchar* p) {
    return ((uint)p[0]) | ((uint)p[1] << 8u) | ((uint)p[2] << 16u) | ((uint)p[3] << 24u);
}

inline uint khushi_le32_global(__global const uchar* p) {
    return ((uint)p[0]) | ((uint)p[1] << 8u) | ((uint)p[2] << 16u) | ((uint)p[3] << 24u);
}

inline void khushi_put_le32(__private uchar* p, uint value) {
    p[0] = (uchar)value;
    p[1] = (uchar)(value >> 8u);
    p[2] = (uchar)(value >> 16u);
    p[3] = (uchar)(value >> 24u);
}

inline ulong khushi_rotl64(ulong value, uint shift) {
    shift &= 63u;
    return shift == 0u ? value : ((value << shift) | (value >> (64u - shift)));
}

inline void khushi_keccak_f1600(__private ulong state[25]) {
    const ulong rc[24] = {
        0x0000000000000001UL, 0x0000000000008082UL, 0x800000000000808aUL, 0x8000000080008000UL,
        0x000000000000808bUL, 0x0000000080000001UL, 0x8000000080008081UL, 0x8000000000008009UL,
        0x000000000000008aUL, 0x0000000000000088UL, 0x0000000080008009UL, 0x000000008000000aUL,
        0x000000008000808bUL, 0x800000000000008bUL, 0x8000000000008089UL, 0x8000000000008003UL,
        0x8000000000008002UL, 0x8000000000000080UL, 0x000000000000800aUL, 0x800000008000000aUL,
        0x8000000080008081UL, 0x8000000000008080UL, 0x0000000080000001UL, 0x8000000080008008UL
    };
    const uint rotations[25] = {
        0u,1u,62u,28u,27u,36u,44u,6u,55u,20u,3u,10u,43u,25u,39u,41u,45u,15u,21u,8u,18u,2u,61u,56u,14u
    };
    for (uint round = 0; round < 24u; ++round) {
        ulong c[5], d[5], b[25];
        for (uint x = 0; x < 5u; ++x)
            c[x] = state[x] ^ state[x+5u] ^ state[x+10u] ^ state[x+15u] ^ state[x+20u];
        for (uint x = 0; x < 5u; ++x)
            d[x] = c[(x+4u)%5u] ^ khushi_rotl64(c[(x+1u)%5u], 1u);
        for (uint y = 0; y < 5u; ++y)
            for (uint x = 0; x < 5u; ++x) state[x+5u*y] ^= d[x];
        for (uint i = 0; i < 25u; ++i) b[i] = 0UL;
        for (uint y = 0; y < 5u; ++y) {
            for (uint x = 0; x < 5u; ++x) {
                uint source = x + 5u*y;
                uint tx = y;
                uint ty = (2u*x + 3u*y) % 5u;
                b[tx + 5u*ty] = khushi_rotl64(state[source], rotations[source]);
            }
        }
        for (uint y = 0; y < 5u; ++y)
            for (uint x = 0; x < 5u; ++x)
                state[x+5u*y] = b[x+5u*y] ^ ((~b[(x+1u)%5u+5u*y]) & b[(x+2u)%5u+5u*y]);
        state[0] ^= rc[round];
    }
}

inline void khushi_keccak512_64(__private const uchar input[64], __private uchar output[64]) {
    ulong state[25];
    for (uint i = 0; i < 25u; ++i) state[i] = 0UL;
    for (uint i = 0; i < 64u; ++i) state[i/8u] ^= ((ulong)input[i]) << ((i%8u)*8u);
    state[8] ^= (ulong)0x01u;
    state[8] ^= ((ulong)0x80u) << 56u;
    khushi_keccak_f1600(state);
    for (uint i = 0; i < 64u; ++i) output[i] = (uchar)(state[i/8u] >> ((i%8u)*8u));
}

inline void khushi_dataset_item(__global const uchar* cache, uint cache_nodes, uint index, __private uchar out[64]) {
    uint base = (index % cache_nodes) * 64u;
    for (uint i = 0; i < 64u; ++i) out[i] = cache[base+i];
    khushi_put_le32(out, khushi_le32_private(out) ^ index);
    uchar hashed[64];
    khushi_keccak512_64(out, hashed);
    for (uint i = 0; i < 64u; ++i) out[i] = hashed[i];
    for (uint parent = 0; parent < KHUSHI_DATASET_PARENTS; ++parent) {
        uint mix_word = khushi_le32_private(out + (parent % 16u) * 4u);
        uint selector = khushi_fnv1(index ^ parent, mix_word);
        uint parent_base = (selector % cache_nodes) * 64u;
        for (uint word = 0; word < 16u; ++word) {
            uint a = khushi_le32_private(out + word*4u);
            uint b = khushi_le32_global(cache + parent_base + word*4u);
            khushi_put_le32(out + word*4u, khushi_fnv1(a, b));
        }
    }
    khushi_keccak512_64(out, hashed);
    for (uint i = 0; i < 64u; ++i) out[i] = hashed[i];
}

inline uint khushi_rotl32(uint v, uint a) { uint s=a&31u; return s==0u?v:rotate(v,s); }
inline uint khushi_rotr32(uint v, uint a) { uint s=a&31u; return s==0u?v:rotate(v,32u-s); }
inline uint khushi_mul_hi32(uint a, uint b) { return mul_hi(a,b); }
inline uint khushi_clz32(uint v) { return v==0u?32u:clz(v); }
inline uint khushi_popcount32(uint v) { return popcount(v); }

inline uint khushi_random_math(uint a, uint b, uint selector) {
    switch (selector % 11u) {
        case 0u: return a+b;
        case 1u: return a*b;
        case 2u: return khushi_mul_hi32(a,b);
        case 3u: return min(a,b);
        case 4u: return khushi_rotl32(a,b);
        case 5u: return khushi_rotr32(a,b);
        case 6u: return a&b;
        case 7u: return a|b;
        case 8u: return a^b;
        case 9u: return khushi_clz32(a)+khushi_clz32(b);
        default: return khushi_popcount32(a)+khushi_popcount32(b);
    }
}

inline uint khushi_random_merge(uint a, uint b, uint selector) {
    uint x=((selector>>16u)%31u)+1u;
    switch (selector%4u) {
        case 0u: return a*33u+b;
        case 1u: return (a^b)*33u;
        case 2u: return khushi_rotl32(a,x)^b;
        default: return khushi_rotr32(a,x)^b;
    }
}

inline void khushi_register_permutations(uint seed_lo, uint seed_hi, __private uint first[32], __private uint second[32]) {
    khushi_kiss99 rng = khushi_program_rng(seed_lo, seed_hi);
    for (uint i=0;i<32u;++i) { first[i]=i; second[i]=i; }
    for (uint i=31u;i>0u;--i) {
        uint j=khushi_kiss99_next(&rng)%(i+1u); uint t=first[i]; first[i]=first[j]; first[j]=t;
        j=khushi_kiss99_next(&rng)%(i+1u); t=second[i]; second[i]=second[j]; second[j]=t;
    }
}

inline void khushi_init_lane(ulong work_seed, uint lane, __private uint mix[32]) {
    uint lo=(uint)work_seed, hi=(uint)(work_seed>>32u);
    khushi_kiss99 rng = {
        khushi_fnv1a(KHUSHI_FNV_OFFSET,lo),
        khushi_fnv1a(KHUSHI_FNV_OFFSET,hi),
        khushi_fnv1a(KHUSHI_FNV_OFFSET,lane),
        khushi_fnv1a(KHUSHI_FNV_OFFSET,lane)
    };
    for (uint i=0;i<32u;++i) mix[i]=khushi_kiss99_next(&rng);
}

inline void khushi_lane_mix(ulong work_seed, uint lane, __private const uchar program_seed[32], __global const uchar* cache, uint cache_nodes, __private uint mix[32]) {
    khushi_init_lane(work_seed,lane,mix);
    uint seed_lo=khushi_le32_private(program_seed), seed_hi=khushi_le32_private(program_seed+4u);
    uint first[32], second[32];
    khushi_register_permutations(seed_lo,seed_hi,first,second);
    khushi_kiss99 rng=khushi_program_rng(seed_lo,seed_hi);
    for (uint round=0;round<KHUSHI_DAG_ROUNDS;++round) {
        uint dag_index=khushi_fnv1(mix[0]^round,lane^khushi_kiss99_next(&rng));
        uchar dag[64]; khushi_dataset_item(cache,cache_nodes,dag_index,dag);
        for(uint load=0;load<KHUSHI_DAG_LOADS;++load){
            uint dst=first[(round*KHUSHI_DAG_LOADS+load)%32u];
            mix[dst]=khushi_random_merge(mix[dst],khushi_le32_private(dag+load*4u),khushi_kiss99_next(&rng));
        }
        for(uint access=0;access<KHUSHI_CACHE_ACCESSES;++access){
            uint src=second[(round+access)%32u];
            uint selector=mix[src]^khushi_kiss99_next(&rng);
            uint cache_node=selector%cache_nodes;
            uint cache_word=khushi_le32_global(cache+cache_node*64u+((selector>>16u)%16u)*4u);
            uint dst=first[(round+KHUSHI_DAG_LOADS+access)%32u];
            mix[dst]=khushi_random_merge(mix[dst],cache_word,khushi_kiss99_next(&rng));
        }
        for(uint op=0;op<KHUSHI_MATH_OPERATIONS;++op){
            uint dst=first[(round+KHUSHI_DAG_LOADS+KHUSHI_CACHE_ACCESSES+op)%32u];
            uint sa=second[(round+op)%32u], sb=second[(round+op+1u)%32u];
            uint value=khushi_random_math(mix[sa],mix[sb],khushi_kiss99_next(&rng));
            mix[dst]=khushi_random_merge(mix[dst],value,khushi_kiss99_next(&rng));
        }
    }
}

inline uint khushi_sha_rotr(uint x,uint n){return (x>>n)|(x<<(32u-n));}

inline void khushi_sha256_compress(__private uint h[8], __private const uchar block[64]) {
    const uint k[64]={
        0x428a2f98u,0x71374491u,0xb5c0fbcfu,0xe9b5dba5u,0x3956c25bu,0x59f111f1u,0x923f82a4u,0xab1c5ed5u,
        0xd807aa98u,0x12835b01u,0x243185beu,0x550c7dc3u,0x72be5d74u,0x80deb1feu,0x9bdc06a7u,0xc19bf174u,
        0xe49b69c1u,0xefbe4786u,0x0fc19dc6u,0x240ca1ccu,0x2de92c6fu,0x4a7484aau,0x5cb0a9dcu,0x76f988dau,
        0x983e5152u,0xa831c66du,0xb00327c8u,0xbf597fc7u,0xc6e00bf3u,0xd5a79147u,0x06ca6351u,0x14292967u,
        0x27b70a85u,0x2e1b2138u,0x4d2c6dfcu,0x53380d13u,0x650a7354u,0x766a0abbu,0x81c2c92eu,0x92722c85u,
        0xa2bfe8a1u,0xa81a664bu,0xc24b8b70u,0xc76c51a3u,0xd192e819u,0xd6990624u,0xf40e3585u,0x106aa070u,
        0x19a4c116u,0x1e376c08u,0x2748774cu,0x34b0bcb5u,0x391c0cb3u,0x4ed8aa4au,0x5b9cca4fu,0x682e6ff3u,
        0x748f82eeu,0x78a5636fu,0x84c87814u,0x8cc70208u,0x90befffau,0xa4506cebu,0xbef9a3f7u,0xc67178f2u};
    uint w[64];
    for(uint i=0;i<16u;++i){uint j=i*4u;w[i]=((uint)block[j]<<24u)|((uint)block[j+1u]<<16u)|((uint)block[j+2u]<<8u)|(uint)block[j+3u];}
    for(uint i=16u;i<64u;++i){uint s0=khushi_sha_rotr(w[i-15u],7u)^khushi_sha_rotr(w[i-15u],18u)^(w[i-15u]>>3u);uint s1=khushi_sha_rotr(w[i-2u],17u)^khushi_sha_rotr(w[i-2u],19u)^(w[i-2u]>>10u);w[i]=w[i-16u]+s0+w[i-7u]+s1;}
    uint a=h[0],b=h[1],c=h[2],d=h[3],e=h[4],f=h[5],g=h[6],hh=h[7];
    for(uint i=0;i<64u;++i){uint s1=khushi_sha_rotr(e,6u)^khushi_sha_rotr(e,11u)^khushi_sha_rotr(e,25u);uint ch=(e&f)^((~e)&g);uint t1=hh+s1+ch+k[i]+w[i];uint s0=khushi_sha_rotr(a,2u)^khushi_sha_rotr(a,13u)^khushi_sha_rotr(a,22u);uint maj=(a&b)^(a&c)^(b&c);uint t2=s0+maj;hh=g;g=f;f=e;e=d+t1;d=c;c=b;b=a;a=t1+t2;}
    h[0]+=a;h[1]+=b;h[2]+=c;h[3]+=d;h[4]+=e;h[5]+=f;h[6]+=g;h[7]+=hh;
}

inline void khushi_sha256_private(__private const uchar* data,uint len,__private uchar digest[32]){
    uint h[8]={0x6a09e667u,0xbb67ae85u,0x3c6ef372u,0xa54ff53au,0x510e527fu,0x9b05688cu,0x1f83d9abu,0x5be0cd19u};
    uint full=len/64u;
    for(uint blockIndex=0;blockIndex<full;++blockIndex){uchar block[64];for(uint i=0;i<64u;++i)block[i]=data[blockIndex*64u+i];khushi_sha256_compress(h,block);}
    uchar tail[128];for(uint i=0;i<128u;++i)tail[i]=0u;uint rem=len%64u;for(uint i=0;i<rem;++i)tail[i]=data[full*64u+i];tail[rem]=0x80u;uint blocks=(rem<=55u)?1u:2u;ulong bits=(ulong)len*8UL;uint end=blocks*64u;for(uint i=0;i<8u;++i)tail[end-1u-i]=(uchar)(bits>>(i*8u));
    for(uint bi=0;bi<blocks;++bi){uchar block[64];for(uint i=0;i<64u;++i)block[i]=tail[bi*64u+i];khushi_sha256_compress(h,block);}
    for(uint i=0;i<8u;++i){digest[i*4u]=(uchar)(h[i]>>24u);digest[i*4u+1u]=(uchar)(h[i]>>16u);digest[i*4u+2u]=(uchar)(h[i]>>8u);digest[i*4u+3u]=(uchar)h[i];}
}

inline void khushi_header_digest(__global const uchar* header,uint header_len,ulong nonce,__private uchar digest[32]){
    const uchar domain[38]={'S','U','D','H','A','R','M','A','-','G','P','U','-','P','O','W','-','V','1','-','R','E','F','E','R','E','N','C','E','-','H','E','A','D','E','R',0,0};
    uchar input[304];uint pos=0u;
    for(uint i=0;i<37u;++i)input[pos++]=domain[i];
    for(uint i=0;i<header_len;++i)input[pos++]=header[i];
    for(uint i=0;i<8u;++i)input[pos++]=(uchar)(nonce>>(i*8u));
    khushi_sha256_private(input,pos,digest);
}

inline ulong khushi_work_seed(__private const uchar digest[32]){ulong seed=0UL;for(uint i=0;i<8u;++i)seed|=((ulong)digest[i])<<(i*8u);return seed;}

inline void khushi_final_digest(__private const uchar header_digest[32],__private const uchar program_seed[32],__global const uchar* cache,uint cache_nodes,__private uchar out[32]){
    uint group[8];for(uint i=0;i<8u;++i)group[i]=KHUSHI_FNV_OFFSET;
    ulong work_seed=khushi_work_seed(header_digest);
    for(uint lane=0;lane<KHUSHI_NUM_LANES;++lane){uint mix[32];khushi_lane_mix(work_seed,lane,program_seed,cache,cache_nodes,mix);uint ld=KHUSHI_FNV_OFFSET;for(uint r=0;r<32u;++r)ld=khushi_fnv1a(ld,mix[r]);uint word=lane%8u;group[word]=khushi_fnv1a(group[word],ld);}
    uchar input[90];const uchar domain[26]={'S','U','D','H','A','R','M','A','-','G','P','U','-','P','O','W','-','V','1','-','F','I','N','A','L',0};
    for(uint i=0;i<26u;++i)input[i]=domain[i];for(uint i=0;i<32u;++i)input[26u+i]=header_digest[i];for(uint w=0;w<8u;++w)khushi_put_le32(input+58u+w*4u,group[w]);
    khushi_sha256_private(input,90u,out);
}

inline int khushi_meets_target(__private const uchar digest[32],__global const uchar* target){for(uint i=0;i<32u;++i){if(digest[i]<target[i])return 1;if(digest[i]>target[i])return 0;}return 1;}

__kernel void khushi_vector(__global const uchar* header,uint header_len,ulong nonce,__global const uchar* program_seed_global,__global const uchar* cache,uint cache_nodes,__global uchar* output){
    if(get_global_id(0)!=0u||header_len>KHUSHI_MAX_HEADER||cache_nodes==0u)return;uchar seed[32];for(uint i=0;i<32u;++i)seed[i]=program_seed_global[i];uchar hd[32],fd[32];khushi_header_digest(header,header_len,nonce,hd);khushi_final_digest(hd,seed,cache,cache_nodes,fd);for(uint i=0;i<32u;++i)output[i]=fd[i];
}

__kernel void khushi_search(__global const uchar* header,uint header_len,__global const uchar* program_seed_global,__global const uchar* cache,uint cache_nodes,__global const uchar* target,ulong nonce_start,ulong nonce_count,volatile __global uint* stale_generation,uint expected_generation,volatile __global uint* found_flag,volatile __global ulong* found_nonce,volatile __global uint* hashes_done){
    ulong gid=(ulong)get_global_id(0);if(gid>=nonce_count||header_len>KHUSHI_MAX_HEADER||cache_nodes==0u)return;if(*stale_generation!=expected_generation)return;ulong nonce=nonce_start+gid;if(nonce<nonce_start)return;uchar seed[32];for(uint i=0;i<32u;++i)seed[i]=program_seed_global[i];uchar hd[32],fd[32];khushi_header_digest(header,header_len,nonce,hd);khushi_final_digest(hd,seed,cache,cache_nodes,fd);if(*stale_generation!=expected_generation)return;atomic_inc(hashes_done);if(khushi_meets_target(fd,target)){if(atomic_cmpxchg(found_flag, 0u, 1u)==0u){*found_nonce = nonce;}}
}
