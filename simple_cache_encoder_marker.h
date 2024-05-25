#ifndef SIMPLE_CACHE_ENCODER_MARKER
#define SIMPLE_CACHE_ENCODER_MARKER

#include <stdlib.h>
#include <stdbool.h>

#define MARKER_MAX_COUNT (1 << 13) - 1 // we have space only for int14

typedef struct
{
    int encoding_size;
    int count;
    bool is_encoded;
} Marker;

uint16_t encode_marker(Marker s);

static uint16_t encoding_size_to_marker[] = {0, 0, 0, 0, 0, 0, 1, 2, 0};
static uint16_t encoding_size_from_marker[] = {4, 6, 7};

bool marker_eq(Marker a, Marker b) { return a.encoding_size == b.encoding_size && a.count == b.count && a.is_encoded == b.is_encoded; }

void log_marker(FILE *out, Marker s)
{
    uint16_t v = encode_marker(s);
    fprintf(out, "marker(%d): encoding_size=%d count=%d is_encoded=%d\n", v, s.encoding_size, s.count, s.is_encoded);
}

uint16_t encode_marker(Marker s)
{
    if (s.count > MARKER_MAX_COUNT)
    {
        fprintf(stderr, "error: count(%d)>%d\n", s.count, MARKER_MAX_COUNT);
        exit(1);
    }

    uint16_t m = (uint16_t)(s.is_encoded ? (int16_t)(s.count) : -(int16_t)(s.count));
    return (m << 2) | encoding_size_to_marker[s.encoding_size];
};

int decode_marker(Marker *s, uint16_t v)
{
    if (v == 0)
    {
        return EOF;
    }
    s->encoding_size = encoding_size_from_marker[v & 3];

    // restore two-s complement
    // remove encoding bits
    bool is_negative = (v & 0x8000) != 0;
    v >>= 2;
    if (is_negative)
    {
        v |= (1 << 15);
        v |= (1 << 14);
    }

    int16_t count = (int16_t)(v);

    s->is_encoded = count >= 0;
    if (!s->is_encoded)
    {
        s->encoding_size = 0;
    }

    s->count = count >= 0 ? (int)(count) : -(int)(count);

    return 0;
};

#endif
