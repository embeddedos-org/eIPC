"""Generate per-segment narration using edge-tts (US English neural voice)."""
import asyncio
import json
import edge_tts
from mutagen.mp3 import MP3

# en-US-GuyNeural = neutral male US voice (Silicon Valley style)
VOICE = "en-US-GuyNeural"
RATE = "+0%"  # natural pace

SEGMENTS = [
    {"id": "intro", "text": "Introducing eIPC. Embedded inter-process communication done right."},
    {"id": "f1", "text": "Feature one. Shared Memory Channels. Lock-free ring buffers deliver microsecond latency between processes."},
    {"id": "f2", "text": "Feature two. Zero-Copy Message Passing. Memory-mapped transfers eliminate serialization and copy overhead."},
    {"id": "f3", "text": "Feature three. Cross-Platform Portability. Runs on Linux, FreeRTOS, Zephyr, and bare metal with the same API."},
    {"id": "arch", "text": "Under the hood, eIPC is built with C, POSIX, and FreeRTOS. The architecture flows from Channel, to Ring Buffer, to Serializer, to Transport, to Discovery."},
    {"id": "cta", "text": "eIPC. Open source and lightning fast. Visit github dot com slash embeddedos-org slash eIPC."}
]


async def generate():
    durations = {}
    audio_files = []

    for seg in SEGMENTS:
        filename = f"seg_{seg['id']}.mp3"
        communicate = edge_tts.Communicate(seg["text"], VOICE, rate=RATE)
        await communicate.save(filename)
        dur = MP3(filename).info.length
        durations[seg["id"]] = round(dur + 0.5, 1)
        audio_files.append(filename)
        print(f"  {seg['id']}: {dur:.1f}s -> padded {durations[seg['id']]}s")

    with open("durations.json", "w") as f:
        json.dump(durations, f, indent=2)

    # Concatenate
    import subprocess
    with open("concat_list.txt", "w") as f:
        for af in audio_files:
            f.write(f"file '{af}'\n")

    subprocess.run([
        "ffmpeg", "-y", "-f", "concat", "-safe", "0",
        "-i", "concat_list.txt", "-c", "copy", "narration.mp3"
    ], check=True)

    total = sum(durations.values())
    print(f"\nVoice: {VOICE}")
    print(f"Total narration: {total:.1f}s")
    print(f"Durations: {json.dumps(durations)}")


asyncio.run(generate())
