"""Render the editable icon design to PNG and Windows ICO. Requires Pillow.

Run from any directory: python assets/generate_icon.py
The drawing matches icon.svg and is supersampled separately at every ICO size.
"""

from io import BytesIO
from pathlib import Path
import math
import struct

from PIL import Image, ImageChops, ImageDraw, ImageOps


ASSETS = Path(__file__).resolve().parent
SIZES = (16, 20, 24, 32, 40, 48, 64, 128, 256)


def render(size):
    scale = max(4, math.ceil(1024 / size))
    edge = size * scale
    unit = edge / 512

    def xy(box):
        return tuple(round(value * unit) for value in box)

    mask = Image.new("L", (edge, edge))
    ImageDraw.Draw(mask).rounded_rectangle(xy((16, 16, 496, 496)), radius=round(108 * unit), fill=255)
    ramp = Image.linear_gradient("L").resize((edge, edge), Image.Resampling.BILINEAR)
    diagonal = ImageChops.add(ramp, ramp.transpose(Image.Transpose.TRANSPOSE), scale=2)
    diagonal = diagonal.point([round(min(255, max(0, (n - 8) * 255 / 239))) for n in range(256)])
    base = ImageOps.colorize(diagonal, (28, 112, 90), (16, 75, 63)).convert("RGBA")
    gradient = base.copy()
    gradient.putalpha(mask)
    draw = ImageDraw.Draw(gradient)
    # An inset / outset pair gives the same centered stroke as the SVG.
    draw.rounded_rectangle(xy((95, 95, 333, 333)), radius=round(59 * unit), fill="#8fd4b9")
    hole = Image.new("L", (edge, edge))
    ImageDraw.Draw(hole).rounded_rectangle(xy((117, 117, 311, 311)), radius=round(37 * unit), fill=255)
    # Restore the original tile gradient inside the outline.
    gradient.paste(base, (0, 0), hole)
    shadow = Image.new("RGBA", (edge, edge))
    ImageDraw.Draw(shadow).rounded_rectangle(xy((192, 202, 408, 418)), radius=round(48 * unit), fill=(13, 63, 53, 89))
    gradient = Image.alpha_composite(gradient, shadow)
    draw = ImageDraw.Draw(gradient)
    draw.rounded_rectangle(xy((192, 192, 408, 408)), radius=round(48 * unit), fill="#f5faf6")
    stroke = round(22 * unit)
    for points in (((248, 300), (352, 300)), ((320, 268), (352, 300), (320, 332))):
        draw.line([xy(point) for point in points], fill="#176c58", width=stroke, joint="curve")
        for x, y in points:
            draw.ellipse(xy((x - 11, y - 11, x + 11, y + 11)), fill="#176c58")
    return gradient.resize((size, size), Image.Resampling.LANCZOS)


def main():
    image = render(512)
    image.save(ASSETS / "icon.png")
    payloads = []
    for size in SIZES:
        stream = BytesIO()
        render(size).save(stream, format="PNG")
        payloads.append(stream.getvalue())
    offset = 6 + 16 * len(SIZES)
    entries = []
    for size, payload in zip(SIZES, payloads):
        entries.append(struct.pack("<BBBBHHII", size % 256, size % 256, 0, 0, 1, 32, len(payload), offset))
        offset += len(payload)
    (ASSETS / "icon.ico").write_bytes(struct.pack("<HHH", 0, 1, len(SIZES)) + b"".join(entries) + b"".join(payloads))
    print("Created assets/icon.png and assets/icon.ico (16–256px)")


if __name__ == "__main__":
    main()
