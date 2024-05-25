#ifndef SIMPLE_CACHE_ENCODER_CACHE
#define SIMPLE_CACHE_ENCODER_CACHE

#include <stdio.h>
#include <stdlib.h>

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
    cache->size = size;
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

uint16_t cache_at(Cache *cache, int i) { return cache->order[i].key; }

int cache_is_full(Cache *cache) { return cache->order_size == cache->size; }

void cache_pop(Cache *cache)
{
    if (cache->order_size > 0)
    {
        cache->order_size--;
    }
}

void cache_add(Cache *cache, uint16_t v)
{
    int idx = cache_index(cache, v);
    if (idx >= 0)
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

    if (idx_new >= idx)
    {
        return;
    }

    // shift entries between [idx_new, idx-1] to the right
    for (int i = idx; i > idx_new; i--)
    {
        cache->order[i] = cache->order[i - 1];
    }
    cache->order[idx_new].count = current_count;
    cache->order[idx_new].key = v;
}

void log_cache_info(FILE *fptr, Cache *cache) { fprintf(fptr, "cache: size=%d order_size=%d\n", cache->size, cache->order_size); }

void log_cache_info_vals(FILE *fptr, Cache *cache)
{
    fprintf(fptr, "cache: vals: ");
    for (int i = 0; i < cache->order_size; i++)
    {
        fprintf(fptr, "%d:%d ", cache->order[i].key, cache->order[i].count);
    }
    fprintf(fptr, "\n");
}

#endif
