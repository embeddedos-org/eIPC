// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project
// ISO/IEC 25000 | ISO/IEC/IEEE 15288:2023

/*
 * EIPC Easy — convenience layer over the low-level C SDK.
 * Provides opaque handles, one-call init, automatic stats,
 * and typed send/receive helpers.
 */

#include "eipc_easy.h"

#include <stdlib.h>
#include <string.h>
#include <stdio.h>

/* ══════════════════════════════════════════════════════════════
 *  Internal struct definitions (hidden from consumers)
 * ══════════════════════════════════════════════════════════════ */

struct eipc_easy_client {
    eipc_client_t    client;
    bool             connected;
    eipc_easy_stats_t stats;
};

struct eipc_easy_server {
    eipc_server_t    server;
    eipc_conn_t      conn;
    bool             listening;
    bool             has_client;
    eipc_easy_stats_t stats;
};

/* ══════════════════════════════════════════════════════════════
 *  Client API
 * ══════════════════════════════════════════════════════════════ */

eipc_status_t eipc_easy_connect(const char *addr, const char *hmac_key,
                                const char *service_id,
                                eipc_easy_client_t **out) {
    eipc_easy_client_t *h;
    eipc_status_t rc;

    if (!addr || !hmac_key || !service_id || !out)
        return EIPC_ERR_INVALID;

    *out = NULL;

    h = (eipc_easy_client_t *)malloc(sizeof(*h));
    if (!h) return EIPC_ERR_NOMEM;
    memset(h, 0, sizeof(*h));

    rc = eipc_client_init(&h->client, service_id);
    if (rc != EIPC_OK) {
        free(h);
        return rc;
    }

    rc = eipc_client_connect(&h->client, addr, hmac_key);
    if (rc != EIPC_OK) {
        free(h);
        return rc;
    }

    h->connected = true;
    *out = h;
    return EIPC_OK;
}

eipc_status_t eipc_easy_send_intent(eipc_easy_client_t *h,
                                    const char *intent, float confidence) {
    eipc_status_t rc;
    if (!h || !h->connected || !intent) return EIPC_ERR_INVALID;

    rc = eipc_client_send_intent(&h->client, intent, confidence);
    if (rc == EIPC_OK)
        h->stats.sent_count++;
    else
        h->stats.error_count++;
    return rc;
}

eipc_status_t eipc_easy_send_tool_request(eipc_easy_client_t *h,
                                          const char *tool,
                                          const eipc_kv_t *args, int arg_count) {
    eipc_status_t rc;
    if (!h || !h->connected || !tool) return EIPC_ERR_INVALID;

    rc = eipc_client_send_tool_request(&h->client, tool, args, arg_count);
    if (rc == EIPC_OK)
        h->stats.sent_count++;
    else
        h->stats.error_count++;
    return rc;
}

eipc_status_t eipc_easy_send_chat(eipc_easy_client_t *h,
                                  const eipc_chat_request_t *req) {
    eipc_status_t rc;
    if (!h || !h->connected || !req) return EIPC_ERR_INVALID;

    rc = eipc_client_send_chat(&h->client, req);
    if (rc == EIPC_OK)
        h->stats.sent_count++;
    else
        h->stats.error_count++;
    return rc;
}

eipc_status_t eipc_easy_send_complete(eipc_easy_client_t *h,
                                      const char *prompt,
                                      const char *session_id) {
    eipc_status_t rc;
    if (!h || !h->connected || !prompt) return EIPC_ERR_INVALID;

    rc = eipc_client_send_complete(&h->client, prompt, session_id);
    if (rc == EIPC_OK)
        h->stats.sent_count++;
    else
        h->stats.error_count++;
    return rc;
}

eipc_status_t eipc_easy_receive(eipc_easy_client_t *h, eipc_message_t *msg) {
    eipc_status_t rc;
    if (!h || !h->connected || !msg) return EIPC_ERR_INVALID;

    rc = eipc_client_receive(&h->client, msg);
    if (rc == EIPC_OK)
        h->stats.received_count++;
    else
        h->stats.error_count++;
    return rc;
}

eipc_status_t eipc_easy_recv_ack(eipc_easy_client_t *h,
                                 eipc_ack_event_t *ack) {
    eipc_message_t msg;
    eipc_status_t rc;

    if (!h || !h->connected || !ack) return EIPC_ERR_INVALID;

    memset(&msg, 0, sizeof(msg));
    rc = eipc_client_receive(&h->client, &msg);
    if (rc != EIPC_OK) {
        h->stats.error_count++;
        return rc;
    }

    if (msg.msg_type != EIPC_MSG_ACK) {
        h->stats.error_count++;
        return EIPC_ERR_PROTOCOL;
    }

    rc = eipc_ack_from_json((const char *)msg.payload, msg.payload_len, ack);
    if (rc != EIPC_OK) {
        h->stats.error_count++;
        return rc;
    }

    h->stats.ack_count++;
    return EIPC_OK;
}

void eipc_easy_close(eipc_easy_client_t *h) {
    if (!h) return;

    fprintf(stderr, "[eipc_easy] client stats: sent=%llu recv=%llu ack=%llu err=%llu\n",
            (unsigned long long)h->stats.sent_count,
            (unsigned long long)h->stats.received_count,
            (unsigned long long)h->stats.ack_count,
            (unsigned long long)h->stats.error_count);

    if (h->connected) {
        eipc_client_close(&h->client);
        h->connected = false;
    }

    free(h);
}

eipc_easy_stats_t eipc_easy_client_stats(const eipc_easy_client_t *h) {
    eipc_easy_stats_t zero;
    if (!h) {
        memset(&zero, 0, sizeof(zero));
        return zero;
    }
    return h->stats;
}

/* ══════════════════════════════════════════════════════════════
 *  Server API
 * ══════════════════════════════════════════════════════════════ */

eipc_status_t eipc_easy_listen(const char *addr, const char *hmac_key,
                               eipc_easy_server_t **out) {
    eipc_easy_server_t *h;
    eipc_status_t rc;

    if (!addr || !hmac_key || !out)
        return EIPC_ERR_INVALID;

    *out = NULL;

    h = (eipc_easy_server_t *)malloc(sizeof(*h));
    if (!h) return EIPC_ERR_NOMEM;
    memset(h, 0, sizeof(*h));

    rc = eipc_server_init(&h->server);
    if (rc != EIPC_OK) {
        free(h);
        return rc;
    }

    rc = eipc_server_listen(&h->server, addr, hmac_key);
    if (rc != EIPC_OK) {
        free(h);
        return rc;
    }

    h->listening = true;
    *out = h;
    return EIPC_OK;
}

eipc_status_t eipc_easy_accept(eipc_easy_server_t *h) {
    eipc_status_t rc;
    if (!h || !h->listening) return EIPC_ERR_INVALID;

    rc = eipc_server_accept(&h->server, &h->conn);
    if (rc != EIPC_OK) {
        h->stats.error_count++;
        return rc;
    }

    h->has_client = true;
    return EIPC_OK;
}

eipc_status_t eipc_easy_recv_intent(eipc_easy_server_t *h,
                                    eipc_intent_event_t *intent_out) {
    eipc_message_t msg;
    eipc_status_t rc;

    if (!h || !h->has_client || !intent_out) return EIPC_ERR_INVALID;

    memset(&msg, 0, sizeof(msg));
    rc = eipc_server_receive(&h->conn, &msg);
    if (rc != EIPC_OK) {
        h->stats.error_count++;
        return rc;
    }

    if (msg.msg_type != EIPC_MSG_INTENT) {
        h->stats.error_count++;
        return EIPC_ERR_PROTOCOL;
    }

    rc = eipc_intent_from_json((const char *)msg.payload, msg.payload_len, intent_out);
    if (rc != EIPC_OK) {
        h->stats.error_count++;
        return rc;
    }

    h->stats.received_count++;
    return EIPC_OK;
}

eipc_status_t eipc_easy_recv_chat(eipc_easy_server_t *h,
                                  eipc_chat_request_t *chat_out) {
    eipc_message_t msg;
    eipc_status_t rc;

    if (!h || !h->has_client || !chat_out) return EIPC_ERR_INVALID;

    memset(&msg, 0, sizeof(msg));
    rc = eipc_server_receive(&h->conn, &msg);
    if (rc != EIPC_OK) {
        h->stats.error_count++;
        return rc;
    }

    if (msg.msg_type != EIPC_MSG_CHAT) {
        h->stats.error_count++;
        return EIPC_ERR_PROTOCOL;
    }

    rc = eipc_chat_request_from_json((const char *)msg.payload, msg.payload_len, chat_out);
    if (rc != EIPC_OK) {
        h->stats.error_count++;
        return rc;
    }

    h->stats.received_count++;
    return EIPC_OK;
}

eipc_status_t eipc_easy_send_ack(eipc_easy_server_t *h,
                                 const char *request_id, const char *status) {
    eipc_status_t rc;
    if (!h || !h->has_client || !status) return EIPC_ERR_INVALID;

    char rid[EIPC_REQUEST_ID_MAX];
    if (!request_id || request_id[0] == '\0') {
        eipc_generate_request_id(rid, sizeof(rid));
        request_id = rid;
    }

    rc = eipc_server_send_ack(&h->conn, request_id, status);
    if (rc == EIPC_OK)
        h->stats.ack_count++;
    else
        h->stats.error_count++;
    return rc;
}

eipc_status_t eipc_easy_send_chat_response(eipc_easy_server_t *h,
                                           const char *session_id,
                                           const char *response,
                                           const char *model,
                                           int tokens_used) {
    eipc_chat_response_t resp;
    char payload_json[EIPC_MAX_PAYLOAD];
    eipc_message_t msg;
    eipc_status_t rc;

    if (!h || !h->has_client || !response) return EIPC_ERR_INVALID;

    memset(&resp, 0, sizeof(resp));
    if (session_id)
        strncpy(resp.session_id, session_id, sizeof(resp.session_id) - 1);
    strncpy(resp.response, response, sizeof(resp.response) - 1);
    if (model)
        strncpy(resp.model, model, sizeof(resp.model) - 1);
    resp.tokens_used = tokens_used;

    rc = eipc_chat_response_to_json(&resp, payload_json, sizeof(payload_json));
    if (rc != EIPC_OK) {
        h->stats.error_count++;
        return rc;
    }

    memset(&msg, 0, sizeof(msg));
    msg.msg_type = EIPC_MSG_CHAT;
    msg.version = EIPC_PROTOCOL_VER;
    strncpy(msg.source, "eipc-easy", sizeof(msg.source) - 1);
    if (session_id)
        strncpy(msg.session_id, session_id, sizeof(msg.session_id) - 1);
    msg.payload_len = (uint32_t)strlen(payload_json);
    memcpy(msg.payload, payload_json, msg.payload_len);

    rc = eipc_server_send_message(&h->conn, &msg);
    if (rc == EIPC_OK)
        h->stats.sent_count++;
    else
        h->stats.error_count++;
    return rc;
}

void eipc_easy_close_server(eipc_easy_server_t *h) {
    if (!h) return;

    fprintf(stderr, "[eipc_easy] server stats: sent=%llu recv=%llu ack=%llu err=%llu\n",
            (unsigned long long)h->stats.sent_count,
            (unsigned long long)h->stats.received_count,
            (unsigned long long)h->stats.ack_count,
            (unsigned long long)h->stats.error_count);

    if (h->has_client) {
        eipc_conn_close(&h->conn);
        h->has_client = false;
    }

    if (h->listening) {
        eipc_server_close(&h->server);
        h->listening = false;
    }

    free(h);
}

eipc_easy_stats_t eipc_easy_server_stats(const eipc_easy_server_t *h) {
    eipc_easy_stats_t zero;
    if (!h) {
        memset(&zero, 0, sizeof(zero));
        return zero;
    }
    return h->stats;
}
