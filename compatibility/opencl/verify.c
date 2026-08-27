#define CL_TARGET_OPENCL_VERSION 120
#include <CL/cl.h>
#include <stdio.h>
#include <stdlib.h>

static void check(cl_int err, const char *where) {
    if (err != CL_SUCCESS) {
        fprintf(stderr, "%s: OpenCL error %d\n", where, err);
        exit(1);
    }
}

static char *read_source(const char *path, size_t *size) {
    FILE *f = fopen(path, "rb");
    if (!f) return NULL;
    if (fseek(f, 0, SEEK_END) != 0) { fclose(f); return NULL; }
    long n = ftell(f);
    if (n < 0) { fclose(f); return NULL; }
    rewind(f);
    char *p = malloc((size_t)n + 1u);
    if (!p) { fclose(f); return NULL; }
    *size = fread(p, 1, (size_t)n, f);
    fclose(f);
    if (*size != (size_t)n) { free(p); return NULL; }
    p[*size] = '\0';
    return p;
}

int main(int argc, char **argv) {
    if (argc != 2) return 2;
    size_t source_size = 0;
    char *source = read_source(argv[1], &source_size);
    if (!source) { fprintf(stderr, "cannot read kernel\n"); return 1; }

    cl_int err;
    cl_platform_id platform;
    cl_device_id device;
    cl_uint count = 0;
    check(clGetPlatformIDs(1, &platform, &count), "clGetPlatformIDs");
    if (count == 0) { fprintf(stderr, "no OpenCL platform\n"); return 1; }
    check(clGetDeviceIDs(platform, CL_DEVICE_TYPE_ALL, 1, &device, &count), "clGetDeviceIDs");
    if (count == 0) { fprintf(stderr, "no OpenCL device\n"); return 1; }

    cl_context context = clCreateContext(NULL, 1, &device, NULL, NULL, &err);
    check(err, "clCreateContext");
    cl_command_queue queue = clCreateCommandQueue(context, device, 0, &err);
    check(err, "clCreateCommandQueue");
    const char *sources[] = {source};
    const size_t sizes[] = {source_size};
    cl_program program = clCreateProgramWithSource(context, 1, sources, sizes, &err);
    check(err, "clCreateProgramWithSource");
    err = clBuildProgram(program, 1, &device, "-cl-std=CL1.2", NULL, NULL);
    if (err != CL_SUCCESS) {
        size_t log_size = 0;
        clGetProgramBuildInfo(program, device, CL_PROGRAM_BUILD_LOG, 0, NULL, &log_size);
        char *log = calloc(log_size + 1u, 1u);
        if (log) {
            clGetProgramBuildInfo(program, device, CL_PROGRAM_BUILD_LOG, log_size, log, NULL);
            fprintf(stderr, "%s\n", log);
            free(log);
        }
        check(err, "clBuildProgram");
    }

    cl_kernel kernel = clCreateKernel(program, "gpupow_v1_kiss99_probe", &err);
    check(err, "clCreateKernel");
    cl_mem output = clCreateBuffer(context, CL_MEM_WRITE_ONLY, 5u * sizeof(cl_uint), NULL, &err);
    check(err, "clCreateBuffer");

    cl_uint z = 362436069u, w = 521288629u, jsr = 123456789u, jcong = 380116160u;
    check(clSetKernelArg(kernel, 0, sizeof(z), &z), "arg0");
    check(clSetKernelArg(kernel, 1, sizeof(w), &w), "arg1");
    check(clSetKernelArg(kernel, 2, sizeof(jsr), &jsr), "arg2");
    check(clSetKernelArg(kernel, 3, sizeof(jcong), &jcong), "arg3");
    check(clSetKernelArg(kernel, 4, sizeof(output), &output), "arg4");

    size_t global = 1;
    check(clEnqueueNDRangeKernel(queue, kernel, 1, NULL, &global, NULL, 0, NULL, NULL), "clEnqueueNDRangeKernel");
    cl_uint got[5] = {0};
    check(clEnqueueReadBuffer(queue, output, CL_TRUE, 0, sizeof(got), got, 0, NULL, NULL), "clEnqueueReadBuffer");
    const cl_uint want[5] = {769445856u, 742012328u, 2121196314u, 2805620942u, 3214428071u};
    for (size_t i = 0; i < 5u; i++) {
        if (got[i] != want[i]) {
            fprintf(stderr, "KISS99 mismatch %zu: got %u want %u\n", i, got[i], want[i]);
            return 1;
        }
    }
    puts("OpenCL KISS99 interoperability passed");

    clReleaseMemObject(output);
    clReleaseKernel(kernel);
    clReleaseProgram(program);
    clReleaseCommandQueue(queue);
    clReleaseContext(context);
    free(source);
    return 0;
}
