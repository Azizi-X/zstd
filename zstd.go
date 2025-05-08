package zstd

/*
#cgo LDFLAGS: -lzstd
#include <stdlib.h>
#include <zstd.h>

typedef struct {
    ZSTD_DCtx*     dctx;
    void*          outBuf;
    size_t         outCap;
    ZSTD_outBuffer out;
} ZstdDCtxWithBuffer;

static ZstdDCtxWithBuffer* zstd_create_ctx(size_t multiplier) {
    size_t outCap = ZSTD_DStreamOutSize() * multiplier;
    ZstdDCtxWithBuffer* ctx = malloc(sizeof(*ctx));
    if (!ctx) return NULL;
    ctx->dctx = ZSTD_createDCtx();
    ctx->outBuf  = malloc(outCap);
    ctx->outCap  = outCap;
    ctx->out.dst  = ctx->outBuf;
    ctx->out.size = outCap;
    ctx->out.pos  = 0;
    return ctx;
}

static void zstd_free_ctx(ZstdDCtxWithBuffer* ctx) {
    if (!ctx) return;
    ZSTD_freeDCtx(ctx->dctx);
    free(ctx->outBuf);
    free(ctx);
}

static size_t zstd_stream_decompress(ZstdDCtxWithBuffer* ctx,
    const void* src, size_t srcSize, size_t offset,
    size_t* outLen, size_t* newOffset, int* done)
{
    ZSTD_inBuffer in = { src, srcSize, offset };
    ctx->out.pos = 0;
    size_t ret = ZSTD_decompressStream(ctx->dctx, &ctx->out, &in);
    if (ZSTD_isError(ret)) return (size_t)-1;

    *outLen    = ctx->out.pos;
    *newOffset = in.pos;
    *done      = (ret == 0 || ((in.pos == in.size) && ctx->out.pos == 0)) ? 1 : 0;
    return ret;
}
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

var (
	maxValue = C.size_t(^uint(0))
)

type ZstdDecoder struct {
	ctx *C.ZstdDCtxWithBuffer
}

func NewZstdDecoder(multiplier int) (*ZstdDecoder, error) {
	ctx := C.zstd_create_ctx(C.size_t(multiplier))
	if ctx == nil {
		return nil, errors.New("zstd: failed to create context")
	}
	return &ZstdDecoder{ctx: ctx}, nil
}

func (d *ZstdDecoder) Close() {
	C.zstd_free_ctx(d.ctx)
}

func (d *ZstdDecoder) Decompress(data []byte) ([]byte, error) {
	var results []byte
	var offset int

	for {
		chunk, done, err := d.streamDecompress(data, &offset)
		if err != nil {
			return nil, fmt.Errorf("zstd: decompression error: %s", err)
		}

		if len(chunk) > 0 {
			results = append(results, chunk...)
		}

		if done {
			return results, nil
		}
	}
}

func (d *ZstdDecoder) streamDecompress(data []byte, offset *int) (chunk []byte, done bool, err error) {
	if *offset >= len(data) {
		return nil, true, nil
	}

	var outLen C.size_t
	var newOff C.size_t
	var cdone C.int

	ret := C.zstd_stream_decompress(
		d.ctx,
		unsafe.Pointer(&data[0]),
		C.size_t(len(data)),
		C.size_t(*offset),
		&outLen,
		&newOff,
		&cdone,
	)

	*offset = int(newOff)

	if ret == maxValue {
		return nil, false, errors.New("zstd: decompression error")
	}

	chunk = unsafe.Slice((*byte)(d.ctx.outBuf), outLen)
	return chunk, cdone != 0, nil
}
