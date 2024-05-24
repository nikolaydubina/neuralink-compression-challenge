#ifndef SIMPLE_CACHE_ENCODER_CACHE
#define SIMPLE_CACHE_ENCODER_CACHE

#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>

typedef struct
{
    uint16_t key;
    int count;
} cache_entry;

typedef struct
{
    cache_entry *order;
    int order_size;
    int size;
} Cache;

Cache *cache_new(int size)
{
    Cache *cache = (Cache *)malloc(sizeof(Cache));
    cache->order = (cache_entry *)calloc(size, sizeof(cache_entry));
    cache->order_size = 0;
    return cache;
}

void cache_free(Cache *s)
{
    if (s == NULL)
    {
        return;
    }
    free(s->order);
    free(s);
}

int cache_index(Cache *cache, uint16_t v)
{
    for (int i = 0; i < cache->order_size; i++)
    {
        if (cache->order[i].key == v)
        {
            return i;
        }
    }
    return -1;
}

int cache_is_full(Cache *cache)
{
    return cache->order_size >= cache->size;
}

void cache_pop(Cache *cache)
{
    if (cache->order_size == 0)
    {
        return;
    }
    cache->order_size--;
}

void cache_add(Cache *cache, uint16_t v)
{
    int idx = cache_index(cache, v);
    if (idx > 0)
    {
        cache->order[idx].count++;
    }
    else
    {
        if (cache_is_full(cache))
        {
            cache_pop(cache);
        }
        cache->order_size++;
        idx = cache->order_size - 1;
        cache->order[idx].key = v;
        cache->order[idx].count = 1;
    }

    // find new position
    int current_count = cache->order[idx].count;
    int idx_new;
    for (idx_new = idx - 1; idx_new >= 0 && cache->order[idx_new].count < current_count; idx_new--)
    {
    }
    idx_new++;

    // shift all entries to right by one
    for (int i = idx_new + 1; i <= idx; i++)
    {
        cache->order[idx] = cache->order[idx - 1];
    }
    cache->order[idx_new].count = current_count;
    cache->order[idx_new].key = v;
}

uint16_t cache_at(Cache *cache, int i)
{
    return cache->order[i].key;
}

#endif
