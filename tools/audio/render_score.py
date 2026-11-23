"""Render Project Nile's original instrumental score from notes, without samples."""
import argparse
from pathlib import Path
import subprocess

import numpy as np


RATE = 24000
# Each tuple is a four-bar reed phrase. Durations are in quarter notes.
PIECES = {
    "first_light": (62, 76, [
        [(0, 2), (3, 1), (5, 1), (7, 3), (5, 1), (3, 2), (2, 1), (0, 1), (-5, 4)],
        [(3, 2), (5, 2), (10, 3), (7, 1), (5, 2), (3, 1), (2, 1), (0, 4)],
        [(7, 3), (10, 1), (12, 2), (10, 2), (7, 2), (3, 2), (5, 4)],
    ]),
    "river_market": (55, 94, [
        [(7, 1), (10, 1), (12, 2), (10, 1), (7, 1), (5, 2), (3, 3), (5, 1), (0, 4)],
        [(3, 1), (5, 1), (7, 2), (5, 1), (7, 1), (10, 2), (12, 3), (10, 1), (7, 4)],
        [(14, 2), (12, 1), (10, 1), (7, 3), (5, 1), (3, 2), (2, 2), (0, 4)],
    ]),
    "evening_barges": (57, 65, [
        [(12, 3), (10, 1), (7, 4), (5, 3), (3, 1), (2, 4)],
        [(5, 2), (7, 2), (10, 3), (7, 1), (3, 4), (0, 4)],
        [(3, 3), (5, 1), (7, 2), (5, 2), (2, 3), (0, 1), (-5, 4)],
    ]),
}


def render(name, destination):
    root, tempo, phrases = PIECES[name]
    beat = 60 / tempo
    mix = np.zeros((round((beat * 104 + 4) * RATE), 2), dtype=np.float32)
    rng = np.random.default_rng(root)

    def note(start, duration, pitch, voice, gain, pan):
        t = np.arange(round(duration * RATE)) / RATE
        hz = 440 * 2 ** ((pitch - 69) / 12)
        phase = 2 * np.pi * hz * t
        if voice == "reed":
            phase += 0.15 * np.sin(2 * np.pi * 4.6 * t)
            wave = np.sin(phase) + 0.17 * np.sin(3 * phase) + 0.035 * rng.normal(size=t.size)
            env = np.minimum(t / 0.14, 1) * np.minimum((duration - t) / 0.27, 1)
        elif voice == "skin":
            wave = np.sin(phase + 7 * (1 - np.exp(-32 * t)))
            env = (1 - np.exp(-260 * t)) * np.exp(-12 * t)
        else:
            wave = np.sin(phase) + 0.45 * np.sin(2 * phase) + 0.25 * np.sin(3.002 * phase)
            env = (1 - np.exp(-180 * t)) * np.exp(-3 * t)
        mono = wave * env * gain * np.minimum((duration - t) / 0.025, 1)
        i = round(start * RATE)
        mix[i:i + t.size, 0] += mono * (1 - pan) ** 0.5
        mix[i:i + t.size, 1] += mono * pan ** 0.5

    for bar in range(26):
        chord = [0, 0, -5, -2, 0, 3, -5, -5][bar % 8]
        for offset, interval in [(0, 0), (1.5, 7), (2.5, 12), (3.5, 7)]:
            note((4 * bar + offset) * beat, 1.8, root - 12 + chord + interval,
                 "harp", 0.20 if offset == 0 else 0.12, 0.3)
        if 2 <= bar < 24:
            for offset in ([0, 1.5, 3] if tempo > 80 else [0, 2.5]):
                note((bar * 4 + offset) * beat, 0.52, 38 if offset == 0 else 48,
                     "skin", 0.18 if offset == 0 else 0.09, 0.58)
    for section, phrase_id in enumerate([0, 1, 0, 2, 1, 0]):
        pos = (4 + 16 * section) * beat
        for pitch, beats in phrases[phrase_id]:
            length = beat * beats
            note(pos + 0.03, length * 0.87, root + pitch, "reed", 0.17, 0.61)
            pos += length
    delay = round(0.195 * RATE)
    mix[delay:] += 0.14 * mix[:-delay, ::-1].copy()
    mix[-4 * RATE:] *= np.linspace(1, 0, 4 * RATE)[:, None]
    pcm = np.asarray(np.clip(mix, -1, 1) * 32767, dtype="<i2")
    subprocess.run(["ffmpeg", "-loglevel", "error", "-y", "-f", "s16le", "-ar",
                    str(RATE), "-ac", "2", "-i", "-", "-c:a", "libvorbis", "-q:a", "3",
                    "-metadata", f"title={name.replace('_', ' ').title()}",
                    str(destination / (name + ".ogg"))], input=pcm.tobytes(), check=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("destination", type=Path)
    args = parser.parse_args()
    args.destination.mkdir(parents=True, exist_ok=True)
    for name in PIECES:
        render(name, args.destination)
