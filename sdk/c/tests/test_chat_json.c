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
    ASSERT(eipc_chat_request_to_json(&req, json, sizeof(json)) == EIPC_OK, "serialize");

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
    ASSERT(eipc_chat_response_to_json(&resp, json, sizeof(json)) == EIPC_OK, "serialize");

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
    ASSERT(eipc_chat_request_to_json(NULL, buf, sizeof(buf)) == EIPC_ERR_INVALID, "null req");
    ASSERT(eipc_chat_response_to_json(NULL, buf, sizeof(buf)) == EIPC_ERR_INVALID, "null resp");
    ASSERT(eipc_chat_request_from_json(NULL, 0, NULL) == EIPC_ERR_INVALID, "null json");
    return 0;
}

int main(void) {
    int p=0,f=0,t=0;
    printf("=== EIPC Chat JSON Tests ===\n\n");
    RUN(test_chat_request_json_roundtrip);
    RUN(test_chat_response_json_roundtrip);
    RUN(test_chat_null_args);
    printf("\n%d/%d passed, %d failed\n",p,t,f);
    return f>0?1:0;
}
