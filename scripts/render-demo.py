#!/usr/bin/env python3
"""Render a native-hook demo from real commands and fake vault data.

Requires Go, Bash, Python 3.11+ and Pillow. Never contacts a real provider.
Run from anywhere: python3 scripts/render-demo.py
"""
from pathlib import Path
import re
import subprocess
import tempfile

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parent.parent
OUTPUT = ROOT / 'assets/workflow.gif'
MARKER = '__BWENV_DEMO_FRAME__'


def render():
    with tempfile.TemporaryDirectory(prefix='bwenv-doc-demo-') as temporary:
        directory = Path(temporary)
        binaries = directory / 'bin'
        binaries.mkdir()
        for target, package in [('bwenv', './cmd/bwenv'), ('bw', './tests/fixtures/fakecli')]:
            subprocess.run(['go', 'build', '-o', str(binaries / target), package], cwd=ROOT, check=True)
        home = directory / 'home'
        config = home / '.config/bwenv'
        config.mkdir(parents=True)
        (config / 'config.json').write_text('{"activation_mode":"shell","show_emoji":false,"show_export_summary":true,"show_direnv_output":false}')
        project = directory / 'project'
        project.mkdir()
        (project / '.bwenv.toml').write_text('version = 1\nprovider = "bitwarden"\n[project]\nfolder_id = "folder-1"\nfolder_name = "Fixture"\nitems = ["item-1"]\n[activation]\nmode = "shell"\n')
        # Use the project's actual wrapper, including its provider-lock handling.
        source = (ROOT / 'internal/shell/wrapper.go').read_text()
        lock = re.search(r'const lockWrapperPOSIX = `(.+?)`', source, re.S)
        wrapper = re.search(r'const wrapperBashZsh = `(.+?)` \+ lockWrapperPOSIX \+ `(.+?)`', source, re.S)
        if not lock or not wrapper:
            raise RuntimeError('Wrapper format changed; update the demo extraction')
        wrapper_file = directory / 'wrapper.sh'
        wrapper_file.write_text(wrapper[1] + lock[1] + wrapper[2])
        script = '''set -e
source "$DEMO_WRAPPER"
eval "$(command bwenv hook bash)"
printf '__BWENV_DEMO_FRAME__\\n$ cd project\\n'
cd "$DEMO_PROJECT"
_bwenv_prompt_hook || true
printf '__BWENV_DEMO_FRAME__\\n$ bwenv login\\n'
bwenv login
_bwenv_prompt_hook
printf '__BWENV_DEMO_FRAME__\\n$ test -n "${API_KEY:-}" && echo "API_KEY is loaded"\\n'
test "$API_KEY" = fake-secret-value
test -n "${API_KEY:-}"
echo 'API_KEY is loaded'
printf '__BWENV_DEMO_FRAME__\\n$ cd ..\\n'
cd ..
_bwenv_prompt_hook
printf '__BWENV_DEMO_FRAME__\\n$ test -z "${API_KEY+x}" && echo "API_KEY is cleared"\\n'
test -z "${API_KEY+x}"
echo 'API_KEY is cleared'
'''
        # Do not inherit real sessions, provider configuration, RC files or tokens.
        environment = {
            'PATH': str(binaries) + ':/usr/bin:/bin',
            'HOME': str(home), 'XDG_CONFIG_HOME': str(home / '.config'),
            'SHELL': '/bin/bash', 'TERM': 'dumb', 'NO_COLOR': '1',
            'DEMO_WRAPPER': str(wrapper_file), 'DEMO_PROJECT': str(project),
        }
        result = subprocess.run(['/bin/bash', '--noprofile', '--norc'], input=script,
                                text=True, env=environment, cwd=directory,
                                stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                timeout=30, check=True)
        transcript = result.stdout
        assert 'fake-secret-value' not in transcript and 'fake-session' not in transcript
        assert 'API_KEY is loaded' in transcript and 'API_KEY is cleared' in transcript
        sections = [section.strip() for section in transcript.split(MARKER) if section.strip()]
        assert len(sections) == 5, transcript

    fonts = ['/System/Library/Fonts/Menlo.ttc', '/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf']
    font_path = next((path for path in fonts if Path(path).exists()), None)
    font = ImageFont.truetype(font_path, 19) if font_path else ImageFont.load_default(size=19)
    small = ImageFont.truetype(font_path, 14) if font_path else ImageFont.load_default(size=14)
    frames = []
    lines = []
    for section in sections:
        for line in section.splitlines():
            # Fit a narrow terminal without relying on the renderer's wrapping.
            while len(line) > 73:
                lines.append(line[:73])
                line = '  ' + line[73:]
            lines.append(line)
        lines.append('')
        canvas = Image.new('RGB', (960, 610), '#0d1117')
        draw = ImageDraw.Draw(canvas)
        draw.rounded_rectangle((12, 12, 948, 598), radius=15, fill='#161b22', outline='#30363d', width=2)
        for x, color in [(35, '#ff5f57'), (59, '#febc2e'), (83, '#28c840')]:
            draw.ellipse((x, 30, x + 12, 42), fill=color)
        draw.text((115, 26), 'bwenv / native shell', font=font, fill='#c9d1d9')
        draw.text((28, 70), 'REAL COMMANDS · FAKE VAULT · NO SECRET VALUES SHOWN', font=small, fill='#8b949e')
        for number, line in enumerate(lines[-15:]):
            color = '#79c0ff' if line.startswith('$') else '#a5d6a7'
            if 'Session locked' in line:
                color = '#e3b341'
            draw.text((28, 115 + number * 28), line, font=font, fill=color)
        draw.text((28, 560), 'Enter → log in → use variables → leave → restore environment', font=small, fill='#8b949e')
        frames.append(canvas)
    frames[0].save(OUTPUT, save_all=True, append_images=frames[1:], duration=[2200, 2200, 2000, 2000, 3200], loop=0, optimize=True)
    print(f'Wrote {OUTPUT} ({OUTPUT.stat().st_size:,} bytes)')
    print(transcript.replace(MARKER + '\n', ''))


if __name__ == '__main__':
    render()
