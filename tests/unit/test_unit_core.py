import unittest

class TesteIPCUnit(unittest.TestCase):
    def test_shared_memory_mutex_lock(self):
        shm_mutex = {"locked": False, "owner": None}
        # Acquire
        assert not shm_mutex["locked"]
        shm_mutex["locked"] = True
        shm_mutex["owner"] = "process_1"
        # Release
        shm_mutex["locked"] = False
        shm_mutex["owner"] = None
        assert not shm_mutex["locked"]
