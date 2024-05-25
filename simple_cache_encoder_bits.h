#ifndef SIMPLE_CACHE_ENCODER_BITS
#define SIMPLE_CACHE_ENCODER_BITS

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>

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
    {.max_key_index = (1 << 4) - 1, .packed_len = 1, .unpacked_len = 2, .encoding_size = 4},
    {},
    {.max_key_index = (1 << 6) - 1, .packed_len = 3, .unpacked_len = 4, .encoding_size = 6},
    {.max_key_index = (1 << 7) - 1, .packed_len = 7, .unpacked_len = 8, .encoding_size = 7},
    {},
};

void pack_2x4(uint8_t *buffer, int size, int encoding_size, uint8_t *packed_buffer, int *packed_size)
{
    packed_buffer[0] = (buffer[1] << 4) | (0x0F & buffer[0]);
    *packed_size = 1;
}

void unpack_2x4(uint8_t *buffer, int size, int encoding_size, uint8_t *packed_buffer, int *packed_size)
{
    buffer[0] = 0x0F & packed_buffer[0];
    buffer[1] = packed_buffer[0] >> 4;
    *packed_size = 2;
}

void pack_4x6(uint8_t *buffer, int size, int encoding_size, uint8_t *packed_buffer, int *packed_size)
{
    packed_buffer[0] = buffer[1] | ((buffer[0] << 2) & 0xC0);
    packed_buffer[1] = buffer[2] | ((buffer[0] << 4) & 0xC0);
    packed_buffer[2] = buffer[3] | ((buffer[0] << 6) & 0xC0);
    *packed_size = 3;
}

void unpack_4x6(uint8_t *buffer, int size, int encoding_size, uint8_t *packed_buffer, int *packed_size)
{
    uint8_t v0 = 0;
    for (int i = 0; i < 3; i++)
    {
        v0 |= (packed_buffer[i] & 0xC0) >> ((i + 1) * 2);
    }
    buffer[0] = v0;
    buffer[1] = packed_buffer[0] & 0x3F;
    buffer[2] = packed_buffer[1] & 0x3F;
    buffer[3] = packed_buffer[2] & 0x3F;
    *packed_size = 4;
}

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
    case 4:
        pack_2x4(buffer, size, encoding_size, packed_buffer, packed_size);
        break;
    case 6:
        pack_4x6(buffer, size, encoding_size, packed_buffer, packed_size);
        break;
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
    case 4:
        unpack_2x4(buffer, size, encoding_size, packed_buffer, packed_size);
        break;
    case 6:
        unpack_4x6(buffer, size, encoding_size, packed_buffer, packed_size);
        break;
    case 7:
        unpack_8x7(buffer, size, encoding_size, packed_buffer, packed_size);
        break;
    default:
        fprintf(stderr, "error: encoding_size=%d\n", encoding_size);
        exit(1);
    }
}

#endif
