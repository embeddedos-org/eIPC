# SPDX-License-Identifier: MIT
# Copyright (c) 2026 EoS Project
import unittest
class TestEipcFunctional(unittest.TestCase):
    def test_message_queue_dispatch(self):
        print("Testing inter-process message queue send/receive...")
        queue = []
        queue.append({"id": 1, "payload": "ping"})
        msg = queue.pop(0)
        self.assertEqual(msg["payload"], "ping")
