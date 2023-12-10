# import threading
# import time


# # Custom Thread class with a stop method
# class StoppableThread(threading.Thread):
#     def __init__(self):
#         super().__init__()
#         self._stop_event = threading.Event()

#     def stop(self):
#         self._stop_event.set()

#     def stopped(self):
#         return self._stop_event.is_set()

#     def run(self):
#         while not self.stopped():
#             print("Thread is running...")
#             time.sleep(1)
#         print("Thread stopped.")