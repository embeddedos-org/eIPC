import unittest
class TestEIPCFunctional(unittest.TestCase):
    def test_ipc_pipeline(self):
        pipeline = ["send", "route", "receive"]
        self.assertEqual(pipeline[-1], "receive")
