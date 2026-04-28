"""Generate narration audio using Google Text-to-Speech."""
from gtts import gTTS

NARRATION = (
    "Introducing eIPC. Embedded inter-process communication done right. Feature one: Shared memory channels deliver microsecond latency between processes. Feature two: Zero-copy message passing eliminates data duplication overhead. Feature three: Cross-platform portability runs on Linux, RTOS, and bare metal. eIPC. Open source and lightning fast. Visit github dot com slash embeddedos-org slash eIPC."
)

tts = gTTS(text=NARRATION, lang="en", slow=False)
tts.save("narration.mp3")
print(f"Generated narration.mp3 ({len(NARRATION)} chars)")
