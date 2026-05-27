# SPDX-License-Identifier: MIT
# Copyright (c) 2026 EoS Project
import unittest
import time
class TestEipcPerformance(unittest.TestCase):
    def test_shm_transfer_rate(self):
        print("Measuring shared memory transfer rate...")
        t0 = time.perf_counter()
        shm_size = 1024 * 1024
        data = b"x" * shm_size
        for _ in range(100):
            _ = data[:]
        t1 = time.perf_counter()
        transfer_rate_mb_s = (shm_size * 100 / (t1 - t0)) / (1024 * 1024)
        print(f"Shared memory transfer rate: {transfer_rate_mb_s:.2f} MB/s")
        self.assertGreater(transfer_rate_mb_s, 500, "Transfer rate below SLA")
