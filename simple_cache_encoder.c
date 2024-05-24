#include <stdio.h>
#include <stdlib.h>

#define READ_SAMPLES_BUFFER_SIZE 16
#define SAMPLE_SIZE_BYTES 2 // uint16

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

    // copy WAV header as is. 44B.
    for (int i = 0; i < 44; i++)
    {
        fputc(fgetc(fptr_from), fptr_to);
    }

    int count = 0;
    int read = 0;
    uint16_t in_sample_buffer[READ_SAMPLES_BUFFER_SIZE];
    while ((read = fread(in_sample_buffer, SAMPLE_SIZE_BYTES, READ_SAMPLES_BUFFER_SIZE, fptr_from)) > 0)
    {
        for (int i = 0; i < read; i++)
        {
            uint16_t out_buffer[1];
            out_buffer[0] = in_sample_buffer[i];
            count++;

            fwrite(out_buffer, sizeof out_buffer[0], 1, fptr_to);
        }
    }

    fprintf(stderr, "num_samples=%d\n", count);

    // TODO
    // read 16B
    // add to buffer
    // if buffer too large, drain buffer
    // draining buffer as before
    // encoding vs non-encoding segments
    // marker
    // non-encoding segment just pass through + increment cache + reorder cache
    // encoding segment get index + encode index + write to file + reorder cache

    fclose(fptr_from);
    fclose(fptr_to);

    return 0;
}
