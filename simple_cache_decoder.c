#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>

#include "simple_cache_encoder_bits.h"
#include "simple_cache_encoder_cache.h"
#include "simple_cache_encoder_marker.h"

#define SAMPLE_SIZE_BYTES 2                  // uint16
#define WAV_HEADER_SIZE_BYTES 44             // standard WAV header
#define ENCODED_SEQ_MAX_LEN (1 << 13) - 1    // marker has 13bits to encode count, this is used in buffer
#define NOT_ENCODED_SEQ_MAX_LEN (1 << 7) - 1 // intentionally small to trigger attempt to encode
#define CACHE_SIZE 1 << 10                   // to fit all possible samples, encoded key space is less or equal to this

int read_into_buffer(uint16_t *buffer, int size, FILE *fptr_from, Cache *cache)
{
    Marker marker;
    uint16_t marker_bytes;
    int i;
    for (i = 0; i < size;)
    {
        if (fread(&marker_bytes, sizeof marker_bytes, 1, fptr_from) <= 0)
        {
            break;
        }
        if (decode_marker(&marker, marker_bytes) == EOF)
        {
            break;
        }

        if (marker.is_encoded)
        {
            uint8_t packed[8] = {0};
            uint8_t unpacked[8] = {0};
            int unpacked_size;
            for (int j = 0; j < marker.count; j += Packers[marker.encoding_size].unpacked_len)
            {
                fread(packed, sizeof packed[0], Packers[marker.encoding_size].packed_len, fptr_from);
                unpack(unpacked, Packers[marker.encoding_size].unpacked_len, Packers[marker.encoding_size].encoding_size, packed, &unpacked_size);
                for (int k = 0; k < Packers[marker.encoding_size].unpacked_len; k++)
                {
                    buffer[i + j + k] = cache_at(cache, unpacked[k]);
                    cache_add(cache, buffer[i + j + k]);
                }
            }
        }
        else
        {
            fread(buffer + i, sizeof buffer[0], marker.count, fptr_from);
            for (int j = 0; j < marker.count; j++)
            {
                cache_add(cache, buffer[i + j]);
            }
        }

        i += marker.count;
    }
    return i;
}

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

    int read = 0;
    uint16_t buffer[ENCODED_SEQ_MAX_LEN];
    while ((read = read_into_buffer(buffer, ENCODED_SEQ_MAX_LEN, fptr_from, cache)) > 0)
    {
        fwrite(buffer, sizeof buffer[0], read, fptr_to);
    }

    fclose(fptr_from);
    fclose(fptr_to);

    return 0;
}
