#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>

#include "simple_cache_encoder_marker.h"

typedef bool (*test_case)(void);

bool test_basic_encoded()
{
    Marker s = {
        .count = 10,
        .encoding_size = 7,
        .is_encoded = true,
    };
    uint16_t v = encode_marker(s);

    Marker d;
    if (decode_marker(&d, v) != 0)
    {
        fprintf(stderr, "cannot decode\n");
        return false;
    }

    if (!(marker_eq(s, d)))
    {
        fprintf(stderr, "exp != got\n");
        log_marker(stderr, s);
        log_marker(stderr, d);
        return false;
    }

    return true;
}

bool test_basic_not_encoded()
{
    Marker s = {
        .count = 10,
        .is_encoded = false,
    };
    uint16_t v = encode_marker(s);

    Marker d;
    if (decode_marker(&d, v) != 0)
    {
        fprintf(stderr, "cannot decode\n");
        return false;
    }

    if (!(marker_eq(s, d)))
    {
        fprintf(stderr, "exp != got\n");
        log_marker(stderr, s);
        log_marker(stderr, d);
        return false;
    }

    return true;
}

bool test_basic_not_encoded_max()
{
    Marker s = {
        .count = 127,
        .is_encoded = false,
    };
    uint16_t v = encode_marker(s);

    Marker d;
    if (decode_marker(&d, v) != 0)
    {
        fprintf(stderr, "cannot decode\n");
        return false;
    }

    if (!(marker_eq(s, d)))
    {
        fprintf(stderr, "exp != got\n");
        log_marker(stderr, s);
        log_marker(stderr, d);
        return false;
    }

    return true;
}

int main()
{
    test_case tests[] = {
        test_basic_encoded,
        test_basic_not_encoded,
        test_basic_not_encoded_max,
    };
    int num_tests = sizeof(tests) / sizeof(tests[0]);
    printf("tests: num_tests=%d\n", num_tests);

    int num_ok = 0;
    for (int i = 0; i < num_tests; i++)
    {
        bool ok = tests[i]();
        printf("tc=%d: %s\n", i, ok ? "ok" : "fail");
        if (ok)
        {
            num_ok++;
        }
    }

    printf("tests: %s\n", num_tests == num_ok ? "ok" : "fail");
    return !(num_ok == num_tests);
}
