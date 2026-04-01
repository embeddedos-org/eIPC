// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project
// ISO/IEC 25000 | ISO/IEC/IEEE 15288:2023
/**
 * @file test_transport.c
 * @brief Unit tests for EIPC transport layer
 */
#include <stdio.h>
#include <string.h>
#include <assert.h>
#include "eipc_types.h"
#include "eipc.h"

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
    eipc_status_t rc = eipc_header_to_json(&hdr, json, sizeof(json));
    assert(rc == EIPC_OK);
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
    eipc_status_t rc = eipc_intent_to_json(&ev, json, sizeof(json));
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
    eipc_status_t rc = eipc_ack_to_json(&ack, json, sizeof(json));
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
    printf("\n=== ALL %d TESTS PASSED ===\n", passed);
    return 0;
}
