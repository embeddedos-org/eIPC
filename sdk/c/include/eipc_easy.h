// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project
// ISO/IEC 25000 | ISO/IEC/IEEE 15288:2023

#ifndef EIPC_EASY_H
#define EIPC_EASY_H

#include "eipc.h"

#ifdef __cplusplus
extern "C" {
#endif

/* ══════════════════════════════════════════════════════════════
 *  Opaque handles (internal structs hidden in eipc_easy.c)
 * ══════════════════════════════════════════════════════════════ */

typedef struct eipc_easy_client eipc_easy_client_t;
typedef struct eipc_easy_server eipc_easy_server_t;

/* ══════════════════════════════════════════════════════════════
 *  Stats snapshot
 * ══════════════════════════════════════════════════════════════ */

typedef struct {
    uint64_t sent_count;
    uint64_t received_count;
    uint64_t ack_count;
    uint64_t error_count;
} eipc_easy_stats_t;

/* ══════════════════════════════════════════════════════════════
 *  Client API (replaces ENI eipc_bridge boilerplate)
 * ══════════════════════════════════════════════════════════════ */

eipc_status_t eipc_easy_connect(const char *addr, const char *hmac_key,
                                const char *service_id,
                                eipc_easy_client_t **out);

eipc_status_t eipc_easy_send_intent(eipc_easy_client_t *h,
                                    const char *intent, float confidence);

eipc_status_t eipc_easy_send_tool_request(eipc_easy_client_t *h,
                                          const char *tool,
                                          const eipc_kv_t *args, int arg_count);

eipc_status_t eipc_easy_send_chat(eipc_easy_client_t *h,
                                  const eipc_chat_request_t *req);

eipc_status_t eipc_easy_send_complete(eipc_easy_client_t *h,
                                      const char *prompt,
                                      const char *session_id);

eipc_status_t eipc_easy_receive(eipc_easy_client_t *h, eipc_message_t *msg);

eipc_status_t eipc_easy_recv_ack(eipc_easy_client_t *h,
                                 eipc_ack_event_t *ack);

void               eipc_easy_close(eipc_easy_client_t *h);
eipc_easy_stats_t  eipc_easy_client_stats(const eipc_easy_client_t *h);

/* ══════════════════════════════════════════════════════════════
 *  Server API (replaces EAI eipc_listener boilerplate)
 * ══════════════════════════════════════════════════════════════ */

eipc_status_t eipc_easy_listen(const char *addr, const char *hmac_key,
                               eipc_easy_server_t **out);

eipc_status_t eipc_easy_accept(eipc_easy_server_t *h);

eipc_status_t eipc_easy_recv_intent(eipc_easy_server_t *h,
                                    eipc_intent_event_t *intent_out);

eipc_status_t eipc_easy_recv_chat(eipc_easy_server_t *h,
                                  eipc_chat_request_t *chat_out);

eipc_status_t eipc_easy_send_ack(eipc_easy_server_t *h,
                                 const char *request_id, const char *status);

eipc_status_t eipc_easy_send_chat_response(eipc_easy_server_t *h,
                                           const char *session_id,
                                           const char *response,
                                           const char *model,
                                           int tokens_used);

void               eipc_easy_close_server(eipc_easy_server_t *h);
eipc_easy_stats_t  eipc_easy_server_stats(const eipc_easy_server_t *h);

#ifdef __cplusplus
}
#endif

#endif /* EIPC_EASY_H */
