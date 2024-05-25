#include <stdio.h>
#include <stdlib.h>

#include "simple_cache_encoder_cache.h"

typedef bool (*test_case)(void);

bool test_basic()
{
    Cache *cache = cache_new(10);
    if (cache == NULL)
    {
        return false;
    }
    cache_free(cache);
    return true;
}

bool test_cache_index()
{
    Cache *cache = cache_new(1024);
    if (cache == NULL)
    {
        return false;
    }
    int index = cache_index(cache, 0);
    if (index != -1)
    {
        fprintf(stderr, "0: expect index 0, got index=%d\n", index);
        log_cache_info_vals(stderr, cache);
        return false;
    }

    cache_add(cache, 56);
    index = cache_index(cache, 56);
    if (index != 0)
    {
        fprintf(stderr, "1: expect index become 0, got index=%d\n", index);
        log_cache_info_vals(stderr, cache);
        return false;
    }

    cache_add(cache, 57);
    index = cache_index(cache, 57);
    if (index != 1)
    {
        fprintf(stderr, "2: expect index become 1, got index=%d\n", index);
        log_cache_info_vals(stderr, cache);
        return false;
    }

    cache_add(cache, 57);
    cache_add(cache, 57);
    index = cache_index(cache, 57);
    if (index != 0)
    {
        fprintf(stderr, "3: expect index become 0, got index=%d\n", index);
        log_cache_info_vals(stderr, cache);
        return false;
    }

    cache_free(cache);
    return true;
}

bool test_cache_pop()
{
    Cache *cache = cache_new(5);
    if (cache == NULL)
    {
        return false;
    }
    int index;

    cache_add(cache, 1);
    cache_add(cache, 2);
    cache_add(cache, 3);
    cache_add(cache, 4);
    cache_add(cache, 5);

    cache_add(cache, 1);
    cache_add(cache, 2);
    cache_add(cache, 3);
    cache_add(cache, 4);

    cache_add(cache, 6);

    // evict last one, add new to end
    index = cache_index(cache, 6);
    if (index != 4)
    {
        fprintf(stderr, "1: expect index become 4, got index=%d\n", index);
        log_cache_info_vals(stderr, cache);
        return false;
    }

    // move back to front
    cache_add(cache, 6);
    cache_add(cache, 6);

    index = cache_index(cache, 6);
    if (index != 0)
    {
        fprintf(stderr, "2: expect index become 0, got index=%d\n", index);
        log_cache_info_vals(stderr, cache);
        return false;
    }

    cache_add(cache, 3);

    // move in middle
    index = cache_index(cache, 3);
    if (index != 1)
    {
        fprintf(stderr, "3: expect index become 1, got index=%d\n", index);
        log_cache_info_vals(stderr, cache);
        return false;
    }

    cache_free(cache);
    return true;
}

int main()
{
    test_case tests[] = {
        test_basic,
        test_cache_index,
        test_cache_pop,
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
