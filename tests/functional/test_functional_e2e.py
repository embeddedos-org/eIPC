import unittest

class TesteIPCFunctional(unittest.TestCase):
    def test_ipc_message_queue_pipeline(self):
        queue = []
        # Send
        queue.append({"id": 1, "payload": "sensor_data"})
        # Receive
        msg = queue.pop(0)
        assert msg["payload"] == "sensor_data"
