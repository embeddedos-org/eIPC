// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project
// ISO/IEC 25000 | ISO/IEC/IEEE 15288:2023
/**
 * @file test_frame.c
 * @brief Unit tests for EIPC frame encode/decode
 */
#include <stdio.h>
#include <string.h>
#include <assert.h>
#include "eipc_types.h"
#include "eipc.h"

static int passed = 0;
#define PASS(name) do { printf("[PASS] %s\n", name); passed++; } while(0)

/* ---- Tests ---- */
static void test_frame_encode_basic(void) {
    eipc_frame_t frame;
    memset(&frame, 0, sizeof(frame));
    frame.version = EIPC_PROTOCOL_VER;
    frame.msg_type = EIPC_MSG_INTENT;
    frame.flags = 0;
    memcpy(frame.header, "{\"service_id\":\"test\"}", 20);
    frame.header_len = 20;
    memcpy(frame.payload, "{\"intent\":\"click\"}", 18);
    frame.payload_len = 18;

    uint8_t buf[1024];
    size_t out_len = 0;
    eipc_status_t rc = eipc_frame_encode(&frame, buf, sizeof(buf), &out_len);
    assert(rc == EIPC_OK);
    assert(out_len > 0);
    assert(out_len == EIPC_FRAME_FIXED_SIZE + 20 + 18);
    PASS("frame_encode_basic");
}

static void test_frame_decode_roundtrip(void) {
    eipc_frame_t original;
    memset(&original, 0, sizeof(original));
    original.version = EIPC_PROTOCOL_VER;
    original.msg_type = EIPC_MSG_HEARTBEAT;
    original.flags = 0;
    const char *hdr = "{\"service\":\"svc1\"}";
    memcpy(original.header, hdr, strlen(hdr));
    original.header_len = (uint32_t)strlen(hdr);
    const char *pay = "{\"status\":\"ok\"}";
    memcpy(original.payload, pay, strlen(pay));
    original.payload_len = (uint32_t)strlen(pay);

    uint8_t buf[1024];
    size_t out_len = 0;
    eipc_frame_encode(&original, buf, sizeof(buf), &out_len);

    eipc_frame_t decoded;
    memset(&decoded, 0, sizeof(decoded));
    eipc_status_t rc = eipc_frame_decode(buf, out_len, &decoded);
    assert(rc == EIPC_OK);
    assert(decoded.version == EIPC_PROTOCOL_VER);
    assert(decoded.msg_type == EIPC_MSG_HEARTBEAT);
    assert(decoded.header_len == original.header_len);
    assert(decoded.payload_len == original.payload_len);
    assert(memcmp(decoded.header, original.header, original.header_len) == 0);
    assert(memcmp(decoded.payload, original.payload, original.payload_len) == 0);
    PASS("frame_decode_roundtrip");
}

static void test_frame_decode_bad_magic(void) {
    uint8_t buf[32] = {0};
    buf[0] = 0xFF; buf[1] = 0xFF; buf[2] = 0xFF; buf[3] = 0xFF;
    eipc_frame_t frame;
    eipc_status_t rc = eipc_frame_decode(buf, sizeof(buf), &frame);
    assert(rc == EIPC_ERR_BAD_MAGIC);
    PASS("frame_decode_bad_magic");
}

static void test_frame_encode_with_hmac_flag(void) {
    eipc_frame_t frame;
    memset(&frame, 0, sizeof(frame));
    frame.version = EIPC_PROTOCOL_VER;
    frame.msg_type = EIPC_MSG_TOOL_REQUEST;
    frame.flags = EIPC_FLAG_HMAC;
    frame.header_len = 0;
    frame.payload_len = 0;

    uint8_t buf[1024];
    size_t out_len = 0;
    eipc_status_t rc = eipc_frame_encode(&frame, buf, sizeof(buf), &out_len);
    assert(rc == EIPC_OK);
    assert(out_len == EIPC_FRAME_FIXED_SIZE + EIPC_MAC_SIZE);
    PASS("frame_encode_with_hmac_flag");
}

static void test_frame_encode_buffer_too_small(void) {
    eipc_frame_t frame;
    memset(&frame, 0, sizeof(frame));
    frame.version = EIPC_PROTOCOL_VER;
    frame.msg_type = EIPC_MSG_INTENT;
    frame.header_len = 100;
    frame.payload_len = 200;

    uint8_t buf[16];
    size_t out_len = 0;
    eipc_status_t rc = eipc_frame_encode(&frame, buf, sizeof(buf), &out_len);
    assert(rc != EIPC_OK);
    PASS("frame_encode_buffer_too_small");
}

static void test_frame_signable_bytes(void) {
    eipc_frame_t frame;
    memset(&frame, 0, sizeof(frame));
    frame.version = EIPC_PROTOCOL_VER;
    frame.msg_type = EIPC_MSG_ACK;
    const char *hdr = "{\"req\":\"123\"}";
    memcpy(frame.header, hdr, strlen(hdr));
    frame.header_len = (uint32_t)strlen(hdr);
    frame.payload_len = 0;

    uint8_t sig_buf[1024];
    size_t sig_len = eipc_frame_signable_bytes(&frame, sig_buf, sizeof(sig_buf));
    assert(sig_len > 0);
    PASS("frame_signable_bytes");
}

static void test_msg_type_constants(void) {
    assert(EIPC_MSG_INTENT == 'i');
    assert(EIPC_MSG_FEATURES == 'f');
    assert(EIPC_MSG_TOOL_REQUEST == 't');
    assert(EIPC_MSG_ACK == 'a');
    assert(EIPC_MSG_POLICY == 'p');
    assert(EIPC_MSG_HEARTBEAT == 'h');
    assert(EIPC_MSG_AUDIT == 'u');
    PASS("msg_type_constants");
}

static void test_protocol_constants(void) {
    assert(EIPC_MAGIC == 0x45495043U);
    assert(EIPC_PROTOCOL_VER == 1);
    assert(EIPC_MAC_SIZE == 32);
    assert(EIPC_FRAME_FIXED_SIZE == 16);
    assert(EIPC_MAX_FRAME == (1U << 20));
    PASS("protocol_constants");
}

static void test_priority_constants(void) {
    assert(EIPC_PRIORITY_P0 == 0);
    assert(EIPC_PRIORITY_P1 == 1);
    assert(EIPC_PRIORITY_P2 == 2);
    assert(EIPC_PRIORITY_P3 == 3);
    PASS("priority_constants");
}

static void test_status_enum(void) {
    assert(EIPC_OK == 0);
    assert(EIPC_ERR_NOMEM == 1);
    assert(EIPC_ERR_INVALID == 2);
    PASS("status_enum");
}

static void test_flag_constants(void) {
    assert(EIPC_FLAG_HMAC == 1);
    assert(EIPC_FLAG_COMPRESS == 2);
    PASS("flag_constants");
}

int main(void) {
    printf("=== eipc Frame Tests ===\n");
    test_frame_encode_basic();
    test_frame_decode_roundtrip();
    test_frame_decode_bad_magic();
    test_frame_encode_with_hmac_flag();
    test_frame_encode_buffer_too_small();
    test_frame_signable_bytes();
    test_msg_type_constants();
    test_protocol_constants();
    test_priority_constants();
    test_status_enum();
    test_flag_constants();
    printf("\n=== ALL %d TESTS PASSED ===\n", passed);
    return 0;
}
