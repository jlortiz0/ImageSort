#define _GNU_SOURCE
#include <stdio.h>
#include <endian.h>
#include <limits.h>
#include <string.h>
#include <sys/stat.h>
#include <errno.h>
#include <stdint.h>
#include <stdlib.h>

typedef struct HashEntry {
    uint8_t *hash;
    int64_t modTime;
    char *name;
} HashEntry;

HashEntry *he_create(char *name, int64_t modTime, uint8_t *hsh) {
    HashEntry *he = (HashEntry *) malloc(sizeof(HashEntry));
    if (he != NULL) {
        he->name = strdup(name);
        he->modTime = modTime;
        he->hash = hsh;
    }
    return he;
}

void he_delete(HashEntry **he) {
    if (he && *he) {
        free((*he)->name);
        free((*he)->hash);
        free(*he);
        *he = NULL;
    }
}

int main(void) {
    FILE *infile = fopen("imgSort.cache", "rb");
    if (infile == NULL) {
        perror("imgSort.cache");
        getchar();
        return 1;
    }
    uint8_t hashSize = getc(infile);
    uint16_t size = hashSize;
    size *= size;
    size /= 8;
    uint32_t entries;
    fread(&entries, sizeof(uint32_t), 1, infile);
    entries = be32toh(entries);
    HashEntry **hashes = calloc(entries, sizeof(HashEntry *));
    char pathbuf[PATH_MAX];
    for (uint32_t i = 0; i < entries; i++) {
        uint16_t strIdx = 0;
        int16_t chr;
        while ((chr = getc(infile)) > 0) {
            pathbuf[strIdx++] = chr;
        }
        pathbuf[strIdx] = 0;
        if (chr == EOF) {
            perror("Error reading from file");
            for (uint32_t j = 0; j < i; j++) {
                he_delete(&hashes[j]);
            }
            free(hashes);
            fclose(infile);
            getchar();
            return 1;
        }
        uint32_t modTimeShort;
        fread(&modTimeShort, sizeof(uint32_t), 1, infile);
        int64_t modTime = be32toh(modTimeShort);
        uint8_t *hsh = malloc(size);
        if (fread(hsh, 1, size, infile) != size) {
            perror("Error reading from file");
            for (uint32_t j = 0; j < i; j++) {
                he_delete(&hashes[j]);
            }
            free(hashes);
            fclose(infile);
            getchar();
            return 1;
        }
        hashes[i] = he_create(pathbuf, modTime, hsh);
    }
    fclose(infile);

    uint32_t removed = 0;
    struct stat statbuf;
    for (uint32_t i = 0; i < entries; i++) {
        if (stat(hashes[i]->name, &statbuf)) {
            if (errno == ENOENT) {
                puts(hashes[i]->name);
                he_delete(&hashes[i]);
                removed++;
            }
        } else if (statbuf.st_mtime != hashes[i]->modTime) {
            if (statbuf.st_mtime + 3600 == hashes[i]->modTime) {
                continue;
            }
            puts(hashes[i]->name);
            he_delete(&hashes[i]);
            removed++;
        }
    }
    if (removed) {
        FILE *outfile = fopen("imgSort.cache", "wb");
        putc(hashSize, outfile);
        uint32_t outEntries = htobe32(entries - removed);
        fwrite(&outEntries, sizeof(uint32_t), 1, outfile);
        for (uint32_t i = 0; i < entries; i++) {
            if (hashes[i] == NULL) {
                continue;
            }
            fputs(hashes[i]->name, outfile);
            putc(0, outfile);
            uint32_t modTimeShort = htobe32((uint32_t) hashes[i]->modTime);
            fwrite(&modTimeShort, sizeof(uint32_t), 1, outfile);
            fwrite(hashes[i]->hash, 1, size, outfile);
        }
        fclose(outfile);
        getchar();
    }
    for (uint32_t j = 0; j < entries; j++) {
        he_delete(&hashes[j]);
    }
    free(hashes);
    return 0;
}