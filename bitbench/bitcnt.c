#include "bitcnt.h"
#include "libpopcnt.h"

#define HASH_DIFF 64

bool blob_similar(uint8_t *a1, uint8_t *a2, size_t n) {
    uint64_t *d1 = (uint64_t *)a1;
    uint64_t *d2 = (uint64_t *)a2;
    uint_fast16_t total = 0;
    for (uint_fast8_t i = 0; i < n / 8; i++) {
        total += popcnt64(*d1 ^ *d2);
        if (total > HASH_DIFF) {
            return false;
        }
        d1++;
        d2++;
    }
    a1 = (uint8_t *)d1;
    a2 = (uint8_t *)d2;
    uint64_t temp = 0;
    for (uint_fast8_t i = 0; i < n % 8; i++) {
        temp <<= 8;
        temp |= a1[i] ^ a2[i];
    }
    return total + popcnt64(temp) <= HASH_DIFF;
}

bool blob_similar_alt(uint8_t *a1, uint8_t *a2, size_t n) {
    uint64_t *d1 = (uint64_t *)a1;
    uint64_t *d2 = (uint64_t *)a2;
    uint_fast16_t total = 0;
    for (uint_fast8_t i = 0; i < n / 8; i++) {
        total += popcnt64(*d1 ^ *d2);
        if (total > HASH_DIFF) {
            return false;
        }
        d1++;
        d2++;
    }
    a1 = (uint8_t *)d1;
    a2 = (uint8_t *)d2;
    for (uint_fast8_t i = 0; i < n % 8; i++) {
        uint8_t temp = a1[i] ^ a2[i];
        total += popcnt64(temp);
    }
    return total <= HASH_DIFF;
}

bool blob_similar_alt2(uint8_t *a1, uint8_t *a2, size_t n) {
    uint64_t *d1 = (uint64_t *)a1;
    uint64_t *d2 = (uint64_t *)a2;
    uint_fast16_t total = 0;
    for (uint_fast8_t i = 0; i < n / 8; i++) {
        total += popcnt64(*d1 ^ *d2);
        if (total > HASH_DIFF) {
            return false;
        }
        d1++;
        d2++;
    }
    a1 = (uint8_t *)d1;
    a2 = (uint8_t *)d2;
    for (uint_fast8_t i = 0; i < n % 8; i++) {
        uint8_t temp = a1[i] ^ a2[i];
        total += popcnt64(temp);
        if (total > HASH_DIFF) {
            return false;
        }
    }
    return total <= HASH_DIFF;
}
