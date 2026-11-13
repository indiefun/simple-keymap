"""Create the website's 1200 x 630 share image. Requires Pillow.

Run on Windows with Microsoft YaHei installed, or pass a CJK font via --font.
"""

import argparse
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parent.parent


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--font", default="C:/Windows/Fonts/msyh.ttc")
    parser.add_argument("--bold-font", default="C:/Windows/Fonts/msyhbd.ttc")
    args = parser.parse_args()
    scale = 2
    image = Image.new("RGB", (1200 * scale, 630 * scale), "#f7f8f5")
    draw = ImageDraw.Draw(image)
    ink, green, muted, line = "#1f3029", "#176c58", "#68766f", "#dde5dd"

    def font(size, bold=False):
        return ImageFont.truetype(args.bold_font if bold else args.font, size * scale)

    def text(point, value, size, color=ink, bold=False):
        draw.text(tuple(n * scale for n in point), value, font=font(size, bold), fill=color)

    def box(bounds, radius, fill, outline=None):
        draw.rounded_rectangle(tuple(n * scale for n in bounds), radius=radius * scale,
                               fill=fill, outline=outline, width=scale * 2)

    icon = Image.open(ROOT / "assets/icon.png").convert("RGBA")
    icon = icon.resize((72 * scale, 72 * scale), Image.Resampling.LANCZOS)
    image.paste(icon, (62 * scale, 54 * scale), icon)
    text((152, 52), "轻映", 34, bold=True)
    text((154, 99), "Simple Keymap", 16, muted)

    text((64, 181), "让按键，", 60, bold=True)
    text((64, 264), "顺你的手。", 60, green, bold=True)
    text((68, 380), "Windows 键盘与鼠标映射工具", 25, muted)
    text((68, 430), "免费使用  ·  便携免安装", 21, green)
    draw.line((68 * scale, 517 * scale, 1132 * scale, 517 * scale), fill=line, width=scale)
    text((68, 551), "qingying.luming.host", 20, muted)
    text((905, 551), "按下与抬起，完整跟随", 18, muted)

    box((690, 183, 1132, 455), 24, "#e9f1e9")
    box((718, 215, 891, 422), 18, "#ffffff", line)
    box((931, 215, 1104, 422), 18, "#ffffff", line)
    text((741, 241), "按下这个键", 16, muted)
    text((954, 241), "变为这个键", 16, muted)
    text((741, 298), "鼠标侧键 1", 22, bold=True)
    text((954, 298), "鼠标中键", 22, green, bold=True)
    text((741, 355), "后退按钮", 16, muted)
    text((954, 355), "按下滚轮", 16, muted)
    draw.line((900 * scale, 321 * scale, 922 * scale, 321 * scale), fill=green, width=3 * scale)
    draw.line((915 * scale, 314 * scale, 922 * scale, 321 * scale, 915 * scale, 328 * scale),
              fill=green, width=3 * scale)

    output = ROOT / "site/assets/social-preview.png"
    image.resize((1200, 630), Image.Resampling.LANCZOS).save(output, optimize=True)
    print(f"Created {output.relative_to(ROOT)}")


if __name__ == "__main__":
    main()
