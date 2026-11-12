// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

#include "eipc.h"
#include <stdio.h>
#include <string.h>

#define ASSERT(c,m) do{if(!(c)){printf("  FAIL: %s\n",m);return 1;}}while(0)
#define RUN(fn) do{printf("  %s ... ",#fn);if(fn()==0){printf("PASS\n");p++;}else f++;t++;}while(0)

static int test_chat_request_json_roundtrip(void) {
    eipc_chat_request_t req;
    memset(&req, 0, sizeof(req));
    strncpy(req.session_id, "sess-123", sizeof(req.session_id) - 1);
    strncpy(req.user_prompt, "hello world", sizeof(req.user_prompt) - 1);
    strncpy(req.model, "llama3", sizeof(req.model) - 1);
    req.max_tokens = 512;

    char json[4096];
    size_t written = 0;
    ASSERT(eipc_chat_request_to_json(&req, json, sizeof(json), &written) == EIPC_OK,
           "serialize");
    ASSERT(written == strlen(json), "written matches the serialized length");

    eipc_chat_request_t parsed;
    ASSERT(eipc_chat_request_from_json(json, strlen(json), &parsed) == EIPC_OK, "deserialize");

    ASSERT(strcmp(parsed.session_id, "sess-123") == 0, "session_id");
    ASSERT(strcmp(parsed.user_prompt, "hello world") == 0, "user_prompt");
    ASSERT(strcmp(parsed.model, "llama3") == 0, "model");
    ASSERT(parsed.max_tokens == 512, "max_tokens");
    return 0;
}

static int test_chat_response_json_roundtrip(void) {
    eipc_chat_response_t resp;
    memset(&resp, 0, sizeof(resp));
    strncpy(resp.session_id, "sess-456", sizeof(resp.session_id) - 1);
    strncpy(resp.response, "AI says hi", sizeof(resp.response) - 1);
    strncpy(resp.model, "gpt4", sizeof(resp.model) - 1);
    resp.tokens_used = 42;

    char json[4096];
    size_t written = 0;
    ASSERT(eipc_chat_response_to_json(&resp, json, sizeof(json), &written) == EIPC_OK,
           "serialize");
    ASSERT(written == strlen(json), "written matches the serialized length");

    eipc_chat_response_t parsed;
    ASSERT(eipc_chat_response_from_json(json, strlen(json), &parsed) == EIPC_OK, "deserialize");

    ASSERT(strcmp(parsed.session_id, "sess-456") == 0, "session_id");
    ASSERT(strcmp(parsed.response, "AI says hi") == 0, "response");
    ASSERT(strcmp(parsed.model, "gpt4") == 0, "model");
    ASSERT(parsed.tokens_used == 42, "tokens_used");
    return 0;
}

static int test_chat_null_args(void) {
    char buf[256];
    size_t written = 0;
    eipc_chat_request_t req;
    eipc_chat_response_t resp;

    memset(&req, 0, sizeof(req));
    memset(&resp, 0, sizeof(resp));

    ASSERT(eipc_chat_request_to_json(NULL, buf, sizeof(buf), &written) == EIPC_ERR_INVALID,
           "null req");
    ASSERT(eipc_chat_response_to_json(NULL, buf, sizeof(buf), &written) == EIPC_ERR_INVALID,
           "null resp");
    ASSERT(eipc_chat_request_from_json(NULL, 0, NULL) == EIPC_ERR_INVALID, "null json");

    /* The out-parameter is mandatory: the implementation dereferences it. */
    ASSERT(eipc_chat_request_to_json(&req, buf, sizeof(buf), NULL) == EIPC_ERR_INVALID,
           "null written rejected for request");
    ASSERT(eipc_chat_response_to_json(&resp, buf, sizeof(buf), NULL) == EIPC_ERR_INVALID,
           "null written rejected for response");
    ASSERT(eipc_chat_request_to_json(&req, NULL, sizeof(buf), &written) == EIPC_ERR_INVALID,
           "null buffer rejected");

    /* A buffer too small must be reported, not truncated silently. */
    ASSERT(eipc_chat_request_to_json(&req, buf, 4, &written) == EIPC_ERR_FRAME_TOO_LARGE,
           "undersized buffer reported");
    return 0;
}

/* The *_from_json decoders must undo JSON string escapes. They used to copy
 * the bytes between the quotes verbatim, so a value written by
 * eipc_ack_to_json() or eipc_intent_to_json() came back with its
 * backslashes, and text from Go's encoding/json, which writes <, > and &
 * as \u003c, \u003e and \u0026, came back as those six-character codes. */
static int test_json_string_escapes_decoded(void) {
    eipc_ack_t ack, ack_parsed;
    eipc_intent_t intent, intent_parsed;
    eipc_chat_response_t resp;
    char json[1024];
    size_t written = 0;

    memset(&ack, 0, sizeof(ack));
    strncpy(ack.request_id, "req-1", sizeof(ack.request_id) - 1);
    strncpy(ack.status, "failed", sizeof(ack.status) - 1);
    strncpy(ack.error, "open \"C:\\tmp\"\nfailed\tagain", sizeof(ack.error) - 1);
    ASSERT(eipc_ack_to_json(&ack, json, sizeof(json), &written) == EIPC_OK, "ack serialize");
    ASSERT(eipc_ack_from_json(json, strlen(json), &ack_parsed) == EIPC_OK, "ack deserialize");
    ASSERT(strcmp(ack_parsed.error, ack.error) == 0, "ack error round-trips");

    memset(&intent, 0, sizeof(intent));
    strncpy(intent.intent, "say \"hi\"", sizeof(intent.intent) - 1);
    strncpy(intent.session_id, "sess-9", sizeof(intent.session_id) - 1);
    intent.confidence = 0.5f;
    ASSERT(eipc_intent_to_json(&intent, json, sizeof(json), &written) == EIPC_OK,
           "intent serialize");
    ASSERT(eipc_intent_from_json(json, strlen(json), &intent_parsed) == EIPC_OK,
           "intent deserialize");
    ASSERT(strcmp(intent_parsed.intent, "say \"hi\"") == 0, "intent round-trips");

    /* Go's json.Marshal writes < and & as \u003c and \u0026; \/, a
     * \u00e9 and a surrogate pair are valid JSON from any encoder. */
    const char *from_go =
        "{\"session_id\":\"s-1\","
        "\"response\":\"a \\u003c b \\u0026 c \\u00e9 \\/ \\ud83d\\ude00\","
        "\"model\":\"m\",\"tokens_used\":3}";
    ASSERT(eipc_chat_response_from_json(from_go, strlen(from_go), &resp) == EIPC_OK,
           "go response deserialize");
    ASSERT(strcmp(resp.response, "a < b & c \xc3\xa9 / \xf0\x9f\x98\x80") == 0,
           "\\u escapes decoded to UTF-8");
    ASSERT(resp.tokens_used == 3, "tokens_used after escaped string");

    /* A NUL would cut the C string short; \u0000 decodes to U+FFFD. */
    const char *with_nul = "{\"intent\":\"a\\u0000b\",\"confidence\":1}";
    ASSERT(eipc_intent_from_json(with_nul, strlen(with_nul), &intent_parsed) == EIPC_OK,
           "intent with \\u0000 deserialize");
    ASSERT(strcmp(intent_parsed.intent, "a\xEF\xBF\xBD" "b") == 0, "\\u0000 decoded to U+FFFD");
    return 0;
}

/* A key is a quoted name followed by a colon. The lookup used strstr() on
 * "error" alone, so it found the status value "error" first and returned
 * an empty error message for every ack that reported a failure. */
static int test_json_key_not_matched_in_value(void) {
    eipc_ack_t ack, parsed;
    eipc_header_t hdr;
    char json[512];
    size_t written = 0;

    memset(&ack, 0, sizeof(ack));
    strncpy(ack.request_id, "req-2", sizeof(ack.request_id) - 1);
    strncpy(ack.status, "error", sizeof(ack.status) - 1);
    strncpy(ack.error, "disk full", sizeof(ack.error) - 1);
    ASSERT(eipc_ack_to_json(&ack, json, sizeof(json), &written) == EIPC_OK, "ack serialize");
    ASSERT(eipc_ack_from_json(json, strlen(json), &parsed) == EIPC_OK, "ack deserialize");
    ASSERT(strcmp(parsed.status, "error") == 0, "status");
    ASSERT(strcmp(parsed.error, "disk full") == 0, "error message after an \"error\" status");

    const char *h = "{\"service_id\":\"priority\",\"priority\":2}";
    ASSERT(eipc_header_from_json(h, strlen(h), &hdr) == EIPC_OK, "header deserialize");
    ASSERT(hdr.priority == 2, "number after a string value equal to its key");
    return 0;
}

int main(void) {
    int p=0,f=0,t=0;
    printf("=== EIPC Chat JSON Tests ===\n\n");
    RUN(test_chat_request_json_roundtrip);
    RUN(test_chat_response_json_roundtrip);
    RUN(test_chat_null_args);
    RUN(test_json_key_not_matched_in_value);
    RUN(test_json_string_escapes_decoded);
    printf("\n%d/%d passed, %d failed\n",p,t,f);
    return f>0?1:0;
}
