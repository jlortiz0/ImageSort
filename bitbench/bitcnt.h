#pragma once

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

bool blob_similar(uint8_t *a1, uint8_t *a2, size_t n);

bool blob_similar_alt2(uint8_t *a1, uint8_t *a2, size_t n);

bool blob_similar_alt(uint8_t *a1, uint8_t *a2, size_t n);
