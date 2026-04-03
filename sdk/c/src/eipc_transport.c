// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project
// ISO/IEC 25000 | ISO/IEC/IEEE 15288:2023

/*
 * EIPC Transport Layer — cross-platform TCP socket implementation.
 * Provides TCP socket framing (4-byte big-endian length-prefixed).
 * Windows: Winsock2, POSIX: sys/socket.h
 */

#include "eipc.h"
#include <string.h>
#include <stdio.h>
#include <stdlib.h>

#ifdef _WIN32
  #include <winsock2.h>
  #include <ws2tcpip.h>
  #pragma comment(lib, "ws2_32.lib")
  static int wsa_initialized = 0;
  static void ensure_wsa(void) {
      if (!wsa_initialized) {
          WSADATA wsa;
          WSAStartup(MAKEWORD(2, 2), &wsa);
          wsa_initialized = 1;
      }
  }
  #define CLOSE_SOCK closesocket
  #define SOCK_ERR   INVALID_SOCKET
#else
  #include <sys/socket.h>
  #include <netinet/in.h>
  #include <arpa/inet.h>
  #include <unistd.h>
  #include <errno.h>
  #define CLOSE_SOCK close
  #define SOCK_ERR   (-1)
  static void ensure_wsa(void) {}
#endif

static int parse_address(const char *address, char *host, size_t host_size, uint16_t *port) {
    const char *colon = strrchr(address, ':');
    if (!colon) return -1;

    size_t hlen = (size_t)(colon - address);
    if (hlen >= host_size) hlen = host_size - 1;
    memcpy(host, address, hlen);
    host[hlen] = '\0';

    *port = (uint16_t)atoi(colon + 1);
    return 0;
}

static int send_all(eipc_socket_t sock, const uint8_t *data, size_t len) {
    size_t sent = 0;
    while (sent < len) {
        int n = send(sock, (const char *)(data + sent), (int)(len - sent), 0);
        if (n <= 0) return -1;
        sent += (size_t)n;
    }
    return 0;
}

static int recv_all(eipc_socket_t sock, uint8_t *data, size_t len) {
    size_t received = 0;
    while (received < len) {
        int n = recv(sock, (char *)(data + received), (int)(len - received), 0);
        if (n <= 0) return -1;
        received += (size_t)n;
    }
    return 0;
}

eipc_status_t eipc_transport_connect(eipc_socket_t *sock, const char *address) {
    char host[256];
    uint16_t port;
    struct sockaddr_in addr;

    if (!sock || !address)
        return EIPC_ERR_INVALID;

    ensure_wsa();

    if (parse_address(address, host, sizeof(host), &port) < 0)
        return EIPC_ERR_INVALID;

    *sock = socket(AF_INET, SOCK_STREAM, 0);
    if (*sock == EIPC_INVALID_SOCKET)
        return EIPC_ERR_CONNECT;

    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons(port);
    if (inet_pton(AF_INET, host, &addr.sin_addr) <= 0) {
        CLOSE_SOCK(*sock);
        *sock = EIPC_INVALID_SOCKET;
        return EIPC_ERR_CONNECT;
    }

    if (connect(*sock, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
        CLOSE_SOCK(*sock);
        *sock = EIPC_INVALID_SOCKET;
        return EIPC_ERR_CONNECT;
    }

    return EIPC_OK;
}

eipc_status_t eipc_transport_listen(eipc_socket_t *sock, const char *address) {
    char host[256];
    uint16_t port;
    struct sockaddr_in addr;
    int optval = 1;

    if (!sock || !address)
        return EIPC_ERR_INVALID;

    ensure_wsa();

    if (parse_address(address, host, sizeof(host), &port) < 0)
        return EIPC_ERR_INVALID;

    *sock = socket(AF_INET, SOCK_STREAM, 0);
    if (*sock == EIPC_INVALID_SOCKET)
        return EIPC_ERR_CONNECT;

    setsockopt(*sock, SOL_SOCKET, SO_REUSEADDR, (const char *)&optval, sizeof(optval));

    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons(port);
    if (strcmp(host, "0.0.0.0") == 0 || host[0] == '\0')
        addr.sin_addr.s_addr = INADDR_ANY;
    else
        inet_pton(AF_INET, host, &addr.sin_addr);

    if (bind(*sock, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
        CLOSE_SOCK(*sock);
        *sock = EIPC_INVALID_SOCKET;
        return EIPC_ERR_CONNECT;
    }

    if (listen(*sock, 8) < 0) {
        CLOSE_SOCK(*sock);
        *sock = EIPC_INVALID_SOCKET;
        return EIPC_ERR_CONNECT;
    }

    return EIPC_OK;
}

eipc_status_t eipc_transport_accept(eipc_socket_t listen_sock,
                                    eipc_socket_t *client_sock,
                                    char *remote_addr,
                                    size_t remote_addr_size) {
    struct sockaddr_in client_addr;
    int addr_len = (int)sizeof(client_addr);

    if (!client_sock)
        return EIPC_ERR_INVALID;

    memset(&client_addr, 0, sizeof(client_addr));
    *client_sock = accept(listen_sock, (struct sockaddr *)&client_addr, (void *)&addr_len);
    if (*client_sock == EIPC_INVALID_SOCKET)
        return EIPC_ERR_CONNECT;

    if (remote_addr && remote_addr_size > 0) {
        char ip[64];
        inet_ntop(AF_INET, &client_addr.sin_addr, ip, sizeof(ip));
        snprintf(remote_addr, remote_addr_size, "%s:%d", ip, ntohs(client_addr.sin_port));
    }

    return EIPC_OK;
}

eipc_status_t eipc_transport_send_frame(eipc_socket_t sock, const eipc_frame_t *frame) {
    uint8_t encoded[EIPC_MAX_FRAME];
    size_t encoded_len = 0;
    eipc_status_t rc;
    uint8_t len_prefix[4];
    size_t total;

    if (!frame)
        return EIPC_ERR_INVALID;

    rc = eipc_frame_encode(frame, encoded, sizeof(encoded), &encoded_len);
    if (rc != EIPC_OK)
        return rc;

    total = (uint32_t)encoded_len;
    len_prefix[0] = (uint8_t)(total >> 24);
    len_prefix[1] = (uint8_t)(total >> 16);
    len_prefix[2] = (uint8_t)(total >> 8);
    len_prefix[3] = (uint8_t)(total);

    if (send_all(sock, len_prefix, 4) < 0)
        return EIPC_ERR_IO;

    if (send_all(sock, encoded, encoded_len) < 0)
        return EIPC_ERR_IO;

    return EIPC_OK;
}

eipc_status_t eipc_transport_recv_frame(eipc_socket_t sock, eipc_frame_t *frame) {
    uint8_t len_prefix[4];
    uint32_t frame_len;
    uint8_t *buf;
    eipc_status_t rc;

    if (!frame)
        return EIPC_ERR_INVALID;

    if (recv_all(sock, len_prefix, 4) < 0)
        return EIPC_ERR_IO;

    frame_len = ((uint32_t)len_prefix[0] << 24) |
                ((uint32_t)len_prefix[1] << 16) |
                ((uint32_t)len_prefix[2] << 8) |
                ((uint32_t)len_prefix[3]);

    if (frame_len > EIPC_MAX_FRAME)
        return EIPC_ERR_FRAME_TOO_LARGE;

    buf = (uint8_t *)malloc(frame_len);
    if (!buf)
        return EIPC_ERR_NOMEM;

    if (recv_all(sock, buf, frame_len) < 0) {
        free(buf);
        return EIPC_ERR_IO;
    }

    rc = eipc_frame_decode(buf, frame_len, frame);
    free(buf);
    return rc;
}

void eipc_transport_close(eipc_socket_t sock) {
    if (sock != EIPC_INVALID_SOCKET)
        CLOSE_SOCK(sock);
}
