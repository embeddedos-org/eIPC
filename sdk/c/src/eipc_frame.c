// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project
// ISO/IEC 25000 | ISO/IEC/IEEE 15288:2023

/*
 * EIPC Frame Codec — encode/decode matching Go protocol.Frame
 *
 * Wire format:
 *   [magic:4][version:2][msg_type:1][flags:1][header_len:4][payload_len:4]
 *   [header][payload][mac:32?]
 *
 * Magic: 0x45495043 ("EIPC")
 */

#include "eipc.h"
#include <string.h>

eipc_status_t eipc_frame_encode(const eipc_frame_t *frame,
                                uint8_t *buf, size_t buf_size,
                                size_t *out_len) {
    size_t total;
    size_t pos = 0;

    if (!frame || !buf || !out_len)
        return EIPC_ERR_INVALID;

    total = EIPC_FRAME_FIXED_SIZE + frame->header_len + frame->payload_len;
    if (frame->flags & EIPC_FLAG_HMAC)
        total += EIPC_MAC_SIZE;

    if (total > buf_size || total > EIPC_MAX_FRAME)
        return EIPC_ERR_FRAME_TOO_LARGE;

    /* Magic bytes (big-endian) */
    buf[pos++] = (uint8_t)(EIPC_MAGIC >> 24);
    buf[pos++] = (uint8_t)(EIPC_MAGIC >> 16);
    buf[pos++] = (uint8_t)(EIPC_MAGIC >> 8);
    buf[pos++] = (uint8_t)(EIPC_MAGIC);

    /* Version (big-endian) */
    buf[pos++] = (uint8_t)(frame->version >> 8);
    buf[pos++] = (uint8_t)(frame->version);

    /* Message type */
    buf[pos++] = frame->msg_type;

    /* Flags */
    buf[pos++] = frame->flags;

    /* Header length (big-endian) */
    buf[pos++] = (uint8_t)(frame->header_len >> 24);
    buf[pos++] = (uint8_t)(frame->header_len >> 16);
    buf[pos++] = (uint8_t)(frame->header_len >> 8);
    buf[pos++] = (uint8_t)(frame->header_len);

    /* Payload length (big-endian) */
    buf[pos++] = (uint8_t)(frame->payload_len >> 24);
    buf[pos++] = (uint8_t)(frame->payload_len >> 16);
    buf[pos++] = (uint8_t)(frame->payload_len >> 8);
    buf[pos++] = (uint8_t)(frame->payload_len);

    /* Header data */
    if (frame->header_len > 0) {
        memcpy(buf + pos, frame->header, frame->header_len);
        pos += frame->header_len;
    }

    /* Payload data */
    if (frame->payload_len > 0) {
        memcpy(buf + pos, frame->payload, frame->payload_len);
        pos += frame->payload_len;
    }

    /* MAC (if HMAC flag set) */
    if (frame->flags & EIPC_FLAG_HMAC) {
        memcpy(buf + pos, frame->mac, EIPC_MAC_SIZE);
        pos += EIPC_MAC_SIZE;
    }

    *out_len = pos;
    return EIPC_OK;
}

eipc_status_t eipc_frame_decode(const uint8_t *buf, size_t buf_len,
                                eipc_frame_t *frame) {
    uint32_t magic;
    size_t pos = 0;

    if (!buf || !frame)
        return EIPC_ERR_INVALID;

    if (buf_len < EIPC_FRAME_FIXED_SIZE)
        return EIPC_ERR_PROTOCOL;

    memset(frame, 0, sizeof(*frame));

    /* Magic bytes */
    magic = ((uint32_t)buf[0] << 24) | ((uint32_t)buf[1] << 16) |
            ((uint32_t)buf[2] << 8) | ((uint32_t)buf[3]);
    if (magic != EIPC_MAGIC)
        return EIPC_ERR_BAD_MAGIC;

    /* Version */
    frame->version = ((uint16_t)buf[4] << 8) | buf[5];
    if (frame->version != EIPC_PROTOCOL_VER)
        return EIPC_ERR_BAD_VERSION;

    /* Message type and flags */
    frame->msg_type = buf[6];
    frame->flags = buf[7];

    /* Header and payload lengths */
    frame->header_len = ((uint32_t)buf[8] << 24) | ((uint32_t)buf[9] << 16) |
                        ((uint32_t)buf[10] << 8) | ((uint32_t)buf[11]);
    frame->payload_len = ((uint32_t)buf[12] << 24) | ((uint32_t)buf[13] << 16) |
                         ((uint32_t)buf[14] << 8) | ((uint32_t)buf[15]);

    if (frame->header_len > EIPC_MAX_HEADER)
        return EIPC_ERR_FRAME_TOO_LARGE;
    if (frame->payload_len > EIPC_MAX_PAYLOAD)
        return EIPC_ERR_FRAME_TOO_LARGE;

    pos = EIPC_FRAME_FIXED_SIZE;

    /* Header data */
    if (frame->header_len > 0) {
        if (pos + frame->header_len > buf_len)
            return EIPC_ERR_PROTOCOL;
        memcpy(frame->header, buf + pos, frame->header_len);
        pos += frame->header_len;
    }

    /* Payload data */
    if (frame->payload_len > 0) {
        if (pos + frame->payload_len > buf_len)
            return EIPC_ERR_PROTOCOL;
        memcpy(frame->payload, buf + pos, frame->payload_len);
        pos += frame->payload_len;
    }

    /* MAC */
    if (frame->flags & EIPC_FLAG_HMAC) {
        if (pos + EIPC_MAC_SIZE > buf_len)
            return EIPC_ERR_PROTOCOL;
        memcpy(frame->mac, buf + pos, EIPC_MAC_SIZE);
    }

    return EIPC_OK;
}

size_t eipc_frame_signable_bytes(const eipc_frame_t *frame,
                                 uint8_t *buf, size_t buf_size) {
    size_t total;
    size_t pos = 0;

    if (!frame || !buf)
        return 0;

    total = EIPC_FRAME_FIXED_SIZE + frame->header_len + frame->payload_len;
    if (total > buf_size)
        return 0;

    /* Preamble */
    buf[pos++] = (uint8_t)(EIPC_MAGIC >> 24);
    buf[pos++] = (uint8_t)(EIPC_MAGIC >> 16);
    buf[pos++] = (uint8_t)(EIPC_MAGIC >> 8);
    buf[pos++] = (uint8_t)(EIPC_MAGIC);
    buf[pos++] = (uint8_t)(frame->version >> 8);
    buf[pos++] = (uint8_t)(frame->version);
    buf[pos++] = frame->msg_type;
    buf[pos++] = frame->flags;
    buf[pos++] = (uint8_t)(frame->header_len >> 24);
    buf[pos++] = (uint8_t)(frame->header_len >> 16);
    buf[pos++] = (uint8_t)(frame->header_len >> 8);
    buf[pos++] = (uint8_t)(frame->header_len);
    buf[pos++] = (uint8_t)(frame->payload_len >> 24);
    buf[pos++] = (uint8_t)(frame->payload_len >> 16);
    buf[pos++] = (uint8_t)(frame->payload_len >> 8);
    buf[pos++] = (uint8_t)(frame->payload_len);

    /* Header */
    if (frame->header_len > 0) {
        memcpy(buf + pos, frame->header, frame->header_len);
        pos += frame->header_len;
    }

    /* Payload */
    if (frame->payload_len > 0) {
        memcpy(buf + pos, frame->payload, frame->payload_len);
        pos += frame->payload_len;
    }

    return pos;
}
