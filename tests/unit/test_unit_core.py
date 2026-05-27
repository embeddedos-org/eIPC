import unittest
class TestEIPCUnit(unittest.TestCase):
    def test_message_queue_dispatch(self):
        queue = []
        queue.append("msg1")
        msg = queue.pop(0)
        self.assertEqual(msg, "msg1")
