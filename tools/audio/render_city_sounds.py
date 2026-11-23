"""Synthesize original river ambience and city interaction sounds without sampled audio."""
from pathlib import Path
import subprocess

import numpy as np

RATE = 24000
OUT = Path(__file__).resolve().parents[2] / "client/assets/sounds"


def render(name, signal):
    peak = np.max(np.abs(signal))
    data = np.asarray(signal * (0.82 / peak), dtype="<f4")
    subprocess.run(["ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "f32le",
                    "-ar", str(RATE), "-ac", "2", "-i", "-", "-c:a", "libvorbis", "-q:a",
                    "4", str(OUT / (name + ".ogg"))], input=data.tobytes(), check=True)


def strikes():
    t = np.arange(RATE) / RATE
    sound = np.zeros(t.shape)
    rng = np.random.default_rng(111)
    for start, freq, volume in [(0, 320, 1), (0.13, 270, 0.55)]:
        age = np.maximum(0, t - start)
        sound += (t >= start) * volume * (1-np.exp(-580*age)) * (
            np.sin(2*np.pi*freq*age)*np.exp(-32*age)
            + 0.53*np.sin(2*np.pi*freq*2.76*age)*np.exp(-41*age)
            + 0.2*rng.normal(size=t.size)*np.exp(-54*age))
    return np.column_stack((sound, 0.9*sound))


def river():
    rng = np.random.default_rng(470)
    t = np.arange(18*RATE) / RATE
    mix = np.zeros((t.size, 2))
    for channel in range(2):
        n = rng.normal(size=t.size)
        shore = np.convolve(n, np.hanning(28)/14, mode="same")
        swell = (0.46 + 0.14*np.sin(2*np.pi*t/3 + channel/7)
                 + 0.17*np.sin(2*np.pi*t/4.5 + 2)) ** 2
        mix[:, channel] = shore*swell
    for start, pitch, length, pan in [(2.1,2350,.19,.25),(2.5,2170,.24,.25),
                                       (9.0,1770,.11,.72),(9.24,2250,.14,.72),(15.2,2570,.23,.42)]:
        v = np.arange(round(length*RATE))/RATE
        hz = pitch + 510*np.sin(np.pi*v/length)
        chirp = np.sin(2*np.pi*np.cumsum(hz)/RATE)*np.sin(np.pi*v/length)**2*.026
        begin = round(start*RATE)
        mix[begin:begin+v.size] += chirp[:,None]*np.array([1-pan,pan])
    envelope = np.minimum(1,np.minimum(t,18-t)*12)
    return mix*envelope[:,None]


if __name__ == "__main__":
    OUT.mkdir(parents=True, exist_ok=True)
    render("riverbank", river())
    render("place", strikes())
