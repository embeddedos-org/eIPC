// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project
/**
 * @file test_timestamp.c
 * @brief eipc_timestamp_now() writes the RFC 3339 UTC form that the Go
 *        endpoints read with time.Parse(time.RFC3339Nano, ...).
 */
#include <stdio.h>
#include <string.h>
#include <time.h>
#include "eipc.h"

static int failed = 0;
static int passed = 0;

#define CHECK(cond, name) do {                                   \
    if (cond) { printf("[PASS] %s\n", name); passed++; }         \
    else      { printf("[FAIL] %s\n", name); failed++; }         \
} while (0)

static int is_digits(const char *s, size_t n) {
    size_t i;
    for (i = 0; i < n; i++) {
        if (s[i] < '0' || s[i] > '9') return 0;
    }
    return 1;
}

/* "YYYY-MM-DDTHH:MM:SSZ" */
static int is_rfc3339_utc(const char *s) {
    return strlen(s) == 20
        && is_digits(s, 4) && s[4] == '-'
        && is_digits(s + 5, 2) && s[7] == '-'
        && is_digits(s + 8, 2) && s[10] == 'T'
        && is_digits(s + 11, 2) && s[13] == ':'
        && is_digits(s + 14, 2) && s[16] == ':'
        && is_digits(s + 17, 2) && s[19] == 'Z';
}

static void test_timestamp_is_rfc3339(void) {
    char buf[EIPC_TIMESTAMP_MAX];
    char year[5];
    time_t now = time(NULL);
    const struct tm *utc = gmtime(&now);

    memset(buf, 0, sizeof(buf));
    eipc_timestamp_now(buf, sizeof(buf));
    printf("  eipc_timestamp_now() = \"%s\"\n", buf);
    CHECK(is_rfc3339_utc(buf), "timestamp is RFC 3339 UTC (YYYY-MM-DDTHH:MM:SSZ)");

    snprintf(year, sizeof(year), "%04d", utc->tm_year + 1900);
    CHECK(strncmp(buf, year, 4) == 0, "timestamp starts with the current UTC year");
}

static void test_timestamp_short_buffer(void) {
    char buf[8];
    memset(buf, 'x', sizeof(buf));
    eipc_timestamp_now(buf, sizeof(buf));
    CHECK(memchr(buf, '\0', sizeof(buf)) != NULL, "short buffer stays NUL-terminated");
}

int main(void) {
    printf("=== EIPC timestamp tests ===\n");
    test_timestamp_is_rfc3339();
    test_timestamp_short_buffer();
    printf("\n%d passed, %d failed\n", passed, failed);
    return failed == 0 ? 0 : 1;
}
