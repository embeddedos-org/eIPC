// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project
// ISO/IEC 25000 | ISO/IEC/IEEE 15288:2023

/*
 * EIPC High-level Server API
 * Mirrors eipc_client.c patterns: init, listen, accept, receive, send_ack, close.
 */

#include "eipc.h"
#include <string.h>
#include <stdio.h>

eipc_status_t eipc_server_init(eipc_server_t *s) {
    if (!s) return EIPC_ERR_INVALID;

    memset(s, 0, sizeof(*s));
    s->listen_sock = EIPC_INVALID_SOCKET;
    return EIPC_OK;
}

eipc_status_t eipc_server_listen(eipc_server_t *s, const char *address,
                                  const char *hmac_key) {
    eipc_status_t rc;
    size_t key_len;

    if (!s || !address || !hmac_key) return EIPC_ERR_INVALID;

    key_len = strlen(hmac_key);
    if (key_len > sizeof(s->hmac_key)) return EIPC_ERR_INVALID;
    memcpy(s->hmac_key, hmac_key, key_len);
    s->hmac_key_len = (uint32_t)key_len;

    rc = eipc_transport_listen(&s->listen_sock, address);
    return rc;
}

eipc_status_t eipc_server_accept(eipc_server_t *s, eipc_conn_t *conn) {
    eipc_status_t rc;

    if (!s || !conn) return EIPC_ERR_INVALID;
    if (s->listen_sock == EIPC_INVALID_SOCKET) return EIPC_ERR_INVALID;

    memset(conn, 0, sizeof(*conn));
    conn->sock = EIPC_INVALID_SOCKET;

    memcpy(conn->hmac_key, s->hmac_key, s->hmac_key_len);
    conn->hmac_key_len = s->hmac_key_len;

    rc = eipc_transport_accept(s->listen_sock, &conn->sock,
                               conn->remote_addr, sizeof(conn->remote_addr));
    return rc;
}

eipc_status_t eipc_server_receive(eipc_conn_t *conn, eipc_message_t *msg) {
    eipc_frame_t frame;
    eipc_header_t hdr;
    eipc_status_t rc;

    if (!conn || !msg) return EIPC_ERR_INVALID;
    if (conn->sock == EIPC_INVALID_SOCKET) return EIPC_ERR_INVALID;

    memset(&frame, 0, sizeof(frame));

    rc = eipc_transport_recv_frame(conn->sock, &frame);
    if (rc != EIPC_OK) return rc;

    /* Verify HMAC if present */
    if (frame.flags & EIPC_FLAG_HMAC) {
        uint8_t signable[EIPC_MAX_FRAME];
        size_t signable_len = eipc_frame_signable_bytes(&frame, signable, sizeof(signable));
        if (signable_len == 0)
            return EIPC_ERR_INTEGRITY;

        if (!eipc_hmac_verify(conn->hmac_key, conn->hmac_key_len,
                              signable, signable_len, frame.mac))
            return EIPC_ERR_AUTH;
    }

    memset(msg, 0, sizeof(*msg));
    msg->msg_type = frame.msg_type;
    msg->version = frame.version;

    rc = eipc_header_from_json((const char *)frame.header, frame.header_len, &hdr);
    if (rc != EIPC_OK) return rc;

    strncpy(msg->source, hdr.service_id, sizeof(msg->source) - 1);
    strncpy(msg->session_id, hdr.session_id, sizeof(msg->session_id) - 1);
    strncpy(msg->request_id, hdr.request_id, sizeof(msg->request_id) - 1);
    msg->priority = hdr.priority;
    strncpy(msg->capability, hdr.capability, sizeof(msg->capability) - 1);

    if (frame.payload_len > 0) {
        if (frame.payload_len > sizeof(msg->payload))
            return EIPC_ERR_FRAME_TOO_LARGE;
        memcpy(msg->payload, frame.payload, frame.payload_len);
        msg->payload_len = frame.payload_len;
    }

    conn->sequence++;
    return EIPC_OK;
}

eipc_status_t eipc_server_send_ack(eipc_conn_t *conn, const char *request_id,
                                    const char *status) {
    eipc_header_t hdr;
    eipc_frame_t frame;
    char header_json[EIPC_MAX_HEADER];
    char payload_json[EIPC_MAX_PAYLOAD];
    int plen;
    size_t signable_len;
    eipc_status_t rc;

    if (!conn || !status) return EIPC_ERR_INVALID;

    memset(&hdr, 0, sizeof(hdr));
    strncpy(hdr.service_id, "eipc-server", sizeof(hdr.service_id) - 1);
    if (request_id)
        strncpy(hdr.request_id, request_id, sizeof(hdr.request_id) - 1);
    hdr.sequence = ++conn->sequence;
    eipc_timestamp_now(hdr.timestamp, sizeof(hdr.timestamp));
    hdr.priority = EIPC_PRIORITY_P0;
    hdr.payload_format = EIPC_PAYLOAD_JSON;

    memset(&frame, 0, sizeof(frame));
    frame.version = EIPC_PROTOCOL_VER;
    frame.msg_type = EIPC_MSG_ACK;
    frame.flags = EIPC_FLAG_HMAC;

    rc = eipc_header_to_json(&hdr, header_json, sizeof(header_json));
    if (rc != EIPC_OK) return rc;

    {
        size_t hdr_len = strlen(header_json);
        memcpy(frame.header, header_json, hdr_len);
        frame.header_len = (uint32_t)hdr_len;
    }

    plen = snprintf(payload_json, sizeof(payload_json),
        "{\"request_id\":\"%s\",\"status\":\"%s\",\"error\":\"\"}",
        request_id ? request_id : "", status);
    if (plen > 0 && (size_t)plen < sizeof(payload_json)) {
        memcpy(frame.payload, payload_json, (size_t)plen);
        frame.payload_len = (uint32_t)plen;
    }

    {
        uint8_t signable[EIPC_MAX_FRAME];
        signable_len = eipc_frame_signable_bytes(&frame, signable, sizeof(signable));
        if (signable_len == 0) return EIPC_ERR_INTEGRITY;
        eipc_hmac_sign(conn->hmac_key, conn->hmac_key_len,
                       signable, signable_len, frame.mac);
    }

    return eipc_transport_send_frame(conn->sock, &frame);
}

eipc_status_t eipc_server_send_message(eipc_conn_t *conn, const eipc_message_t *msg) {
    eipc_header_t hdr;
    eipc_frame_t frame;
    char header_json[EIPC_MAX_HEADER];
    size_t signable_len;
    eipc_status_t rc;

    if (!conn || !msg) return EIPC_ERR_INVALID;

    memset(&hdr, 0, sizeof(hdr));
    strncpy(hdr.service_id, msg->source, sizeof(hdr.service_id) - 1);
    strncpy(hdr.session_id, msg->session_id, sizeof(hdr.session_id) - 1);
    strncpy(hdr.request_id, msg->request_id, sizeof(hdr.request_id) - 1);
    hdr.sequence = ++conn->sequence;
    eipc_timestamp_now(hdr.timestamp, sizeof(hdr.timestamp));
    hdr.priority = msg->priority;
    strncpy(hdr.capability, msg->capability, sizeof(hdr.capability) - 1);
    hdr.payload_format = EIPC_PAYLOAD_JSON;

    memset(&frame, 0, sizeof(frame));
    frame.version = EIPC_PROTOCOL_VER;
    frame.msg_type = msg->msg_type;
    frame.flags = EIPC_FLAG_HMAC;

    rc = eipc_header_to_json(&hdr, header_json, sizeof(header_json));
    if (rc != EIPC_OK) return rc;

    {
        size_t hdr_len = strlen(header_json);
        memcpy(frame.header, header_json, hdr_len);
        frame.header_len = (uint32_t)hdr_len;
    }

    if (msg->payload_len > 0) {
        if (msg->payload_len > sizeof(frame.payload))
            return EIPC_ERR_FRAME_TOO_LARGE;
        memcpy(frame.payload, msg->payload, msg->payload_len);
        frame.payload_len = msg->payload_len;
    }

    {
        uint8_t signable[EIPC_MAX_FRAME];
        signable_len = eipc_frame_signable_bytes(&frame, signable, sizeof(signable));
        if (signable_len == 0) return EIPC_ERR_INTEGRITY;
        eipc_hmac_sign(conn->hmac_key, conn->hmac_key_len,
                       signable, signable_len, frame.mac);
    }

    return eipc_transport_send_frame(conn->sock, &frame);
}

void eipc_conn_close(eipc_conn_t *conn) {
    if (!conn) return;
    if (conn->sock != EIPC_INVALID_SOCKET) {
        eipc_transport_close(conn->sock);
        conn->sock = EIPC_INVALID_SOCKET;
    }
}

void eipc_server_close(eipc_server_t *s) {
    if (!s) return;
    if (s->listen_sock != EIPC_INVALID_SOCKET) {
        eipc_transport_close(s->listen_sock);
        s->listen_sock = EIPC_INVALID_SOCKET;
    }
}
