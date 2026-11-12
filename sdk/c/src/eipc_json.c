// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project
// ISO/IEC 25000 | ISO/IEC/IEEE 15288:2023

/*
 * EIPC Minimal JSON Serializer/Deserializer
 * No external dependencies. Uses snprintf for serialization and
 * simple string scanning for deserialization.
 */

#include "eipc.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* ---------- JSON Parse Helpers ---------- */

#define JSON_BAD_HEX 0xFFFFFFFFu

/* The value of four hex digits at p, or JSON_BAD_HEX. */
static uint32_t json_hex4(const char *p) {
    uint32_t v = 0;
    int i;
    for (i = 0; i < 4; i++) {
        char c = p[i];
        v <<= 4;
        if (c >= '0' && c <= '9')
            v |= (uint32_t)(c - '0');
        else if (c >= 'a' && c <= 'f')
            v |= (uint32_t)(c - 'a' + 10);
        else if (c >= 'A' && c <= 'F')
            v |= (uint32_t)(c - 'A' + 10);
        else
            return JSON_BAD_HEX;
    }
    return v;
}

/* Write code point cp as UTF-8 into out (room for 4 bytes); return the length. */
static size_t json_utf8(uint32_t cp, char *out) {
    if (cp < 0x80) {
        out[0] = (char)cp;
        return 1;
    }
    if (cp < 0x800) {
        out[0] = (char)(0xC0 | (cp >> 6));
        out[1] = (char)(0x80 | (cp & 0x3F));
        return 2;
    }
    if (cp < 0x10000) {
        out[0] = (char)(0xE0 | (cp >> 12));
        out[1] = (char)(0x80 | ((cp >> 6) & 0x3F));
        out[2] = (char)(0x80 | (cp & 0x3F));
        return 3;
    }
    out[0] = (char)(0xF0 | (cp >> 18));
    out[1] = (char)(0x80 | ((cp >> 12) & 0x3F));
    out[2] = (char)(0x80 | ((cp >> 6) & 0x3F));
    out[3] = (char)(0x80 | (cp & 0x3F));
    return 4;
}

/* The start of the value for key, or NULL. Only a "key" followed by a colon
 * is a key: strstr() alone also matched the same text as a string value, so
 * in {"status":"error","error":"disk full"} the lookup for "error" landed on
 * the status value and returned nothing. */
static const char *json_find_value(const char *json, const char *key) {
    char pattern[128];
    const char *p = json;
    size_t plen;

    snprintf(pattern, sizeof(pattern), "\"%s\"", key);
    plen = strlen(pattern);
    while ((p = strstr(p, pattern)) != NULL) {
        const char *v = p + plen;
        while (*v == ' ' || *v == '\t' || *v == '\n' || *v == '\r')
            v++;
        if (*v == ':') {
            v++;
            while (*v == ' ' || *v == '\t' || *v == '\n' || *v == '\r')
                v++;
            return v;
        }
        p += plen;
    }
    return NULL;
}

static int json_find_string(const char *json, const char *key,
                            char *out, size_t out_size) {
    const char *start, *end;
    size_t len;

    start = json_find_value(json, key);
    if (!start || *start != '"') {
        out[0] = '\0';
        return -1;
    }
    start++;

    /* Decode the escapes as the value is copied: the encoders write \" \\
     * \n \r \t (json_escape), and other JSON encoders also write \/ \b \f
     * and \uXXXX (Go's encoding/json writes < > & as \u003c \u003e
     * \u0026). A value longer than out_size - 1 bytes is truncated as
     * before; a decoded \uXXXX is never cut in half. */
    len = 0;
    end = start;
    while (*end && *end != '"') {
        char enc[4];
        size_t n = 1;

        if (*end != '\\') {
            enc[0] = *end++;
        } else if (end[1] == 'u') {
            uint32_t cp = json_hex4(end + 2);
            end += (cp == JSON_BAD_HEX) ? 2 : 6;
            if (cp >= 0xD800 && cp <= 0xDBFF && end[0] == '\\' && end[1] == 'u') {
                uint32_t lo = json_hex4(end + 2);
                if (lo >= 0xDC00 && lo <= 0xDFFF) {
                    cp = 0x10000 + ((cp - 0xD800) << 10) + (lo - 0xDC00);
                    end += 6;
                }
            }
            /* A NUL would cut the C string short, so \u0000 becomes
             * U+FFFD, like a malformed escape or an unpaired surrogate. */
            if (cp == 0 || cp == JSON_BAD_HEX || (cp >= 0xD800 && cp <= 0xDFFF))
                cp = 0xFFFD;
            n = json_utf8(cp, enc);
        } else if (end[1]) {
            switch (end[1]) {
                case 'b': enc[0] = '\b'; break;
                case 'f': enc[0] = '\f'; break;
                case 'n': enc[0] = '\n'; break;
                case 'r': enc[0] = '\r'; break;
                case 't': enc[0] = '\t'; break;
                default:  enc[0] = end[1]; break; /* \" \\ \/ */
            }
            end += 2;
        } else {
            break; /* a lone backslash at the end of the input */
        }

        if (len + n >= out_size)
            break;
        memcpy(out + len, enc, n);
        len += n;
    }

    out[len] = '\0';
    return 0;
}

static int json_find_uint64(const char *json, const char *key, uint64_t *out) {
    const char *start = json_find_value(json, key);
    if (!start) {
        *out = 0;
        return -1;
    }

    if (*start == '"')
        start++;
    *out = (uint64_t)strtoull(start, NULL, 10);
    return 0;
}

static int json_find_int(const char *json, const char *key, int *out) {
    uint64_t val;
    if (json_find_uint64(json, key, &val) < 0) {
        *out = 0;
        return -1;
    }
    *out = (int)val;
    return 0;
}

static int json_find_float(const char *json, const char *key, float *out) {
    const char *start = json_find_value(json, key);
    if (!start) {
        *out = 0.0f;
        return -1;
    }

    if (*start == '"')
        start++;
    *out = (float)strtod(start, NULL);
    return 0;
}

/* ---------- JSON Escape Helper ---------- */

static size_t json_escape(const char *src, char *dst, size_t dst_size) {
    size_t si = 0, di = 0;
    if (!src || !dst || dst_size == 0) return 0;

    while (src[si] && di < dst_size - 1) {
        switch (src[si]) {
            case '"':
            case '\\':
                if (di + 2 >= dst_size) goto done;
                dst[di++] = '\\';
                dst[di++] = src[si];
                break;
            case '\n':
                if (di + 2 >= dst_size) goto done;
                dst[di++] = '\\';
                dst[di++] = 'n';
                break;
            case '\r':
                if (di + 2 >= dst_size) goto done;
                dst[di++] = '\\';
                dst[di++] = 'r';
                break;
            case '\t':
                if (di + 2 >= dst_size) goto done;
                dst[di++] = '\\';
                dst[di++] = 't';
                break;
            default:
                dst[di++] = src[si];
                break;
        }
        si++;
    }
done:
    dst[di] = '\0';
    return di;
}

/* ---------- Header JSON ---------- */

eipc_status_t eipc_header_to_json(const eipc_header_t *hdr,
                                  char *buf, size_t buf_size,
                                  size_t *out_len) {
    int n;

    if (!hdr || !buf || !out_len)
        return EIPC_ERR_INVALID;

    n = snprintf(buf, buf_size,
        "{\"service_id\":\"%s\","
        "\"session_id\":\"%s\","
        "\"request_id\":\"%s\","
        "\"sequence\":%llu,"
        "\"timestamp\":\"%s\","
        "\"priority\":%d,"
        "\"capability\":\"%s\","
        "\"payload_format\":%d}",
        hdr->service_id,
        hdr->session_id,
        hdr->request_id,
        (unsigned long long)hdr->sequence,
        hdr->timestamp,
        hdr->priority,
        hdr->capability,
        hdr->payload_format);

    if (n < 0 || (size_t)n >= buf_size)
        return EIPC_ERR_BUFFER;

    *out_len = (size_t)n;
    return EIPC_OK;
}

/* ---------- Chat Request JSON ---------- */

eipc_status_t eipc_chat_request_to_json(const eipc_chat_request_t *req,
                                        char *buf, size_t buf_size, size_t *written) {
    int n;
    if (!req || !buf || !written) return EIPC_ERR_INVALID;
    n = snprintf(buf, buf_size,
        "{\"session_id\":\"%s\","
        "\"user_prompt\":\"%s\","
        "\"model\":\"%s\","
        "\"max_tokens\":%d}",
        req->session_id, req->user_prompt, req->model, req->max_tokens);
    if (n < 0 || (size_t)n >= buf_size) return EIPC_ERR_FRAME_TOO_LARGE;
    *written = (size_t)n;
    return EIPC_OK;
}

eipc_status_t eipc_chat_request_from_json(const char *json, size_t json_len,
                                          eipc_chat_request_t *req) {
    int ival;
    (void)json_len;
    if (!json || !req) return EIPC_ERR_INVALID;
    memset(req, 0, sizeof(*req));
    json_find_string(json, "session_id", req->session_id, sizeof(req->session_id));
    json_find_string(json, "user_prompt", req->user_prompt, sizeof(req->user_prompt));
    json_find_string(json, "model", req->model, sizeof(req->model));
    if (json_find_int(json, "max_tokens", &ival) == 0) req->max_tokens = ival;
    return EIPC_OK;
}

/* ---------- Chat Response JSON ---------- */

eipc_status_t eipc_chat_response_to_json(const eipc_chat_response_t *resp,
                                         char *buf, size_t buf_size, size_t *written) {
    int n;
    if (!resp || !buf || !written) return EIPC_ERR_INVALID;
    n = snprintf(buf, buf_size,
        "{\"session_id\":\"%s\","
        "\"response\":\"%s\","
        "\"model\":\"%s\","
        "\"tokens_used\":%d}",
        resp->session_id, resp->response, resp->model, resp->tokens_used);
    if (n < 0 || (size_t)n >= buf_size) return EIPC_ERR_FRAME_TOO_LARGE;
    *written = (size_t)n;
    return EIPC_OK;
}

eipc_status_t eipc_chat_response_from_json(const char *json, size_t json_len,
                                           eipc_chat_response_t *resp) {
    int ival;
    (void)json_len;
    if (!json || !resp) return EIPC_ERR_INVALID;
    memset(resp, 0, sizeof(*resp));
    json_find_string(json, "session_id", resp->session_id, sizeof(resp->session_id));
    json_find_string(json, "response", resp->response, sizeof(resp->response));
    json_find_string(json, "model", resp->model, sizeof(resp->model));
    if (json_find_int(json, "tokens_used", &ival) == 0) resp->tokens_used = ival;
    return EIPC_OK;
}

eipc_status_t eipc_header_from_json(const char *json, size_t json_len,
                                    eipc_header_t *hdr) {
    uint64_t val;
    int ival;
    (void)json_len;

    if (!json || !hdr)
        return EIPC_ERR_INVALID;

    memset(hdr, 0, sizeof(*hdr));

    json_find_string(json, "service_id", hdr->service_id, sizeof(hdr->service_id));
    json_find_string(json, "session_id", hdr->session_id, sizeof(hdr->session_id));
    json_find_string(json, "request_id", hdr->request_id, sizeof(hdr->request_id));

    if (json_find_uint64(json, "sequence", &val) == 0)
        hdr->sequence = val;
    json_find_string(json, "timestamp", hdr->timestamp, sizeof(hdr->timestamp));

    if (json_find_int(json, "priority", &ival) == 0)
        hdr->priority = ival;

    json_find_string(json, "capability", hdr->capability, sizeof(hdr->capability));

    if (json_find_int(json, "payload_format", &ival) == 0)
        hdr->payload_format = ival;

    return EIPC_OK;
}

/* ---------- Intent JSON ---------- */

eipc_status_t eipc_intent_to_json(const eipc_intent_t *intent,
                                  char *buf, size_t buf_size,
                                  size_t *out_len) {
    int n;
    char escaped_intent[256];
    char escaped_session[128];

    if (!intent || !buf || !out_len)
        return EIPC_ERR_INVALID;

    json_escape(intent->intent, escaped_intent, sizeof(escaped_intent));
    json_escape(intent->session_id, escaped_session, sizeof(escaped_session));

    n = snprintf(buf, buf_size,
        "{\"intent\":\"%s\","
        "\"confidence\":%.2f,"
        "\"session_id\":\"%s\"}",
        escaped_intent,
        intent->confidence,
        escaped_session);

    if (n < 0 || (size_t)n >= buf_size)
        return EIPC_ERR_BUFFER;

    *out_len = (size_t)n;
    return EIPC_OK;
}

eipc_status_t eipc_intent_from_json(const char *json, size_t json_len,
                                    eipc_intent_t *intent) {
    (void)json_len;

    if (!json || !intent)
        return EIPC_ERR_INVALID;

    memset(intent, 0, sizeof(*intent));

    json_find_string(json, "intent", intent->intent, sizeof(intent->intent));
    json_find_float(json, "confidence", &intent->confidence);
    json_find_string(json, "session_id", intent->session_id, sizeof(intent->session_id));

    return EIPC_OK;
}

/* ---------- Ack JSON ---------- */

eipc_status_t eipc_ack_to_json(const eipc_ack_t *ack,
                               char *buf, size_t buf_size,
                               size_t *out_len) {
    int n;
    char escaped_error[512];

    if (!ack || !buf || !out_len)
        return EIPC_ERR_INVALID;

    json_escape(ack->error, escaped_error, sizeof(escaped_error));

    n = snprintf(buf, buf_size,
        "{\"request_id\":\"%s\","
        "\"status\":\"%s\","
        "\"error\":\"%s\"}",
        ack->request_id,
        ack->status,
        escaped_error);

    if (n < 0 || (size_t)n >= buf_size)
        return EIPC_ERR_BUFFER;

    *out_len = (size_t)n;
    return EIPC_OK;
}

eipc_status_t eipc_ack_from_json(const char *json, size_t json_len,
                                 eipc_ack_t *ack) {
    (void)json_len;

    if (!json || !ack)
        return EIPC_ERR_INVALID;

    memset(ack, 0, sizeof(*ack));

    json_find_string(json, "request_id", ack->request_id, sizeof(ack->request_id));
    json_find_string(json, "status", ack->status, sizeof(ack->status));
    json_find_string(json, "error", ack->error, sizeof(ack->error));

    return EIPC_OK;
}

/* ---------- Tool Request JSON ---------- */

eipc_status_t eipc_tool_request_to_json(const eipc_tool_request_t *req,
                                        char *buf, size_t buf_size,
                                        size_t *out_len) {
    int n;
    char args_json[2048];
    size_t ai = 0;
    int i;

    if (!req || !buf || !out_len)
        return EIPC_ERR_INVALID;

    args_json[ai++] = '{';
    for (i = 0; i < req->arg_count && i < EIPC_MAX_TOOL_ARGS; i++) {
        char ek[128], ev[256];
        int written;

        if (i > 0)
            args_json[ai++] = ',';

        json_escape(req->args[i].key, ek, sizeof(ek));
        json_escape(req->args[i].value, ev, sizeof(ev));

        written = snprintf(args_json + ai, sizeof(args_json) - ai,
                          "\"%s\":\"%s\"", ek, ev);
        if (written < 0 || (size_t)written >= sizeof(args_json) - ai)
            return EIPC_ERR_BUFFER;
        ai += (size_t)written;
    }
    args_json[ai++] = '}';
    args_json[ai] = '\0';

    n = snprintf(buf, buf_size,
        "{\"tool\":\"%s\","
        "\"args\":%s,"
        "\"permission\":\"%s\"}",
        req->tool,
        args_json,
        req->permission);

    if (n < 0 || (size_t)n >= buf_size)
        return EIPC_ERR_BUFFER;

    *out_len = (size_t)n;
    return EIPC_OK;
}

/* ---------- Heartbeat JSON ---------- */

eipc_status_t eipc_heartbeat_to_json(const eipc_heartbeat_t *hb,
                                     char *buf, size_t buf_size,
                                     size_t *out_len) {
    int n;

    if (!hb || !buf || !out_len)
        return EIPC_ERR_INVALID;

    n = snprintf(buf, buf_size,
        "{\"service\":\"%s\","
        "\"status\":\"%s\"}",
        hb->service,
        hb->status);

    if (n < 0 || (size_t)n >= buf_size)
        return EIPC_ERR_BUFFER;

    *out_len = (size_t)n;
    return EIPC_OK;
}
