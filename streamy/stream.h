#pragma once

#include <libavutil/buffer.h>
#include <stdint.h>
#include <stdbool.h>

typedef struct LibavReader LibavReader;

int libavreader_new(const char *fName, LibavReader **ptr, char *user_agent, bool loop);

int libavreader_next(LibavReader *l, uint8_t *buf);

typedef struct {
    int x;
    int y;
} pair_int;

pair_int libavreader_dimensions(const LibavReader *l);

void libavreader_destroy(LibavReader *l);

float libavreader_fps(const LibavReader *l);
