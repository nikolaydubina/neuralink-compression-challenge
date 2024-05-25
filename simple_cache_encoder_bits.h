#ifndef SIMPLE_CACHE_ENCODER_BITS
#define SIMPLE_CACHE_ENCODER_BITS

#include <stdio.h>
#include <stdlib.h>

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

void pack_8x7(uint8_t *buffer, int size, int encoding_size, uint8_t *packed_buffer, int *packed_size)
{
    packed_buffer[0] = buffer[1] | ((buffer[0] << 1) & 0x80);
    packed_buffer[1] = buffer[2] | ((buffer[0] << 2) & 0x80);
    packed_buffer[2] = buffer[3] | ((buffer[0] << 3) & 0x80);
    packed_buffer[3] = buffer[4] | ((buffer[0] << 4) & 0x80);
    packed_buffer[4] = buffer[5] | ((buffer[0] << 5) & 0x80);
    packed_buffer[5] = buffer[6] | ((buffer[0] << 6) & 0x80);
    packed_buffer[6] = buffer[7] | ((buffer[0] << 7) & 0x80);
    *packed_size = 7;
}

void unpack_8x7(uint8_t *buffer, int size, int encoding_size, uint8_t *packed_buffer, int *packed_size)
{
    uint8_t v0 = 0;
    for (int i = 0; i < 7; i++)
    {
        v0 |= (packed_buffer[i] & 0x80) >> (i + 1);
    }
    buffer[0] = v0;
    buffer[1] = packed_buffer[0] & 0x7F;
    buffer[2] = packed_buffer[1] & 0x7F;
    buffer[3] = packed_buffer[2] & 0x7F;
    buffer[4] = packed_buffer[3] & 0x7F;
    buffer[5] = packed_buffer[4] & 0x7F;
    buffer[6] = packed_buffer[5] & 0x7F;
    buffer[7] = packed_buffer[6] & 0x7F;
    *packed_size = 8;
}

void pack(uint8_t *buffer, int size, int encoding_size, uint8_t *packed_buffer, int *packed_size)
{
    switch (encoding_size)
    {
    case 7:
        pack_8x7(buffer, size, encoding_size, packed_buffer, packed_size);
        break;
    default:
        fprintf(stderr, "error: encoding_size=%d\n", encoding_size);
        exit(1);
    }
}

void unpack(uint8_t *buffer, int size, int encoding_size, uint8_t *packed_buffer, int *packed_size)
{
    switch (encoding_size)
    {
    case 7:
        unpack_8x7(buffer, size, encoding_size, packed_buffer, packed_size);
        break;
    default:
        fprintf(stderr, "error: encoding_size=%d\n", encoding_size);
        exit(1);
    }
}

#endif
