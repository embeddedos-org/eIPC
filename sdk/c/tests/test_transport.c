// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project
// ISO/IEC 25000 | ISO/IEC/IEEE 15288:2023
/**
 * @file test_transport.c
 * @brief Unit tests for EIPC transport layer
 */
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <assert.h>
#include "eipc_types.h"
#include "eipc.h"

#if !defined(_WIN32)
#include <sys/socket.h>
#include <sys/types.h>
#include <pthread.h>
#include <unistd.h>
#endif

static int passed = 0;
#define PASS(name) do { printf("[PASS] %s\n", name); passed++; } while(0)

/* ---- Tests ---- */
static void test_timestamp_now(void) {
    char buf[EIPC_TIMESTAMP_MAX];
    memset(buf, 0, sizeof(buf));
    eipc_timestamp_now(buf, sizeof(buf));
    assert(strlen(buf) > 0);
    PASS("timestamp_now");
}

static void test_generate_request_id(void) {
    char buf[EIPC_REQUEST_ID_MAX];
    memset(buf, 0, sizeof(buf));
    eipc_generate_request_id(buf, sizeof(buf));
    assert(strlen(buf) > 0);
    PASS("generate_request_id");
}

static void test_request_ids_unique(void) {
    char id1[EIPC_REQUEST_ID_MAX], id2[EIPC_REQUEST_ID_MAX];
    eipc_generate_request_id(id1, sizeof(id1));
    eipc_generate_request_id(id2, sizeof(id2));
    assert(strcmp(id1, id2) != 0);
    PASS("request_ids_unique");
}

static void test_client_init(void) {
    eipc_client_t client;
    memset(&client, 0, sizeof(client));
    eipc_status_t rc = eipc_client_init(&client, "test-service");
    assert(rc == EIPC_OK);
    assert(strcmp(client.service_id, "test-service") == 0);
    assert(client.connected == false);
    assert(client.sequence == 0);
    PASS("client_init");
}

static void test_server_init(void) {
    eipc_server_t server;
    memset(&server, 0, sizeof(server));
    eipc_status_t rc = eipc_server_init(&server);
    assert(rc == EIPC_OK);
    PASS("server_init");
}

static void test_header_json_roundtrip(void) {
    eipc_header_t hdr;
    memset(&hdr, 0, sizeof(hdr));
    strncpy(hdr.service_id, "eai-agent", EIPC_SERVICE_ID_MAX - 1);
    strncpy(hdr.session_id, "sess-001", EIPC_SESSION_ID_MAX - 1);
    strncpy(hdr.request_id, "req-001", EIPC_REQUEST_ID_MAX - 1);
    hdr.sequence = 42;
    hdr.priority = EIPC_PRIORITY_P1;
    strncpy(hdr.capability, "tool_exec", EIPC_CAPABILITY_MAX - 1);

    char json[EIPC_MAX_HEADER];
    size_t written = 0;
    eipc_status_t rc = eipc_header_to_json(&hdr, json, sizeof(json), &written);
    assert(rc == EIPC_OK);
    assert(written == strlen(json));
    assert(strstr(json, "eai-agent") != NULL);
    assert(strstr(json, "sess-001") != NULL);

    eipc_header_t parsed;
    memset(&parsed, 0, sizeof(parsed));
    rc = eipc_header_from_json(json, strlen(json), &parsed);
    assert(rc == EIPC_OK);
    assert(strcmp(parsed.service_id, "eai-agent") == 0);
    assert(parsed.sequence == 42);
    PASS("header_json_roundtrip");
}

static void test_intent_json_roundtrip(void) {
    eipc_intent_event_t ev;
    memset(&ev, 0, sizeof(ev));
    strncpy(ev.intent, "move_cursor", EIPC_INTENT_MAX - 1);
    ev.confidence = 0.95f;
    strncpy(ev.session_id, "sess-002", EIPC_SESSION_ID_MAX - 1);

    char json[1024];
    size_t written = 0;
    eipc_status_t rc = eipc_intent_to_json(&ev, json, sizeof(json), &written);
    assert(rc == EIPC_OK);
    assert(strstr(json, "move_cursor") != NULL);

    eipc_intent_event_t parsed;
    memset(&parsed, 0, sizeof(parsed));
    rc = eipc_intent_from_json(json, strlen(json), &parsed);
    assert(rc == EIPC_OK);
    assert(strcmp(parsed.intent, "move_cursor") == 0);
    assert(parsed.confidence > 0.9f);
    PASS("intent_json_roundtrip");
}

static void test_ack_json_roundtrip(void) {
    eipc_ack_event_t ack;
    memset(&ack, 0, sizeof(ack));
    strncpy(ack.request_id, "req-123", EIPC_REQUEST_ID_MAX - 1);
    strncpy(ack.status, "success", 31);

    char json[512];
    size_t written = 0;
    eipc_status_t rc = eipc_ack_to_json(&ack, json, sizeof(json), &written);
    assert(rc == EIPC_OK);

    eipc_ack_event_t parsed;
    memset(&parsed, 0, sizeof(parsed));
    rc = eipc_ack_from_json(json, strlen(json), &parsed);
    assert(rc == EIPC_OK);
    assert(strcmp(parsed.request_id, "req-123") == 0);
    assert(strcmp(parsed.status, "success") == 0);
    PASS("ack_json_roundtrip");
}

static void test_kv_struct_sizes(void) {
    assert(sizeof(eipc_kv_t) == EIPC_KV_KEY_MAX + EIPC_KV_VALUE_MAX);
    PASS("kv_struct_sizes");
}

static void test_transport_constants(void) {
    assert(EIPC_SERVICE_ID_MAX == 64);
    assert(EIPC_SESSION_ID_MAX == 64);
    assert(EIPC_REQUEST_ID_MAX == 64);
    assert(EIPC_CAPABILITY_MAX == 64);
    assert(EIPC_TIMESTAMP_MAX == 40);
    assert(EIPC_MAX_ARGS == 16);
    PASS("transport_constants");
}

static void test_wire_ceiling_constants(void) {
    /* Shared with Go: MaxFrameSize (1 MiB header + payload) + 16-byte
     * preamble + 32-byte MAC = 1,048,624 bytes on the wire
     * (Go transport/maxEncodedFrameSize). */
    assert(EIPC_MAX_WIRE_FRAME ==
           EIPC_MAX_FRAME + EIPC_FRAME_FIXED_SIZE + EIPC_MAC_SIZE);
    assert(EIPC_MAX_WIRE_FRAME == 1048624);
    PASS("wire_ceiling_constants");
}

#if !defined(_WIN32)
/* Writer thread: pushes a whole buffer into one end of a socketpair so the
 * reader under test can drain it concurrently. A 1 MiB frame does not fit
 * in a socket buffer, so a single-threaded write-then-read would deadlock. */
struct writer_arg {
    int fd;
    const uint8_t *data;
    size_t len;
};

static void *writer_thread(void *arg) {
    struct writer_arg *w = (struct writer_arg *)arg;
    size_t sent = 0;
    while (sent < w->len) {
        ssize_t n = send(w->fd, (const char *)(w->data + sent), w->len - sent, 0);
        assert(n > 0);
        sent += (size_t)n;
    }
    return NULL;
}

/* Build a length-prefixed wire frame: 4-byte big-endian length, then a valid
 * empty-body preamble (magic, version 1, no flags, header_len = 0,
 * payload_len = 0) followed by zero padding to the requested wire length.
 * The padding is ignored by eipc_frame_decode; what this exercises is the
 * receiver's length-prefix ceiling, not a 1 MiB payload — the C struct caps
 * header at 1024 and payload at 4096, so the C side can never emit a
 * max-size frame itself. */
static uint8_t *build_wire_frame(uint32_t wire_len, size_t *out_total) {
    uint8_t *buf = (uint8_t *)malloc((size_t)wire_len + 4);
    assert(buf != NULL);
    buf[0] = (uint8_t)(wire_len >> 24);
    buf[1] = (uint8_t)(wire_len >> 16);
    buf[2] = (uint8_t)(wire_len >> 8);
    buf[3] = (uint8_t)wire_len;
    memset(buf + 4, 0, wire_len);
    buf[4] = 0x45; buf[5] = 0x49; buf[6] = 0x50; buf[7] = 0x43; /* "EIPC" */
    buf[8] = 0; buf[9] = 1; /* protocol version 1 */
    /* msg_type, flags, header_len, payload_len are all zero. */
    *out_total = (size_t)wire_len + 4;
    return buf;
}

/* A wire frame exactly at the shared ceiling (1 MiB header + payload worth
 * of wire length) must be accepted: before the fix the receiver capped the
 * length prefix at EIPC_MAX_FRAME (1 MiB) and rejected it with
 * EIPC_ERR_FRAME_TOO_LARGE. */
static void test_recv_frame_accepts_max_wire_length(void) {
    int sv[2];
    assert(socketpair(AF_UNIX, SOCK_STREAM, 0, sv) == 0);

    size_t total = 0;
    uint8_t *wire = build_wire_frame(EIPC_MAX_WIRE_FRAME, &total);
    struct writer_arg warg = { sv[0], wire, total };
    pthread_t writer;
    assert(pthread_create(&writer, NULL, writer_thread, &warg) == 0);

    eipc_frame_t frame;
    memset(&frame, 0, sizeof(frame));
    eipc_status_t rc = eipc_transport_recv_frame(sv[1], &frame);
    assert(rc == EIPC_OK);
    assert(frame.header_len == 0);
    assert(frame.payload_len == 0);

    pthread_join(writer, NULL);
    free(wire);
    close(sv[0]);
    close(sv[1]);
    PASS("recv_frame_accepts_max_wire_length");
}

/* One byte over the shared ceiling must be rejected before anything is
 * allocated or read for the body — only the length prefix is sent. */
static void test_recv_frame_rejects_one_byte_over_max_wire_length(void) {
    int sv[2];
    assert(socketpair(AF_UNIX, SOCK_STREAM, 0, sv) == 0);

    uint8_t prefix[4];
    uint32_t over = EIPC_MAX_WIRE_FRAME + 1;
    prefix[0] = (uint8_t)(over >> 24);
    prefix[1] = (uint8_t)(over >> 16);
    prefix[2] = (uint8_t)(over >> 8);
    prefix[3] = (uint8_t)over;
    struct writer_arg warg = { sv[0], prefix, sizeof(prefix) };
    pthread_t writer;
    assert(pthread_create(&writer, NULL, writer_thread, &warg) == 0);

    eipc_frame_t frame;
    memset(&frame, 0, sizeof(frame));
    eipc_status_t rc = eipc_transport_recv_frame(sv[1], &frame);
    assert(rc == EIPC_ERR_FRAME_TOO_LARGE);

    pthread_join(writer, NULL);
    close(sv[0]);
    close(sv[1]);
    PASS("recv_frame_rejects_one_byte_over_max_wire_length");
}
#endif /* !defined(_WIN32) */

int main(void) {
    printf("=== eipc Transport Tests ===\n");
    test_timestamp_now();
    test_generate_request_id();
    test_request_ids_unique();
    test_client_init();
    test_server_init();
    test_header_json_roundtrip();
    test_intent_json_roundtrip();
    test_ack_json_roundtrip();
    test_kv_struct_sizes();
    test_transport_constants();
    test_wire_ceiling_constants();
#if !defined(_WIN32)
    test_recv_frame_accepts_max_wire_length();
    test_recv_frame_rejects_one_byte_over_max_wire_length();
#else
    printf("[SKIP] recv_frame wire-ceiling tests (POSIX socketpair only)\n");
#endif
    printf("\n=== ALL %d TESTS PASSED ===\n", passed);
    return 0;
}
