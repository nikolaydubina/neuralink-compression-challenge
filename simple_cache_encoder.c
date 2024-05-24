#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>

#include "simple_cache_encoder_cache.h"
#include "simple_cache_encoder_marker.h"

#define SAMPLE_SIZE_BYTES 2                  // uint16
#define WAV_HEADER_SIZE_BYTES 44             // standard WAV header
#define ENCODED_SEQ_MAX_LEN (1 << 13) - 1    // marker has 13bits to encode count, this is used in buffer
#define NOT_ENCODED_SEQ_MAX_LEN (1 << 7) - 1 // intentionally small to trigger attempt to encode
#define CACHE_SIZE 1 << 10                   // to fit all possible samples, encoded key space is less or equal to this

uint8_t encode_one(Cache *cache, uint16_t v, int encoding_size);
int count_flush_buffer_hits(uint16_t *buffer, int size, int *encoding_size, Cache *cache);
int count_flush_buffer_not_hits(uint16_t *buffer, int size, Cache *cache);
void flush_buffer_hits(uint16_t *buffer, int size, int count, int encoding_size, Cache *cache, FILE *fptr_to);
void flush_buffer_not_hits(uint16_t *buffer, int size, int count, Cache *cache, FILE *fptr_to);

struct Packer
{
    int max_key_index;
    int packed_len;
    int unpacked_len;
    int encoding_size;
};
typedef struct Packer Packer;

static struct Packer Packers[9] = {
    {}, // empty, zero-padding
    {},
    {},
    {},
    {}, // TODO: 4bit packer
    {},
    {},                                                                                      // TODO: 6bit packer
    {.max_key_index = (1 << 7) - 1, .packed_len = 7, .unpacked_len = 8, .encoding_size = 7}, // 7bit packer
    {},
};

void drain_buffer(uint16_t *buffer, int size, Cache *cache, FILE *fptr_to)
{
    int encoding_size;
    for (int count_hits = 0, count_not_hits = 0; size > 0; size -= count_hits + count_not_hits)
    {
        count_hits = count_flush_buffer_hits(buffer, size, &encoding_size, cache);
        count_not_hits = count_flush_buffer_not_hits(buffer + count_hits, size - count_hits, cache);

        // there samples to flush, but they are not hits,
        // and if they are hits they can not be encoded.
        // flush them unencoded.
        if (count_hits == 0 && count_not_hits == 0)
        {
            count_not_hits = Packers[7].unpacked_len > size ? size : Packers[7].unpacked_len;
        }

        flush_buffer_hits(buffer, size, count_hits, encoding_size, cache, fptr_to);
        flush_buffer_not_hits(buffer + count_hits, size - count_hits, count_not_hits, cache, fptr_to);

        buffer += count_hits + count_not_hits;
    }
}

int count_flush_buffer_hits(uint16_t *buffer, int size, int *encoding_size, Cache *cache)
{
    int encoded_count_by_packer[8] = {0};

    for (int encoding_size = 1; encoding_size <= 7; encoding_size++)
    {
        if (Packers[encoding_size].max_key_index == 0)
        {
            continue;
        }

        int count = 0;
        for (int j = 0; j < size && cache_index(cache, buffer[j]) >= 0 && cache_index(cache, buffer[j]) <= Packers[encoding_size].max_key_index; j++)
        {
            count++;
        }
        count = count - (count % Packers[encoding_size].unpacked_len);
    }

    int best_encoding_size = -1, min_num_bytes = 0;
    for (int encoding_size = 1; encoding_size <= 7; encoding_size++)
    {
        int num_bytes = encoded_count_by_packer[encoding_size] * Packers[encoding_size].encoding_size;
        if (num_bytes > 0 && (min_num_bytes == 0 || num_bytes < min_num_bytes))
        {
            best_encoding_size = encoding_size;
            min_num_bytes = num_bytes;
        }
    }

    *encoding_size = best_encoding_size;
    return encoded_count_by_packer[best_encoding_size];
}

int count_flush_buffer_not_hits(uint16_t *buffer, int size, Cache *cache)
{
    int count = 0;
    for (int i = 0; i < size && cache_index(cache, buffer[i]) < 0; i++)
    {
        count++;
    }
    return count > NOT_ENCODED_SEQ_MAX_LEN ? NOT_ENCODED_SEQ_MAX_LEN : count;
}

uint8_t encode_one(Cache *cache, uint16_t v, int encoding_size)
{

    int i = cache_index(cache, v);
    if (i < 0 || i > Packers[encoding_size].max_key_index)
    {
        fprintf(stderr, "value(%d) got index(%d) is out of bound for encoded key, expected [0, %d]", v, i, Packers[encoding_size].max_key_index);
        exit(1);
    }
    return (uint8_t)(i);
}

void flush_buffer_hits(uint16_t *buffer, int size, int count, int encoding_size, Cache *cache, FILE *fptr_to)
{
    if ((count = count < size ? count : size) <= 0)
    {
        return;
    }

    Marker marker = {
        .count = count,
        .is_encoded = true,
        .encoding_size = encoding_size,
    };
    uint16_t v = encode_marker(marker);

    uint16_t marker_bytes[] = {encode_marker(marker)};
    fwrite(marker_bytes, sizeof marker_bytes[0], 1, fptr_to);

    // TODO: heap? malloc dynamic packed/unpacked len.
    uint8_t unpacked[8] = {0};
    uint8_t packed[7] = {0};
    for (int i = 0; i < count; i += Packers[encoding_size].unpacked_len)
    {
        for (int j = 0; j < Packers[encoding_size].unpacked_len; j++)
        {
            unpacked[j] = encode_one(cache, Packers[encoding_size].encoding_size, buffer[i + j]);
        }

        fwrite(packed, sizeof packed[0], Packers[encoding_size].packed_len, fptr_to);
        cache_add(cache, buffer[i]);
    }
}

void flush_buffer_not_hits(uint16_t *buffer, int size, int count, Cache *cache, FILE *fptr_to)
{
    if ((count = count < size ? count : size) <= 0)
    {
        return;
    }

    Marker marker = {
        .count = count,
        .is_encoded = false,
    };
    uint16_t v = encode_marker(marker);

    uint16_t marker_bytes[] = {encode_marker(marker)};
    fwrite(marker_bytes, sizeof marker_bytes[0], 1, fptr_to);
    fwrite(buffer, sizeof buffer[0], count, fptr_to);

    for (int i = 0; i < count; i++)
    {
        cache_add(cache, buffer[i]);
    }
}

// this code lacks error handling, WAV header checks, but ok for prototype
int main(int argc, char *argv[])
{
    FILE *fptr_from, *fptr_to;

    if ((fptr_from = fopen(argv[1], "r")) == NULL)
    {
        return 1;
    }

    if ((fptr_to = fopen(argv[2], "w")) == NULL)
    {
        return 1;
    }

    // copy WAV header as is
    uint8_t wave_header[WAV_HEADER_SIZE_BYTES] = {0};
    fread(wave_header, sizeof wave_header[0], WAV_HEADER_SIZE_BYTES, fptr_from);
    fwrite(wave_header, sizeof wave_header[0], WAV_HEADER_SIZE_BYTES, fptr_to);

    Cache *cache = cache_new(CACHE_SIZE);

    int count = 0;
    int read = 0;
    uint16_t buffer[ENCODED_SEQ_MAX_LEN];
    while ((read = fread(buffer, SAMPLE_SIZE_BYTES, ENCODED_SEQ_MAX_LEN, fptr_from)) > 0)
    {
        fprintf(stderr, "num_samples_buffer_read=%d\n", read);
        drain_buffer(buffer, read, cache, fptr_to);
    }

    fprintf(stderr, "num_samples=%d\n", count);

    fclose(fptr_from);
    fclose(fptr_to);

    return 0;
}
