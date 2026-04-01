// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project
// ISO/IEC 25000 | ISO/IEC/IEEE 15288:2023

/*
 * Unit tests for eipc_easy convenience layer.
 * Tests NULL-safety, lifecycle failure paths, and stats.
 */

#include "eipc_easy.h"
#include <stdio.h>
#include <string.h>
#include <stdlib.h>

static int pass_count = 0;
static int fail_count = 0;

#define ASSERT_EQ(a, b, msg) do { \
    if ((a) == (b)) { pass_count++; } \
    else { fail_count++; fprintf(stderr, "FAIL: %s (got %d, expected %d)\n", msg, (int)(a), (int)(b)); } \
} while (0)

#define ASSERT_NULL(p, msg) do { \
    if ((p) == NULL) { pass_count++; } \
    else { fail_count++; fprintf(stderr, "FAIL: %s (expected NULL)\n", msg); } \
} while (0)

#define ASSERT_ZERO(v, msg) do { \
    if ((v) == 0) { pass_count++; } \
    else { fail_count++; fprintf(stderr, "FAIL: %s (expected 0, got %llu)\n", msg, (unsigned long long)(v)); } \
} while (0)

/* ── NULL-safety: client API ── */
static void test_client_null_safety(void) {
    eipc_easy_client_t *out = NULL;
    eipc_message_t msg;
    eipc_ack_event_t ack;

    ASSERT_EQ(eipc_easy_connect(NULL, "key", "svc", &out), EIPC_ERR_INVALID,
              "connect: NULL addr");
    ASSERT_EQ(eipc_easy_connect("addr", NULL, "svc", &out), EIPC_ERR_INVALID,
              "connect: NULL hmac_key");
    ASSERT_EQ(eipc_easy_connect("addr", "key", NULL, &out), EIPC_ERR_INVALID,
              "connect: NULL service_id");
    ASSERT_EQ(eipc_easy_connect("addr", "key", "svc", NULL), EIPC_ERR_INVALID,
              "connect: NULL out");

    ASSERT_EQ(eipc_easy_send_intent(NULL, "test", 0.5f), EIPC_ERR_INVALID,
              "send_intent: NULL handle");
    ASSERT_EQ(eipc_easy_send_tool_request(NULL, "tool", NULL, 0), EIPC_ERR_INVALID,
              "send_tool_request: NULL handle");
    ASSERT_EQ(eipc_easy_send_chat(NULL, NULL), EIPC_ERR_INVALID,
              "send_chat: NULL handle");
    ASSERT_EQ(eipc_easy_send_complete(NULL, "prompt", "sess"), EIPC_ERR_INVALID,
              "send_complete: NULL handle");
    ASSERT_EQ(eipc_easy_receive(NULL, &msg), EIPC_ERR_INVALID,
              "receive: NULL handle");
    ASSERT_EQ(eipc_easy_recv_ack(NULL, &ack), EIPC_ERR_INVALID,
              "recv_ack: NULL handle");

    eipc_easy_close(NULL);
    pass_count++;
}

/* ── NULL-safety: server API ── */
static void test_server_null_safety(void) {
    eipc_easy_server_t *out = NULL;
    eipc_intent_event_t intent;
    eipc_chat_request_t chat;

    ASSERT_EQ(eipc_easy_listen(NULL, "key", &out), EIPC_ERR_INVALID,
              "listen: NULL addr");
    ASSERT_EQ(eipc_easy_listen("addr", NULL, &out), EIPC_ERR_INVALID,
              "listen: NULL hmac_key");
    ASSERT_EQ(eipc_easy_listen("addr", "key", NULL), EIPC_ERR_INVALID,
              "listen: NULL out");

    ASSERT_EQ(eipc_easy_accept(NULL), EIPC_ERR_INVALID,
              "accept: NULL handle");
    ASSERT_EQ(eipc_easy_recv_intent(NULL, &intent), EIPC_ERR_INVALID,
              "recv_intent: NULL handle");
    ASSERT_EQ(eipc_easy_recv_chat(NULL, &chat), EIPC_ERR_INVALID,
              "recv_chat: NULL handle");
    ASSERT_EQ(eipc_easy_send_ack(NULL, "rid", "ok"), EIPC_ERR_INVALID,
              "send_ack: NULL handle");
    ASSERT_EQ(eipc_easy_send_chat_response(NULL, "s", "r", "m", 0), EIPC_ERR_INVALID,
              "send_chat_response: NULL handle");

    eipc_easy_close_server(NULL);
    pass_count++;
}

/* ── Client connect failure path ── */
static void test_client_connect_failure(void) {
    eipc_easy_client_t *h = NULL;
    eipc_status_t rc = eipc_easy_connect("tcp://127.0.0.1:0", "testkey", "svc", &h);

    /* Transport connect to port 0 or unresolvable should fail */
    if (rc != EIPC_OK) {
        ASSERT_NULL(h, "connect failure: out should be NULL");
    } else {
        /* If connect somehow succeeded (unlikely), clean up */
        eipc_easy_close(h);
        pass_count++;
    }
}

/* ── Server listen failure path ── */
static void test_server_listen_failure(void) {
    eipc_easy_server_t *h = NULL;
    eipc_status_t rc = eipc_easy_listen("tcp://127.0.0.1:0", "testkey", &h);

    if (rc != EIPC_OK) {
        ASSERT_NULL(h, "listen failure: out should be NULL");
    } else {
        eipc_easy_close_server(h);
        pass_count++;
    }
}

/* ── Stats zero-initialized ── */
static void test_client_stats_null(void) {
    eipc_easy_stats_t s = eipc_easy_client_stats(NULL);
    ASSERT_ZERO(s.sent_count, "null client stats: sent_count");
    ASSERT_ZERO(s.received_count, "null client stats: received_count");
    ASSERT_ZERO(s.ack_count, "null client stats: ack_count");
    ASSERT_ZERO(s.error_count, "null client stats: error_count");
}

static void test_server_stats_null(void) {
    eipc_easy_stats_t s = eipc_easy_server_stats(NULL);
    ASSERT_ZERO(s.sent_count, "null server stats: sent_count");
    ASSERT_ZERO(s.received_count, "null server stats: received_count");
    ASSERT_ZERO(s.ack_count, "null server stats: ack_count");
    ASSERT_ZERO(s.error_count, "null server stats: error_count");
}

int main(void) {
    printf("=== eipc_easy unit tests ===\n");

    test_client_null_safety();
    test_server_null_safety();
    test_client_connect_failure();
    test_server_listen_failure();
    test_client_stats_null();
    test_server_stats_null();

    printf("\nResults: %d passed, %d failed\n", pass_count, fail_count);
    return fail_count > 0 ? 1 : 0;
}
